package tests

import (
	"fmt"
	"testing"

	runner "github.com/mpmcintyre/process-party/internal"
)

// Creates a process with non-default values and waiting values
func createWaitProcess(command string, args []string) runner.Process {
	var tpExit runner.ExitCommand = "wait"
	var tpColour runner.ColourCode = "blue"
	var tpDelays int = 1

	return runner.Process{
		Name:            "wait",
		Command:         command,
		Args:            args,
		Prefix:          "wait",
		RestartAttempts: 0,
		Color:           tpColour,
		OnFailure:       tpExit,
		OnComplete:      tpExit,
		DisplayPid:      true,
		Delay:           tpDelays,
		TimeoutOnExit:   tpDelays,
		RestartDelay:    tpDelays,
		Status:          runner.ExitStatusRunning,
		// These must be set by the config file not the process
		ShowTimestamp:    false,
		SeperateNewLines: false,
	}
}

// Creates a process with non-default values
func createRestartProcess(command string, args []string) runner.Process {
	var tpExit runner.ExitCommand = "restart"
	var tpColour runner.ColourCode = "yellow"
	var tpDelays int = 1

	return runner.Process{
		Name:            "restart",
		Command:         command,
		Args:            args,
		Prefix:          "restart",
		RestartAttempts: 0,
		Color:           tpColour,
		OnFailure:       tpExit,
		OnComplete:      tpExit,
		DisplayPid:      true,
		Delay:           tpDelays,
		TimeoutOnExit:   tpDelays,
		RestartDelay:    tpDelays,
		Status:          runner.ExitStatusRunning,
		// These must be set by the config file not the process
		ShowTimestamp:    false,
		SeperateNewLines: false,
	}
}

// Creates a process with non-default values
func createBuzzkillProcess(command string, args []string) runner.Process {
	var tpExit runner.ExitCommand = "buzzkill"
	var tpColour runner.ColourCode = "green"
	var tpDelays int = 1

	return runner.Process{
		Name:            "buzzkill",
		Command:         command,
		Args:            args,
		Prefix:          "buzzkill",
		RestartAttempts: 0,
		Color:           tpColour,
		OnFailure:       tpExit,
		OnComplete:      tpExit,
		DisplayPid:      true,
		Delay:           tpDelays,
		TimeoutOnExit:   tpDelays,
		RestartDelay:    tpDelays,
		Status:          runner.ExitStatusRunning,
		// These must be set by the config file not the process
		ShowTimestamp:    false,
		SeperateNewLines: false,
	}
}

func TestRunner(t *testing.T) {

	// config := runner.CreateConfig()

	// // Test that each one works by creating a file with the name of the process
	// buzkillProcess := createBuzzkillProcess("cmd", []string{"echo", ">>", "buzzkill"})

	// // command := "cmd"
	// args := []
	// config.Processes := append(config.Processes)
	fmt.Printf("Test")
}
