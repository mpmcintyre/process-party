package runner

import (
	"io"
	"os"
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

// Create the private variables for the command to start running, MUST BE CALLED INSIDE context.Run()
func (c *BaseContext) setupCmd() {
	c.cmd = exec.Command(c.process.Command, c.process.Args...)
	c.cmd.Env = os.Environ() // Set the full environment, including PATH
	// Create IO
	r, w := io.Pipe()
	c.readPipe = r
	c.writePipe = w
	// Write into the command
	c.infoWriter = &customWriter{w: os.Stdout, severity: "info", process: c.process}   // Write info out
	c.errorWriter = &customWriter{w: os.Stdout, severity: "error", process: c.process} // Write errors out
	// Set IO
	c.cmd.Stdout = c.infoWriter
	c.cmd.Stderr = c.errorWriter
	c.cmd.Stdin = r
}
