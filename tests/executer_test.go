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

	context.Start()
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

	context.Start()
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

	buzzkilled := false
	bkChan := context.GetBuzkillEmitter()
	// Checks for signals
	notificationChannel := context.GetProcessNotificationChannel()
	var exitRecieved atomic.Bool
	var failedRecieved atomic.Bool
	var startRecieved atomic.Bool
	var restartRecieved atomic.Bool
	var notStartedRecieved atomic.Bool
	var unknownRecieved atomic.Bool

	go func() {
		for {
			value, ok := <-notificationChannel
			if ok {
				switch value {
				case pp.ProcessStatusExited:
					exitRecieved.Store(true)
					t.Log("EOC recieved")
				case pp.ProcessStatusFailed:
					failedRecieved.Store(true)
					t.Log("Failure signal recieved")
				case pp.ProcessStatusRunning:
					startRecieved.Store(true)
					t.Log("Process started running")
				case pp.ProcessStatusRestarting:
					restartRecieved.Store(true)
					t.Log("Process signalled restarting")
				case pp.ProcessStatusNotStarted:
					notStartedRecieved.Store(true)
					t.Log("Process signalled not started")
				default:
					unknownRecieved.Store(true)
				}
			} else {
				break
			}
		}

	}()

	go func() {
		buzzkilled = <-bkChan
		t.Log("Buzkill recieved")
	}()

	context.Start()
	t1 := time.Now()

	wg.Wait()

	// Status checks
	assert.True(t, exitRecieved.Load())
	assert.False(t, failedRecieved.Load())
	assert.True(t, startRecieved.Load())
	assert.False(t, restartRecieved.Load())
	assert.True(t, notStartedRecieved.Load())
	assert.False(t, unknownRecieved.Load())
	// Runtime check
	assert.Greater(t, time.Since(t1), time.Duration(sleepDuration))
	assert.False(t, buzzkilled)
	assert.Equal(t, context.Status, pp.ProcessStatusExited)

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
	sleepTask.DisplayPid = false

	context := sleepTask.CreateContext(
		&wg,
	)
	buzzkilled := false
	bkChan := context.GetBuzkillEmitter()
	// Checks for signals
	notificationChannel := context.GetProcessNotificationChannel()
	var exitRecieved atomic.Bool
	var failedRecieved atomic.Bool
	var startRecieved atomic.Bool
	var restartRecieved atomic.Bool
	var notStartedRecieved atomic.Bool
	var unknownRecieved atomic.Bool
	processRunCounter := 0

	go func() {
		for {
			value, ok := <-notificationChannel
			if ok {
				switch value {
				case pp.ProcessStatusExited:
					exitRecieved.Store(true)
					t.Log("EOC recieved")
				case pp.ProcessStatusFailed:
					failedRecieved.Store(true)
					t.Log("Failure signal recieved")
				case pp.ProcessStatusRunning:
					processRunCounter++
					startRecieved.Store(true)
					t.Log("Process started running")
				case pp.ProcessStatusRestarting:
					restartRecieved.Store(true)
					t.Log("Process signalled restarting")
				case pp.ProcessStatusNotStarted:
					notStartedRecieved.Store(true)
					t.Log("Process signalled not started")
				default:
					unknownRecieved.Store(true)
				}
			} else {
				break
			}
		}

	}()

	go func() {
		buzzkilled = <-bkChan
		t.Log("Buzkill recieved")
	}()

	context.Start()
	t1 := time.Now()

	wg.Wait()

	assert.True(t, exitRecieved.Load())
	assert.False(t, failedRecieved.Load())
	assert.True(t, startRecieved.Load())
	assert.True(t, restartRecieved.Load())
	assert.True(t, notStartedRecieved.Load())
	assert.False(t, unknownRecieved.Load())

	assert.Equal(t, restartAttempts, processRunCounter)
	assert.Greater(t, time.Since(t1), time.Duration(sleepDuration))
	assert.False(t, buzzkilled)
	assert.Equal(t, pp.ProcessStatusExited, context.Status)
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

	buzzkilled := false
	bkChan := context.GetBuzkillEmitter()
	// Checks for signals
	notificationChannel := context.GetProcessNotificationChannel()
	var exitRecieved atomic.Bool
	var failedRecieved atomic.Bool
	var startRecieved atomic.Bool
	var restartRecieved atomic.Bool
	var notStartedRecieved atomic.Bool
	var unknownRecieved atomic.Bool

	go func() {
		for {
			value, ok := <-notificationChannel
			if ok {
				switch value {
				case pp.ProcessStatusExited:
					exitRecieved.Store(true)
					t.Log("EOC recieved")
				case pp.ProcessStatusFailed:
					failedRecieved.Store(true)
					t.Log("Failure signal recieved")
				case pp.ProcessStatusRunning:
					startRecieved.Store(true)
					t.Log("Process started running")
				case pp.ProcessStatusRestarting:
					restartRecieved.Store(true)
					t.Log("Process signalled restarting")
				case pp.ProcessStatusNotStarted:
					notStartedRecieved.Store(true)
					t.Log("Process signalled not started")
				default:
					unknownRecieved.Store(true)
				}
			} else {
				break
			}
		}

	}()
	go func() {
		buzzkilled = <-bkChan
		t.Log("Buzkill recieved")
	}()

	context.Start()
	t1 := time.Now()
	wg.Wait()

	assert.True(t, exitRecieved.Load())
	assert.True(t, failedRecieved.Load())
	assert.True(t, startRecieved.Load())
	assert.True(t, restartRecieved.Load())
	assert.True(t, notStartedRecieved.Load())
	assert.False(t, unknownRecieved.Load())

	assert.Greater(t, time.Since(t1), time.Duration(restartDelay*restartAttempts)*time.Second)
	assert.False(t, buzzkilled)
	assert.Equal(t, pp.ProcessStatusExited, context.Status)
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

	buzzkilled := false
	bkChan := context.GetBuzkillEmitter()
	// Checks for signals
	notificationChannel := context.GetProcessNotificationChannel()
	var exitRecieved atomic.Bool
	var failedRecieved atomic.Bool
	var startRecieved atomic.Bool
	var restartRecieved atomic.Bool
	var notStartedRecieved atomic.Bool
	var unknownRecieved atomic.Bool

	go func() {
		for {
			value, ok := <-notificationChannel
			if ok {
				switch value {
				case pp.ProcessStatusExited:
					exitRecieved.Store(true)
					t.Log("EOC recieved")
				case pp.ProcessStatusFailed:
					failedRecieved.Store(true)
					t.Log("Failure signal recieved")
				case pp.ProcessStatusRunning:
					startRecieved.Store(true)
					t.Log("Process started running")
				case pp.ProcessStatusRestarting:
					restartRecieved.Store(true)
					t.Log("Process signalled restarting")
				case pp.ProcessStatusNotStarted:
					notStartedRecieved.Store(true)
					t.Log("Process signalled not started")
				default:
					unknownRecieved.Store(true)
				}
			} else {
				break
			}
		}
	}()
	go func() {
		buzzkilled = <-bkChan
		t.Log("Buzkill recieved")
	}()

	context.Start()
	t1 := time.Now()
	wg.Wait()

	assert.True(t, exitRecieved.Load())
	assert.False(t, failedRecieved.Load())
	assert.True(t, startRecieved.Load())
	assert.False(t, restartRecieved.Load())
	assert.True(t, notStartedRecieved.Load())
	assert.False(t, unknownRecieved.Load())

	if time.Since(t1) < time.Duration(sleepDuration) {
		t.Fatal("Process did not run to completion")
	}
	if buzzkilled {
		t.Fatal("Context recieved buzzkill")
	}
	if context.Status == pp.ProcessStatusRunning {
		t.Fatal("Context run status is still running.")
	}
}
