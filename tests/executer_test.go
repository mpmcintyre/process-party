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
		ShowTimestamp:   false,
		Delay:           startDelay,
		RestartAttempts: 0,
		OnFailure:       tpExit,
		OnComplete:      tpExit,
		RestartDelay:    tpDelays,
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
		ShowTimestamp:   true,
		Delay:           tpDelays,
		RestartAttempts: restartAttempts,
		OnFailure:       tpExit,
		OnComplete:      tpExit,
		RestartDelay:    restartDelay,
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
		ShowTimestamp:   false,
		Delay:           tpDelays,
		RestartAttempts: 0,
		OnFailure:       tpExit,
		OnComplete:      tpExit,
		RestartDelay:    tpDelays,
	}
}

// Ensure that the internal buzzkill command works
func TestInternalBuzzkill(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup
	cmdSettings := testHelpers.CreateFailCmdSettings()
	buzzkillTask := createBuzzkillProcess(cmdSettings.Cmd, cmdSettings.Args)
	context := buzzkillTask.CreateContext(
		&wg,
	)
	buzzkillTask.Silent = true

	var buzzkilled atomic.Bool
	bkChan := context.GetBuzkillEmitter()
	go func() {
		<-bkChan
		buzzkilled.Store(true)
	}()

	context.Start()
	wg.Wait()
	assert.True(t, buzzkilled.Load())
}

// Test if the process exits from an external buzzkill command
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
	buzzkillTask.Silent = true

	context := buzzkillTask.CreateContext(
		&wg,
	)

	context.Start()
	time.Sleep(time.Duration(delay) * time.Millisecond)
	t1 := time.Now()
	context.BuzzkillProcess()
	wg.Wait()
	assert.Less(t, time.Since(t1), time.Duration(sleepDuration)*time.Second)
}

