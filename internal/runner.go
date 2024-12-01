package runner

// Custom writers-  https://medium.com/@shubhamagrawal094/custom-writer-in-golang-171dd2cac7e0
import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	color "github.com/fatih/color"
)

type Context struct {
	Process      Process
	StdIn        chan string
	EndOfCommand chan string
	BuzzKill     chan bool
	wg           *sync.WaitGroup
	// outb, errb   bytes.Buffer
}
type customWriter struct {
	w        io.Writer
	severity string
	process  Process
}

func (e customWriter) Write(p []byte) (int, error) {

	prefix := "[" + e.process.Prefix + "]"
	message := string(p)
	now := time.Now()

	// Format the time as HH:MM:SS:MS
	timeString := fmt.Sprintf("%02d:%02d:%02d:%03d",
		now.Hour(),
		now.Minute(),
		now.Second(),
		now.Nanosecond()/1e6) // Convert nanoseconds to milliseconds

	x := strings.Split(message, "\n")

	if e.process.SeperateNewLines {
		for _, message := range x {

			prefix = color.BlueString(prefix)
			if e.severity == "error" {
				message = color.RedString(message)
			}
			if e.process.ShowTimestamp {
				message = timeString + "	" + message
			}
			n, err := e.w.Write([]byte(prefix + " " + message + "\n"))
			if err != nil {
				return n, err
			}

		}
	} else {

		prefix = color.BlueString(prefix)
		if e.severity == "error" {
			message = color.RedString(message)
		}
		if e.process.ShowTimestamp {
			message = timeString + "	" + message
		}
		n, err := e.w.Write([]byte(prefix + " " + message + "\n"))
		if err != nil {
			return n, err
		}
	}

	return len(p), nil
}

func (c Context) Write(p []byte) (int, error) {
	color.Red("Prints %s in blue.", "text")
	n, err := os.Stdout.Write(p)
	if err != nil {
		return n, err
	}
	if n != len(p) {
		return n, io.ErrShortWrite
	}
	return len(p), nil
}

func CreateContext(p Process, wg *sync.WaitGroup, chanIn chan string, eoc chan string, bk chan bool) Context {
	return Context{
		Process:      p,
		StdIn:        chanIn,
		EndOfCommand: eoc,
		BuzzKill:     bk,
		wg:           wg,
	}
}

func (c *Context) Run() {
	// c.wg.Add(1)

	log.Printf("Starting command %s", c.Process.Name)
	cmd := exec.Command(c.Process.Command, c.Process.Args...)
	cmd.Env = os.Environ() // Set the full environment, including PATH

	cmdIn, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	infoWriter := &customWriter{w: os.Stdout, severity: "info", process: c.Process}
	errorWriter := &customWriter{w: os.Stdout, severity: "error", process: c.Process}
	cmd.Stdout = infoWriter
	cmd.Stderr = errorWriter
	// exec.CommandContext()
	displayedPid := false
	startErr := cmd.Start()

cmdLoop:
	for {

		select {
		case <-c.StdIn:
			log.Printf("Writing to input")
			cmdIn.Write([]byte(<-c.StdIn))
		case kill := <-c.BuzzKill:
			if kill {
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
			}

		default:
			// Handle the process exiting

			if cmd.ProcessState.ExitCode() > 0 || startErr != nil {
				if cmd.ProcessState.ExitCode() == 0 {
					startErr = c.handleCloseConditions(*errorWriter, cmd, c.Process.OnFailure)
				} else if startErr != nil {
					errorWriter.Write([]byte(startErr.Error()))
					startErr = c.handleCloseConditions(*errorWriter, cmd, c.Process.OnFailure)
				} else {
					infoWriter.Write([]byte("Process exited"))
					c.EndOfCommand <- c.Process.Name
					startErr = c.handleCloseConditions(*infoWriter, cmd, c.Process.OnComplete)
				}

				// If the end commands are anything else we dont give a shit
				// If the process is restarted this should be false
				if cmd.ProcessState.ExitCode() > 0 || startErr != nil {
					break cmdLoop
				}
			}
			if !displayedPid {
				infoWriter.Write([]byte(fmt.Sprintf("PID = %d", cmd.Process.Pid)))
				displayedPid = true
			}

			if cmd.ProcessState == nil {
				startErr = c.handleCloseConditions(*errorWriter, cmd, c.Process.OnFailure)
				if cmd.ProcessState == nil || cmd.ProcessState.ExitCode() > 0 || startErr != nil {
					break cmdLoop
				}
			}

		}
	}
	cmd.Wait()
	infoWriter.Write([]byte("~Exiting context~"))
	c.wg.Done()
}

func (c *Context) handleCloseConditions(writer customWriter, cmd *exec.Cmd, exitHandler ExitCommand) error {
	if exitHandler == ExitCommandBuzzkill {
		writer.Write([]byte("Process exited - Buzzkilling"))
		c.BuzzKill <- true
		return errors.New("exit code 1")
	}
	if exitHandler == ExitCommandRestart {
		writer.Write([]byte("Process exited - Restarting"))
		return cmd.Start()
	}
	return errors.New("exit code 1")
}
