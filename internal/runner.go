package runner

// Custom writers-  https://medium.com/@shubhamagrawal094/custom-writer-in-golang-171dd2cac7e0
import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"

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

func (c Context) Write(p []byte) (int, error) {
	color.Red("Prints %s in blue.", "text")
	n, err := c.w.Write(p)
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
	log.Printf("Starting command %s", c.Process.Name)
	cmd := exec.Command(c.Process.Command, c.Process.Args...)
	cmdIn, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	cmdOut, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	cmdErr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	buffOut := []byte{}
	buffErr := []byte{}

cmdLoop:
	for {

		select {
		case <-c.StdIn:
			log.Printf("Writing to input")
			cmdIn.Write([]byte(<-c.StdIn))
		case <-c.BuzzKill:
			log.Printf("Sending buzzkill command")
			cmd.Process.Signal(os.Kill)
			break cmdLoop
		default:
			// Handle the process exiting
			if cmd.ProcessState.ExitCode() > 0 {
				log.Printf("Process exited: %s", c.Process.Name)

				if cmd.ProcessState.ExitCode() == 0 {
					if c.Process.OnComplete == ExitCommandBuzzkill {
						c.BuzzKill <- true
						break
					}
					if c.Process.OnComplete == ExitCommandRestart {
						cmd = exec.Command(c.Process.Command, c.Process.Args...)
						cmdIn, _ = cmd.StdinPipe()
						cmdOut, _ = cmd.StdoutPipe()
						cmd.Start()
						cmd.Stdout = c.w
					}
				} else {
					c.EndOfCommand <- c.Process.Name
					if c.Process.OnFailure == ExitCommandBuzzkill {
						c.BuzzKill <- true
						break
					}
					if c.Process.OnFailure == ExitCommandRestart {
						cmd = exec.Command(c.Process.Command, c.Process.Args...)
						cmdIn, _ = cmd.StdinPipe()
						cmdOut, _ = cmd.StdoutPipe()
						cmd.Start()
						cmd.Stdout = c.w
					}
				}

				// If the end commands are anything else we dont give a shit
				// If the process is restarted this should be false
				if cmd.ProcessState.Exited() {
					break
				}
			}
			cmdErr.Read(buffErr)
			cmdOut.Read(buffOut)

			if len(buffOut) > 0 {
				color.Blue("[%s]:", c.Process.Name)
				fmt.Printf("%s", string(buffOut))
			}
			if len(buffErr) > 0 {
				color.Blue("[%s]:", c.Process.Name)
				fmt.Printf("%s", string(buffOut))
			}
		}
	}

	log.Printf("Exiting context: %s", c.Process.Name)
	// cmdIn.Write([]byte("hello grep\ngoodbye grep"))
	// grepBytes, _ := io.ReadAll(cmdOut)
	// cmdIn.Close()
	// cmdOut.Close()
	// cmdErr.Close()
	cmd.Wait()
	c.wg.Done()
}
