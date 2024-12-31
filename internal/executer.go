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

// Communication from task to main thread
type (
	ExecutionExitEvent int

	// All contexts of running processes will have these fields
	ExecutionContext struct {
		cmd                      *exec.Cmd
		infoWriter               *customWriter
		errorWriter              *customWriter
		readPipe                 *io.PipeReader //
		writePipe                *io.PipeWriter
		Process                  *Process
		wg                       *sync.WaitGroup
		exitEvent                ExecutionExitEvent
		buzzkillEmitters         []chan bool               // Allow external processes to monitor to trigger a buzzkill event
		internalExitNotifiers    []chan bool               // All related internal goroutines should lock onto this notifier to exit when the process is killed
		externalProcessNotifiers []chan ProcessStatus      // Allow external processes to hook into process notifications (running, failed, exited, restarting etc,)
		externalExitNotifiers    []chan ExecutionExitEvent // Allow external processes to hook into true exit notifications (i.e. the process is no longer running)
		triggers                 []chan bool
		stdIn                    chan string
		exitCode                 int
		cancel                   bool
	}
)

const (
	ExitEventInternal ExecutionExitEvent = iota
	ExitEventBuzzkilled
	ExitEventBuzzkiller
)

// Creates an execution context
func (p *Process) CreateContext(wg *sync.WaitGroup) ExecutionContext {
	context := ExecutionContext{
		Process:                  p,
		wg:                       wg,
		internalExitNotifiers:    make([]chan bool, 0),
		externalProcessNotifiers: make([]chan ProcessStatus, 0),
		externalExitNotifiers:    make([]chan ExecutionExitEvent, 0),
		stdIn:                    make(chan string),
		buzzkillEmitters:         make([]chan bool, 0),
		triggers:                 make([]chan bool, 0),
	}

	wg.Add(1)
	// Write into the command
	context.infoWriter = &customWriter{w: os.Stdout, severity: "info", process: context.Process}   // Write info out
	context.errorWriter = &customWriter{w: os.Stdout, severity: "error", process: context.Process} // Write errors out
	// Set IO
	context.readPipe, context.writePipe = io.Pipe()
	fsExitChannel := context.getInternalExitNotifier()

	fsTrigger := context.CreateFsTrigger(fsExitChannel)

	if fsTrigger != nil {
		context.triggers = append(context.triggers, fsTrigger)
	}

	return context
}

// Returns a listening channel to listen for a buzzkill event comming FROM the process
func (e *ExecutionContext) GetBuzkillEmitter() chan bool {
	channel := make(chan bool)
	e.buzzkillEmitters = append(e.buzzkillEmitters, channel)
	return channel
}

// Emits the buzzkill command FROM INSIDE the process
func (e *ExecutionContext) emitBuzkill() {
	if len(e.buzzkillEmitters) == 0 {
		return
	}

	// Send external notifications
	for _, channel := range e.buzzkillEmitters {
		if channel != nil {
			select {
			case channel <- true:
			default:
				// Channel is not ready to receive, skip it
			}
		}
	}
	// Shut down process goroutines
	e.BuzzkillProcess()
}

// Tell all goroutines related to this process to stop and exit
func (e *ExecutionContext) BuzzkillProcess() {
	e.cancel = true
	if len(e.internalExitNotifiers) == 0 {
		return
	}

	for _, channel := range e.internalExitNotifiers {
		if channel != nil {
			select {
			case channel <- true:
			default:
				// Channel is not ready to receive, skip it
			}
		}
	}
}

// Updates the process status and sends external notifications of said status
func (e *ExecutionContext) setProcessStatus(status ProcessStatus) {
	e.Process.Status = status
	for _, channel := range e.externalProcessNotifiers {
		if channel != nil {
			select {
			case channel <- status:
			default:
				// Channel is not ready to receive, skip it
			}
		}
	}
}

// Returns an output channel of the processes internal status (running, exiting, restarting, failing etc.)
func (e *ExecutionContext) GetProcessNotificationChannel() chan ProcessStatus {
	channel := make(chan ProcessStatus)
	e.externalProcessNotifiers = append(e.externalProcessNotifiers, channel)
	return channel
}

// Get an instance of the internal exit notifier channel that outputs a signal when the process is buzzkilled
func (e *ExecutionContext) getInternalExitNotifier() chan bool {
	channel := make(chan bool)
	e.internalExitNotifiers = append(e.internalExitNotifiers, channel)
	return channel
}

func (e *ExecutionContext) handleProcessExit() {
	exitCommand := ExitCommandWait
	if e.exitEvent != ExitEventBuzzkilled {
		if e.Process.Status == ProcessStatusFailed || e.Process.Status == ProcessStatusNotStarted {
			e.errorWriter.Write([]byte("Process failed"))
			exitCommand = e.Process.OnFailure
		} else {
			e.errorWriter.Write([]byte("Process exited"))
			exitCommand = e.Process.OnComplete
		}
	}

	switch exitCommand {
	case ExitCommandBuzzkill:
		e.errorWriter.Write([]byte("Buzzkilling"))
		e.exitEvent = ExitEventBuzzkiller
		e.emitBuzkill()
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
		e.setProcessStatus(ProcessStatusRestarting)
		e.infoWriter.Write([]byte(fmt.Sprintf("Process exited - Restarting, %d second restart delay, %d attempts remaining", e.Process.RestartDelay, e.Process.RestartAttempts)))
		if e.Process.RestartDelay > 0 {
			time.Sleep(time.Duration(e.Process.RestartDelay) * time.Second)
		}
		// Recursive call
		e.execute()

	case ExitCommandWait:
		e.BuzzkillProcess()
	}

}

