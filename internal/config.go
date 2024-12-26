package runner

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	color "github.com/fatih/color"
	"gopkg.in/yaml.v3"
)

type (
	ExitCommand string
	ColourCode  string
	ExitStatus  int

	Process struct {
		Name             string     `toml:"name" json:"name" yaml:"name"`
		Command          string     `toml:"command" json:"command" yaml:"command"`
		Args             []string   `toml:"args" json:"args" yaml:"args"`
		Prefix           string     `toml:"prefix" json:"prefix" yaml:"prefix"`
		Color            ColourCode `toml:"color" json:"color" yaml:"color"`
		SeperateNewLines bool       `toml:"seperate_new_lines" json:"seperate_new_lines" yaml:"seperate_new_lines"`
		DisplayPid       bool       `toml:"show_pid" json:"show_pid" yaml:"show_pid"`
		Delay            int        `toml:"delay" json:"delay" yaml:"delay"`
		StartStream      string     `toml:"stdin_on_start" json:"stdin_on_start" yaml:"stdin_on_start"`
		Silent           bool       `toml:"silent" json:"silent" yaml:"silent"`
		ShowTimestamp    bool
		Status           ExitStatus
		Pid              string
	}

	ProcessExitContext struct {
		RestartDelay    int         `toml:"restart_delay" json:"restart_delay" yaml:"restart_delay"`
		OnFailure       ExitCommand `toml:"on_failure" json:"on_failure" yaml:"on_failure"`
		OnComplete      ExitCommand `toml:"on_complete,omitempty" json:"on_complete,omitempty" yaml:"on_complete,omitempty"`
		RestartAttempts int         `toml:"restart_attempts" json:"restart_attempts" yaml:"restart_attempts"`
		TimeoutOnExit   int         `toml:"timeout_on_exit" json:"timeout_on_exit" yaml:"timeout_on_exit"`
	}

	WatcherContext struct {
		Watch  []string `toml:"watch" json:"watch" yaml:"watch"`
		Ingore []string `toml:"ignore" json:"ignore" yaml:"ignore"`
	}

	RunTask struct {
		ProcessExitContext
		Process
	}

	WatchTask struct {
		WatcherContext
		Process
	}

	Config struct {
		Processes        []RunTask   `toml:"processes" json:"processes" yaml:"processes"`
		WatchTasks       []WatchTask `toml:"watchers" json:"watchers" yaml:"watchers"`
		SeperateNewLines bool        `toml:"indicate_every_line" json:"indicate_every_line" yaml:"indicate_every_line"`
		ShowTimestamp    bool        `toml:"show_timestamp" json:"show_timestamp" yaml:"show_timestamp"`
		filePresent      bool
	}
)

const (
	ExitCommandBuzzkill ExitCommand = "buzzkill"
	ExitCommandWait     ExitCommand = "wait"
	ExitCommandRestart  ExitCommand = "restart"
)

const (
	ExitStatusNotStarted ExitStatus = iota
	ExitStatusRunning
	ExitStatusExited
	ExitStatusFailed
	ExitStatusRestarting
)

const (
	ColourCmdYellow  ColourCode = "yellow"
	ColourCmdBlue    ColourCode = "blue"
	ColourCmdGreen   ColourCode = "green"
	ColourCmdRed     ColourCode = "red"
	ColourCmdCyan    ColourCode = "cyan"
	ColourCmdWhite   ColourCode = "white"
	ColourCmdMagenta ColourCode = "magenta"
)

func (p *Process) GetFgColour() func(format string, a ...interface{}) string {
	switch p.Color {
	case ColourCmdYellow:
		return color.YellowString
	case ColourCmdBlue:
		return color.BlueString
	case ColourCmdGreen:
		return color.GreenString
	case ColourCmdRed:
		return color.RedString
	case ColourCmdCyan:
		return color.CyanString
	case ColourCmdWhite:
		return color.WhiteString
	case ColourCmdMagenta:
		return color.MagentaString
	default:
		return color.WhiteString
	}
}

func (c *Process) GetStatusAsStr() string {
	switch c.Status {
	case ExitStatusNotStarted:
		return "Not started"
	case ExitStatusRunning:
		return "Running"
	case ExitStatusExited:
		return "Exited"
	case ExitStatusFailed:
		return "Failed"
	}
	return "Unknown"
}

func CreateConfig() *Config {
	return &Config{
		Processes:        []RunTask{},
		WatchTasks:       []WatchTask{},
		SeperateNewLines: true,
		ShowTimestamp:    true,
		filePresent:      false,
	}
}

func (c *Config) ParseInlineCmd(cmd string) error {
	s := strings.Split(cmd, " ")
	if len(s) == 0 {
		return errors.New("empty command provided")
	}
	command := s[0]
	args := []string{}
	prefix := command
	if len(s) > 1 {
		args = s[:1]
	}

	// Count how many processes have the same name and increment it for the prefix
	count := 0
	for _, process := range c.Processes {
		if strings.Contains(process.Command, command) {
			count++
		}
	}

	if count != 0 {
		prefix = fmt.Sprintf("cmd%d", count)
	}

	p := RunTask{
		ProcessExitContext: ProcessExitContext{
			OnFailure:  ExitCommandBuzzkill,
			OnComplete: ExitCommandWait,
		},
		Process: Process{
			Command: command,
			Args:    args,
			Prefix:  prefix,
		}}
	c.Processes = append(c.Processes, p)

	return nil
}

func (c *Config) ParseFile(path string) error {
	buffer, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(buffer)

	extensions := strings.Split(path, ".")
	switch extensions[len(extensions)-1] {
	case "toml":
		_, err := toml.Decode(text, &c)
		if err != nil {
			return err
		}

	case "json":
		err = json.Unmarshal([]byte(text), &c)
		if err != nil {
			return err
		}

	case "yaml":
		err = yaml.Unmarshal([]byte(text), &c)
		if err != nil {
			return err
		}
	case "yml":
		err = yaml.Unmarshal([]byte(text), &c)
		if err != nil {
			return err
		}
	default:
		return errors.New("unsupported filetype provided")
	}

	fmt.Printf("Found %d processes in %s\nProcesses:\n", len(c.Processes), path)
	fmt.Print("[")
	for i := range c.Processes {
		output := fmt.Sprintf("%#v", c.Processes[i].Name)
		if c.Processes[i].Silent {
			output = fmt.Sprintf("%#v (silent)", c.Processes[i].Name)
		}
		if i == len(c.Processes)-1 {
			fmt.Printf("%s", output)
		} else {
			fmt.Printf("%s, ", output)
		}
		c.Processes[i].SeperateNewLines = c.SeperateNewLines
		c.Processes[i].ShowTimestamp = c.ShowTimestamp
	}
	fmt.Printf("]\n\n")

	fmt.Printf("Found %d watchers in %s\nWatcher tasks:\n", len(c.Processes), path)
	fmt.Print("[")
	for i := range c.Processes {
		output := fmt.Sprintf("%#v", c.Processes[i].Name)
		if c.Processes[i].Silent {
			output = fmt.Sprintf("%#v (silent)", c.Processes[i].Name)
		}
		if i == len(c.Processes)-1 {
			fmt.Printf("%s", output)
		} else {
			fmt.Printf("%s, ", output)
		}
		c.Processes[i].SeperateNewLines = c.SeperateNewLines
		c.Processes[i].ShowTimestamp = c.ShowTimestamp
	}
	fmt.Printf("]\n\n")

	c.filePresent = true

	return nil
}
