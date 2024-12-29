package pp

import (
	"io"
	"os/exec"
	"sync"
)

// Communication from task to main thread
type (
	TaskChannelsOut struct {
		Buzzkill   chan bool
		ExitStatus chan int
	}

	// Communication from main thread to all threads
	MainChannelsOut struct {
		Buzzkill chan bool
		StdIn    chan string
	}

	// All contexts of running processes will have these fields
	RunContext struct {
		cmd             *exec.Cmd
		infoWriter      *customWriter
		errorWriter     *customWriter
		readPipe        *io.PipeReader
		writePipe       *io.PipeWriter
		Process         *Process
		MainChannelsOut MainChannelsOut
		TaskChannelsOut TaskChannelsOut
		wg              *sync.WaitGroup
	}
)

func (p *Process) CreateContext(wg *sync.WaitGroup, mc MainChannelsOut, tc TaskChannelsOut) RunContext {
	return RunContext{
		Process:         p,
		wg:              wg,
		MainChannelsOut: mc,
		TaskChannelsOut: tc,
	}
}

func (config *Config) GenerateRunTaskContexts(wg *sync.WaitGroup) []RunContext {

	// Create context and channel groups
	contexts := []RunContext{}
	mainChannels := []MainChannelsOut{}
	// Keep track of number of running procesess to exit main app
	runningProcessCount := len(config.Processes)

	for index, process := range config.Processes {

		// Create the task output channels
		taskChannel := TaskChannelsOut{
			Buzzkill:   make(chan bool),
			ExitStatus: make(chan int),
		}

		// Create the task input channels
		mainChannels = append(mainChannels,
			MainChannelsOut{
				Buzzkill: make(chan bool),
				StdIn:    make(chan string),
			})

		// Create context
		contexts = append(contexts, process.CreateContext(
			wg,
			mainChannels[index],
			taskChannel,
		))

		// Start listening to the threads channels fo multi-channel communcation
		go func() {
		monitorLoop:
			for {
				select {
				case <-taskChannel.Buzzkill:
					for i := range len(config.Processes) {
						// Send to all other channels (not including this one)
						if i != index {
							mainChannels[i].Buzzkill <- true
						}
					}
					break monitorLoop
				case <-taskChannel.ExitStatus:
					runningProcessCount--
					// if runningProcessCount <= 0 {
					// 	fmt.Println("All processes exited")
					// 	break monitorLoop
					// }
				}
			}

		}()
	}

	return contexts
}
