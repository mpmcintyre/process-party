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
var generateConfig *bool

func createSectionHeading(length int, character string, title string) string {
	wraplength := (length - len(title)) / 2
	wrap := strings.Repeat(character, wraplength)
	title = wrap + title + wrap
	return title
}

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

		// Print ascii art
		color.HiGreen("   ___                             ___           __      \n  / _ \\_______  _______ ___ ___   / _ \\___ _____/ /___ __\n / ___/ __/ _ \\/ __/ -_|_-<(_-<  / ___/ _ `/ __/ __/ // /\n/_/  /_/  \\___/\\__/\\__/___/___/ /_/   \\_,_/_/  \\__/\\_, / \n                                                  /___/  ")

		sectionHeadingLength := 80
		headingChar := "-"
		// Create the configuration to store settings and process configurations
		config := pp.CreateConfig()

		fmt.Println()
		color.HiBlack(createSectionHeading(sectionHeadingLength, headingChar, "Parsing inputs"))

		// If the user wishes to generate an empty config, asist in generating a config
		if *generateConfig {
			path := "process-party.yml"
			if len(args) != 0 {
				path = args[0]
			}
			err := config.GenerateExampleConfig(path)
			return err
		}

		// Parse the input file if the user passes in an argument
		if len(args) != 0 {
			// Parse the input file path
			err := config.ParseFile(args[0], false)
			if err != nil {
				return err
			}
		} else {
			// Check if there is a process-party file in parent dir

			targetFile, err := config.ScanDir(".")
			if err != nil {
				return err
			}
			if targetFile != "" {
				err := config.ParseFile(targetFile, false)
				if err != nil {
					return err
				}
			}
		}

		// Parse the inline commands (-e or --execute flag)
		for _, cmd := range execCommands {
			err := config.ParseInlineCmd(cmd)
			if err != nil {
				return err
			}
		}

		color.HiBlack("Input is active - std in to commands using [all] or specific command using [<cmd prefix>]")
		color.HiBlack("Get the status using \"status\" or \"s\", or quit the party using \"exit\" or ctrl+c")
		fmt.Println()
		color.HiBlack(createSectionHeading(sectionHeadingLength, headingChar, "Linking triggers"))
		fmt.Println()

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

				// Apply shorthands
				if target == "s" {
					target = "status"
				}

				switch target {
				case "all":
					if len(s) < 2 {
						color.HiBlack("No input provided")
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

				case "help":
					color.HiBlack(`The input allows you to view the status of all
commands with the "status" command, pipe input to a
specific command using <command name|command prefix>:<input>
e.g. "cmd:echo hello", or pipe input to all commands using 
"all:<input>". Gracefully shutdown all processes using ctrl+c 
or input "exit" into the command line.`)

				case "exit":
					color.HiBlack("Exiting all")
					for _, context := range runContexts {
						context.BuzzkillProcess()
					}
					break input_loop

				default:
					if len(s) < 2 {
						color.HiBlack("No input provided")
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
								fmt.Printf("The %s command is %s, cannot write to process\n", target, context.GetStatusAsStr())
							}
						}
					}
					if !found {
						fmt.Printf("%s not found", target)
					}
				}
			}
		}()

		fmt.Println()
		color.HiBlack(createSectionHeading(sectionHeadingLength, headingChar, "Launching"))
		fmt.Println()
		if len(runContexts) == 0 {
			return errors.New("no processes to run")
		}

		// Start the tasks
		for _, context := range runContexts {
			context.Start()
		}

		// // Listen to signals
		// sigc := make(chan os.Signal, 1)
		// signal.Notify(sigc,
		// 	syscall.SIGHUP,
		// 	syscall.SIGINT,
		// 	syscall.SIGTERM,
		// 	syscall.SIGQUIT)
		// go func() {
		// 	<-sigc
		// 	color.HiBlack("Recieved exit signal, exiting all")
		// 	for _, context := range runContexts {
		// 		context.BuzzkillProcess()
		// 	}
		// }()

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
	generateConfig = rootCmd.Flags().BoolP("generate", "g", false, "Generate blank config")
}
