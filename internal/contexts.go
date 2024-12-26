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
	BaseContext struct {
		cmd         *exec.Cmd
		infoWriter  *customWriter
		errorWriter *customWriter
		readPipe    *io.PipeReader
		writePipe   *io.PipeWriter
		process     *Process
	}

	// Watchers are fire and forget, but pipe all info to the main screen
	WatchTaskContext struct {
		Task *WatchTask
		BaseContext
	}

	// Runners use channels to communicate to the main thread as well as use the wait group
	RunTaskContext struct {
		Task            *RunTask
		MainChannelsOut MainChannelsOut
		TaskChannelsOut TaskChannelsOut
		wg              *sync.WaitGroup
		BaseContext
		// outb, errb   bytes.Buffer
	}
)

func (p *RunTask) CreateContext(wg *sync.WaitGroup, mc MainChannelsOut, tc TaskChannelsOut) RunTaskContext {
	return RunTaskContext{
		Task:            p,
		wg:              wg,
		MainChannelsOut: mc,
		TaskChannelsOut: tc,
		BaseContext: BaseContext{
			process: &p.Process,
		},
	}
}

func (p *WatchTask) CreateContext(wg *sync.WaitGroup, mc MainChannelsOut, tc TaskChannelsOut) WatchTaskContext {
	return WatchTaskContext{
		Task: p,
		BaseContext: BaseContext{
			process: &p.Process,
		},
	}
}
