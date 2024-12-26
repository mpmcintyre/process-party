package tests

import (
	"sync"
	"testing"
	"time"

	runner "github.com/mpmcintyre/process-party/internal"
	testHelpers "github.com/mpmcintyre/process-party/test_helpers"
)

// Creates a process with non-default values and waiting values
func createWaitProcess(command string, args []string, startDelay int) runner.RunTask {
	var tpExit runner.ExitCommand = "wait"
	var tpColour runner.ColourCode = "blue"
	var tpDelays int = 0

	return runner.RunTask{
		Process: runner.Process{
			Name:    "wait",
			Command: command,
			Args:    args,
			Prefix:  "wait",

			Color:      tpColour,
			DisplayPid: true,
			Delay:      startDelay,
			Status:     runner.ExitStatusRunning,
			Silent:     true,
			// These must be set by the config file not the process
			ShowTimestamp:    false,
			SeperateNewLines: false,
		},
		ProcessExitContext: runner.ProcessExitContext{
			RestartAttempts: 0,
			OnFailure:       tpExit,
			OnComplete:      tpExit,
			RestartDelay:    tpDelays,
			TimeoutOnExit:   tpDelays,
		},
	}
}

// Creates a process with non-default values
func createRestartProcess(command string, args []string, restartAttempts int, restartDelay int) runner.RunTask {
	var tpExit runner.ExitCommand = "restart"
	var tpColour runner.ColourCode = "yellow"
	var tpDelays int = 0

	return runner.RunTask{
		Process: runner.Process{
			Name:       "restart",
			Command:    command,
			Args:       args,
			Prefix:     "restart",
			Color:      tpColour,
			DisplayPid: true,
			Delay:      tpDelays,
			Status:     runner.ExitStatusRunning,
			Silent:     true,
			// These must be set by the config file not the process
			ShowTimestamp:    true,
			SeperateNewLines: true,
		},
		ProcessExitContext: runner.ProcessExitContext{
			RestartAttempts: restartAttempts,
			OnFailure:       tpExit,
			OnComplete:      tpExit,
			RestartDelay:    restartDelay,
			TimeoutOnExit:   tpDelays,
		},
	}
}

// Creates a process with non-default values
func createBuzzkillProcess(command string, args []string) runner.RunTask {
	var tpExit runner.ExitCommand = "buzzkill"
	var tpColour runner.ColourCode = "green"
	var tpDelays int = 0

	return runner.RunTask{
		Process: runner.Process{
			Name:       "buzzkill",
			Command:    command,
			Args:       args,
			Prefix:     "buzzkill",
			Color:      tpColour,
			DisplayPid: true,
			Delay:      tpDelays,
			Status:     runner.ExitStatusRunning,
			Silent:     true,
			// These must be set by the config file not the process
			ShowTimestamp:    false,
			SeperateNewLines: false,
		},
		ProcessExitContext: runner.ProcessExitContext{
			RestartAttempts: 0,
			OnFailure:       tpExit,
			OnComplete:      tpExit,
			RestartDelay:    tpDelays,
			TimeoutOnExit:   tpDelays,
		},
	}
}

func TestInternalBuzzkill(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup

	// Test that each one works by creating a file with the name of the process
	// delay := 100 //ms
	cmdSettings := testHelpers.CreateFailCmdSettings()
	buzkillTask := createBuzzkillProcess(cmdSettings.Cmd, cmdSettings.Args)

	wg.Add(1)

	// Create the task output channels
	taskChannel := runner.TaskChannelsOut{
		Buzzkill:   make(chan bool),
		ExitStatus: make(chan int),
	}

	mainChannel := runner.MainChannelsOut{
		Buzzkill: make(chan bool),
		StdIn:    make(chan string),
	}

	context := buzkillTask.CreateContext(
		&wg,
		mainChannel,
		taskChannel,
	)

	buzzkilled := false
	go func() {
		<-taskChannel.ExitStatus
		t.Log("EOC recieved")
	}()
	go func() {
		buzzkilled = <-taskChannel.Buzzkill
		t.Log("Buzkill recieved")
	}()
	go context.Run()
	wg.Wait()

	if !buzzkilled {
		t.Fatal("Process ran to completion. Context did not buzzkill on exit")
	}
}

