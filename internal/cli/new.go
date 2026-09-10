package cli

import (
	"fmt"
	"os"

	"github.com/joaomdsg/packets/internal/packet"
	"github.com/spf13/cobra"
)

func newNewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "new",
		Short: "Write a template packet.yaml to the current directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			const path = "packet.yaml"
			if err := os.WriteFile(path, []byte(packet.Template), 0o600); err != nil {
				return fmt.Errorf("new: write %s: %s", path, err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		},
	}
}
