package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	watchInterval time.Duration
	watchCount    int
)

var watchCmd = &cobra.Command{
	Use:   "watch [filters...]",
	Short: "Stream connection events as json frames",
	Long: `Stream connection events as json frames.
	
Filters are specified in key=value format. For example:
  snitchx watch proto=tcp state=established

Available filters:
  proto, state, pid, proc, lport, rport, user, laddr, raddr, contains
`,
	Run: func(cmd *cobra.Command, args []string) {
		runWatchCommand(args)
	},
}

func runWatchCommand(args []string) {
	filters, err := BuildFilters(args)
	if err != nil {
		log.Fatalf("Error parsing filters: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	ticker := time.NewTicker(watchInterval)
	defer ticker.Stop()

	count := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			connections, err := FetchConnections(filters)
			if err != nil {
				log.Printf("Error getting connections: %v", err)
				continue
			}

			frame := map[string]interface{}{
				"timestamp":   time.Now().Format(time.RFC3339Nano),
				"connections": connections,
				"count":       len(connections),
			}

			jsonOutput, err := json.Marshal(frame)
			if err != nil {
				log.Printf("Error marshaling JSON: %v", err)
				continue
			}

			fmt.Println(string(jsonOutput))

			count++
			if watchCount > 0 && count >= watchCount {
				return
			}
		}
	}
}

func init() {
	rootCmd.AddCommand(watchCmd)

	// watch-specific flags
	watchCmd.Flags().DurationVarP(&watchInterval, "interval", "i", time.Second, "Refresh interval (e.g., 500ms, 2s)")
	watchCmd.Flags().IntVarP(&watchCount, "count", "c", 0, "Number of frames to emit (0 = unlimited)")

	// shared filter flags
	addFilterFlags(watchCmd)
}
