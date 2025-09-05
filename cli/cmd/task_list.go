package cmd

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/rudraprasaaad/task-scheduler/cli/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	limit  int
	offset int
)

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks in the system",
	Run: func(cmd *cobra.Command, args []string) {
		apiURL := viper.GetString("api_url")
		cli, err := client.NewClient(apiURL)
		if err != nil {
			log.Fatalf("Failed to create API client: %v", err)
		}

		tasks, err := cli.ListTasks(limit, offset)
		if err != nil {
			log.Fatalf("Failed to list tasks: %v", err)
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks found.")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Println(w, "ID\tNAME\tTYPE\tSTATUS\tCREATED AT")
		for _, task := range tasks {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", task.ID, task.Name, task.Type, task.Status, task.CreatedAt.Format("2006-01-02 15:04:05"))
		}
		w.Flush()
	},
}

func init() {
	taskCmd.AddCommand(taskListCmd)

	taskListCmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of tasks to return")
	taskListCmd.Flags().IntVarP(&offset, "offset", "o", 0, "offset for pagination")
}
