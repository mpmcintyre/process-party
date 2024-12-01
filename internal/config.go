package runner

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type (
	ExitCommand string

	OnFailure struct {
		Id      string
		Command ExitCommand
	}
	OnComplete struct {
		Id      string
		Command ExitCommand
	}

	Process struct {
		Name       string      `toml:"name"`
		Command    string      `toml:"command"`
		Args       []string    `toml:"args"`
		Prefix     string      `toml:"prefix"`
		Color      string      `toml:"color"`
		OnFailure  ExitCommand `toml:"on_failure"`
		OnComplete ExitCommand `toml:"on_complete,omitempty"`
	}

	Config struct {
		Processes   []Process `toml:"processes"`
		filePresent bool
	}
)

const (
	ExitCommandBuzzkill ExitCommand = "buzzkill"
	ExitCommandWait     ExitCommand = "wait"
	ExitCommandRestart  ExitCommand = "restart"
)

func CreateConfig() *Config {
	return &Config{}
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
	for _, c := range c.Processes {
		fmt.Printf("%#v\n", c.Name)
	}
	c.filePresent = true
	return nil
}
