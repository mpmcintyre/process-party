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
	pp "github.com/mpmcintyre/process-party/internal"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
)

var execCommands []string

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

		// Print ascii art
		color.HiGreen("\n   ___                             ___           __      \n  / _ \\_______  _______ ___ ___   / _ \\___ _____/ /___ __\n / ___/ __/ _ \\/ __/ -_|_-<(_-<  / ___/ _ `/ __/ __/ // /\n/_/  /_/  \\___/\\__/\\__/___/___/ /_/   \\_,_/_/  \\__/\\_, / \n                                                  /___/  ")
		// Create the configuration to store settings and process configurations
		config := pp.CreateConfig()

		// Parse the input file if the user passes in an argument
		if len(args) != 0 {
			err := config.ParseFile(args[0])
			if err != nil {
				return err
			}
		}

		// Parse the inline commands (-e or --execute flag)
		for _, cmd := range execCommands {
			err := config.ParseInlineCmd(cmd)
			if err != nil {
				return err
			}
		}

		// Create the waitgroup
		var wg sync.WaitGroup

		// Generate the contexts for all processes in the config
		runContexts := config.GenerateRunTaskContexts(&wg)
		// Link contexts with their triggers
		err := pp.LinkProcessTriggers(runContexts)
		if err != nil {
			return err
		}

		// Start an input stream monitor
		go func() {
			reader := bufio.NewReader(os.Stdin)
			fmt.Println("Input is active - std in to commands using [all] or specific command using [<cmd prefix>]")
			fmt.Println("Get the status using \"status\", or quit the party using \"quit\" or ctrl+c")
			fmt.Println("-------------------------------------------------------")

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
					for _, context := range runContexts {
						if context.Status == pp.ProcessStatusRunning {
							context.Write(strings.Join(input, ""))
						}
					}

					// Broadcast to all processes
				case "status":
					// Print runcontexts status
					if len(runContexts) > 0 {
						fmt.Println()
						// Print status of every command
						headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
						columnFmt := color.New(color.FgYellow).SprintfFunc()
						tbl := table.New("Index", "Name", "Prefix", "Command", "Status")
						tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)
						for index, context := range runContexts {
							tbl.AddRow(index, context.Process.Name, context.Process.Prefix, context.Process.Command, context.GetStatusAsStr())
						}
						tbl.Print()
						fmt.Println()
					}

				case "exit":
					fmt.Println("Exiting all")
					for _, context := range runContexts {
						context.BuzzkillProcess()
					}
					break input_loop

				default:
					if len(s) < 2 {
						fmt.Println("No input provided")
						continue
					}
					found := false
					input := s[1:]
					for _, context := range runContexts {
						if context.Process.Name == target || context.Process.Prefix == target {
							found = true
							if context.Status == pp.ProcessStatusRunning {
								context.Write(strings.Join(input, ""))
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

		// Start the tasks
		for _, context := range runContexts {
			context.Start()
		}

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

func init() {
	rootCmd.Flags().StringSliceVarP(&execCommands, "execute", "e", execCommands, "Execute command (can be used multiple times)")
}
