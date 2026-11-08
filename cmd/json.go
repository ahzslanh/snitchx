package cmd

import (
	"github.com/spf13/cobra"
)

var jsonCmd = &cobra.Command{
	Use:   "json [filters...]",
	Short: "One-shot json output of connections",
	Long:  `One-shot json output of connections. This is an alias for "ls -o json".`,
	Run: func(cmd *cobra.Command, args []string) {
		runListCommand("json", args)
	},
}

func init() {
	rootCmd.AddCommand(jsonCmd)
	addFilterFlags(jsonCmd)
}
