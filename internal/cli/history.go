package cli

import (
	"fmt"
	"time"

	"github.com/Tresor-Kasend/apix/internal/history"
	"github.com/Tresor-Kasend/apix/internal/output"
	"github.com/spf13/cobra"
)

func newHistoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "Show request execution history",
		Long:  "Display the most recent requests executed by apix.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clearHistory, _ := cmd.Flags().GetBool("clear")
			if clearHistory {
				if err := history.Clear(); err != nil {
					return err
				}
				output.PrintSuccess("History cleared")
				return nil
			}

			limit, _ := cmd.Flags().GetInt("limit")
			entries, err := history.Read(limit)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				output.PrintInfo("No history entries found.")
				return nil
			}

			fmt.Println()
			for _, entry := range entries {
				status := "-"
				if entry.Status > 0 {
					status = fmt.Sprintf("%d", entry.Status)
				}
				fmt.Printf(
					"  %s  %-6s %-4s %s (%dms)\n",
					entry.Timestamp.Format(time.RFC3339),
					entry.Method,
					status,
					entry.Path,
					entry.DurationMS,
				)
			}
			fmt.Println()
			return nil
		},
	}

	cmd.Flags().Int("limit", 20, "Maximum number of entries to display")
	cmd.Flags().Bool("clear", false, "Clear history and exit")

	return cmd
}
