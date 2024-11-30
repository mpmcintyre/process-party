/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	runner "github.com/mpmcintyre/process-party/internal"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "process-party",
	Short: "A brief description of your application",
	Args:  cobra.MaximumNArgs(1),
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 && len(execCommands) == 0 {
			fmt.Println("Please provide either a directory or execution commands to run in parallel")
			return
		}

		config := runner.CreateConfig()
		execPrefix := ""

		if len(args) != 0 {
			err := config.ParseFile(args[0])
			if err != nil {
				fmt.Println(err)
				return
			}
			execPrefix = "-e"
		}

		for _, cmd := range execCommands {
			p := runner.Process{
				Command: cmd,
				Prefix:  execPrefix,
			}
			config.Processes = append(config.Processes, p)
		}

		contexts := []runner.Context{}

		for _, process := range config.Processes {
			contexts = append(contexts, runner.CreateContext(
				process,
			))
		}

		context := runner.CreateContext()

	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var execCommands []string

func init() {
	rootCmd.Flags().StringSliceVar(&execCommands, "e", execCommands, "Execute command (can be used multiple times)")
}
