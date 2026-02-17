package cli

import (
	"fmt"

	"github.com/Tresor-Kasend/apix/internal/request"
	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show a saved request",
		Long:  "Display the YAML content of a saved request from requests/<name>.yaml.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			content, err := request.ReadRaw(name)
			if err != nil {
				return err
			}

			fmt.Println()
			fmt.Printf("# requests/%s.yaml\n", name)
			fmt.Println(content)
			return nil
		},
	}
}
