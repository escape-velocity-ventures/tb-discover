package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/tinkerbelle-io/tb-discover/internal/agent"
	"github.com/tinkerbelle-io/tb-discover/internal/config"
	"github.com/tinkerbelle-io/tb-discover/internal/logging"
)

var (
	flagClusterID      string
	flagIdleTimeout    time.Duration
	flagScanInterval   time.Duration
	flagDaemonProfile  string
	flagGatewayURL     string
	flagSaaSURL        string
	flagPermissions    []string
	flagMaxSessions    int
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run as a persistent agent with scanning and terminal support",
	Long: `Run tb-discover as a daemon that continuously scans infrastructure,
reports to TinkerBelle SaaS, and serves terminal sessions via WebSocket.

The daemon runs two concurrent loops:
  1. Scan loop: periodic infrastructure scan + upload to edge-ingest
  2. WebSocket loop: terminal session multiplexing via gateway

Use --url for the SaaS base URL (Supabase, for upload).
Use --gateway for the WebSocket gateway URL (for terminal sessions).
Either or both can be specified.`,
	RunE: runDaemon,
}

func init() {
	daemonCmd.Flags().StringVar(&flagClusterID, "cluster-id", "", "Cluster identifier")
	daemonCmd.Flags().DurationVar(&flagIdleTimeout, "idle-timeout", 30*time.Minute, "Terminal session idle timeout")
	daemonCmd.Flags().DurationVar(&flagScanInterval, "scan-interval", 5*time.Minute, "Scan interval (e.g., 5m, 30s)")
	daemonCmd.Flags().StringVar(&flagDaemonProfile, "profile", "standard", "Scan profile: minimal, standard, full")
	daemonCmd.Flags().StringVar(&flagGatewayURL, "gateway", "", "Gateway WebSocket URL for terminal sessions (env: TB_GATEWAY_URL)")
	daemonCmd.Flags().StringVar(&flagSaaSURL, "saas-url", "", "SaaS base URL for upload (env: TB_URL, defaults to --url)")
	daemonCmd.Flags().StringSliceVar(&flagPermissions, "permissions", []string{"scan"}, "Agent permissions: scan, terminal")
	daemonCmd.Flags().IntVar(&flagMaxSessions, "max-sessions", 10, "Maximum concurrent terminal sessions")
	rootCmd.AddCommand(daemonCmd)
}

func runDaemon(cmd *cobra.Command, args []string) error {
	logging.Setup(flagLogLevel)

	// Load config file for defaults (permissions, etc.)
	cfg, _ := config.Load(flagConfig)

	token := resolveToken()
	saasURL := resolveSaaSURL()
	gatewayURL := resolveGatewayURL()

	// Need at least one mode of operation
	if token == "" {
		return cmd.Help()
	}

	// Merge permissions: flag overrides config file
	permissions := flagPermissions
	if !cmd.Flags().Changed("permissions") && cfg != nil && len(cfg.Permissions) > 0 {
		permissions = cfg.Permissions
	}

	// Build scan loop config
	var scanCfg *agent.ScanLoopConfig
	if saasURL != "" {
		scanCfg = &agent.ScanLoopConfig{
			Profile:   flagDaemonProfile,
			Interval:  flagScanInterval,
			UploadURL: saasURL,
			Token:     token,
			Version:   rootCmd.Version,
		}
	}

	a := agent.New(agent.Config{
		WSURL:       gatewayURL,
		Token:       token,
		ClusterID:   flagClusterID,
		IdleTimeout: flagIdleTimeout,
		ScanConfig:  scanCfg,
		Permissions: permissions,
		MaxSessions: flagMaxSessions,
	})

	return a.Run(context.Background())
}

// resolveSaaSURL returns the SaaS URL for uploading scan results.
func resolveSaaSURL() string {
	if flagSaaSURL != "" {
		return flagSaaSURL
	}
	// Fall back to --url / TB_URL
	return resolveURL()
}

// resolveGatewayURL returns the gateway WebSocket URL.
func resolveGatewayURL() string {
	if flagGatewayURL != "" {
		return flagGatewayURL
	}
	return resolveEnv("TB_GATEWAY_URL")
}

func resolveEnv(key string) string {
	if v, ok := lookupEnv(key); ok {
		return v
	}
	return ""
}
