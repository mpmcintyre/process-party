package tests

import (
	"io/fs"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pp "github.com/mpmcintyre/process-party/internal"
	testHelpers "github.com/mpmcintyre/process-party/test_helpers"
	"github.com/stretchr/testify/assert"
)

// Creates a process with non-default values
func createBaseProcess(command string, args []string, restartAttempts int, restartDelay int) pp.Process {
	var tpExit pp.ExitCommand = "wait"
	var tpColour pp.ColourCode = "yellow"
	var tpDelays int = 0

	return pp.Process{
		Name:       "trigger",
		Command:    command,
		Args:       args,
		Prefix:     "trigger",
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

func TestFsIlligals(t *testing.T) {
	t.Parallel()

	createProcess := func() {
		tempDir := "./.tmp/triggers/nonExistant"
		cmdSettings := testHelpers.CreateSleepCmdSettings(0)
		process := createBaseProcess(cmdSettings.Cmd, cmdSettings.Args, 0, 0)
		process.Silent = true

		var wg sync.WaitGroup
		context := process.CreateContext(&wg)
		context.Process.Trigger.FileSystem.Watch = []string{tempDir}
		context.Process.Trigger.FileSystem.Ignore = []string{}
		context.Process.Trigger.FileSystem.ContainFilters = []string{}
		err := pp.LinkProcessTriggers([]*pp.ExecutionContext{context})

	}

	assert.NotNil(t, err, "The linking process should errror if a file is not found")

}

// Test basic default functionality (recursive watching and allowing it to run to completion)
func TestFsTriggersBasic(t *testing.T) {
	t.Parallel()
	tempDir := "./.tmp/triggers/basic"
	subDir := "./.tmp/triggers/basic/subfolder"
	filename := "triggerFile"
	createdFiles := 3
	createdFilesInSubdirectory := 3
	createdDirectories := 3
	expectedRuns := 10
	triggerInterval := 50
	cmdSettings := testHelpers.CreateSleepCmdSettings(0)
	process := createBaseProcess(cmdSettings.Cmd, cmdSettings.Args, 0, 0)
	process.Silent = true

	var wg sync.WaitGroup
	context := process.CreateContext(&wg)
	context.Process.Trigger.FileSystem.Watch = []string{tempDir}
	context.Process.Trigger.FileSystem.Ignore = []string{}
	context.Process.Trigger.FileSystem.ContainFilters = []string{}

	os.RemoveAll(tempDir)
	os.MkdirAll(tempDir, fs.ModeDir)

	err := pp.LinkProcessTriggers([]*pp.ExecutionContext{context})

	assert.Nil(t, err, "File/Folder not found")
	notificationsChannel := context.GetProcessNotificationChannel()
	buzzkillChannel := context.GetBuzkillEmitter()

	buzzkilled := false
	runCounter := 0

	var exitRecieved atomic.Bool
	var failedRecieved atomic.Bool
	var startRecieved atomic.Bool
	var restartRecieved atomic.Bool
	var notStartedRecieved atomic.Bool
	var waitingForTrigger atomic.Bool
	var unknownRecieved atomic.Bool

	go func() {
	monitorLoop:
		for {
			value, ok := <-notificationsChannel
			if ok {
				switch value {
				case pp.ProcessStatusExited:
					exitRecieved.Store(true)
				case pp.ProcessStatusFailed:
					failedRecieved.Store(true)
				case pp.ProcessStatusRunning:
					runCounter++
					startRecieved.Store(true)
				case pp.ProcessStatusRestarting:
					restartRecieved.Store(true)
				case pp.ProcessStatusNotStarted:
					notStartedRecieved.Store(true)
				case pp.ProcessStatusWaitingTrigger:
					waitingForTrigger.Store(true)
				default:
					t.Log("Undeclared state recieved: " + context.GetStatusAsStr())
					unknownRecieved.Store(true)
				}
			} else {
				break monitorLoop
			}

		}

	}()

	go func() {
		buzzkilled = <-buzzkillChannel
	}()

	context.Start()

	go func() {
		time.Sleep(time.Duration(100) * time.Millisecond)
		// Create files in watched directory
		for i := range createdFiles {
			time.Sleep(time.Duration(triggerInterval) * time.Millisecond)
			os.Create(tempDir + "/" + filename + strconv.Itoa(i))
		}
		// Make sure the trigger does not run when an empty directory is created
		for i := range createdDirectories {
			time.Sleep(time.Duration(triggerInterval) * time.Millisecond)
			os.Mkdir(tempDir+"/"+filename+"dir"+strconv.Itoa(i), fs.ModeDir)
		}
		// Create subdirectory for creating files inside a subdirectory
		time.Sleep(time.Duration(triggerInterval) * time.Millisecond)
		os.MkdirAll(subDir, fs.ModeDir)
		time.Sleep(time.Duration(triggerInterval) * time.Millisecond)

		for i := range createdFilesInSubdirectory {
			time.Sleep(time.Duration(triggerInterval) * time.Millisecond)
			os.Create(subDir + "/" + filename + strconv.Itoa(i))
		}

		time.Sleep(time.Duration(triggerInterval) * time.Millisecond)
		context.BuzzkillProcess()
	}()

	wg.Wait()

	assert.True(t, exitRecieved.Load(), "Should recieve exit signal")
	assert.False(t, failedRecieved.Load(), "Should not recieve failed signal")
	assert.True(t, startRecieved.Load(), "Should recieve running signal")
	assert.False(t, restartRecieved.Load(), "Should not recieve restarted signal")
	assert.True(t, notStartedRecieved.Load(), "Should recieve not started signal")
	assert.True(t, waitingForTrigger.Load(), "Should recieve waiting for trigger")
	assert.False(t, unknownRecieved.Load(), "Should not recieve uknown signal")

	assert.False(t, buzzkilled, "Should not recieve buzzkill signal")
	assert.Equal(t, expectedRuns, runCounter, "Should run the on every propper trigger")

	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, createdDirectories+createdDirectories+1, len(files))

	t.Cleanup(func() {
		time.Sleep(time.Duration(1000) * time.Millisecond)
		err := os.RemoveAll(subDir)
		if err != nil {
		}
		time.Sleep(time.Duration(100) * time.Millisecond)
		err = os.RemoveAll(tempDir)
		if err != nil {
		}
	})
}

// The trigger should not have more runs if the process is currently running
func TestFsTriggersNoDoubleProcessing(t *testing.T) {
	t.Parallel()
	tempDir := "./.tmp/triggers/double"
	filename := "triggerFile"
	createdFiles := 10
	expectedRuns := 1
	triggerIntervals := 50
	runtimeSec := 1
	cmdSettings := testHelpers.CreateSleepCmdSettings(runtimeSec)
	process := createBaseProcess(cmdSettings.Cmd, cmdSettings.Args, 0, 0)
	process.Silent = true

	assert.Less(t, triggerIntervals*createdFiles/1000, runtimeSec, "The intervals across all triggers cannot be longer that the total runtime")
	var wg sync.WaitGroup
	context := process.CreateContext(&wg)
	context.Process.Trigger.FileSystem.Watch = []string{tempDir}
	context.Process.Trigger.FileSystem.Ignore = []string{}
	context.Process.Trigger.FileSystem.ContainFilters = []string{}

	os.RemoveAll(tempDir)
	os.MkdirAll(tempDir, fs.ModeDir)

	err := pp.LinkProcessTriggers([]*pp.ExecutionContext{context})

	assert.Nil(t, err, "File/Folder not found")
	notificationsChannel := context.GetProcessNotificationChannel()
	buzzkillChannel := context.GetBuzkillEmitter()

	buzzkilled := false
	runCounter := 0

	var exitRecieved atomic.Bool
	var failedRecieved atomic.Bool
	var startRecieved atomic.Bool
	var restartRecieved atomic.Bool
	var notStartedRecieved atomic.Bool
	var waitingForTrigger atomic.Bool
	var unknownRecieved atomic.Bool

	go func() {
	monitorLoop:
		for {
			value, ok := <-notificationsChannel
			if ok {
				switch value {
				case pp.ProcessStatusExited:
					exitRecieved.Store(true)
				case pp.ProcessStatusFailed:
					failedRecieved.Store(true)
				case pp.ProcessStatusRunning:
					runCounter++
					startRecieved.Store(true)
				case pp.ProcessStatusRestarting:
					restartRecieved.Store(true)
				case pp.ProcessStatusNotStarted:
					notStartedRecieved.Store(true)
				case pp.ProcessStatusWaitingTrigger:
					waitingForTrigger.Store(true)
				default:
					t.Log("Undeclared state recieved: " + context.GetStatusAsStr())
					unknownRecieved.Store(true)
				}
			} else {
				break monitorLoop
			}

		}

	}()

	go func() {
		buzzkilled = <-buzzkillChannel
	}()

	context.Start()

	go func() {
		time.Sleep(time.Duration(100) * time.Millisecond)
		// Create files in watched directory
		for i := range createdFiles {
			time.Sleep(time.Duration(triggerIntervals) * time.Millisecond)
			os.Create(tempDir + "/" + filename + strconv.Itoa(i))
		}

		time.Sleep(time.Duration(runtimeSec)*time.Second - time.Duration(triggerIntervals*createdFiles)*time.Millisecond)
		context.BuzzkillProcess()
	}()

	wg.Wait()

	assert.True(t, exitRecieved.Load(), "Should recieve exit signal")
	assert.False(t, failedRecieved.Load(), "Should not recieve failed signal")
	assert.True(t, startRecieved.Load(), "Should recieve running signal")
	assert.False(t, restartRecieved.Load(), "Should not recieve restarted signal")
	assert.True(t, notStartedRecieved.Load(), "Should recieve not started signal")
	assert.True(t, waitingForTrigger.Load(), "Should recieve waiting for trigger")
	assert.False(t, unknownRecieved.Load(), "Should not recieve uknown signal")

	assert.False(t, buzzkilled, "Should not recieve buzzkill signal")
	assert.Equal(t, expectedRuns, runCounter, "Should run the on every propper trigger")

	t.Cleanup(func() {
		time.Sleep(time.Duration(1000) * time.Millisecond)
		err = os.RemoveAll(tempDir)
		if err != nil {
		}
	})
}
