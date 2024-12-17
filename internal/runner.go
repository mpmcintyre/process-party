package runner

// Custom writers-  https://medium.com/@shubhamagrawal094/custom-writer-in-golang-171dd2cac7e0
import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Communication from task to main thread
type TaskChannelsOut struct {
	Buzzkill     chan bool
	EndOfCommand chan string
}

// Communication from main thread to all threads
type MainChannelsOut struct {
	Buzzkill chan bool
	StdIn    chan string
}

type Context struct {
	Process         *Process
	MainChannelsOut MainChannelsOut
	TaskChannelsOut TaskChannelsOut
	wg              *sync.WaitGroup
	cmd             *exec.Cmd
	infoWriter      *customWriter
	errorWriter     *customWriter
	readPipe        *io.PipeReader
	writePipe       *io.PipeWriter
	// outb, errb   bytes.Buffer
}

func CreateContext(p *Process, wg *sync.WaitGroup, mc MainChannelsOut, tc TaskChannelsOut) Context {
	return Context{
		Process:         p,
		wg:              wg,
		MainChannelsOut: mc,
		TaskChannelsOut: tc,
	}
}

func (c *Context) GetStatusAsStr() string {
	switch c.Process.Status {
	case ExitStatusNotStarted:
		return "Not started"
	case ExitStatusRunning:
		return "Running"
	case ExitStatusExited:
		return "Exited"
	case ExitStatusFailed:
		return "Failed"
	}
	return "Unknown"
}

func (c *Context) setupCmd() {
	c.cmd = exec.Command(c.Process.Command, c.Process.Args...)
	c.cmd.Env = os.Environ() // Set the full environment, including PATH
	// Create IO
	r, w := io.Pipe()
	c.readPipe = r
	c.writePipe = w
	// Write into the command
	c.infoWriter = &customWriter{w: os.Stdout, severity: "info", process: c.Process}   // Write info out
	c.errorWriter = &customWriter{w: os.Stdout, severity: "error", process: c.Process} // Write errors out
	// Set IO
	c.cmd.Stdout = c.infoWriter
	c.cmd.Stderr = c.errorWriter
	c.cmd.Stdin = r
}

func (c *Context) Run() {
	defer c.wg.Done()
	// Create command
	c.setupCmd()
	displayedPid := false // Simple bool to show boolean at the start of the process
	c.Process.Status = ExitStatusRunning

	// Wait for the start delay
	c.infoWriter.Write([]byte(fmt.Sprintf("Starting process - %d second delay", c.Process.Delay)))
	time.Sleep(time.Duration(c.Process.Delay) * time.Second)
	// Start the command
	startErr := c.cmd.Start()
	// Go wait somewhere else lamo (*insert you cant sit with us meme*)
	go c.cmd.Wait()

commandLoop:
	for {

		select {
		case value := <-c.MainChannelsOut.StdIn: // Received std in
			c.writePipe.Write([]byte(value + "\n"))
		case <-c.MainChannelsOut.Buzzkill: // Recieved buzzkill
			c.Process.Status = ExitStatusExited
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
					c.infoWriter.Write([]byte("Gracefull shutdown timed out"))
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
			// Handle the process exiting
			if c.cmd.ProcessState.ExitCode() >= 0 || startErr != nil {
				if c.cmd.ProcessState.ExitCode() == 0 {
					c.Process.Status = ExitStatusExited
					startErr = c.handleCloseConditions(*c.infoWriter, c.Process.OnComplete)
				} else if startErr != nil {
					c.infoWriter.Write([]byte("Failed to start"))
					c.errorWriter.Write([]byte(startErr.Error()))
					c.Process.Status = ExitStatusFailed
					startErr = c.handleCloseConditions(*c.errorWriter, c.Process.OnFailure)
				} else {
					c.Process.Status = ExitStatusExited
					// Note! This will block if not listened to in tests
					c.TaskChannelsOut.EndOfCommand <- c.Process.Name
					startErr = c.handleCloseConditions(*c.infoWriter, c.Process.OnComplete)
				}

				// If the end commands are anything else we dont give a shit
				// If the process is restarted this should be false
				if c.cmd.ProcessState.ExitCode() >= 0 || startErr != nil {
					break commandLoop
				}
				c.Process.Status = ExitStatusRunning
			}
			// Stream the initial start stream values and set it to an empty string
			if c.Process.StartStream != "" {
				c.writePipe.Write([]byte(c.Process.StartStream + "\n"))
				c.Process.StartStream = ""
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	c.writePipe.Close()
	c.readPipe.Close()
	c.cmd.Wait()
	// The process exits so quick we need to delay to ensure that the buzkill command is sent
	time.Sleep(time.Duration(10) * time.Millisecond)
}

func (c *Context) handleCloseConditions(writer customWriter, exitHandler ExitCommand) error {
	if exitHandler == ExitCommandBuzzkill {
		writer.Write([]byte("Process exited - Buzzkilling"))
		c.TaskChannelsOut.Buzzkill <- true
		return errors.New("exit code 1")
	}
	if exitHandler == ExitCommandRestart {
		// We cannot reuse the c.cmd, so we use recursion with counters
		if c.Process.RestartAttempts == 0 {
			return errors.New("exit code 1")
		}
		// Remove one attempt (negative numbers imply to always restart)
		if c.Process.RestartAttempts+1 > 0 {
			c.Process.RestartAttempts = c.Process.RestartAttempts - 1
		}

		c.Process.Status = ExitStatusRestarting
		writer.Write([]byte(fmt.Sprintf("Process exited - Restarting, %d second restart delay, %d attempts remaining", c.Process.RestartDelay, c.Process.RestartAttempts)))
		time.Sleep(time.Duration(c.Process.RestartDelay) * time.Second)
		// Add to the wait group, create a new context, and run the new command syncrounously
		// This might cause some issues if he process needs to restart indefinitely
		c.wg.Add(1)
		c := CreateContext(
			c.Process, c.wg, c.MainChannelsOut, c.TaskChannelsOut,
		)
		c.Run()
		return errors.New("exit code 1")
	}
	if exitHandler == ExitCommandWait {
		writer.Write([]byte("Process exited - waiting"))
		// Note! This will block if not listened to in tests
		c.TaskChannelsOut.EndOfCommand <- c.Process.Name
		return errors.New("waiting for other processes - exit code 1")
	}
	return errors.New("unknown exit condition - exit status 1")
}
