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
	case ExitStatusRunning:
		return "Running"
	case ExitStatusExited:
		return "Exited"
	case ExitStatusFailed:
		return "Failed"
	}
	return "Unknown"
}

func (c *Context) Run() {
	// Create command
	cmd := exec.Command(c.Process.Command, c.Process.Args...)
	cmd.Env = os.Environ() // Set the full environment, including PATH
	// Create IO
	r, w := io.Pipe()                                                                 // Write into the command
	infoWriter := &customWriter{w: os.Stdout, severity: "info", process: c.Process}   // Write info out
	errorWriter := &customWriter{w: os.Stdout, severity: "error", process: c.Process} // Write errors out
	// Set IO
	cmd.Stdout = infoWriter
	cmd.Stderr = errorWriter
	cmd.Stdin = r

	displayedPid := false // Simple bool to show boolean at the start of the process
	c.Process.Status = ExitStatusRunning

	// Wait for the start delay
	infoWriter.Write([]byte(fmt.Sprintf("Starting process - %d second delay", c.Process.Delay)))
	time.Sleep(time.Duration(c.Process.Delay) * time.Second)
	// Start the command
	startErr := cmd.Start()
	// Go wait somewhere else lamo (*insert you cant sit with us meme*)
	go cmd.Wait()

cmdLoop:
	for {

		select {
		case value := <-c.MainChannelsOut.StdIn: // Received std in
			w.Write([]byte(value + "\n"))
		case <-c.MainChannelsOut.Buzzkill: // Recieved buzzkill
			c.Process.Status = ExitStatusExited
			infoWriter.Write([]byte("Recieved buzzkill command"))
			// Wait for timeout_on_exit duration
			startTime := time.Now()
			timeout := time.Duration(c.Process.TimeoutOnExit) * time.Second
			if cmd.Process != nil {
				cmd.Process.Signal(os.Kill)
			}
			for {
				elapsed := time.Since(startTime)

				if elapsed > timeout {
					infoWriter.Write([]byte("Gracefull shutdown timed out"))
					if cmd.Process != nil {
						cmd.Process.Kill()
					}
					break
				}
				if cmd.ProcessState != nil {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}

			break cmdLoop

		default:
			// Display the PID on the first line
			if !displayedPid && cmd.Process != nil {
				infoWriter.Write([]byte(fmt.Sprintf("PID = %d", cmd.Process.Pid)))
				c.Process.Pid = fmt.Sprintf("%d", cmd.Process.Pid)
				displayedPid = true
			}
			// Handle the process exiting
			if cmd.ProcessState.ExitCode() > 0 || startErr != nil {
				if cmd.ProcessState.ExitCode() == 0 {
					c.Process.Status = ExitStatusExited
					startErr = c.handleCloseConditions(*infoWriter, cmd, c.Process.OnComplete)
				} else if startErr != nil {
					errorWriter.Write([]byte(startErr.Error()))
					c.Process.Status = ExitStatusFailed
					startErr = c.handleCloseConditions(*errorWriter, cmd, c.Process.OnFailure)
				} else {
					c.Process.Status = ExitStatusExited
					fmt.Printf("%s process (%s) exited, saying irish goodbye\n", c.Process.Name, c.Process.Prefix)
					c.TaskChannelsOut.EndOfCommand <- c.Process.Name
					startErr = c.handleCloseConditions(*infoWriter, cmd, c.Process.OnComplete)
				}

				// If the end commands are anything else we dont give a shit
				// If the process is restarted this should be false
				if cmd.ProcessState.ExitCode() > 0 || startErr != nil {
					break cmdLoop
				}
				c.Process.Status = ExitStatusRunning
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	r.Close()
	w.Close()
	cmd.Wait()
	infoWriter.Write([]byte("~Exiting context~"))
	c.wg.Done()
}

func (c *Context) handleCloseConditions(writer customWriter, cmd *exec.Cmd, exitHandler ExitCommand) error {
	if exitHandler == ExitCommandBuzzkill {
		writer.Write([]byte("Process exited - Buzzkilling"))
		c.TaskChannelsOut.Buzzkill <- true
		return errors.New("exit code 1")
	}
	if exitHandler == ExitCommandRestart {
		if c.Process.RestartAttempts == 0 {
			return errors.New("exit code 1")
		}
		// Remove one attempt (negative numbers imply to always restart)
		if c.Process.RestartAttempts > 0 {
			c.Process.RestartAttempts = c.Process.RestartAttempts - 1
		}
		c.Process.Status = ExitStatusRestarting
		writer.Write([]byte(fmt.Sprintf("Process exited - Restarting, %d second restart delay, %d attempts remaining", c.Process.RestartDelay, c.Process.RestartAttempts)))
		time.Sleep(time.Duration(c.Process.RestartDelay) * time.Second)
		err := cmd.Start()
		return err
	}
	if exitHandler == ExitCommandWait {
		writer.Write([]byte("Process exited - Buzzkilling"))
		c.TaskChannelsOut.Buzzkill <- true
		return errors.New("waiting for other processes - exit code 1")
	}
	return errors.New("unknown exit condition - exit status 1")
}
