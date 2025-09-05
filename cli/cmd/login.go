package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/rudraprasaaad/task-scheduler/cli/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with the Task Scheduler service",
	Run: func(cmd *cobra.Command, args []string) {
		apiURL := viper.GetString("api_url")
		cli, err := client.NewClient(apiURL)
		if err != nil {
			log.Fatalf("Failed to create API client: %v", err)
		}

		var email string
		fmt.Print("Enter email: ")
		fmt.Scanln(&email)

		fmt.Print("Enter password: ")
		bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			log.Fatalf("Failed to read password: %v", err)
		}
		fmt.Println()

		if err := cli.Login(email, string(bytePassword)); err != nil {
			log.Fatalf("Login failed: %v", err)
		}

		fmt.Println("Login successful. Session cookies saved.")
	},
}
