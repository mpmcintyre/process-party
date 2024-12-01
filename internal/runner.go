package runner

// Custom writers-  https://medium.com/@shubhamagrawal094/custom-writer-in-golang-171dd2cac7e0
import (
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
	w            io.Writer
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
	cmdIn, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err != nil {
		log.Fatal(err)
	}
	if err != nil {
		log.Fatal(err)
	}

	infoWriter := &customWriter{w: os.Stdout, severity: "info", process: c.Process}
	errorWriter := &customWriter{w: os.Stdout, severity: "error", process: c.Process}
	cmd.Stdout = infoWriter
	cmd.Stderr = errorWriter
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
				cmd.Process.Signal(os.Kill)
				break cmdLoop
			}

		default:
			// Handle the process exiting
			if cmd.ProcessState.ExitCode() > 0 || startErr != nil {
				if cmd.ProcessState.ExitCode() == 0 {
					c.handleCloseConditions(*errorWriter, cmd, c.Process.OnFailure)
				} else if startErr != nil {
					errorWriter.Write([]byte(startErr.Error()))
					c.handleCloseConditions(*errorWriter, cmd, c.Process.OnFailure)
				} else {
					infoWriter.Write([]byte("Process exited"))
					c.EndOfCommand <- c.Process.Name
					c.handleCloseConditions(*infoWriter, cmd, c.Process.OnComplete)
				}

				// If the end commands are anything else we dont give a shit
				// If the process is restarted this should be false
				break cmdLoop
			}

		}
	}

	infoWriter.Write([]byte("Exiting context"))

	cmd.Wait()
	c.wg.Done()
}

func (c *Context) handleCloseConditions(writer customWriter, cmd *exec.Cmd, exitHandler ExitCommand) {
	writer.Write([]byte(c.Process.OnComplete))
	writer.Write([]byte(c.Process.OnFailure))

	if exitHandler == ExitCommandBuzzkill {
		writer.Write([]byte("Process exited - Buzzkilling"))
		c.BuzzKill <- true
	}
	if exitHandler == ExitCommandRestart {
		writer.Write([]byte("Process exited - Restarting"))
		cmd = exec.Command(c.Process.Command, c.Process.Args...)
		cmd.Start()
		cmd.Stdout = c.w
	}
}
