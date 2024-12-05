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
	// c.wg.Add(1)

	cmd := exec.Command(c.Process.Command, c.Process.Args...)
	cmd.Env = os.Environ() // Set the full environment, including PATH
	r, w := io.Pipe()
	infoWriter := &customWriter{w: os.Stdout, severity: "info", process: c.Process}
	errorWriter := &customWriter{w: os.Stdout, severity: "error", process: c.Process}
	cmd.Stdout = infoWriter
	cmd.Stderr = errorWriter
	cmd.Stdin = r
	displayedPid := false
	c.Process.Status = ExitStatusRunning
	startErr := cmd.Start()
	// Go wait somewhere else lamo (*insert you cant sit with us meme*)
	go cmd.Wait()

cmdLoop:
	for {

		select {
		case value := <-c.MainChannelsOut.StdIn:
			w.Write([]byte(value + "\n"))
		case <-c.MainChannelsOut.Buzzkill:
			infoWriter.Write([]byte("Recieved buzzkill command"))
			startTime := time.Now()
			timeout := 3 * time.Second
			timedOut := false
			for {
				elapsed := time.Since(startTime)
				if elapsed > timeout {
					timedOut = true
					break
				}
				if cmd.Process != nil {
					break
				}

			}
			if !timedOut {
				cmd.Process.Signal(os.Kill)
			}
			break cmdLoop

		default:
			// Handle the process exiting
			if !displayedPid && cmd.Process != nil {
				infoWriter.Write([]byte(fmt.Sprintf("PID = %d", cmd.Process.Pid)))
				displayedPid = true
			}
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
		writer.Write([]byte("Process exited - Restarting"))
		return cmd.Start()
	}
	return errors.New("exit code 1")
}
