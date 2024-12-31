package tests

import (
	"sync"
	"testing"
	"time"

	pp "github.com/mpmcintyre/process-party/internal"
	testHelpers "github.com/mpmcintyre/process-party/test_helpers"
)

// Creates a process with non-default values and waiting values
func createWaitProcess(command string, args []string, startDelay int) pp.Process {
	var tpExit pp.ExitCommand = "wait"
	var tpColour pp.ColourCode = "blue"
	var tpDelays int = 0

	return pp.Process{
		Name:    "wait",
		Command: command,
		Args:    args,
		Prefix:  "wait",

		Color:      tpColour,
		DisplayPid: true,
		Status:     pp.ProcessStatusRunning,
		Silent:     true,
		// These must be set by the config file not the process
		ShowTimestamp:    false,
		SeperateNewLines: false,
		Delay:            startDelay,
		RestartAttempts:  0,
		OnFailure:        tpExit,
		OnComplete:       tpExit,
		RestartDelay:     tpDelays,
		TimeoutOnExit:    tpDelays,
	}
}

// Creates a process with non-default values
func createRestartProcess(command string, args []string, restartAttempts int, restartDelay int) pp.Process {
	var tpExit pp.ExitCommand = "restart"
	var tpColour pp.ColourCode = "yellow"
	var tpDelays int = 0

	return pp.Process{
		Name:       "restart",
		Command:    command,
		Args:       args,
		Prefix:     "restart",
		Color:      tpColour,
		DisplayPid: true,
		Status:     pp.ProcessStatusRunning,
		Silent:     true,
		// These must be set by the config file not the process
		ShowTimestamp:    true,
		SeperateNewLines: true,
		Delay:            tpDelays,
		RestartAttempts:  restartAttempts,
		OnFailure:        tpExit,
		OnComplete:       tpExit,
		RestartDelay:     restartDelay,
		TimeoutOnExit:    tpDelays,
	}
}

// Creates a process with non-default values
func createBuzzkillProcess(command string, args []string) pp.Process {
	var tpExit pp.ExitCommand = "buzzkill"
	var tpColour pp.ColourCode = "green"
	var tpDelays int = 0

	return pp.Process{
		Name:       "buzzkill",
		Command:    command,
		Args:       args,
		Prefix:     "buzzkill",
		Color:      tpColour,
		DisplayPid: true,
		Status:     pp.ProcessStatusRunning,
		Silent:     true,
		// These must be set by the config file not the process
		ShowTimestamp:    false,
		SeperateNewLines: false,
		Delay:            tpDelays,
		RestartAttempts:  0,
		OnFailure:        tpExit,
		OnComplete:       tpExit,
		RestartDelay:     tpDelays,
		TimeoutOnExit:    tpDelays,
	}
}

func TestInternalBuzzkill(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup

	// Test that each one works by creating a file with the name of the process
	// delay := 100 //ms
	cmdSettings := testHelpers.CreateFailCmdSettings()
	buzzkillTask := createBuzzkillProcess(cmdSettings.Cmd, cmdSettings.Args)

	wg.Add(1)

	// Create the task output channels
	taskChannel := pp.TaskChannelsOut{
		Buzzkill:    make(chan bool),
		ProcessExit: make(chan int),
	}

	context := buzzkillTask.CreateContext(
		&wg,
		taskChannel,
	)

	buzzkilled := false
	go func() {
		<-taskChannel.ProcessExit
		t.Log("EOC recieved")
	}()
	go func() {
		buzzkilled = <-taskChannel.Buzzkill
		t.Log("Buzkill recieved")
	}()
	go context.Start()
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
	buzzkillTask := createBuzzkillProcess(cmdSettings.Cmd, cmdSettings.Args)

	wg.Add(1)

	// Create the task output channels
	taskChannel := pp.TaskChannelsOut{
		Buzzkill:    make(chan bool),
		ProcessExit: make(chan int),
	}

	context := buzzkillTask.CreateContext(
		&wg,
		taskChannel,
	)
	go context.Start()
	time.Sleep(time.Duration(delay) * time.Millisecond)
	t1 := time.Now()
	context.Buzzkill()
	wg.Wait()
	if time.Since(t1) > time.Duration(sleepDuration)*time.Second {
		t.Fatal("Process ran to completion. Context did not exit on buzzkill")
	}
	if context.Process.Status == pp.ProcessStatusRunning {
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
	taskChannel := pp.TaskChannelsOut{
		Buzzkill:    make(chan bool),
		ProcessExit: make(chan int),
	}

	context := completeTask.CreateContext(
		&wg,
		taskChannel,
	)
	buzzkilled := false
	go context.Start()
	t1 := time.Now()
	go func() {
		<-taskChannel.ProcessExit
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
	if context.Process.Status == pp.ProcessStatusRunning {
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
	taskChannel := pp.TaskChannelsOut{
		Buzzkill:    make(chan bool),
		ProcessExit: make(chan int),
	}

	context := sleepTask.CreateContext(
		&wg,
		taskChannel,
	)
	buzzkilled := false
	go context.Start()
	t1 := time.Now()
	go func() {
		for attempt := range restartAttempts {
			exit := <-taskChannel.ProcessExit
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
	if context.Process.Status == pp.ProcessStatusRunning {
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
	taskChannel := pp.TaskChannelsOut{
		Buzzkill:    make(chan bool),
		ProcessExit: make(chan int),
	}

	context := restartTask.CreateContext(
		&wg,
		taskChannel,
	)
	buzzkilled := false
	go context.Start()
	t1 := time.Now()
	go func() {
		for attempt := range restartAttempts {
			exit := <-taskChannel.ProcessExit
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
	if context.Process.Status == pp.ProcessStatusRunning {
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
	taskChannel := pp.TaskChannelsOut{
		Buzzkill:    make(chan bool),
		ProcessExit: make(chan int),
	}

	context := waitTask.CreateContext(
		&wg,
		taskChannel,
	)
	buzzkilled := false
	go context.Start()
	t1 := time.Now()
	go func() {
		<-taskChannel.ProcessExit
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
	if context.Process.Status == pp.ProcessStatusRunning {
		t.Fatal("Context run status is still running.")
	}
}