func (e *ExecutionContext) Write(input string) {
	e.stdIn <- input
}

func (config *Config) GenerateRunTaskContexts(wg *sync.WaitGroup) []ExecutionContext {

	// Create context and channel groups
	contexts := []ExecutionContext{}

	for index, process := range config.Processes {

		// Create context
		newContext := process.CreateContext(
			wg,
		)
		contexts = append(contexts, newContext)
		externalBuzzkill := newContext.getInternalExitNotifier()
		internalBuzzkill := newContext.GetBuzkillEmitter()
		// processNotificationChannel := newContext.GetProcessNotificationChannel()

		// Start listening to the threads channels fo multi-channel communcation
		go func() {
		monitorLoop:
			for {
				select {
				case <-internalBuzzkill:
					for i := range len(config.Processes) {
						// Send to all other channels (not including this one)
						if i != index {
							contexts[i].BuzzkillProcess()
						}
					}
					break monitorLoop

				case <-externalBuzzkill:
					break monitorLoop
				}
			}

		}()
	}

	// Apply process triggers to contexts here

	return contexts
}

func (c *ExecutionContext) execute() {
	c.setProcessStatus(ProcessStatusNotStarted)
	// Create command
	c.cmd = exec.Command(c.Process.Command, c.Process.Args...)
	c.cmd.Env = os.Environ() // Set the full environment, including PATH
	// Create IO

	c.cmd.Stdout = c.infoWriter
	c.cmd.Stderr = c.errorWriter
	c.cmd.Stdin = c.readPipe

	displayedPid := false // Simple bool to show boolean at the start of the process
	// Wait for the start delay
	c.infoWriter.Write([]byte(fmt.Sprintf("Starting process - %d second delay", c.Process.Delay)))
	if c.Process.Delay > 0 {
		time.Sleep(time.Duration(c.Process.Delay) * time.Second)
	}
	// Start the command
	startErr := c.cmd.Start()
	if startErr == nil {
		c.setProcessStatus(ProcessStatusRunning)
	}
	// Go wait somewhere else lamo (*insert you cant sit with us meme*)
	go c.cmd.Wait()

	buzzkillChannel := c.getInternalExitNotifier()

commandLoop:
	for {
		select {
		case value := <-c.stdIn: // Received std in
			c.writePipe.Write([]byte(value + "\n"))
		case <-buzzkillChannel: // Recieved buzzkill
			c.exitEvent = ExitEventBuzzkilled
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
			c.setProcessStatus(ProcessStatusExited)
			break commandLoop

		default:
			if c.cancel {
				c.exitEvent = ExitEventBuzzkilled
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
				c.setProcessStatus(ProcessStatusExited)
				break commandLoop
			}
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
					c.exitEvent = ExitEventInternal
					c.setProcessStatus(ProcessStatusExited)
				} else if c.cmd.ProcessState.ExitCode() > 0 {
					c.errorWriter.Write([]byte("Detected Process failure"))
					c.setProcessStatus(ProcessStatusFailed)
				} else if startErr != nil {
					c.errorWriter.Write([]byte("Failed to start"))
					c.errorWriter.Write([]byte(startErr.Error()))
					c.setProcessStatus(ProcessStatusFailed)
					c.exitEvent = ExitEventInternal
				} else {
					c.setProcessStatus(ProcessStatusExited)
				}
				c.exitCode = c.cmd.ProcessState.ExitCode()
				c.cmd.Wait() // This is likely redundant as we listen up top, but best be sure
				break commandLoop
			}

			// Don't spin too hard
			time.Sleep(10 * time.Millisecond)

		}
	}

	// The process exits so quick we need to delay to ensure that the buzzkill command is sent
	time.Sleep(time.Duration(10) * time.Millisecond)
	c.handleProcessExit()
}

// Cleanup operations on remaining channels
func (e *ExecutionContext) end() {
	if e.writePipe != nil {
		e.writePipe.Close()
	}
	if e.readPipe != nil {
		e.readPipe.Close()
	}
	// This unfortunately needs to be there to let things settle properly
	time.Sleep(time.Millisecond * 10)
	e.wg.Done()
}

func (e *ExecutionContext) Start() {

	// buzzkill := e.getInternalExitNotifier()

	// TODO: Check for process trigger here too
	fmt.Printf("Trigger length = %d\n", len(e.triggers))
	if len(e.triggers) == 0 {
		e.execute()
	} else {
		e.execute()

		// monitorLoop:
		// for {
		// 	for _, trigger := range e.triggers {
		// 		select {
		// 		case <-trigger:
		// 			e.execute()
		// 			// Break monitoring if buzzkill commited
		// 			if e.exitEvent != ExitEventInternal {
		// 				break monitorLoop
		// 			}

		// 		case <-buzzkill:
		// 			break monitorLoop
		// 		}

		// 	}
		// }
	}
	e.end()
}
