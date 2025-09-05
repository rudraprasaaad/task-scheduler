package cmd

import "github.com/spf13/cobra"

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks in the scheduler",
	Long:  `The task command provides a suite of tools to create, list, inspect, and manage tasks within the Task Scheduler service.`,
}

func init() {
	rootCmd.AddCommand(taskCmd)
}
