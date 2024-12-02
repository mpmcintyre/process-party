package runner

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	color "github.com/fatih/color"
)

type (
	ExitCommand string
	ColourCode  string

	Process struct {
		Name             string      `toml:"name"`
		Command          string      `toml:"command"`
		Args             []string    `toml:"args"`
		Prefix           string      `toml:"prefix"`
		Color            ColourCode  `toml:"color"`
		OnFailure        ExitCommand `toml:"on_failure"`
		OnComplete       ExitCommand `toml:"on_complete,omitempty"`
		SeperateNewLines bool
		ShowTimestamp    bool
	}

	Config struct {
		Processes        []Process `toml:"processes"`
		SeperateNewLines bool      `toml:"indicate_every_line"`
		ShowTimestamp    bool      `toml:"show_timestamp"`
		filePresent      bool
	}
)

const (
	ExitCommandBuzzkill ExitCommand = "buzzkill"
	ExitCommandWait     ExitCommand = "wait"
	ExitCommandRestart  ExitCommand = "restart"
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

	_, err2 := toml.Decode(text, &c)
	if err2 != nil {
		return err2
	}

	fmt.Printf("Found %d processes in %s\n", len(c.Processes), path)
	for i := range c.Processes {
		fmt.Printf("%#v\n", c.Processes[i].Name)
		c.Processes[i].SeperateNewLines = c.SeperateNewLines
		c.Processes[i].ShowTimestamp = c.ShowTimestamp
	}
	c.filePresent = true
	return nil
}
