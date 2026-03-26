package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"websocket-server/config"
	"websocket-server/db"
	"websocket-server/server"
)

var (
	flagPort            int
	flagModule          string
	flagCreateIfMissing bool
	flagCreateAdmin     bool
)

// Root command
var rootCmd = &cobra.Command{
	Use:     os.Args[0] + " [command]",
	Short:   "Ekhoes Server",
	Long:    "CLI to start and manage Ekhoes Server.",
	Version: config.Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		config.Init()

		if flagPort != 0 {
			config.Conf.Port = flagPort
		}
	},

	// -> Put here code to be executed without commands <-
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start server",
	RunE: func(cmd *cobra.Command, args []string) error {

		if config.Local() && flagCreateIfMissing {
			log.Printf("Checking database '%s'...", flagModule)

			if !db.CheckLocal(flagModule) {
				log.Printf("Creating database '%s'...", flagModule)

				if err := db.CreateLocal(flagModule); err != nil {
					log.Fatal(err)
				}

				if err := db.OpenAndInit(flagModule, flagCreateAdmin); err != nil {
					log.Fatal(err)
				}
			}
		}

		// Manteniamo la semantica: Start([]string) int -> exit code
		exitCode := server.Start()
		if exitCode != 0 {
			os.Exit(exitCode)
		}
		return nil
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initializes a module",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := db.OpenAndInit(flagModule, flagCreateAdmin)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	//log.SetFlags(log.Ldate | log.Ltime)

	rootCmd.PersistentFlags().BoolVarP(&config.Runtime.Local, "local", "l", false, "Local mode")
	rootCmd.SetVersionTemplate(`{{.Version}}`)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(initCmd)

	startCmd.Flags().IntVarP(&flagPort, "port", "p", 9876, "Server port")
	startCmd.Flags().BoolVarP(&flagCreateIfMissing, "create-db", "C", false, "Create local database if not exists (local mode only)")
	startCmd.Flags().BoolVarP(&flagCreateAdmin, "create-admin", "A", false, "Create default admin user")

	initCmd.Flags().StringVarP(&flagModule, "module", "m", "ekhoes", "Module to be initialized")
	initCmd.Flags().BoolVarP(&flagCreateAdmin, "create-admin", "A", false, "Create default admin user")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		// Cobra normalmente stampa già l'errore; usiamo log.Fatal come fallback.
		log.Fatal(err)
	}
}
