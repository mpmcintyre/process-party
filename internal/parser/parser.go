package parser

import "fmt"

type OnCommandEnd int

const (
	Buzzkill OnCommandEnd = iota
	Wait
	IrishGoodBye
	Restart
)

type Command struct {
	Cmd    string
	Color  string
	Prefix string
	Silent bool
	OnEnd  OnCommandEnd
}

type CliCommands struct {
	Commands  []Command
	LogOutput bool
}

func Run() (*CliCommands, error) {

	fmt.Print("HELLOO")
	return nil, nil
}
