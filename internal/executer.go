package pp

// Custom writers-  https://medium.com/@shubhamagrawal094/custom-writer-in-golang-171dd2cac7e0
import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Creates an execution context
func (p *Process) CreateContext(wg *sync.WaitGroup, tc TaskChannelsOut) ExecutionContext {
	return ExecutionContext{
		Process:               p,
		wg:                    wg,
		TaskChannelsOut:       tc,
		internalExitNotifiers: make([]chan bool, 0),
		stdIn:                 make(chan string),
	}
}

// Tell all goroutines related to this process to stop and exit
func (e *ExecutionContext) Buzzkill() {
	for _, channel := range e.internalExitNotifiers {
		if channel != nil {
			channel <- true
		}
	}
}

func (e *ExecutionContext) getBuzzkillChannel() chan bool {
	channel := make(chan bool)
	e.internalExitNotifiers = append(e.internalExitNotifiers, channel)
	return channel
}

func (e *ExecutionContext) handleProcessExit() {
	exitCommand := ExitCommandWait
	if e.Process.Status == ProcessStatusFailed || e.Process.Status == ProcessStatusNotStarted {
		e.errorWriter.Write([]byte("Process failed"))
		exitCommand = e.Process.OnFailure
	} else {
		e.errorWriter.Write([]byte("Process exited"))
		exitCommand = e.Process.OnComplete
	}

	switch exitCommand {
	case ExitCommandBuzzkill:
		e.errorWriter.Write([]byte("Buzzkilling"))
		e.exitStatus = ExitStatusBuzzkilled
		e.TaskChannelsOut.Buzzkill <- true
	case ExitCommandRestart:
		// We cannot reuse the c.cmd, so we use recursion with counters
		// Remove one attempt (negative numbers imply to always restart)
		if e.Process.RestartAttempts+1 > 0 {
			e.Process.RestartAttempts = e.Process.RestartAttempts - 1
		}

		if e.Process.RestartAttempts == 0 {
			e.infoWriter.Write([]byte("No restart attempts left, exiting"))
			return
		}
		e.Process.Status = ProcessStatusRestarting
		e.infoWriter.Write([]byte(fmt.Sprintf("Process exited - Restarting, %d second restart delay, %d attempts remaining", e.Process.RestartDelay, e.Process.RestartAttempts)))
		if e.Process.RestartDelay > 0 {
			time.Sleep(time.Duration(e.Process.RestartDelay) * time.Second)
		}
		// Add to the wait group, create a new context, and run the new command syncrounously
		// This might cause some issues if he process needs to restart indefinitely
		e.wg.Add(1)

		e.execute()

		// x := e.Process.CreateContext(
		// 	e.wg, e.TaskChannelsOut,
		// )
		// x.Start()
	}
}

func (e *ExecutionContext) Write(input string) {
	e.stdIn <- input
}

func (config *Config) GenerateRunTaskContexts(wg *sync.WaitGroup) []ExecutionContext {

	// Create context and channel groups
	contexts := []ExecutionContext{}

	for index, process := range config.Processes {

		// Create the task output channels
		taskChannel := TaskChannelsOut{
			Buzzkill:    make(chan bool),
			ProcessExit: make(chan int),
		}

		// Create context
		newContext := process.CreateContext(
			wg,
			taskChannel,
		)
		contexts = append(contexts, newContext)

		// Start listening to the threads channels fo multi-channel communcation
		go func() {
		monitorLoop:
			for {
				select {
				case <-taskChannel.Buzzkill:
					for i := range len(config.Processes) {
						// Send to all other channels (not including this one)
						if i != index {
							contexts[i].Buzzkill()
							// mainChannels[i].buzzkill <- true
						}
					}
					break monitorLoop
				case <-taskChannel.ProcessExit:
					if newContext.Process.OnComplete != ExitCommandRestart &&
						newContext.Process.OnComplete != ExitCommandBuzzkill {
						break monitorLoop
					}
					// if runningProcessCount <= 0 {
					// 	fmt.Println("All processes exited")
					// 	break monitorLoop
					// }
				case <-newContext.getBuzzkillChannel():
					break monitorLoop
				}
			}

		}()
	}

	return contexts
}

// Communication from task to main thread
type (
	ExecutionExitStatus int

	TaskChannelsOut struct {
		Buzzkill    chan bool
		ProcessExit chan int
	}

	// All contexts of running processes will have these fields
	ExecutionContext struct {
		cmd                   *exec.Cmd
		infoWriter            *customWriter
		errorWriter           *customWriter
		readPipe              *io.PipeReader //
		writePipe             *io.PipeWriter
		Process               *Process
		TaskChannelsOut       TaskChannelsOut
		wg                    *sync.WaitGroup
		exitStatus            ExecutionExitStatus
		internalExitNotifiers []chan bool // All related internal goroutines should lock onto this notifier to exit when the process is killed
		stdIn                 chan string
	}
)

const (
	ExitStatusOk ExecutionExitStatus = iota
	ExitStatusBuzzkilled
)

