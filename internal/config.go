package runner

import (
	"encoding/json"
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
		Name             string      `toml:"name" json:"name" yaml:"name"`
		Command          string      `toml:"command" json:"command" yaml:"command"`
		Args             []string    `toml:"args" json:"args" yaml:"args"`
		Prefix           string      `toml:"prefix" json:"prefix" yaml:"prefix"`
		Color            ColourCode  `toml:"color" json:"color" yaml:"color"`
		OnFailure        ExitCommand `toml:"on_failure" json:"on_failure" yaml:"on_failure"`
		OnComplete       ExitCommand `toml:"on_complete,omitempty" json:"on_complete,omitempty" yaml:"on_complete,omitempty"`
		SeperateNewLines bool        `json:"seperate_new_lines" yaml:"seperate_new_lines"`
		ShowTimestamp    bool
		Status           ExitStatus
	}
	Config struct {
		Processes        []Process `toml:"processes" json:"processes" yaml:"processes"`
		SeperateNewLines bool      `toml:"indicate_every_line" json:"indicate_every_line" yaml:"indicate_every_line"`
		ShowTimestamp    bool      `toml:"show_timestamp" json:"show_timestamp" yaml:"show_timestamp"`
		filePresent      bool
	}
)

const (
	ExitCommandBuzzkill ExitCommand = "buzzkill"
	ExitCommandWait     ExitCommand = "wait"
	ExitCommandRestart  ExitCommand = "restart"
)

const (
	ExitStatusRunning ExitStatus = iota
	ExitStatusExited
	ExitStatusFailed
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

func CreateConfig() *Config {
	return &Config{
		Processes:        []Process{},
		SeperateNewLines: true,
		ShowTimestamp:    true,
		filePresent:      false,
	}
}

func (c *Config) ParseFile(path string) error {
	buffer, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(buffer)

	extensions := strings.Split(path, ".")
	switch extensions[len(extensions)] {
	case "toml":

		_, err := toml.Decode(text, &c)
		if err != nil {
			return err
		}

		fmt.Printf("Found %d processes in %s\n", len(c.Processes), path)
		for i := range c.Processes {
			fmt.Printf("%#v\n", c.Processes[i].Name)
			c.Processes[i].SeperateNewLines = c.SeperateNewLines
			c.Processes[i].ShowTimestamp = c.ShowTimestamp
		}
		c.filePresent = true

	case "json":
		jsonData, err := json.Marshal(text)
		if err != nil {
			return err
		}
		err = json.Unmarshal(jsonData, &c)
		if err != nil {
			return err
		}

	case "yaml":
		yamlData, err := yaml.Marshal(text)
		if err != nil {
			return err
		}
		err = yaml.Unmarshal(yamlData, &c)
		if err != nil {
			return err
		}
	case "yml":
		yamlData, err := yaml.Marshal(text)
		if err != nil {
			return err
		}
		err = yaml.Unmarshal(yamlData, &c)
		if err != nil {
			return err
		}
	}

	return nil
}
