package cmd

import (
	"fmt"
	"os"

	"github.com/ahzslanh/snitchx/internal/config"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
)

var rootCmd = &cobra.Command{
	Use:   "snitchx",
	Short: "snitchx - cross-platform security inspection",
	Long: `snitchx is a cross-platform security inspection tool.

Network. Processes. Services. Compliance.

Inspect TCP/UDP endpoints and the processes that own them, and audit services,
startup items, and compliance posture across macOS, Linux, and Windows.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if _, err := config.Load(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Error loading config: %v\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// default to top - flags are shared so they work here too
		topCmd.Run(cmd, args)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/snitchx/snitchx.toml)")
	rootCmd.PersistentFlags().Bool("debug", false, "enable debug logs to stderr")

	// add top's flags to root so `snitchx -l` works (defaults to top command)
	cfg := config.Get()
	rootCmd.Flags().StringVar(&topTheme, "theme", cfg.Defaults.Theme, "Theme for TUI (see 'snitchx themes')")
	rootCmd.Flags().DurationVarP(&topInterval, "interval", "i", 0, "Refresh interval (default 1s)")

	// shared flags for root command
	addFilterFlags(rootCmd)
	addResolutionFlags(rootCmd)
}