// Test a process successfully exiting and waiting
func TestWait(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup

	// Test that each one works by creating a file with the name of the process
	sleepDuration := 1 // Seconds
	delay := 100       //ms

	if delay/1000 > sleepDuration/2 {
		t.Fatalf("delay duration cannot be larger than sleepDuration/2, delay=%d ms, sleep=%d s", delay, sleepDuration)
	}
	cmdSettings := testHelpers.CreateSleepCmdSettings(0)
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
	var waitingForTriggerRecieved atomic.Bool
	var unknownRecieved atomic.Bool

	go func() {
		for {
			value, ok := <-notificationChannel
			if ok {
				switch value {
				case pp.ProcessStatusExited:
					exitRecieved.Store(true)
				case pp.ProcessStatusFailed:
					failedRecieved.Store(true)
				case pp.ProcessStatusRunning:
					startRecieved.Store(true)
				case pp.ProcessStatusRestarting:
					restartRecieved.Store(true)
				case pp.ProcessStatusNotStarted:
					notStartedRecieved.Store(true)
				case pp.ProcessStatusWaitingTrigger:
					waitingForTriggerRecieved.Store(true)
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
	}()

	context.Start()
	t1 := time.Now()

	wg.Wait()

	// Status checks
	assert.True(t, exitRecieved.Load(), "Should recieve exit status")
	assert.False(t, failedRecieved.Load(), "Should not recieve failed status")
	assert.True(t, startRecieved.Load(), "Should recieve running status")
	assert.False(t, restartRecieved.Load(), "Should not recieve restarting status")
	assert.True(t, notStartedRecieved.Load(), "Should recieve not started status (preparing)")
	assert.False(t, waitingForTriggerRecieved.Load(), "Should not recieve waiting status")
	assert.False(t, unknownRecieved.Load(), "Should not recieve unknown status")
	// Runtime check
	assert.Greater(t, time.Since(t1), time.Duration(sleepDuration), "Process did not run to completion")
	assert.False(t, buzzkilled, "Should not emit buzzkill during test")
	assert.Equal(t, context.Status, pp.ProcessStatusExited)

}

// Test process restart functionality
func TestRestart(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup

	// Test that each one works by creating a file with the name of the process
	sleepDuration := 1 // Seconds
	restartAttempts := 3

	cmdSettings := testHelpers.CreateSleepCmdSettings(0)
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
	var waitingForTriggerRecieved atomic.Bool
	var unknownRecieved atomic.Bool
	processRunCounter := 0

	go func() {
		for {
			value, ok := <-notificationChannel
			if ok {
				switch value {
				case pp.ProcessStatusExited:
					exitRecieved.Store(true)
				case pp.ProcessStatusFailed:
					failedRecieved.Store(true)
				case pp.ProcessStatusRunning:
					processRunCounter++
					startRecieved.Store(true)
				case pp.ProcessStatusRestarting:
					restartRecieved.Store(true)
				case pp.ProcessStatusNotStarted:
					notStartedRecieved.Store(true)
				case pp.ProcessStatusWaitingTrigger:
					waitingForTriggerRecieved.Store(true)
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
	}()

	context.Start()
	t1 := time.Now()
	wg.Wait()

	assert.True(t, exitRecieved.Load(), "Should recieve exit status")
	assert.False(t, failedRecieved.Load(), "Should not recieve failed status")
	assert.True(t, startRecieved.Load(), "Should recieve running status")
	assert.True(t, restartRecieved.Load(), "Should recieve restart status")
	assert.True(t, notStartedRecieved.Load(), "Should recieve not started status (preparing)")
	assert.False(t, waitingForTriggerRecieved.Load(), "Should not recieve waiting status")
	assert.False(t, unknownRecieved.Load(), "Should not recieve unknown status")

	assert.Equal(t, restartAttempts, processRunCounter, "Should have run the amount of times restarted")
	assert.Greater(t, time.Since(t1), time.Duration(sleepDuration), "Process did not run to completion")
	assert.False(t, buzzkilled, "Should not emit buzzkill during test")
	assert.Equal(t, pp.ProcessStatusExited, context.Status, "Final status should be exited")
}

// Ensure that restarts have propper delays
func TestRestartWithDelays(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup

	// Test that each one works by creating a file with the name of the process
	restartDelay := 1 // Seconds
	restartAttempts := 4

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
	var waitingForTriggerRecieved atomic.Bool
	var unknownRecieved atomic.Bool

	go func() {
		for {
			value, ok := <-notificationChannel
			if ok {
				switch value {
				case pp.ProcessStatusExited:
					exitRecieved.Store(true)
				case pp.ProcessStatusFailed:
					failedRecieved.Store(true)
				case pp.ProcessStatusRunning:
					startRecieved.Store(true)
				case pp.ProcessStatusRestarting:
					restartRecieved.Store(true)
				case pp.ProcessStatusNotStarted:
					notStartedRecieved.Store(true)
				case pp.ProcessStatusWaitingTrigger:
					waitingForTriggerRecieved.Store(true)
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
	}()

	context.Start()
	t1 := time.Now()
	wg.Wait()

	assert.True(t, exitRecieved.Load(), "Should recieve exit status")
	assert.True(t, failedRecieved.Load())
	assert.True(t, startRecieved.Load(), "Should recieve running status")
	assert.True(t, restartRecieved.Load(), "Should recieve restart status")
	assert.True(t, notStartedRecieved.Load(), "Should recieve not started status (preparing)")
	assert.False(t, waitingForTriggerRecieved.Load(), "Should not recieve waiting status")
	assert.False(t, unknownRecieved.Load(), "Should not recieve unknown status")

	// There is no delay on the starting process, so minus one
	assert.Greater(t, time.Since(t1), time.Duration(restartDelay*restartAttempts-1)*time.Second, "Should take longer than run duration with restart delays")
	assert.False(t, buzzkilled, "Should not emit buzzkill during test")
	assert.Equal(t, pp.ProcessStatusExited, context.Status, "Final status should be exited")
}

// Ensure that the standard process delays actually work
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
	var waitingForTriggerRecieved atomic.Bool
	var unknownRecieved atomic.Bool

	go func() {
		for {
			value, ok := <-notificationChannel
			if ok {
				switch value {
				case pp.ProcessStatusExited:
					exitRecieved.Store(true)
				case pp.ProcessStatusFailed:
					failedRecieved.Store(true)
				case pp.ProcessStatusRunning:
					startRecieved.Store(true)
				case pp.ProcessStatusRestarting:
					restartRecieved.Store(true)
				case pp.ProcessStatusNotStarted:
					notStartedRecieved.Store(true)
				case pp.ProcessStatusWaitingTrigger:
					waitingForTriggerRecieved.Store(true)
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
	}()

	context.Start()
	t1 := time.Now()
	wg.Wait()

	assert.True(t, exitRecieved.Load(), "Should recieve exit status")
	assert.False(t, failedRecieved.Load(), "Should not recieve failed status")
	assert.True(t, startRecieved.Load(), "Should recieve running status")
	assert.False(t, restartRecieved.Load(), "Should not recieve restarting status")
	assert.True(t, notStartedRecieved.Load(), "Should recieve not started status (preparing)")
	assert.False(t, waitingForTriggerRecieved.Load(), "Should not recieve waiting status")
	assert.False(t, unknownRecieved.Load(), "Should not recieve unknown status")

	// Runtime check
	assert.Greater(t, time.Since(t1), time.Duration(sleepDuration), "Process did not run to completion")
	assert.False(t, buzzkilled, "Should not emit buzzkill during test")
	assert.Equal(t, context.Status, pp.ProcessStatusExited)
}
