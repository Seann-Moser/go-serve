package cmd

import (
	"github.com/spf13/cobra"

	"github.com/Seann-Moser/go-serve/internal/client_builder"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "client-builder",
	Short: "A brief description of your command",
	RunE:  client_builder.Runner,
}

func init() {
	serveCmd.Flags().AddFlagSet(client_builder.Flags())
	rootCmd.AddCommand(serveCmd)
}
