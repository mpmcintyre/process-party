package pp

// Custom writers-  https://medium.com/@shubhamagrawal094/custom-writer-in-golang-171dd2cac7e0
import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// Communication from task to main thread
type (
	ExecutionExitEvent int
	ProcessStatus      int

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
		buzzkillEmitters         []chan bool          // Allow external processes to monitor to trigger a buzzkill event
		internalExitNotifiers    []chan bool          // All related internal goroutines should lock onto this notifier to exit when the process is killed
		externalProcessNotifiers []chan ProcessStatus // Allow external processes to hook into process notifications (running, failed, exited, restarting etc,)
		triggers                 []chan string
		stdIn                    chan string
		exitCode                 int
		executionMutex           *sync.RWMutex
		Status                   ProcessStatus
		restartCounter           int
	}
)

const (
	ExitEventInternal ExecutionExitEvent = iota
	ExitEventBuzzkilled
	ExitEventBuzzkiller
)

const (
	ProcessStatusNotStarted ProcessStatus = iota
	ProcessStatusRunning
	ProcessStatusExited
	ProcessStatusFailed
	ProcessStatusRestarting
	ProcessStatusWaitingTrigger
)

// Returns the executions current status as a string
func (c *ExecutionContext) GetStatusAsStr() string {
	switch c.Status {
	case ProcessStatusNotStarted:
		return "Not started"
	case ProcessStatusRunning:
		return "Running"
	case ProcessStatusExited:
		return "Exited"
	case ProcessStatusFailed:
		return "Failed"
	case ProcessStatusWaitingTrigger:
		return "Waiting for trigger"
	case ProcessStatusRestarting:
		return "Restarting"
	}
	return "Unknown"
}

// Creates an execution context
func (p *Process) CreateContext(wg *sync.WaitGroup) *ExecutionContext {
	context := &ExecutionContext{
		Process:                  p,
		wg:                       wg,
		internalExitNotifiers:    make([]chan bool, 0),
		externalProcessNotifiers: make([]chan ProcessStatus, 0),
		stdIn:                    make(chan string, 10),
		buzzkillEmitters:         make([]chan bool, 0),
		triggers:                 make([]chan string, 0),
		executionMutex:           &sync.RWMutex{},
	}

	// Write into the command
	context.infoWriter = &customWriter{w: os.Stdout, severity: "info", process: context.Process}   // Write info out
	context.errorWriter = &customWriter{w: os.Stdout, severity: "error", process: context.Process} // Write errors out
	// Set IO
	context.readPipe, context.writePipe = io.Pipe()

	return context
}

// Returns a listening channel to listen for a buzzkill event comming FROM the process
// This channel can close so be sure to check with _,ok:= <- emitted
func (e *ExecutionContext) GetBuzkillEmitter() chan bool {
	e.executionMutex.Lock()
	defer e.executionMutex.Unlock()

	channel := make(chan bool, 10)
	e.buzzkillEmitters = append(e.buzzkillEmitters, channel)
	return channel
}

// Returns an output channel of the processes internal status (running, exiting, restarting, failing etc.)
// This channel can close so be sure to check with _,ok:= <- status
func (e *ExecutionContext) GetProcessNotificationChannel() chan ProcessStatus {
	e.executionMutex.Lock()
	defer e.executionMutex.Unlock()

	channel := make(chan ProcessStatus, 10)
	e.externalProcessNotifiers = append(e.externalProcessNotifiers, channel)
	return channel
}

// Get an instance of the internal exit notifier channel that outputs a signal when the process is buzzkilled
// This channel can close so be sure to check with _,ok:= <- status
func (e *ExecutionContext) getInternalExitNotifier() chan bool {
	e.executionMutex.Lock()
	defer e.executionMutex.Unlock()

	channel := make(chan bool, 10)
	e.internalExitNotifiers = append(e.internalExitNotifiers, channel)
	return channel
}

// Close all the channels
func (e *ExecutionContext) closeChannels() {
	e.executionMutex.Lock()
	defer e.executionMutex.Unlock()

	for i := range e.internalExitNotifiers {
		if e.internalExitNotifiers[i] != nil {
			close(e.internalExitNotifiers[i])
			e.internalExitNotifiers[i] = nil
		}
	}

	for i := range e.externalProcessNotifiers {
		if e.externalProcessNotifiers[i] != nil {
			close(e.externalProcessNotifiers[i])
			e.externalProcessNotifiers[i] = nil
		}
	}

	for i := range e.buzzkillEmitters {
		if e.buzzkillEmitters[i] != nil {
			close(e.buzzkillEmitters[i])
			e.buzzkillEmitters[i] = nil
		}
	}

	close(e.stdIn)
	e.stdIn = nil

	// Triggers should close themselves on internal exit emitted (dont close here)
}

