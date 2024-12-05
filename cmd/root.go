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

	"github.com/fatih/color"
	runner "github.com/mpmcintyre/process-party/internal"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "process-party ./path/to/config.yml -e \"tailwindcss ...\" -e \"go run main.go\"",
	Short: "Run multiple processes concurrently with input and grouping",
	Args:  cobra.MaximumNArgs(1),
	Long: `Start a process, or start a party
Run multiple processes concurrently with the same configuration on any system. 
Simple usage, high flexibility, robust use cases. The input allows
you to view the status of all commands with the "status" command, pipe input to 
a specific command using <command name|command prefix>:<input> e.g. "cmd:echo hello",
or pipe input to all commands using "all:<input>". Gracefully shutdown all processes 
using ctrl+c or input "exit" into the command line.
	`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 && len(execCommands) == 0 {
			return errors.New("please provide either a directory or execution commands to run in parallel")
		}

		// Create the configuration to store settings and process configurations
		config := runner.CreateConfig()

		// Parse the input file if the user passes in an argument
		if len(args) != 0 {
			err := config.ParseFile(args[0])
			if err != nil {
				return err
			}
		}

		// Parse the inline commands (--e flag)
		for _, cmd := range execCommands {
			config.ParseInlineCmd(cmd)
		}

		// Create wait group for each spawned process
		var wg sync.WaitGroup
		wg.Add(len(config.Processes))

		// Create context and channel groups
		contexts := []runner.Context{}
		mainChannels := []runner.MainChannelsOut{}
		// Keep track of number of running procesess to exit main app
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
				&process,
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
		input_loop:
			for {
				text, _ := reader.ReadString('\n')
				text = strings.TrimSpace(text) // Remove leading/trailing whitespace including newlines
				s := strings.Split(text, ":")  // Split by ":"
				if len(s) < 1 {
					continue
				}
				target := ""
				if len(s) == 1 {
					target = text
				} else {
					target = s[0]
				}
				switch target {
				case "all":
					if len(s) < 2 {
						fmt.Println("No input provided")
						continue
					}
					input := s[1:]
					for _, context := range contexts {
						if context.Process.Status == runner.ExitStatusRunning {
							context.MainChannelsOut.StdIn <- strings.Join(input, "")
						}
					}

					// Broadcast to all processes
				case "status":
					fmt.Println()
					// Print status of every command
					headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
					columnFmt := color.New(color.FgYellow).SprintfFunc()
					tbl := table.New("Index", "Name", "Prefix", "Command", "Status")
					tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)
					for index, context := range contexts {
						tbl.AddRow(index, context.Process.Name, context.Process.Prefix, context.Process.Command, context.GetStatusAsStr())
					}
					tbl.Print()
					fmt.Println()
				case "exit":
					fmt.Println("Exiting all")

					for _, context := range contexts {
						if context.Process.Status == runner.ExitStatusRunning {
							context.MainChannelsOut.Buzzkill <- true
						}
					}

					break input_loop

				default:
					if len(s) < 2 {
						fmt.Println("No input provided")
						continue
					}
					found := false
					input := s[1:]
					for _, context := range contexts {
						if context.Process.Name == target || context.Process.Prefix == target {
							found = true
							if context.Process.Status == runner.ExitStatusRunning {
								context.MainChannelsOut.StdIn <- strings.Join(input, "")
							} else {
								fmt.Printf("The %s command has exited, cannot write to process\n", target)
							}
						}
					}
					if !found {
						fmt.Printf("%s not found", target)
					}
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
	rootCmd.Flags().StringSliceVarP(&execCommands, "execute", "e", execCommands, "Execute command (can be used multiple times)")
}
