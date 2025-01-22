package tests

import (
	"context"
	"os"
	"path/filepath"
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
		ShowTimestamp:   true,
		Delay:           tpDelays,
		RestartAttempts: restartAttempts,
		OnFailure:       tpExit,
		OnComplete:      tpExit,
		RestartDelay:    restartDelay,
		TimeoutOnExit:   tpDelays,
	}
}

// Ensure the trigger linker works as intended
func TestLinkErrors(t *testing.T) {
	t.Parallel()

	existingDirPath := filepath.Join(".tmp", "existingDir")
	nonExistentDir := filepath.Join(".tmp", "i-no-existo")
	existingProcessName := "linkTest1"
	nonExistingProcessName := "non-link-testslol"
	numberOfProcesses := 10

	err := os.Mkdir(existingDirPath, 0755)
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

	err = pp.LinkProcessTriggers(contexts)
	assert.Nil(t, err)

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
		{"Allow duplicate triggers", false, []string{existingDirPath, existingDirPath}, []string{}, []string{}, 0},
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

	err = pp.LinkProcessTriggers(contexts)
	assert.Nil(t, err)

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
		{"Allow duplicate triggers", false, []string{existingProcessName, existingProcessName}, []string{}, []string{}, 0},
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

	contexts = []*pp.ExecutionContext{}
	for i := range numberOfProcesses {
		cmdSettings := testHelpers.CreateSleepCmdSettings(0)
		process := createBaseProcess(cmdSettings.Cmd, cmdSettings.Args, 0, 0, "linkTest"+strconv.Itoa(i))
		process.Silent = true
		context := process.CreateContext(&wg)
		contexts = append(contexts, context)
	}

	t.Run("Recursive Process OnComplete", func(t *testing.T) {
		err := pp.LinkProcessTriggers(contexts)
		assert.Nil(t, err)
		contexts[0].Process.Trigger.Process.OnComplete = []string{contexts[1].Process.Name}
		contexts[1].Process.Trigger.Process.OnComplete = []string{contexts[2].Process.Name}
		contexts[2].Process.Trigger.Process.OnComplete = []string{contexts[0].Process.Name}
		err = pp.LinkProcessTriggers(contexts)
		assert.Nil(t, err)
	})

	contexts = []*pp.ExecutionContext{}
	for i := range numberOfProcesses {
		cmdSettings := testHelpers.CreateSleepCmdSettings(0)
		process := createBaseProcess(cmdSettings.Cmd, cmdSettings.Args, 0, 0, "linkTest"+strconv.Itoa(i))
		process.Silent = true
		context := process.CreateContext(&wg)
		contexts = append(contexts, context)
	}

	t.Run("Recursive Process OnError", func(t *testing.T) {
		err := pp.LinkProcessTriggers(contexts)
		assert.Nil(t, err)
		contexts[0].Process.Trigger.Process.OnError = []string{contexts[1].Process.Name}
		contexts[1].Process.Trigger.Process.OnError = []string{contexts[2].Process.Name}
		contexts[2].Process.Trigger.Process.OnError = []string{contexts[0].Process.Name}
		err = pp.LinkProcessTriggers(contexts)
		assert.Nil(t, err)
	})

	contexts = []*pp.ExecutionContext{}
	for i := range numberOfProcesses {
		cmdSettings := testHelpers.CreateSleepCmdSettings(0)
		process := createBaseProcess(cmdSettings.Cmd, cmdSettings.Args, 0, 0, "linkTest"+strconv.Itoa(i))
		process.Silent = true
		context := process.CreateContext(&wg)
		contexts = append(contexts, context)
	}

	t.Run("Recursive Process OnStart", func(t *testing.T) {
		err := pp.LinkProcessTriggers(contexts)
		assert.Nil(t, err)
		contexts[0].Process.Trigger.Process.OnStart = []string{contexts[1].Process.Name}
		contexts[1].Process.Trigger.Process.OnStart = []string{contexts[2].Process.Name}
		contexts[2].Process.Trigger.Process.OnStart = []string{contexts[0].Process.Name}
		err = pp.LinkProcessTriggers(contexts)
		assert.Nil(t, err)
	})

	t.Cleanup(func() {
		os.RemoveAll(existingDirPath)
	})

}

