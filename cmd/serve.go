package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/swishcloud/filesync-web/server"
	"github.com/swishcloud/identity-provider/flagx"
)

const SERVER_CONFIG_FILE = "IDENTITY_PROVIDER_CONFIG"

var serveCmd = &cobra.Command{
	Use: "serve",
	Run: func(cmd *cobra.Command, args []string) {
		path := flagx.MustGetString(cmd, "config")
		skip_tls_verify, err := cmd.Flags().GetBool("skip-tls-verify")
		if err != nil {
			log.Fatal(err)
		}
		tcpServer := server.NewTcpServer(path, 2003)
		webServer := server.NewFileSyncWebServer(path, skip_tls_verify)
		go tcpServer.Serve()
		webServer.Serve()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringP("config", "c", "config.yaml", "server config file")
	serveCmd.Flags().Bool("skip-tls-verify", false, "skip tls verify")
	serveCmd.MarkFlagRequired("config")
}
