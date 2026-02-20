package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tinkerbelle-io/tb-discover/internal/install"
	"github.com/tinkerbelle-io/tb-discover/internal/logging"
)

var flagPurge bool

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove tb-discover system service",
	Long: `Stop and remove the tb-discover system service.

By default, the config file at /etc/tb-discover/ is preserved.
Use --purge to also remove the config directory.`,
	RunE: runUninstall,
}

func init() {
	uninstallCmd.Flags().BoolVar(&flagPurge, "purge", false, "Also remove config files")
	rootCmd.AddCommand(uninstallCmd)
}

func runUninstall(cmd *cobra.Command, args []string) error {
	logging.Setup(flagLogLevel)

	if err := install.Uninstall(flagPurge); err != nil {
		return fmt.Errorf("uninstall failed: %w", err)
	}

	fmt.Println("tb-discover service removed.")
	if flagPurge {
		fmt.Println("Config files purged.")
	} else {
		fmt.Printf("Config preserved at %s (use --purge to remove)\n", install.DefaultConfigDir)
	}
	return nil
}