// Emits the buzzkill command FROM INSIDE the process
func (e *ExecutionContext) emitBuzkill() {
	e.executionMutex.RLock()
	// Send external notifications
	for _, channel := range e.buzzkillEmitters {
		if channel != nil {
			channel <- true
		}
	}
	e.executionMutex.RUnlock()

	// Shut down process goroutines
	e.closeChannels()

}

// Tell all goroutines related to this process to stop and exit
func (e *ExecutionContext) BuzzkillProcess() {
	e.infoWriter.Printf("Buzzkill  called")

	e.executionMutex.RLock()
	for i := range e.internalExitNotifiers {
		if e.internalExitNotifiers[i] != nil {
			e.internalExitNotifiers[i] <- true
		}
	}
	e.executionMutex.RUnlock()

	// Wait for the status to change to exited before closing all channels
	go func() {
		statusChannel := e.GetProcessNotificationChannel()
		for {
			status := <-statusChannel
			if status == ProcessStatusExited {
				break
			}
		}
		e.closeChannels()
	}()
}

// Updates the process status and sends external notifications of said status
func (e *ExecutionContext) setProcessStatus(status ProcessStatus) {
	e.executionMutex.RLock()
	defer e.executionMutex.RUnlock()

	e.Status = status
	for _, channel := range e.externalProcessNotifiers {
		if channel != nil {
			channel <- status
		}
	}

}

// Handles how the execution context behaves on exit, depending on exit behaviour
func (e *ExecutionContext) handleProcessExit() {
	exitCommand := ExitCommandWait
	if e.exitEvent != ExitEventBuzzkilled {
		if e.Status == ProcessStatusFailed || e.Status == ProcessStatusNotStarted {
			exitCommand = e.Process.OnFailure
		} else {
			exitCommand = e.Process.OnComplete
		}
	}

	switch exitCommand {
	case ExitCommandBuzzkill:
		e.errorWriter.Write([]byte("Buzzkilling other processes"))
		e.exitEvent = ExitEventBuzzkiller
		e.emitBuzkill()
	case ExitCommandRestart:

		e.restartCounter++

		if e.restartCounter >= e.Process.RestartAttempts && e.Process.RestartAttempts >= 0 {
			e.infoWriter.Write([]byte("No restart attempts left, exiting"))
			return
		}

		e.setProcessStatus(ProcessStatusRestarting)
		if e.Process.RestartAttempts > 0 {
			e.infoWriter.Write([]byte(fmt.Sprintf("Process exited - Restarting, %d second restart delay, %d attempts remaining", e.Process.RestartDelay, e.Process.RestartAttempts-e.restartCounter)))
		}
		if e.Process.RestartDelay > 0 {
			time.Sleep(time.Duration(e.Process.RestartDelay) * time.Second)
		}
		// Recursive call
		e.execute()

	case ExitCommandWait:
		if len(e.triggers) == 0 {
			e.BuzzkillProcess()
		}
	}

	e.setProcessStatus(ProcessStatusExited)
}

// Writes to the executing execution context
func (e *ExecutionContext) Write(input string) {
	if e.stdIn != nil {
		e.stdIn <- input
	}
}

// Loops over the processes in the config and provides a list of pointers to execution contexts for each process
func (config *Config) GenerateRunTaskContexts(wg *sync.WaitGroup) []*ExecutionContext {

	// Create context and channel groups
	contexts := []*ExecutionContext{}

	for index, process := range config.Processes {

		// Create context
		newContext := process.CreateContext(
			wg,
		)

		// processNotificationChannel := newContext.GetProcessNotificationChannel()

		// Start listening to the threads channels fo multi-channel communcation
		go func() {
			externalKillCommand := newContext.getInternalExitNotifier()
			buzzKillEmitter := newContext.GetBuzkillEmitter()

		monitorLoop:
			for {
				select {
				case _, ok := <-buzzKillEmitter:
					if ok {
						for i := range len(config.Processes) {
							if i != index {
								contexts[i].BuzzkillProcess()
							}
						}
					}
					break monitorLoop

				case <-externalKillCommand:
					break monitorLoop
				}
			}
		}()
		contexts = append(contexts, newContext)
	}

	// Apply process triggers to contexts here

	return contexts
}

