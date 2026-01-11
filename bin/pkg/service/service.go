package service

import (
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type ServiceType int

const (
	TypeCore ServiceType = iota
	TypeAgent
	TypeUser
)

type RestartPolicy int

const (
	RestartNever RestartPolicy = iota
	RestartOnFailure
	RestartAlways
)

type ServiceState int

const (
	StateStopped ServiceState = iota
	StateStarting
	StateRunning
	StateStopping
	StateFailed
)

type Service struct {
	Name      string
	Command   string
	Args      []string
	Type      ServiceType
	Restart   RestartPolicy
	DependsOn []string
	Env       map[string]string

	state     ServiceState
	cmd       *exec.Cmd
	pid       int
	exitCode  int
	startTime time.Time
	mu        sync.RWMutex
}

func (s *Service) State() ServiceState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *Service) PID() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pid
}

func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == StateRunning {
		return nil
	}

	s.state = StateStarting
	s.cmd = exec.Command(s.Command, s.Args...)
	
	// Set environment
	s.cmd.Env = os.Environ()
	for k, v := range s.Env {
		s.cmd.Env = append(s.cmd.Env, k+"="+v)
	}

	// Set process group for clean shutdown
	s.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := s.cmd.Start(); err != nil {
		s.state = StateFailed
		return err
	}

	s.pid = s.cmd.Process.Pid
	s.state = StateRunning
	s.startTime = time.Now()

	return nil
}

func (s *Service) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != StateRunning {
		return nil
	}

	s.state = StateStopping

	// Send SIGTERM first
	if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// If SIGTERM fails, try SIGKILL
		s.cmd.Process.Kill()
	}

	// Wait for process to exit
	done := make(chan error, 1)
	go func() {
		done <- s.cmd.Wait()
	}()

	select {
	case <-done:
		// Process exited
	case <-time.After(5 * time.Second):
		// Force kill after timeout
		s.cmd.Process.Kill()
		<-done
	}

	s.state = StateStopped
	if s.cmd.ProcessState != nil {
		s.exitCode = s.cmd.ProcessState.ExitCode()
	}

	return nil
}

func (s *Service) Wait() (int, error) {
	if s.cmd == nil {
		return -1, nil
	}
	
	err := s.cmd.Wait()
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.cmd.ProcessState != nil {
		s.exitCode = s.cmd.ProcessState.ExitCode()
	}
	s.state = StateStopped
	
	return s.exitCode, err
}

func (s *Service) ExitCode() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.exitCode
}

func (s *Service) Uptime() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.state != StateRunning {
		return 0
	}
	return time.Since(s.startTime)
}