func TestExternalBuzzkill(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup

	// Test that each one works by creating a file with the name of the process
	sleepDuration := 5 // Seconds
	delay := 100       //ms

	if delay/1000 > sleepDuration/2 {
		t.Fatalf("delay duration cannot be larger than sleepDuration/2, delay=%d ms, sleep=%d s", delay, sleepDuration)
	}
	cmdSettings := testHelpers.CreateSleepCmdSettings(sleepDuration)
	buzkillTask := createBuzzkillProcess(cmdSettings.Cmd, cmdSettings.Args)

	wg.Add(1)

	// Create the task output channels
	taskChannel := runner.TaskChannelsOut{
		Buzzkill:   make(chan bool),
		ExitStatus: make(chan int),
	}

	mainChannel := runner.MainChannelsOut{
		Buzzkill: make(chan bool),
		StdIn:    make(chan string),
	}

	context := buzkillTask.CreateContext(
		&wg,
		mainChannel,
		taskChannel,
	)
	go context.Run()
	time.Sleep(time.Duration(delay) * time.Millisecond)
	t1 := time.Now()
	mainChannel.Buzzkill <- true
	wg.Wait()
	if time.Since(t1) > time.Duration(sleepDuration)*time.Second {
		t.Fatal("Process ran to completion. Context did not exit on buzzkill")
	}
	if context.Task.Status == runner.ExitStatusRunning {
		t.Fatal("Context run status is still running. Context did not exit on buzzkill")
	}
}

func TestWait(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup

	// Test that each one works by creating a file with the name of the process
	sleepDuration := 1 // Seconds
	delay := 100       //ms

	if delay/1000 > sleepDuration/2 {
		t.Fatalf("delay duration cannot be larger than sleepDuration/2, delay=%d ms, sleep=%d s", delay, sleepDuration)
	}
	cmdSettings := testHelpers.CreateSleepCmdSettings(sleepDuration)
	completeTask := createWaitProcess(cmdSettings.Cmd, cmdSettings.Args, 0)

	wg.Add(1)

	// Create the task output channels
	taskChannel := runner.TaskChannelsOut{
		Buzzkill:   make(chan bool),
		ExitStatus: make(chan int),
	}

	mainChannel := runner.MainChannelsOut{
		Buzzkill: make(chan bool),
		StdIn:    make(chan string),
	}

	context := completeTask.CreateContext(
		&wg,
		mainChannel,
		taskChannel,
	)
	buzzkilled := false
	go context.Run()
	t1 := time.Now()
	go func() {
		<-taskChannel.ExitStatus
		t.Log("EOC recieved")
	}()
	go func() {
		buzzkilled = <-taskChannel.Buzzkill
		t.Log("Buzkill recieved")
	}()
	wg.Wait()
	if time.Since(t1) < time.Duration(sleepDuration) {
		t.Fatal("Process did not run to completion")
	}
	if buzzkilled {
		t.Fatal("Context recieved buzzkill")

	}
	if context.Task.Status == runner.ExitStatusRunning {
		t.Fatal("Context run status is still running.")
	}
}

func TestRestart(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup

	// Test that each one works by creating a file with the name of the process
	sleepDuration := 1 // Seconds
	restartAttempts := 3

	cmdSettings := testHelpers.CreateSleepCmdSettings(sleepDuration)
	sleepTask := createRestartProcess(cmdSettings.Cmd, cmdSettings.Args, restartAttempts, 0)
	sleepTask.Silent = true
	wg.Add(1)

	// Create the task output channels
	taskChannel := runner.TaskChannelsOut{
		Buzzkill:   make(chan bool),
		ExitStatus: make(chan int),
	}

	mainChannel := runner.MainChannelsOut{
		Buzzkill: make(chan bool),
		StdIn:    make(chan string),
	}

	context := sleepTask.CreateContext(
		&wg,
		mainChannel,
		taskChannel,
	)
	buzzkilled := false
	go context.Run()
	t1 := time.Now()
	go func() {
		for attempt := range restartAttempts {
			exit := <-taskChannel.ExitStatus
			t.Logf("Attept %d - Exit status %d recieved\n", attempt+1, exit)
		}
	}()
	go func() {
		for attempt := range restartAttempts {
			buzzkilled = <-taskChannel.Buzzkill
			t.Logf("Attept %d - buzzkill recieved\n", attempt+1)
		}
	}()
	wg.Wait()
	if time.Since(t1) < time.Duration(sleepDuration*restartAttempts)*time.Second {
		t.Fatal("Process did not run to completion")
	}
	if buzzkilled {
		t.Fatal("Context recieved buzzkill")

	}
	if context.Task.Status == runner.ExitStatusRunning {
		t.Fatal("Context run status is still running.")
	}
}

