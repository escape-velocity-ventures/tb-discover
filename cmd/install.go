package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tinkerbelle-io/tb-manage/internal/install"
	"github.com/tinkerbelle-io/tb-manage/internal/logging"
)

var flagInstallProfile string

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install tb-manage as a system service",
	Long: `Install tb-manage as a systemd service (Linux) or launchd daemon (macOS).

This command:
  1. Validates the provided token against the SaaS URL
  2. Writes a config file to /etc/tb-manage/config.yaml
  3. Creates and enables a system service
  4. Starts the service immediately

The service runs 'tb-manage daemon' with the configured profile.`,
	RunE: runInstall,
}

func init() {
	installCmd.Flags().StringVar(&flagInstallProfile, "profile", "standard", "Scan profile: minimal, standard, full")
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	logging.Setup(flagLogLevel)

	token := resolveToken()
	url := resolveURL()

	if token == "" {
		return fmt.Errorf("--token or TB_TOKEN is required")
	}
	if url == "" {
		return fmt.Errorf("--url or TB_URL is required")
	}

	fmt.Println("Installing tb-manage...")

	cfg := install.InstallConfig{
		Token:   token,
		URL:     url,
		Profile: flagInstallProfile,
	}

	if err := install.Install(cfg); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	fmt.Println("tb-manage installed and running.")
	fmt.Printf("  Config: %s\n", install.DefaultConfigFile)
	fmt.Printf("  Profile: %s\n", flagInstallProfile)
	fmt.Println("\nCheck status with: tb-manage status")
	return nil
}
