package cmd

import (
	"github.com/spf13/cobra"
	"github.com/swishcloud/filesync-web/server"
)

const SERVER_CONFIG_FILE = "IDENTITY_PROVIDER_CONFIG"

var serveCmd = &cobra.Command{
	Use: "serve",
	Run: func(cmd *cobra.Command, args []string) {
		server.NewFileSyncWebServer().Serve()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
