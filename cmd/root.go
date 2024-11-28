/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	party "github.com/mpmcintyre/process-party/internal"
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
		commander := party.New()
		if len(args) != 0 {
			err := commander.AddFile(args[0])
			if err != nil {
				fmt.Println(err)
				return
			}
		}

		for _, arg := range args {
			fmt.Println(arg)
		}

		for _, flags := range execCommands {
			fmt.Println(flags)
			// exec.Command("sh", "-c", arg).Run()
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var execCommands []string

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.pparty.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().StringSliceVar(&execCommands, "e", execCommands, "Execute command (can be used multiple times)")
	rootCmd.Flags().String("file", "", "cfg ./tools")

	// rootCmd.Flags().BoolVarP(&onlyDigits, "digits", "d", false, "Count only digits")
}
