package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/rudraprasaaad/task-scheduler/cli/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var taskGetCmd = &cobra.Command{
	Use:   "get [TASK_ID]",
	Short: "Get detailed information about a single task",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		taskID := args[0]
		apiURL := viper.GetString("api_url")
		cli, err := client.NewClient(apiURL)

		if err != nil {
			log.Fatalf("Failed to create API client: %v", err)
		}

		task, err := cli.GetTask(taskID)

		if err != nil {
			log.Fatalf("Failed to get task :%v", err)
		}

		fmt.Printf("--- Task Details: %s ---\n", task.ID)
		fmt.Printf("Name:\t\t%s\n", task.Name)
		fmt.Printf("Type:\t\t%s\n", task.Type)
		fmt.Printf("Status:\t\t%s\n", task.Status)
		fmt.Printf("Priority:\t%d\n", task.Priority)
		fmt.Printf("Retries:\t%d / %d\n", task.Retries, task.MaxRetries)
		fmt.Printf("Created At:\t%s\n", task.CreatedAt.Format(time.RFC1123))
		fmt.Printf("Scheduled At:\t%s\n", task.ScheduledAt.Format(time.RFC1123))

		if task.WorkerID != "" {
			fmt.Printf("Worker ID:\t%s\n", task.WorkerID)
		}

		if task.Error != "" {
			fmt.Printf("Last Error:\t%s\n", task.Error)
		}

		payload, _ := json.MarshalIndent(task.Payload, "", " ")
		fmt.Printf("Payload:\n%s\n", string(payload))
	},
}

func init() {
	taskCmd.AddCommand(taskGetCmd)
}