func TestRestartWithDelays(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup

	// Test that each one works by creating a file with the name of the process
	restartDelay := 1 // Seconds
	restartAttempts := 3

	cmdSettings := testHelpers.CreateFailCmdSettings()
	restartTask := createRestartProcess(cmdSettings.Cmd, cmdSettings.Args, restartAttempts, restartDelay)
	restartTask.Prefix = "restart-delay"
	restartTask.Silent = true

	wg.Add(1)

	// Create the task output channels
	taskChannel := runner.TaskChannelsOut{
		Buzzkill:   make(chan bool),
		ExitStatus: make(chan int),
	}

	mainChannel := runner.MainChannelsOut{
		Buzzkill: make(chan bool),
		StdIn:    make(chan string),
	}

	context := restartTask.CreateContext(
		&wg,
		mainChannel,
		taskChannel,
	)
	buzzkilled := false
	go context.Run()
	t1 := time.Now()
	go func() {
		for attempt := range restartAttempts {
			exit := <-taskChannel.ExitStatus
			t.Logf("Attept %d - Exit status %d recieved\n", attempt+1, exit)
		}
	}()
	go func() {
		for attempt := range restartAttempts {
			buzzkilled = <-taskChannel.Buzzkill
			t.Logf("Attept %d - buzzkill recieved\n", attempt+1)
		}
	}()
	wg.Wait()
	if time.Since(t1) < time.Duration(restartDelay*restartAttempts)*time.Second {
		t.Fatal("Process did not run to completion")
	}
	if buzzkilled {
		t.Fatal("Context recieved buzzkill")

	}
	if context.Task.Status == runner.ExitStatusRunning {
		t.Fatal("Context run status is still running.")
	}
}

func TestStartDelay(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup

	// Test that each one works by creating a file with the name of the process
	sleepDuration := 1 // Seconds
	delay := 100       //ms

	if delay/1000 > sleepDuration/2 {
		t.Fatalf("delay duration cannot be larger than sleepDuration/2, delay=%d ms, sleep=%d s", delay, sleepDuration)
	}
	cmdSettings := testHelpers.CreateSleepCmdSettings(0)
	waitTask := createWaitProcess(cmdSettings.Cmd, cmdSettings.Args, 0)
	waitTask.Delay = sleepDuration

	wg.Add(1)

	// Create the task output channels
	taskChannel := runner.TaskChannelsOut{
		Buzzkill:   make(chan bool),
		ExitStatus: make(chan int),
	}

	mainChannel := runner.MainChannelsOut{
		Buzzkill: make(chan bool),
		StdIn:    make(chan string),
	}

	context := waitTask.CreateContext(
		&wg,
		mainChannel,
		taskChannel,
	)
	buzzkilled := false
	go context.Run()
	t1 := time.Now()
	go func() {
		<-taskChannel.ExitStatus
		t.Log("EOC recieved")
	}()
	go func() {
		buzzkilled = <-taskChannel.Buzzkill
		t.Log("Buzkill recieved")
	}()
	wg.Wait()
	if time.Since(t1) < time.Duration(sleepDuration) {
		t.Fatal("Process did not run to completion")
	}
	if buzzkilled {
		t.Fatal("Context recieved buzzkill")
	}
	if context.Task.Status == runner.ExitStatusRunning {
		t.Fatal("Context run status is still running.")
	}
}
