package cli

import (
	"context"
	"fmt"
	"gopublic/internal/client/config"
	"gopublic/internal/client/inspector"
	"gopublic/internal/client/tunnel"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gopublic",
	Short: "A secure request tunneling tool",
}

// ServerAddr should be injected via ldflags. Default for dev.
var ServerAddr = "localhost:4443"

func Init(serverAddr string) {
	if serverAddr != "" {
		ServerAddr = serverAddr
	}

	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(startCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var authCmd = &cobra.Command{
	Use:   "auth [token]",
	Short: "Save authentication token",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		token := args[0]
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}
		cfg.Token = token
		if err := config.SaveConfig(cfg); err != nil {
			log.Fatalf("Error saving config: %v", err)
		}
		path, _ := config.GetConfigPath()
		fmt.Printf("Token saved to %s\n", path)
	},
}

var startCmd = &cobra.Command{
	Use:   "start [port]",
	Short: "Start a public tunnel to a local port",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}

		if cfg.Token == "" {
			log.Fatal("No token found. Run 'gopublic auth <token>' first.")
		}

		// Setup context for graceful shutdown
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Handle shutdown signals
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigChan
			fmt.Println("\nShutdown signal received, closing tunnel...")
			cancel()
		}()

		// Start Inspector
		inspector.Start("4040")

		// Check for project config (gopublic.yaml)
		allFlag, _ := cmd.Flags().GetBool("all")
		projectCfg, projectErr := config.LoadProjectConfig("")

		if projectErr == nil && (allFlag || len(args) == 0) {
			// Multi-tunnel mode from gopublic.yaml
			fmt.Println("Loading tunnels from gopublic.yaml...")
			fmt.Println("Inspector UI: http://localhost:4040")

			manager := tunnel.NewTunnelManager(ServerAddr, cfg.Token)

			// Set first tunnel port for replay (use first tunnel's port)
			for _, t := range projectCfg.Tunnels {
				inspector.SetLocalPort(t.Addr)
				break
			}

			for name, t := range projectCfg.Tunnels {
				manager.AddTunnel(name, t.Addr, t.Subdomain)
			}

			if err := manager.StartAll(ctx); err != nil {
				if err != context.Canceled {
					log.Fatalf("Tunnel error: %v", err)
				}
			}
		} else if len(args) == 1 {
			// Single tunnel mode (legacy)
			port := args[0]
			fmt.Printf("Starting tunnel to localhost:%s on server %s\n", port, ServerAddr)
			fmt.Println("Inspector UI: http://localhost:4040")

			// Configure replay with local port
			inspector.SetLocalPort(port)

			t := tunnel.NewTunnel(ServerAddr, cfg.Token, port)
			if err := t.StartWithReconnect(ctx, nil); err != nil {
				if err != context.Canceled {
					log.Fatalf("Tunnel error: %v", err)
				}
			}
		} else {
			log.Fatal("Either provide a port or create gopublic.yaml config file")
		}

		fmt.Println("Tunnel closed")
	},
}

func init() {
	startCmd.Flags().BoolP("all", "a", false, "Start all tunnels from gopublic.yaml")
}