// Test basic default functionality (recursive watching and allowing it to run to completion)
func TestFsTriggersBasic(t *testing.T) {
	t.Parallel()
	tempDir := filepath.Join(".tmp", "triggers", "basic")
	subDir := filepath.Join(".tmp", "triggers", "basic", "subfolder")
	filename := "triggerFile"
	createdFiles := 3
	createdFilesInSubdirectory := 3
	createdDirectories := 3
	expectedRuns := 10
	triggerInterval := 60
	cmdSettings := testHelpers.CreateSleepCmdSettings(0)
	process := createBaseProcess(cmdSettings.Cmd, cmdSettings.Args, 0, 0, "trigger")
	process.Silent = true

	var wg sync.WaitGroup
	context := process.CreateContext(&wg)
	context.Process.Trigger.FileSystem.Watch = []string{tempDir}
	context.Process.Trigger.FileSystem.Ignore = []string{}
	context.Process.Trigger.FileSystem.ContainFilters = []string{}

	os.RemoveAll(tempDir)
	err := os.MkdirAll(tempDir, 0755)
	assert.Nil(t, err, "Could not create the temp folder")

	err = pp.LinkProcessTriggers([]*pp.ExecutionContext{context})

	assert.Nil(t, err, "Error when creating trigger links")
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

	ended := make(chan bool)

	go func() {
	monitorLoop:
		for {
			value, ok := <-notificationsChannel
			if ok {
				switch value {
				case pp.ProcessStatusExited:
					exitRecieved.Store(true)
					ended <- true
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
			f, err := os.Create(filepath.Join(tempDir, filename+strconv.Itoa(i)))
			if err != nil {
				t.Errorf("Failed to create file %s: %v", filepath.Join(tempDir, filename+strconv.Itoa(i)), err)
				continue
			}
			f.Sync()
			f.Close()
			if i < createdFiles {
				<-ended
			}
		}
		// Make sure the trigger does not run when an empty directory is created
		for i := range createdDirectories {
			os.Mkdir(filepath.Join(tempDir, filename+"dir"+strconv.Itoa(i)), 0755)
			if i < createdDirectories {
				<-ended
			}
		}
		// Create subdirectory for creating files inside a subdirectory
		time.Sleep(time.Duration(triggerInterval) * time.Millisecond)
		os.MkdirAll(subDir, 0755)
		<-ended

		for i := range createdFilesInSubdirectory {
			f, err := os.Create(filepath.Join(subDir, filename+strconv.Itoa(i)))
			if err != nil {
				t.Errorf("Failed to create file %s: %v", filepath.Join(subDir, filename+strconv.Itoa(i)), err)
				continue
			}
			f.Sync()
			f.Close()
			if i < createdFilesInSubdirectory {
				<-ended
			}
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

// Test basic default functionality (recursive watching and allowing it to run to completion)
func TestFsOnStart(t *testing.T) {
	t.Parallel()
	tempDir := filepath.Join(".tmp", "triggers", "fs_on_start")
	expectedRunsA := 1
	expectedRunsB := 0
	cmdSettings := testHelpers.CreateSleepCmdSettings(0)

	processA := createBaseProcess(cmdSettings.Cmd, cmdSettings.Args, 0, 0, "triggerA")
	processA.Prefix = "triggerA"
	processA.Trigger.RunOnStart = true
	processA.Silent = true

	processB := createBaseProcess(cmdSettings.Cmd, cmdSettings.Args, 0, 0, "triggerB")
	processB.Prefix = "triggerB"
	processB.Trigger.RunOnStart = false
	processB.Silent = true

	var wg sync.WaitGroup
	contextA := processA.CreateContext(&wg)
	contextA.Process.Trigger.FileSystem.Watch = []string{tempDir}

	contextB := processB.CreateContext(&wg)
	contextB.Process.Trigger.FileSystem.Watch = []string{tempDir}

	os.RemoveAll(tempDir)
	err := os.MkdirAll(tempDir, 0755)
	assert.Nil(t, err, "Could not create the temp folder")

	err = pp.LinkProcessTriggers([]*pp.ExecutionContext{contextA, contextB})

	assert.Nil(t, err, "Error when creating trigger links")
	notificationsChannelA := contextA.GetProcessNotificationChannel()
	notificationsChannelB := contextB.GetProcessNotificationChannel()
	buzzkillChannelA := contextA.GetBuzkillEmitter()
	buzzkillChannelB := contextB.GetBuzkillEmitter()

	buzzkilledA := false
	buzzkilledB := false
	runCounterA := 0
	runCounterB := 0

	var exitRecievedA atomic.Bool
	var failedRecievedA atomic.Bool
	var startRecievedA atomic.Bool
	var restartRecievedA atomic.Bool
	var notStartedRecievedA atomic.Bool
	var waitingForTriggerA atomic.Bool
	var unknownRecievedA atomic.Bool

	var exitRecievedB atomic.Bool
	var failedRecievedB atomic.Bool
	var startRecievedB atomic.Bool
	var restartRecievedB atomic.Bool
	var notStartedRecievedB atomic.Bool
	var waitingForTriggerB atomic.Bool
	var unknownRecievedB atomic.Bool

	go func() {
	monitorLoop:
		for {
			value, ok := <-notificationsChannelA
			if ok {
				switch value {
				case pp.ProcessStatusExited:
					exitRecievedA.Store(true)
				case pp.ProcessStatusFailed:
					failedRecievedA.Store(true)
				case pp.ProcessStatusRunning:
					runCounterA++
					startRecievedA.Store(true)
				case pp.ProcessStatusRestarting:
					restartRecievedA.Store(true)
				case pp.ProcessStatusNotStarted:
					notStartedRecievedA.Store(true)
				case pp.ProcessStatusWaitingTrigger:
					waitingForTriggerA.Store(true)
				default:
					t.Log("Undeclared state recieved: " + contextA.GetStatusAsStr())
					unknownRecievedA.Store(true)
				}
			} else {
				break monitorLoop
			}

		}
	}()

	go func() {
	monitorLoop:
		for {
			value, ok := <-notificationsChannelB
			if ok {
				switch value {
				case pp.ProcessStatusExited:
					exitRecievedB.Store(true)
				case pp.ProcessStatusFailed:
					failedRecievedB.Store(true)
				case pp.ProcessStatusRunning:
					runCounterB++
					startRecievedB.Store(true)
				case pp.ProcessStatusRestarting:
					restartRecievedB.Store(true)
				case pp.ProcessStatusNotStarted:
					notStartedRecievedB.Store(true)
				case pp.ProcessStatusWaitingTrigger:
					waitingForTriggerB.Store(true)
				default:
					t.Log("Undeclared state recieved: " + contextB.GetStatusAsStr())
					unknownRecievedB.Store(true)
				}
			} else {
				break monitorLoop
			}

		}
	}()

	go func() {
		buzzkilledA = <-buzzkillChannelA
	}()

	go func() {
		buzzkilledB = <-buzzkillChannelB
	}()

	contextA.Start()
	contextB.Start()

	go func() {
		time.Sleep(time.Duration(1000) * time.Millisecond)
		contextA.BuzzkillProcess()
		contextB.BuzzkillProcess()
	}()

	wg.Wait()

	assert.True(t, exitRecievedA.Load(), "Should recieve exit signal")
	assert.False(t, failedRecievedA.Load(), "Should not recieve failed signal")
	assert.True(t, startRecievedA.Load(), "Should recieve running signal")
	assert.False(t, restartRecievedA.Load(), "Should not recieve restarted signal")
	assert.True(t, notStartedRecievedA.Load(), "Should recieve not started signal")
	assert.True(t, waitingForTriggerA.Load(), "Should recieve waiting for trigger")
	assert.False(t, unknownRecievedA.Load(), "Should not recieve uknown signal")

	assert.False(t, buzzkilledA, "Should not recieve buzzkill signal")
	assert.Equal(t, expectedRunsA, runCounterA, "Should run the on every propper trigger")

	assert.False(t, exitRecievedB.Load(), "Should recieve exit signal")
	assert.False(t, failedRecievedB.Load(), "Should not recieve failed signal")
	assert.False(t, startRecievedB.Load(), "Should recieve running signal")
	assert.False(t, restartRecievedB.Load(), "Should not recieve restarted signal")
	assert.False(t, notStartedRecievedB.Load(), "Should recieve not started signal")
	assert.True(t, waitingForTriggerB.Load(), "Should recieve waiting for trigger")
	assert.False(t, unknownRecievedB.Load(), "Should not recieve uknown signal")

	assert.False(t, buzzkilledB, "Should not recieve buzzkill signal")
	assert.Equal(t, expectedRunsB, runCounterB, "Should run the on every propper trigger")

	t.Cleanup(func() {
		time.Sleep(time.Duration(1000) * time.Millisecond)
		time.Sleep(time.Duration(100) * time.Millisecond)
		err = os.RemoveAll(tempDir)
		if err != nil {
		}
	})
}

// The trigger should not have more runs if the process is currently running
func TestDebounce(t *testing.T) {
	t.Parallel()

	const (
		tempDir         = ".tmp/triggers/debounce"
		filename        = "triggerFile"
		createdFiles    = 10
		expectedRuns    = 1
		triggerInterval = 50 * time.Millisecond
		debounceTime    = 100
		runtimeSec      = 1
		setupWaitTime   = 100 * time.Millisecond
		cleanupWaitTime = time.Second
		processTimeout  = 5 * time.Second // Safety timeout
	)

	// Ensure clean state
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Use a channel to signal cleanup completion
		done := make(chan error, 1)
		go func() {
			time.Sleep(cleanupWaitTime)
			done <- os.RemoveAll(tempDir)
		}()

		select {
		case err := <-done:
			if err != nil {
				t.Logf("Cleanup failed: %v", err)
			}
		case <-ctx.Done():
			t.Logf("Cleanup timed out")
		}
	})

	// Setup
	if err := os.RemoveAll(tempDir); err != nil {
		t.Fatalf("Failed to cleanup before test: %v", err)
	}
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	cmdSettings := testHelpers.CreateSleepCmdSettings(runtimeSec)
	process := createBaseProcess(cmdSettings.Cmd, cmdSettings.Args, 0, 0, "trigger")
	process.Silent = true
	process.Trigger.FileSystem.DebounceTime = debounceTime

	var wg sync.WaitGroup
	context := process.CreateContext(&wg)
	context.Process.Trigger.FileSystem.Watch = []string{tempDir}
	context.Process.Trigger.FileSystem.Ignore = []string{}
	context.Process.Trigger.FileSystem.ContainFilters = []string{}

	if err := pp.LinkProcessTriggers([]*pp.ExecutionContext{context}); err != nil {
		t.Fatalf("Failed to link process triggers: %v", err)
	}

	notificationsChannel := context.GetProcessNotificationChannel()
	buzzkillChannel := context.GetBuzkillEmitter()

	// Use atomic values for thread-safe state tracking
	states := struct {
		exit           atomic.Bool
		failed         atomic.Bool
		start          atomic.Bool
		restart        atomic.Bool
		notStarted     atomic.Bool
		waitingTrigger atomic.Bool
		unknown        atomic.Bool
		runCounter     atomic.Int32
	}{}

	go func() {
		for value := range notificationsChannel {
			switch value {
			case pp.ProcessStatusExited:
				states.exit.Store(true)
			case pp.ProcessStatusFailed:
				states.failed.Store(true)
			case pp.ProcessStatusRunning:
				states.runCounter.Add(1)
				states.start.Store(true)
			case pp.ProcessStatusRestarting:
				states.restart.Store(true)
			case pp.ProcessStatusNotStarted:
				states.notStarted.Store(true)
			case pp.ProcessStatusWaitingTrigger:
				states.waitingTrigger.Store(true)
			default:
				t.Logf("Undeclared state received: %s", context.GetStatusAsStr())
				states.unknown.Store(true)
			}
		}
	}()

	// Monitor buzzkill
	var buzzkilled atomic.Bool
	go func() {
		killed := <-buzzkillChannel
		buzzkilled.Store(killed)
	}()

	// Start process and wait for initial setup
	context.Start()
	time.Sleep(setupWaitTime)

	// Create files with synchronized completion
	fileCreationDone := make(chan struct{})
	go func() {
		defer close(fileCreationDone)
		for i := 0; i < createdFiles; i++ {
			filePath := filepath.Join(tempDir, filename+strconv.Itoa(i))
			f, err := os.Create(filePath)
			if err != nil {
				t.Errorf("Failed to create file %s: %v", filePath, err)
				return
			}
			if err := f.Sync(); err != nil {
				t.Errorf("Failed to sync file %s: %v", filePath, err)
			}
			f.Close()
			time.Sleep(triggerInterval)
		}
	}()

	// Wait for file creation to complete and additional runtime
	<-fileCreationDone
	time.Sleep(time.Duration(runtimeSec) * time.Second)

	// Cleanup and verify
	context.BuzzkillProcess()
	wg.Wait()

	// Assert final states
	assert.True(t, states.exit.Load(), "Should receive at least one exit signal")
	assert.False(t, states.failed.Load(), "Should not receive failed signal")
	assert.True(t, states.start.Load(), "Should receive running signal")
	assert.False(t, states.restart.Load(), "Should not receive restarted signal")
	assert.True(t, states.notStarted.Load(), "Should receive not started signal")
	assert.True(t, states.waitingTrigger.Load(), "Should receive waiting for trigger")
	assert.False(t, states.unknown.Load(), "Should not receive unknown signal")
	assert.False(t, buzzkilled.Load(), "Should not receive buzzkill signal")
	assert.Equal(t, expectedRuns, int(states.runCounter.Load()), "Should run on every proper trigger")
}

// The trigger should not have more runs if the process is currently running
func TestTerminationFSOnTrigger(t *testing.T) {
	t.Parallel()

	tempDir := filepath.Join(".tmp", "triggers", "fsterm")
	filename := "triggerFile"
	createdFiles := 10
	triggerIntervals := 50
	runtimeSec := 1
	cmdSettings := testHelpers.CreateSleepCmdSettings(runtimeSec)
	process := createBaseProcess(cmdSettings.Cmd, cmdSettings.Args, 0, 0, "trigger")
	process.Silent = false
	process.Trigger.FileSystem.DebounceTime = 0
	process.Trigger.EndOnNew = true

	assert.Less(t, triggerIntervals*createdFiles/1000, runtimeSec+int(process.Trigger.FileSystem.DebounceTime), "The intervals across all triggers cannot be longer that the total runtime")
	var wg sync.WaitGroup
	context := process.CreateContext(&wg)
	context.Process.Trigger.FileSystem.Watch = []string{tempDir}
	context.Process.Trigger.FileSystem.Ignore = []string{}
	context.Process.Trigger.FileSystem.ContainFilters = []string{}

	os.RemoveAll(tempDir)
	os.MkdirAll(tempDir, 0755)

	err := pp.LinkProcessTriggers([]*pp.ExecutionContext{context})

	assert.Nil(t, err, "File/Folder not found")
	notificationsChannel := context.GetProcessNotificationChannel()
	buzzkillChannel := context.GetBuzkillEmitter()

	buzzkilled := false
	runCounter := 0
	completedCounter := 0

	var exitRecieved atomic.Bool
	var failedRecieved atomic.Bool
	var startRecieved atomic.Bool
	var restartRecieved atomic.Bool
	var notStartedRecieved atomic.Bool
	var waitingForTrigger atomic.Bool
	var unknownRecieved atomic.Bool

	started := make(chan bool)

	go func() {
	monitorLoop:
		for {
			value, ok := <-notificationsChannel
			if ok {
				switch value {
				case pp.ProcessStatusExited:
					exitRecieved.Store(true)
					completedCounter++
				case pp.ProcessStatusFailed:
					failedRecieved.Store(true)
				case pp.ProcessStatusRunning:
					started <- true
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
				t.Log("Monitor loop close")
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
			f, err := os.Create(filepath.Join(tempDir, filename+strconv.Itoa(i)))
			if err != nil {
				t.Errorf("Failed to create file %s: %v", filepath.Join(tempDir, filename+strconv.Itoa(i)), err)
				continue
			}
			f.Sync()
			f.Close()
			if i < createdFiles {
				<-started
			}
		}
		time.Sleep(time.Duration(runtimeSec+1) * time.Second)
		context.BuzzkillProcess()
	}()

	wg.Wait()

	assert.True(t, exitRecieved.Load(), "Should recieve exit signal")
	// assert.True(t, failedRecieved.Load(), "Should not recieve failed signal") // This can be trie or false
	assert.True(t, startRecieved.Load(), "Should recieve running signal")
	assert.False(t, restartRecieved.Load(), "Should not recieve restarted signal")
	assert.True(t, notStartedRecieved.Load(), "Should recieve not started signal")
	assert.True(t, waitingForTrigger.Load(), "Should recieve waiting for trigger")
	assert.False(t, unknownRecieved.Load(), "Should not recieve uknown signal")

	assert.False(t, buzzkilled, "Should not recieve buzzkill signal")
	assert.Equal(t, runCounter, createdFiles, "Should run the on every propper trigger")

	t.Cleanup(func() {
		time.Sleep(time.Duration(1000) * time.Millisecond)
		err = os.RemoveAll(tempDir)
		if err != nil {
		}
	})
}
