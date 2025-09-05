package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	apiURL  string
)

var rootCmd = &cobra.Command{
	Use:   "task-cli",
	Short: "A CLI to interact with the Task Scheduler service",
	Long:  `task-cli is a command-line interface to create, manage, and monitor tasks on your self-hosted Task Scheduler.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.task-cli.yaml)")
	rootCmd.PersistentFlags().StringVarP(&apiURL, "api-url", "a", "http://localhost:8080", "The base URL of the Task Scheduler API")

	viper.BindPFlag("api_url", rootCmd.PersistentFlags().Lookup("api-url"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".task-cli")
	}

	viper.SetDefault("api_url", "http://localhost:8080")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
