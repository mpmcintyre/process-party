package runner

// Custom writers-  https://medium.com/@shubhamagrawal094/custom-writer-in-golang-171dd2cac7e0
import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"
)

type Context struct {
	Process      Process
	StdIn        chan string
	EndOfCommand chan string
	BuzzKill     chan bool
	wg           *sync.WaitGroup
	// outb, errb   bytes.Buffer
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
