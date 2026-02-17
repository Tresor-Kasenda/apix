package cli

import (
	"fmt"

	"github.com/Tresor-Kasend/apix/internal/output"
	"github.com/Tresor-Kasend/apix/internal/request"
	"github.com/spf13/cobra"
)

func newSaveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "save <name>",
		Short: "Save the last request",
		Long:  "Save the most recently executed request to requests/<name>.yaml for later replay.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			fromLast, _ := cmd.Flags().GetBool("from-last")
			if !fromLast {
				// Current behavior already saves from last request; this flag keeps intent explicit.
			}

			last, err := request.LoadLast()
			if err != nil {
				return fmt.Errorf("no previous request to save (run a request first)")
			}

			if err := request.Save(name, *last); err != nil {
				return err
			}

			output.PrintSuccess(fmt.Sprintf("Request saved as %q (requests/%s.yaml)", name, name))
			return nil
		},
	}
	cmd.Flags().Bool("from-last", false, "Explicitly save from the last executed request")
	return cmd
}
