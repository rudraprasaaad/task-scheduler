package cmd

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/rudraprasaaad/task-scheduler/cli/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display the current status of the scheduler system",
	Run: func(cmd *cobra.Command, args []string) {
		apiURL := viper.GetString("api_url")
		cli, err := client.NewClient(apiURL)

		if err != nil {
			log.Fatalf("Failed to create API client: %v", err)
		}

		status, err := cli.GetSystemStatus()
		if err != nil {
			log.Fatalf("Failed to get sytem status: %v", err)
		}

		fmt.Println("--- System Summary ---")
		summaryTable := tablewriter.NewWriter(os.Stdout)
		summaryTable.Header([]string{"Metric", "Value"})
		summaryTable.Append([]string{"Pending Tasks in Queue", fmt.Sprintf("%d", status.QueueSize)})
		summaryTable.Append([]string{"Total Workers", fmt.Sprintf("%d", len(status.Workers))})
		summaryTable.Render()
		fmt.Println()

		if len(status.Workers) > 0 {
			fmt.Println("--- Worker Details ---")
			workerTable := tablewriter.NewWriter(os.Stdout)
			workerTable.Header([]string{"Worker ID", "Status", "Tasks Run", "Last Seen"})

			for _, worker := range status.Workers {
				workerTable.Append([]string{
					worker.ID,
					worker.Status,
					fmt.Sprintf("%d", worker.TasksRun),
					worker.LastSeen.Format(time.RFC1123),
				})
			}
			workerTable.Render()
		} else {
			fmt.Println("No workers are currently registered with the system.")
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
