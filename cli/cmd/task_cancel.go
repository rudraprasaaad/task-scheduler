package cmd

import (
	"fmt"
	"log"

	"github.com/rudraprasaaad/task-scheduler/cli/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var taskCancelCmd = &cobra.Command{
	Use:   "cancel [TASK_ID]",
	Short: "Cancel a pending task",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		taskID := args[0]
		apiURL := viper.GetString("api_url")
		cli, err := client.NewClient(apiURL)

		if err != nil {
			log.Fatalf("Failed to create API client: %v", err)
		}

		if err := cli.CancelTaskk(taskID); err != nil {
			log.Fatalf("Failed to cancel task: %v", err)
		}

		fmt.Printf("✅ Task %s successfully requested for cancellation.\n", taskID)
		fmt.Println("Note: Cancellation is only possilbe if the task is still in 'pending' state.")
	},
}

func init() {
	taskCmd.AddCommand(taskCancelCmd)
}
