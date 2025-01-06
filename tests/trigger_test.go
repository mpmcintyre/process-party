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
func createBaseProcess(command string, args []string, restartAttempts int, restartDelay int, name string) pp.Process {
	var tpExit pp.ExitCommand = "wait"
	var tpColour pp.ColourCode = "yellow"
	var tpDelays int = 0

	return pp.Process{
		Name:       name,
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

func TestLinkErrors(t *testing.T) {
	t.Parallel()

	existingDirPath := "./.tmp/existingDir"
	nonExistentDir := "./.tmp/i-no-existo"
	existingProcessName := "linkTest1"
	nonExistingProcessName := "non-link-testslol"
	numberOfProcesses := 10

	err := os.Mkdir(existingDirPath, fs.ModeDir)
	assert.Nil(t, err, "Have to make a test directory")

	var wg sync.WaitGroup

	contexts := []*pp.ExecutionContext{}
	for i := range numberOfProcesses {
		cmdSettings := testHelpers.CreateSleepCmdSettings(0)
		process := createBaseProcess(cmdSettings.Cmd, cmdSettings.Args, 0, 0, "linkTest"+strconv.Itoa(i))
		process.Silent = true
		context := process.CreateContext(&wg)
		contexts = append(contexts, context)
	}

	fsTests := []struct {
		name            string
		linkErrors      bool
		Watch           []string
		Ignore          []string
		ContainFilters  []string
		restartAttempts int
	}{
		{"No triggers", false, []string{}, []string{}, []string{}, 0},
		{"Single existing dir", false, []string{existingDirPath}, []string{}, []string{}, 0},
		{"Non existent dir", true, []string{nonExistentDir}, []string{}, []string{}, 0},
		{"No restarts allowed - restart once", true, []string{nonExistentDir}, []string{}, []string{}, 1},
		{"No restarts allowed - restart forever", true, []string{nonExistentDir}, []string{}, []string{}, -1},
	}

	t.Run("FileSystem", func(t *testing.T) {
		for _, tt := range fsTests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.restartAttempts != 0 {
					contexts[0].Process.OnComplete = pp.ExitCommandRestart
					contexts[0].Process.RestartAttempts = tt.restartAttempts
				} else {
					contexts[0].Process.OnComplete = pp.ExitCommandWait
					contexts[0].Process.RestartAttempts = 0
				}

				contexts[0].Process.Trigger.FileSystem.Watch = tt.Watch
				contexts[0].Process.Trigger.FileSystem.Ignore = tt.Ignore
				contexts[0].Process.Trigger.FileSystem.ContainFilters = tt.ContainFilters
				err := pp.LinkProcessTriggers(contexts)

				if tt.linkErrors {
					assert.NotNil(t, err, tt.name+" should have errrored when linking")
				} else {
					assert.Nil(t, err, tt.name+" should not have errrored when linking")
				}

			})
		}
	})
	contexts = []*pp.ExecutionContext{}
	for i := range numberOfProcesses {
		cmdSettings := testHelpers.CreateSleepCmdSettings(0)
		process := createBaseProcess(cmdSettings.Cmd, cmdSettings.Args, 0, 0, "linkTest"+strconv.Itoa(i))
		process.Silent = true
		context := process.CreateContext(&wg)
		contexts = append(contexts, context)
	}

	processTests := []struct {
		name            string
		linkErrors      bool
		OnComplete      []string
		OnError         []string
		OnStart         []string
		restartAttempts int
	}{
		{"No triggers", false, []string{}, []string{}, []string{}, 0},
		{"Single process - on complete", false, []string{existingProcessName}, []string{}, []string{}, 0},
		{"Single process - on complete (non-existent)", true, []string{nonExistingProcessName}, []string{}, []string{}, 0},
		{"Single process - on error", false, []string{}, []string{existingProcessName}, []string{}, 0},
		{"Single process - on error (non-existent)", true, []string{}, []string{nonExistingProcessName}, []string{}, 0},
		{"Single process - on start", false, []string{}, []string{}, []string{existingProcessName}, 0},
		{"Single process - on start (non-existent)", true, []string{}, []string{}, []string{nonExistingProcessName}, 0},
		{"Allow multiple triggers", false, []string{existingProcessName}, []string{existingProcessName}, []string{existingProcessName}, 0},
		{"Can't have the same name", true, []string{contexts[0].Process.Name}, []string{}, []string{}, 0},
		{"No restarts allowed - restart once", true, []string{contexts[0].Process.Name}, []string{}, []string{}, 1},
		{"No restarts allowed - restart forever", true, []string{contexts[0].Process.Name}, []string{}, []string{}, -1},
	}

	t.Run("Processes", func(t *testing.T) {
		assert.NotEqual(t, contexts[0].Process.RestartAttempts, -1)
		assert.NotEqual(t, contexts[0].Process.RestartAttempts, 1)
		for _, tt := range processTests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.restartAttempts != 0 {
					contexts[0].Process.OnComplete = pp.ExitCommandRestart
					contexts[0].Process.RestartAttempts = tt.restartAttempts
				} else {
					contexts[0].Process.OnComplete = pp.ExitCommandWait
					contexts[0].Process.RestartAttempts = 0
				}

				contexts[0].Process.Trigger.Process.OnComplete = tt.OnComplete
				contexts[0].Process.Trigger.Process.OnError = tt.OnError
				contexts[0].Process.Trigger.Process.OnStart = tt.OnStart
				err := pp.LinkProcessTriggers(contexts)

				if tt.linkErrors {
					assert.NotNil(t, err, tt.name+" should have errrored when linking")
				} else {
					assert.Nil(t, err, tt.name+" should not have errrored when linking")
				}

			})
			contexts[0].Process.Trigger.Process.OnComplete = []string{}
			contexts[0].Process.Trigger.Process.OnError = []string{}
			contexts[0].Process.Trigger.Process.OnStart = []string{}
		}
	})

	t.Cleanup(func() {
		os.RemoveAll(existingDirPath)
	})

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
	process := createBaseProcess(cmdSettings.Cmd, cmdSettings.Args, 0, 0, "trigger")
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
	process := createBaseProcess(cmdSettings.Cmd, cmdSettings.Args, 0, 0, "trigger")
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
