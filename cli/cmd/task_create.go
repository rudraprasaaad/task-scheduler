package cmd

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/rudraprasaaad/task-scheduler/cli/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	taskName     string
	taskType     string
	taskPayload  string
	taskPriority int
)

var taskCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a new task",
	Example: `task-cli task create --name "Process video" --type "video_processing" --payload '{"url": "https://hey.com/me.mp4"}'`,
	Run: func(cmd *cobra.Command, args []string) {
		apiURL := viper.GetString("api_url")
		cli, err := client.NewClient(apiURL)
		if err != nil {
			log.Fatalf("Failed to create API clientL %v", err)
		}

		if !json.Valid([]byte(taskPayload)) {
			log.Fatalf("Error: --payload is not valid JSON.")
		}

		payload := client.CreateTaskPayload{
			Name:     taskName,
			Type:     taskType,
			Payload:  []byte(taskPayload),
			Priority: taskPriority,
		}

		createdTask, err := cli.CreateTask(payload)
		if err != nil {
			log.Fatalf("Failed to create task: %v", err)
		}

		fmt.Printf("Task created successfully!\n")
		fmt.Printf(" ID: %s\n", createdTask.ID)
		fmt.Printf(" Name: %s\n", createdTask.Name)
	},
}

func init() {
	taskCmd.AddCommand(taskCreateCmd)

	taskCreateCmd.Flags().StringVarP(&taskName, "name", "n", "", "Name of the task (required)")
	taskCreateCmd.Flags().StringVarP(&taskType, "type", "t", "", "Type of the task (required)")
	taskCreateCmd.Flags().StringVarP(&taskPayload, "payload", "p", "{}", "JSON payload for the task")
	taskCreateCmd.Flags().IntVarP(&taskPriority, "priority", "", 5, "Task priority (0 - 10)")

	taskCreateCmd.MarkFlagRequired("name")
	taskCreateCmd.MarkFlagRequired("type")
}
