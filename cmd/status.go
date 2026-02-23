package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tinkerbelle-io/tb-manage/internal/config"
	"github.com/tinkerbelle-io/tb-manage/internal/install"
	"github.com/tinkerbelle-io/tb-manage/internal/logging"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show tb-manage service status",
	Long:  `Display the current state of the tb-manage service, config, and binary.`,
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	logging.Setup(flagLogLevel)

	s := install.Status()

	fmt.Printf("Platform:   %s\n", s.Platform)
	fmt.Printf("Binary:     %s\n", valueOrNA(s.BinaryPath))
	fmt.Printf("Config:     %s\n", s.ConfigPath)
	fmt.Printf("Installed:  %s\n", boolStatus(s.Installed))
	fmt.Printf("Running:    %s\n", boolStatus(s.Running))

	// Show config summary if present
	if s.Installed {
		cfg, err := config.Load(install.DefaultConfigFile)
		if err == nil {
			fmt.Println()
			fmt.Println("Configuration:")
			fmt.Printf("  URL:      %s\n", maskEnd(cfg.URL, 20))
			fmt.Printf("  Token:    %s\n", maskToken(cfg.Token))
			fmt.Printf("  Profile:  %s\n", cfg.Profile)
			fmt.Printf("  Interval: %s\n", cfg.ScanInterval)
		}
	}

	// Show version
	fmt.Printf("\nVersion:    %s\n", rootCmd.Version)

	// Exit code 1 if not running (useful for scripts)
	if !s.Running {
		os.Exit(1)
	}
	return nil
}

func boolStatus(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func valueOrNA(s string) string {
	if s == "" {
		return "n/a"
	}
	return s
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

func maskEnd(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
