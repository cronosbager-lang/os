package supervisor

import (
	"fmt"
	"sync"
	"time"

	"mixos.dev/init/pkg/config"
	"mixos.dev/init/pkg/ipc"
	"mixos.dev/init/pkg/service"
)

type EventType int

const (
	EventServiceStarted EventType = iota
	EventServiceStopped
	EventServiceFailed
	EventServiceRestarted
)

type Event struct {
	Type     EventType
	Service  string
	PID      int
	ExitCode int
	Error    string
	Time     time.Time
}

type Supervisor struct {
	config   *config.Config
	ipc      *ipc.Server
	services map[string]*service.Service
	order    []string // startup order based on dependencies
	events   chan Event
	mu       sync.RWMutex
	wg       sync.WaitGroup
}

func New(cfg *config.Config, ipcServer *ipc.Server) *Supervisor {
	return &Supervisor{
		config:   cfg,
		ipc:      ipcServer,
		services: make(map[string]*service.Service),
		order:    []string{},
		events:   make(chan Event, 100),
	}
}

func (s *Supervisor) Register(svc *service.Service) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.services[svc.Name] = svc
	s.order = s.resolveOrder()
}

func (s *Supervisor) resolveOrder() []string {
	// Topological sort based on dependencies
	visited := make(map[string]bool)
	order := []string{}
	
	var visit func(name string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true
		
		svc, ok := s.services[name]
		if !ok {
			return
		}
		
		for _, dep := range svc.DependsOn {
			visit(dep)
		}
		order = append(order, name)
	}
	
	for name := range s.services {
		visit(name)
	}
	
	return order
}

func (s *Supervisor) StartAll() error {
	s.mu.RLock()
	order := s.order
	s.mu.RUnlock()

	for _, name := range order {
		if err := s.StartService(name); err != nil {
			return fmt.Errorf("failed to start %s: %w", name, err)
		}
		// Small delay between service starts
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

func (s *Supervisor) StartService(name string) error {
	s.mu.RLock()
	svc, ok := s.services[name]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("service not found: %s", name)
	}

	// Check dependencies
	for _, dep := range svc.DependsOn {
		s.mu.RLock()
		depSvc, ok := s.services[dep]
		s.mu.RUnlock()
		
		if !ok {
			return fmt.Errorf("dependency not found: %s", dep)
		}
		if depSvc.State() != service.StateRunning {
			return fmt.Errorf("dependency not running: %s", dep)
		}
	}

	if err := svc.Start(); err != nil {
		s.events <- Event{
			Type:    EventServiceFailed,
			Service: name,
			Error:   err.Error(),
			Time:    time.Now(),
		}
		return err
	}

	s.events <- Event{
		Type:    EventServiceStarted,
		Service: name,
		PID:     svc.PID(),
		Time:    time.Now(),
	}

	// Monitor service in background
	s.wg.Add(1)
	go s.monitor(svc)

	return nil
}

func (s *Supervisor) monitor(svc *service.Service) {
	defer s.wg.Done()

	exitCode, _ := svc.Wait()

	s.events <- Event{
		Type:     EventServiceStopped,
		Service:  svc.Name,
		ExitCode: exitCode,
		Time:     time.Now(),
	}

	// Handle restart policy
	switch svc.Restart {
	case service.RestartAlways:
		s.restartService(svc)
	case service.RestartOnFailure:
		if exitCode != 0 {
			s.restartService(svc)
		}
	}
}

func (s *Supervisor) restartService(svc *service.Service) {
	// Exponential backoff
	delays := []time.Duration{
		1 * time.Second,
		2 * time.Second,
		5 * time.Second,
		10 * time.Second,
		30 * time.Second,
	}

	for i, delay := range delays {
		time.Sleep(delay)
		
		if err := svc.Start(); err != nil {
			if i == len(delays)-1 {
				s.events <- Event{
					Type:    EventServiceFailed,
					Service: svc.Name,
					Error:   fmt.Sprintf("failed after %d restart attempts: %v", i+1, err),
					Time:    time.Now(),
				}
				return
			}
			continue
		}

		s.events <- Event{
			Type:    EventServiceRestarted,
			Service: svc.Name,
			PID:     svc.PID(),
			Time:    time.Now(),
		}

		// Continue monitoring
		s.wg.Add(1)
		go s.monitor(svc)
		return
	}
}

func (s *Supervisor) StopAll() {
	s.mu.RLock()
	order := s.order
	s.mu.RUnlock()

	// Stop in reverse order
	for i := len(order) - 1; i >= 0; i-- {
		s.StopService(order[i])
	}

	s.wg.Wait()
}

func (s *Supervisor) StopService(name string) error {
	s.mu.RLock()
	svc, ok := s.services[name]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("service not found: %s", name)
	}

	return svc.Stop()
}

func (s *Supervisor) Reload(cfg *config.Config) {
	s.mu.Lock()
	s.config = cfg
	s.mu.Unlock()
}

func (s *Supervisor) Events() <-chan Event {
	return s.events
}

func (s *Supervisor) Status() map[string]service.ServiceState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := make(map[string]service.ServiceState)
	for name, svc := range s.services {
		status[name] = svc.State()
	}
	return status
}