// Actual execution of the desired process/execution context.
func (c *ExecutionContext) execute() {
	c.setProcessStatus(ProcessStatusNotStarted)
	c.executionMutex.Lock()
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
	c.executionMutex.Unlock()
	// if startErr == nil {
	// }

	processDone := make(chan struct{}, 1)

	// Go wait somewhere else lamo (*insert you cant sit with us meme*)
	go func() {
		c.cmd.Wait()
		close(processDone)
	}()

	buzzkillChannel := c.getInternalExitNotifier()

commandLoop:
	for {
		select {
		case <-processDone:
			state, err := c.cmd.Process.Wait()
			if err != nil {
				c.errorWriter.Write([]byte(fmt.Sprintf("Process wait error: %v", err)))
				c.setProcessStatus(ProcessStatusFailed)
			}

			// Check if process was terminated by a signal
			if status, ok := state.Sys().(syscall.WaitStatus); ok {
				if status.Signaled() {
					c.errorWriter.Write([]byte(fmt.Sprintf("Process terminated by signal: %v", status.Signal())))
					c.exitCode = 128 + int(status.Signal())
					c.setProcessStatus(ProcessStatusFailed)
				} else {
					c.exitCode = status.ExitStatus()
					if c.exitCode != 0 {
						c.setProcessStatus(ProcessStatusFailed)
					}
				}
			}

			break commandLoop
		case value := <-c.stdIn: // Received std in
			if c.cmd.Process != nil && c.cmd.ProcessState.ExitCode() < 0 && startErr != nil {
				c.writePipe.Write([]byte(value + "\n"))
			}
		case <-buzzkillChannel: // Recieved buzzkill
			c.exitEvent = ExitEventBuzzkilled
			c.infoWriter.Write([]byte("Recieved buzzkill command"))
			// Wait for timeout_on_exit duration
			startTime := time.Now()
			timeout := time.Duration(c.Process.TimeoutOnExit) * time.Second
			if c.cmd.Process != nil {
				c.cmd.Process.Signal(os.Kill)
			} else {
				break commandLoop
			}
			time.Sleep(100 * time.Millisecond)
			for {
				elapsed := time.Since(startTime)
				if c.cmd.ProcessState != nil {
					break
				}
				if elapsed > timeout {
					if c.cmd.Process != nil {
						c.infoWriter.Write([]byte("Gracefull shutdown timed out - killing process"))
						c.cmd.Process.Kill()
					}
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			c.setProcessStatus(ProcessStatusExited)
			break commandLoop

		default:

			// Display the PID on the first line
			if !displayedPid && c.cmd.Process != nil {
				c.setProcessStatus(ProcessStatusRunning)
				c.infoWriter.Write([]byte(fmt.Sprintf("PID = %d", c.cmd.Process.Pid)))
				c.Process.Pid = fmt.Sprintf("%d", c.cmd.Process.Pid)
				displayedPid = true
			}

			// Handle the process exiting
			if c.cmd.Process != nil && c.cmd.ProcessState.ExitCode() >= 0 ||
				startErr != nil ||
				c.Status == ProcessStatusFailed ||
				c.Status == ProcessStatusExited {
				if c.cmd.ProcessState.ExitCode() == 0 {
					c.errorWriter.Write([]byte("Detected Process exit"))
					c.exitEvent = ExitEventInternal
				} else if c.cmd.ProcessState.ExitCode() > 0 {
					c.errorWriter.Write([]byte("Detected Process failure"))
					c.setProcessStatus(ProcessStatusFailed)
				} else if startErr != nil {
					c.errorWriter.Write([]byte("Failed to start"))
					c.errorWriter.Write([]byte(startErr.Error()))
					c.setProcessStatus(ProcessStatusFailed)
					c.exitEvent = ExitEventInternal
				}
				c.exitCode = c.cmd.ProcessState.ExitCode()
				break commandLoop
			}

			// Stream the initial start stream values and set it to an empty string
			if c.Process.StartStream != "" {
				c.writePipe.Write([]byte(c.Process.StartStream + "\n"))
				c.Process.StartStream = ""
			}

			// Don't spin too hard
			time.Sleep(10 * time.Millisecond)
		}
	}
	c.cmd.Wait() // This is likely redundant as we listen up top, but best be sure

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
	time.Sleep(time.Millisecond * 100)
	e.wg.Done()
}

// Starts execution of the process or monitoring for any trigger events to trigger the process. Exits with buzzkill
func (e *ExecutionContext) Start() {
	e.wg.Add(1)
	exitNotifier := e.getInternalExitNotifier()

	go func() {
		defer e.end()

		if len(e.triggers) == 0 {
			e.execute()
			return
		}

		e.setProcessStatus(ProcessStatusWaitingTrigger)

		// Start a goroutine for each trigger to forward messages
		triggerChan := make(chan string, 50)
		for _, trigger := range e.triggers {
			go func(t chan string) {
				for msg := range t {
					triggerChan <- msg
				}
			}(trigger)
		}

	monitorLoop:
		for {
			select {
			case message := <-triggerChan:
				e.infoWriter.Printf("%s\n", message)
				if e.Status != ProcessStatusWaitingTrigger {
					e.errorWriter.Printf("Can't start process, process is already running")
					continue
				}

				if e.exitEvent != ExitEventInternal {
					break monitorLoop
				}
				e.execute()
				e.Process.Pid = ""
				e.setProcessStatus(ProcessStatusWaitingTrigger)
				time.Sleep(time.Duration(10) * time.Millisecond)

			case <-exitNotifier:
				e.BuzzkillProcess()
				break monitorLoop
			}
		}
	}()
}