func (c *ExecutionContext) execute() {

	// Create command
	c.cmd = exec.Command(c.Process.Command, c.Process.Args...)
	c.cmd.Env = os.Environ() // Set the full environment, including PATH
	// Create IO
	c.readPipe, c.writePipe = io.Pipe()

	// Write into the command
	c.infoWriter = &customWriter{w: os.Stdout, severity: "info", process: c.Process}   // Write info out
	c.errorWriter = &customWriter{w: os.Stdout, severity: "error", process: c.Process} // Write errors out
	// Set IO
	c.cmd.Stdout = c.infoWriter
	c.cmd.Stderr = c.errorWriter
	c.cmd.Stdin = c.readPipe

	displayedPid := false // Simple bool to show boolean at the start of the process
	c.Process.Status = ProcessStatusRunning

	// Wait for the start delay
	c.infoWriter.Write([]byte(fmt.Sprintf("Starting process - %d second delay", c.Process.Delay)))
	if c.Process.Delay > 0 {
		time.Sleep(time.Duration(c.Process.Delay) * time.Second)
	}
	// Start the command
	startErr := c.cmd.Start()
	// Go wait somewhere else lamo (*insert you cant sit with us meme*)
	go c.cmd.Wait()

	buzzkillChannel := c.getBuzzkillChannel()

commandLoop:
	for {
		select {
		case value := <-c.stdIn: // Received std in
			c.writePipe.Write([]byte(value + "\n"))
		case <-buzzkillChannel: // Recieved buzzkill
			c.exitStatus = ExitStatusBuzzkilled
			c.Process.Status = ProcessStatusExited
			c.infoWriter.Write([]byte("Recieved buzzkill command"))
			// Wait for timeout_on_exit duration
			startTime := time.Now()
			timeout := time.Duration(c.Process.TimeoutOnExit) * time.Second
			if c.cmd.Process != nil {
				c.cmd.Process.Signal(os.Kill)
			}
			for {
				elapsed := time.Since(startTime)

				if elapsed > timeout {
					c.infoWriter.Write([]byte("Gracefull shutdown timed out - killing process"))
					if c.cmd.Process != nil {
						c.cmd.Process.Kill()
					}
					break
				}
				if c.cmd.ProcessState != nil {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}

			break commandLoop

		default:
			// Display the PID on the first line
			if !displayedPid && c.cmd.Process != nil {
				c.infoWriter.Write([]byte(fmt.Sprintf("PID = %d", c.cmd.Process.Pid)))
				c.Process.Pid = fmt.Sprintf("%d", c.cmd.Process.Pid)
				displayedPid = true
			}

			// Stream the initial start stream values and set it to an empty string
			if c.Process.StartStream != "" {
				c.writePipe.Write([]byte(c.Process.StartStream + "\n"))
				c.Process.StartStream = ""
			}

			// Handle the process exiting
			if c.cmd.ProcessState.ExitCode() >= 0 || startErr != nil {
				if c.cmd.ProcessState.ExitCode() == 0 {
					c.exitStatus = ExitStatusOk
					c.Process.Status = ProcessStatusExited
				} else if c.cmd.ProcessState.ExitCode() > 0 {
					c.errorWriter.Write([]byte("Detected Process failure"))
					c.Process.Status = ProcessStatusFailed
				} else if startErr != nil {
					c.errorWriter.Write([]byte("Failed to start"))
					c.errorWriter.Write([]byte(startErr.Error()))
					c.Process.Status = ProcessStatusNotStarted
				} else {
					c.Process.Status = ProcessStatusExited
				}

				// Note! This will block if not listened to in tests
				c.TaskChannelsOut.ProcessExit <- c.cmd.ProcessState.ExitCode()
				c.handleProcessExit()

				// If the end commands are anything else we dont give a shit
				// If the process is restarted this should be false
				break commandLoop
			}

			// Don't spin too hard
			time.Sleep(10 * time.Millisecond)

		}
	}
	c.writePipe.Close()
	c.readPipe.Close()
	c.cmd.Wait()
	// The process exits so quick we need to delay to ensure that the buzzkill command is sent
	time.Sleep(time.Duration(10) * time.Millisecond)
	c.wg.Done()
}

func (e *ExecutionContext) Start() {
	// buzzkill := c.getBuzzkillChannel()
	fsTrigger, done := e.CreateFsTrigger()

	triggers := make([]chan bool, 0)
	if fsTrigger != nil {
		triggers = append(triggers, make(chan bool))
	}

	// TODO: Check for process trigger here too
	if len(triggers) == 0 {
		e.execute()
	} else {
		e.execute()

		// monitorLoop:
		// 	for {
		// 		for _, trigger := range triggers {
		// 			select {
		// 			case <-trigger:
		// 				e.execute()
		// 				// Break monitoring if buzzkill commited
		// 				if e.exitStatus != ExitStatusOk {
		// 					break monitorLoop
		// 				}

		// 			case <-buzzkill:
		// 				break monitorLoop
		// 			}

		// 		}
		// 	}
	}

	done <- true
}
