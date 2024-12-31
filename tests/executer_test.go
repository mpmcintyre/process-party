package tests

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pp "github.com/mpmcintyre/process-party/internal"
	testHelpers "github.com/mpmcintyre/process-party/test_helpers"
	"github.com/stretchr/testify/assert"
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
		SeperateNewLines: true,
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
		SeperateNewLines: true,
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
	cmdSettings := testHelpers.CreateFailCmdSettings()
	buzzkillTask := createBuzzkillProcess(cmdSettings.Cmd, cmdSettings.Args)
	context := buzzkillTask.CreateContext(
		&wg,
	)

	var buzzkilled atomic.Bool
	bkChan := context.GetBuzkillEmitter()
	go func() {
		<-bkChan
		buzzkilled.Store(true)
		t.Log("Buzkill recieved")
	}()
	go context.Start()
	wg.Wait()
	assert.True(t, buzzkilled.Load())
}

func TestExternalBuzzkill(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup

	// Test that each one works by creating a file with the name of the process
	sleepDuration := 5 // Seconds
	delay := 500       //ms

	if delay/1000 > sleepDuration/2 {
		t.Fatalf("delay duration cannot be larger than sleepDuration/2, delay=%d ms, sleep=%d s", delay, sleepDuration)
	}
	cmdSettings := testHelpers.CreateSleepCmdSettings(sleepDuration)
	buzzkillTask := createBuzzkillProcess(cmdSettings.Cmd, cmdSettings.Args)

	context := buzzkillTask.CreateContext(
		&wg,
	)
	go context.Start()
	time.Sleep(time.Duration(delay) * time.Millisecond)
	t1 := time.Now()
	t.Log("Sending buzzkill")
	context.BuzzkillProcess()
	wg.Wait()
	assert.Less(t, time.Since(t1), time.Duration(sleepDuration)*time.Second)
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

	context := completeTask.CreateContext(
		&wg,
	)
	notificationChannel := context.GetProcessNotificationChannel()
	buzzkilled := false
	bkChan := context.GetBuzkillEmitter()
	go func() {
		value := <-notificationChannel
		switch value {
		case pp.ProcessStatusExited:
			t.Log("EOC recieved")
		case pp.ProcessStatusFailed:
			t.Log("Failure signal recieved")
		case pp.ProcessStatusRunning:
			t.Log("Process started running")
		case pp.ProcessStatusRestarting:
			t.Log("Process signalled restarting")
		}
	}()
	go func() {
		buzzkilled = <-bkChan
		t.Log("Buzkill recieved")
	}()
	go context.Start()
	t1 := time.Now()

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

	context := sleepTask.CreateContext(
		&wg,
	)
	notificationChannel := context.GetProcessNotificationChannel()
	buzzkilled := false
	bkChan := context.GetBuzkillEmitter()
	go func() {
		value := <-notificationChannel
		switch value {
		case pp.ProcessStatusExited:
			t.Log("EOC recieved")
		case pp.ProcessStatusFailed:
			t.Log("Failure signal recieved")
		case pp.ProcessStatusRunning:
			t.Log("Process started running")
		case pp.ProcessStatusRestarting:
			t.Log("Process signalled restarting")
		}

	}()
	go func() {
		buzzkilled = <-bkChan
		t.Log("Buzkill recieved")
	}()
	go context.Start()
	t1 := time.Now()

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

	context := restartTask.CreateContext(
		&wg,
	)
	notificationChannel := context.GetProcessNotificationChannel()
	buzzkilled := false
	bkChan := context.GetBuzkillEmitter()
	go func() {
		value := <-notificationChannel
		switch value {
		case pp.ProcessStatusExited:
			t.Log("EOC recieved")
		case pp.ProcessStatusFailed:
			t.Log("Failure signal recieved")
		case pp.ProcessStatusRunning:
			t.Log("Process started running")
		case pp.ProcessStatusRestarting:
			t.Log("Process signalled restarting")
		}

	}()
	go func() {
		buzzkilled = <-bkChan
		t.Log("Buzkill recieved")
	}()
	go context.Start()
	t1 := time.Now()
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

	context := waitTask.CreateContext(
		&wg,
	)
	notificationChannel := context.GetProcessNotificationChannel()
	buzzkilled := false
	bkChan := context.GetBuzkillEmitter()
	go func() {
		value := <-notificationChannel
		switch value {
		case pp.ProcessStatusExited:
			t.Log("EOC recieved")
		case pp.ProcessStatusFailed:
			t.Log("Failure signal recieved")
		case pp.ProcessStatusRunning:
			t.Log("Process started running")
		case pp.ProcessStatusRestarting:
			t.Log("Process signalled restarting")
		}

	}()
	go func() {
		buzzkilled = <-bkChan
		t.Log("Buzkill recieved")
	}()
	go context.Start()
	t1 := time.Now()
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
