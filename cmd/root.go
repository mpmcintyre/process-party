/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

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
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 && len(execCommands) == 0 {
			return errors.New("please provide either a directory or execution commands to run in parallel")
		}

		config := runner.CreateConfig()

		if len(args) != 0 {
			err := config.ParseFile(args[0])
			if err != nil {
				return err
			}
		}

		for _, cmd := range execCommands {
			config.ParseInlineCmd(cmd)
		}

		var wg sync.WaitGroup
		wg.Add(len(config.Processes))

		contexts := []runner.Context{}

		mainChannels := []runner.MainChannelsOut{}
		runningProcessCount := len(config.Processes)

		for index, process := range config.Processes {

			// Create the task output channels
			taskChannel := runner.TaskChannelsOut{
				Buzzkill:     make(chan bool),
				EndOfCommand: make(chan string),
			}

			// Create the task input channels
			mainChannels = append(mainChannels,
				runner.MainChannelsOut{
					Buzzkill: make(chan bool),
					StdIn:    make(chan string),
				})

			// Create context
			contexts = append(contexts, runner.CreateContext(
				process,
				&wg,
				mainChannels[index],
				taskChannel,
			))

			// Start listening to the threads channels fo multi-channel communcation
			go func() {
			monitorLoop:
				for {
					select {
					case <-taskChannel.Buzzkill:
						for i := range len(config.Processes) {
							// Send to all other channels (not including this one)
							if i != index {
								mainChannels[i].Buzzkill <- true
							}
						}
						break monitorLoop
					case <-taskChannel.EndOfCommand:
						runningProcessCount--
						if runningProcessCount <= 0 {
							fmt.Println("All processes exited")
							break monitorLoop
						}
					}
				}

			}()
		}
		// Start the task
		for _, context := range contexts {
			go context.Run()
		}

		go func() {
			reader := bufio.NewReader(os.Stdin)
			fmt.Println("Input is active - std in to commands using [all] or specific command using [<cmd prefix>]")
			fmt.Println("Get the status using \"status\", or quit the party using \"quit\" or ctrl+c")
			for {
				text, _ := reader.ReadString('\n')
				s := strings.Split(text, "]")
				// If there are multiple "] " values, re-add them into the provided text past the first entry
				if len(s) > 1 {
					text = strings.Join(s[:1], "]")
				}
				// If the target is provided as [all]test test we need the "all" value, however we can provide "status" or "quit" as is
				target := strings.Replace(s[0], "[", "", -1)
				switch strings.ToLower(target) {
				case "all":
					// Broadcast to all processes
				case "status":
					// Print status of every command
				case "exit":
					fmt.Println("Exiting all")
					for _, x := range mainChannels {
						x.Buzzkill <- true
					}
				}
				if runningProcessCount > 0 {
					fmt.Println(text)
				}
			}
		}()

		wg.Wait()

		return nil
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
