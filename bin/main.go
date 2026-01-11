// MixOS Init System
// Go-based init (PID 1) for MixOS custom kernel
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"mixos.dev/init/pkg/config"
	"mixos.dev/init/pkg/ipc"
	"mixos.dev/init/pkg/service"
	"mixos.dev/init/pkg/supervisor"
)

const (
	Version = "0.1.0"
	Banner  = `
 __  __ _       ___  ____  
|  \/  (_)_  __/ _ \/ ___| 
| |\/| | \ \/ / | | \___ \ 
| |  | | |>  <| |_| |___) |
|_|  |_|_/_/\_\\___/|____/ 
                           
MixOS Init System v%s
`
)

func main() {
	fmt.Printf(Banner, Version)

	// Check if running as PID 1
	if os.Getpid() != 1 {
		fmt.Println("[init] Warning: Not running as PID 1, some features may not work")
	}

	// Load configuration
	cfg, err := config.Load("/etc/mixos/init.toml")
	if err != nil {
		fmt.Printf("[init] Using default configuration: %v\n", err)
		cfg = config.Default()
	}

	// Initialize IPC server
	ipcServer, err := ipc.NewServer(cfg.IPC.SocketPath)
	if err != nil {
		fmt.Printf("[init] Failed to create IPC server: %v\n", err)
		os.Exit(1)
	}
	defer ipcServer.Close()

	// Start IPC server in background
	go func() {
		if err := ipcServer.Listen(); err != nil {
			fmt.Printf("[init] IPC server error: %v\n", err)
		}
	}()
	fmt.Printf("[init] IPC server listening on %s\n", cfg.IPC.SocketPath)

	// Initialize service supervisor
	sup := supervisor.New(cfg, ipcServer)

	// Register core services
	sup.Register(&service.Service{
		Name:    "ocaml-broker",
		Command: "/svc/broker/broker",
		Args:    []string{},
		Type:    service.TypeCore,
		Restart: service.RestartAlways,
	})

	sup.Register(&service.Service{
		Name:    "ocaml-pkgmgr",
		Command: "/svc/pkgmgr/pkgmgr",
		Args:    []string{},
		Type:    service.TypeCore,
		Restart: service.RestartAlways,
		DependsOn: []string{"ocaml-broker"},
	})

	sup.Register(&service.Service{
		Name:    "ocaml-builder",
		Command: "/svc/builder/builder",
		Args:    []string{},
		Type:    service.TypeCore,
		Restart: service.RestartAlways,
		DependsOn: []string{"ocaml-broker"},
	})

	sup.Register(&service.Service{
		Name:    "ocaml-resolver",
		Command: "/svc/resolver/resolver",
		Args:    []string{},
		Type:    service.TypeCore,
		Restart: service.RestartAlways,
		DependsOn: []string{"ocaml-broker"},
	})

	sup.Register(&service.Service{
		Name:    "ocaml-cache",
		Command: "/svc/cache/cache",
		Args:    []string{},
		Type:    service.TypeCore,
		Restart: service.RestartAlways,
		DependsOn: []string{"ocaml-broker"},
	})

	sup.Register(&service.Service{
		Name:    "python-agent",
		Command: "/agent/run.sh",
		Args:    []string{},
		Type:    service.TypeAgent,
		Restart: service.RestartOnFailure,
		DependsOn: []string{"ocaml-broker", "ocaml-pkgmgr"},
	})

	// Start all services
	fmt.Println("[init] Starting services...")
	if err := sup.StartAll(); err != nil {
		fmt.Printf("[init] Failed to start services: %v\n", err)
	}

	// Setup signal handlers
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	// Main loop
	fmt.Println("[init] System ready")
	for {
		select {
		case sig := <-sigChan:
			switch sig {
			case syscall.SIGTERM, syscall.SIGINT:
				fmt.Println("[init] Received shutdown signal")
				sup.StopAll()
				fmt.Println("[init] Shutdown complete")
				os.Exit(0)
			case syscall.SIGHUP:
				fmt.Println("[init] Reloading configuration...")
				cfg, _ = config.Load("/etc/mixos/init.toml")
				sup.Reload(cfg)
			}
		case event := <-sup.Events():
			handleEvent(event, sup)
		}
	}
}

func handleEvent(event supervisor.Event, sup *supervisor.Supervisor) {
	switch event.Type {
	case supervisor.EventServiceStarted:
		fmt.Printf("[init] Service started: %s (PID %d)\n", event.Service, event.PID)
	case supervisor.EventServiceStopped:
		fmt.Printf("[init] Service stopped: %s (exit code %d)\n", event.Service, event.ExitCode)
	case supervisor.EventServiceFailed:
		fmt.Printf("[init] Service failed: %s - %s\n", event.Service, event.Error)
	}
}
