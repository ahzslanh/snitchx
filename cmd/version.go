package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ahzslanh/snitchx/internal/errutil"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"

	versionJSON bool
)

type versionInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	Go      string `json:"go"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version/build info",
	Run: func(cmd *cobra.Command, args []string) {
		info := versionInfo{
			Version: Version,
			Commit:  Commit,
			Date:    Date,
			Go:      runtime.Version(),
			OS:      runtime.GOOS,
			Arch:    runtime.GOARCH,
		}

		if versionJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(info); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		}

		bold := color.New(color.Bold)
		cyan := color.New(color.FgCyan)
		faint := color.New(color.Faint)

		errutil.Print(bold, "snitchx ")
		errutil.Println(cyan, Version)
		fmt.Println()

		errutil.Print(faint, "  commit  ")
		fmt.Println(Commit)

		errutil.Print(faint, "  built   ")
		fmt.Println(Date)

		errutil.Print(faint, "  go      ")
		fmt.Println(runtime.Version())

		errutil.Print(faint, "  os      ")
		fmt.Printf("%s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	versionCmd.Flags().BoolVar(&versionJSON, "json", false, "Output version information as JSON")
	rootCmd.AddCommand(versionCmd)
}
