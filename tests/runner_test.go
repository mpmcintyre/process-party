package tests

import (
	"sync"
	"testing"
	"time"

	runner "github.com/mpmcintyre/process-party/internal"
	testHelpers "github.com/mpmcintyre/process-party/test_helpers"
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

func TestExternalBuzzkill(t *testing.T) {
	var wg sync.WaitGroup

	// Test that each one works by creating a file with the name of the process
	sleepDuration := 5 // Seconds
	delay := 100       //ms

	if delay/1000 > sleepDuration/2 {
		t.Fatalf("delay duration cannot be larger than sleepDuration/2, delay=%d ms, sleep=%d s", delay, sleepDuration)
	}
	cmdSettings := testHelpers.CreateSleepCmdSettings(sleepDuration)
	buzkillProcess := createBuzzkillProcess(cmdSettings.Cmd, cmdSettings.Args)

	wg.Add(1)

	// Create the task output channels
	taskChannel := runner.TaskChannelsOut{
		Buzzkill:     make(chan bool),
		EndOfCommand: make(chan string),
	}

	mainChannel := runner.MainChannelsOut{
		Buzzkill: make(chan bool),
		StdIn:    make(chan string),
	}

	context := runner.CreateContext(
		&buzkillProcess,
		&wg,
		mainChannel,
		taskChannel,
	)
	go context.Run()
	time.Sleep(time.Duration(delay) * time.Millisecond)
	t1 := time.Now()
	mainChannel.Buzzkill <- true
	wg.Wait()
	if time.Since(t1) < time.Duration(sleepDuration) {
		t.Fatal("Process ran to completion. Context did not exit on buzzkill")
	}
	if context.Process.Status == runner.ExitStatusRunning {
		t.Fatal("Context run status is still running. Context did not exit on buzzkill")
	}
}

func TestWait(t *testing.T) {

}

func TestRestart(t *testing.T) {

}
