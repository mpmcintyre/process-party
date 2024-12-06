package tests

import (
	"io/fs"
	"os"
	"reflect"
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
	ShowTimestamp:    true,
	Status:           runner.ExitStatusRunning,
	Pid:              tp1PID,
}

var tp2String string = "tp2"
var tp2Exit runner.ExitCommand = "wait"
var tp2Colour runner.ColourCode = "red"
var tp2Delays int = 1
var tp2PID string = "456"
var testProcess2 = runner.Process{
	Name:             tp2String,
	Command:          tp2String,
	Args:             []string{tp2String},
	Prefix:           tp2String,
	Color:            tp2Colour,
	OnFailure:        tp2Exit,
	OnComplete:       tp2Exit,
	SeperateNewLines: true,
	DisplayPid:       true,
	Delay:            tp2Delays,
	TimeoutOnExit:    tp2Delays,
	RestartDelay:     tp2Delays,
	ShowTimestamp:    true,
	Status:           runner.ExitStatusRunning,
	Pid:              tp2PID,
}

var tp3String string = "tp3"
var tp3Exit runner.ExitCommand = "wait"
var tp3Colour runner.ColourCode = "red"
var tp3Delays int = 1
var tp3PID string = "789"
var testProcess3 = runner.Process{
	Name:             tp3String,
	Command:          tp3String,
	Args:             []string{tp3String},
	Prefix:           tp3String,
	Color:            tp3Colour,
	OnFailure:        tp3Exit,
	OnComplete:       tp3Exit,
	SeperateNewLines: true,
	DisplayPid:       true,
	Delay:            tp3Delays,
	TimeoutOnExit:    tp3Delays,
	RestartDelay:     tp3Delays,
	ShowTimestamp:    true,
	Status:           runner.ExitStatusRunning,
	Pid:              tp3PID,
}

func TestConfigParsing(t *testing.T) {

	os.Mkdir("./.tmp", fs.ModeDir)
	defaultProcess := runner.Process{}

	config := runner.CreateConfig()

	if len(config.Processes) > 0 {
		t.Fatalf("non empty processes in config")
	}

	config.Processes = append(config.Processes, testProcess1, testProcess2, testProcess3)

	for _, process := range config.Processes {
		if reflect.DeepEqual(process, defaultProcess) {
			t.Fatalf("default processes in config")
		}
	}

	// Set the global settings in the config to non default values

	t.Cleanup(func() {
		// Remove temp directory
		os.Remove("./.tmp")
	})
}
