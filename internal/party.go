package party

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type (
	Process struct {
		Name       string   `toml:"name"`
		Command    string   `toml:"command"`
		Args       []string `toml:"args"`
		Prefix     string   `toml:"prefix"`
		Color      string   `toml:"color"`
		OnFailure  string   `toml:"on_failure"`
		OnComplete string   `toml:"on_complete,omitempty"`
	}

	// Updated struct to match TOML structure
	Config struct {
		Processes []Process `toml:"processes"`
	}

	Command struct {
		Process      Process
		StdOut       chan string
		EndOfCommand chan bool
		BuzzKill     chan bool
	}

	Commander struct {
		Commands []Command
	}
)

func New() *Config {
	return &Config{
		Processes: []Process{},
	}
}

func (c *Config) AddFile(path string) error {
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
	fmt.Println(text)
	for _, c := range c.Processes {
		fmt.Printf("%#v\n", c.Name)
	}
	return nil
}
