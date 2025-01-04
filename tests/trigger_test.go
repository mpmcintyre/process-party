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

func TestFsTriggersBasic(t *testing.T) {
	t.Parallel()
	tempDir := "./.tmp/fstrigger"
	subDir := "./.tmp/fstrigger/subfolder"
	filename := "triggerFile"
	createdFiles := 3
	createdFilesInSubdirectory := 3
	createdDirectories := 3
	expectedRuns := 6
	cmdSettings := testHelpers.CreateSleepCmdSettings(0)
	process := createBaseProcess(cmdSettings.Cmd, cmdSettings.Args, 0, 0)

	var wg sync.WaitGroup
	context := process.CreateContext(&wg)
	context.Process.Trigger.FileSystem.Watch = []string{tempDir}
	context.Process.Trigger.FileSystem.Ignore = []string{}
	context.Process.Trigger.FileSystem.ContainFilters = []string{}

	os.RemoveAll(tempDir)
	os.MkdirAll(tempDir, fs.ModeDir)

	pp.LinkProcessTriggers([]*pp.ExecutionContext{context})

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

	go func() {
		time.Sleep(time.Duration(100) * time.Millisecond)
		// Create files in watched directory
		for i := range createdFiles {
			time.Sleep(time.Duration(50) * time.Millisecond)
			os.Create(tempDir + "/" + filename + strconv.Itoa(i))
		}
		// Make sure the trigger does not run when an empty directory is created
		for i := range createdDirectories {
			time.Sleep(time.Duration(50) * time.Millisecond)
			os.Mkdir(tempDir+"/"+filename+strconv.Itoa(i), fs.ModeDir)
		}
		// Create subdirectory for creating files inside a subdirectory
		os.MkdirAll(subDir, fs.ModeDir)
		for i := range createdDirectories {
			time.Sleep(time.Duration(50) * time.Millisecond)
			os.Create(subDir + "/" + filename + strconv.Itoa(i))
		}

		time.Sleep(time.Duration(1000) * time.Millisecond)
		context.BuzzkillProcess()
	}()

	context.Start()

	wg.Wait()

	assert.True(t, exitRecieved.Load(), "Should recieve exit signal")
	assert.False(t, failedRecieved.Load())
	assert.True(t, startRecieved.Load())
	assert.False(t, restartRecieved.Load())
	assert.True(t, notStartedRecieved.Load())
	assert.True(t, waitingForTrigger.Load())
	assert.False(t, unknownRecieved.Load())

	assert.False(t, buzzkilled)
	assert.Equal(t, expectedRuns, runCounter)

	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, createdDirectories+createdFilesInSubdirectory, len(files))

}
