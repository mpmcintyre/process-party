package tests

import (
	"fmt"
	"testing"

	runner "github.com/mpmcintyre/process-party/internal"
)

var tp1String string = "tp1"
var tp1Exit runner.ExitCommand = "wait"
var tp1Colour runner.ColourCode = "green"
var tp1Delays int = 1
var tp1PID string = "123"
var testProcess1 = runner.Process{
	Name:             tp1String,
	Command:          tp1String,
	Args:             []string{tp1String},
	Prefix:           tp1String,
	Color:            tp1Colour,
	OnFailure:        tp1Exit,
	OnComplete:       tp1Exit,
	SeperateNewLines: true,
	DisplayPid:       true,
	Delay:            tp1Delays,
	TimeoutOnExit:    tp1Delays,
	RestartDelay:     tp1Delays,
	ShowTimestamp:    false,
	Status:           runner.ExitStatusRunning,
	Pid:              tp1PID,
}

func TestConfig(t *testing.T) {
	fmt.Printf("Test")
}
