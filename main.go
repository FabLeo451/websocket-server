package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"ekhoes-server/config"
	"ekhoes-server/db"
	"ekhoes-server/server"
)

var (
	flagPort            int
	flagModule          string
	flagInstallIfMissing bool
	flagAdminEmail      string
)

// Root command
var rootCmd = &cobra.Command{
	Use:     filepath.Base(os.Args[0]) + " [command]",
	Short:   "Ekhoes Server",
	Long:    "CLI to start and manage Ekhoes Server.",
	Version: config.Version(),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		config.Init()

		if flagPort != 0 {
			config.SetPort(flagPort)
		}
	},

	// -> Put here code to be executed without commands <-
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start server",
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagInstallIfMissing {
			log.Println("Checking if database exists...")

			dbExists, err := db.CheckDatabaseExists()

			if err != nil {
				log.Fatal(err)
				os.Exit(1)
			}

			if !dbExists {
				if err := Install(); err != nil {
					log.Fatal("Aborted")
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

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Create database and execute init script",
	RunE: func(cmd *cobra.Command, args []string) error {
		return Install()
	},
}

func Install() error {
	dbExists, err := db.CheckDatabaseExists()

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	if !dbExists {
		log.Println("Creating database...")

		if err := db.CreateDatabase(); err != nil {
			log.Fatal(err)
		}
	}

	err = db.OpenAndInit(flagModule)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if flagAdminEmail != "" {
		log.Printf("Creating administrator user %s...", flagAdminEmail)
		err := db.CreateAdmin(flagAdminEmail)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	log.Println("Initialization successful")

	return nil
}

func init() {
	//log.SetFlags(log.Ldate | log.Ltime)

	rootCmd.PersistentFlags().BoolVarP(&config.Runtime.Local, "local", "l", false, "Local database on disk")
	rootCmd.SetVersionTemplate(`{{.Version}}`)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(installCmd)

	startCmd.Flags().IntVarP(&flagPort, "port", "p", 9876, "Server port")
	startCmd.Flags().BoolVarP(&flagInstallIfMissing, "install-missing", "I", false, "Create and init database if not exists")
	startCmd.Flags().StringVarP(&flagAdminEmail, "create-admin", "A", "", "Create admin user (with -I)")

	//installCmd.Flags().StringVarP(&flagModule, "module", "m", "ekhoes", "Module to be initialized")
	//installCmd.Flags().BoolVarP(&flagInstallIfMissing, "create-db", "C", false, "Create database if not exists")
	installCmd.Flags().StringVarP(&flagAdminEmail, "create-admin", "A", "", "Create admin user")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		// Cobra normalmente stampa già l'errore; usiamo log.Fatal come fallback.
		log.Fatal(err)
	}
}
