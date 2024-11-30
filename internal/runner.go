package runner

// Custom writers-  https://medium.com/@shubhamagrawal094/custom-writer-in-golang-171dd2cac7e0
import (
	"fmt"
	"io"
	"os/exec"

	color "github.com/fatih/color"
)

type Context struct {
	Process      Process
	StdOut       chan string
	StdIn        chan string
	EndOfCommand chan bool
	BuzzKill     chan bool
	w            io.Writer
	// outb, errb   bytes.Buffer
}

func (c Context) Write(p []byte) (int, error) {
	color.Blue("Prints %s in blue.", "text")
	n, err := c.w.Write(p)
	if err != nil {
		return n, err
	}
	if n != len(p) {
		return n, io.ErrShortWrite
	}
	return len(p), nil
}

func CreateContext(p Process) Context {
	return Context{
		Process:      p,
		StdOut:       make(chan string),
		StdIn:        make(chan string),
		EndOfCommand: make(chan bool),
		BuzzKill:     make(chan bool),
	}
}

func (c *Context) Run() {
	cmd := exec.Command(c.Process.Command, c.Process.Args...)
	cmdIn, _ := cmd.StdinPipe()
	grepOut, _ := cmd.StdoutPipe()
	cmd.Start()
	cmd.Stdout = c.w
	// buffIn := []byte{}
	buffOut := []byte{}

	for {
		grepOut.Read(buffOut)
		if len(buffOut) > 0 {
			fmt.Printf("")
		}
		select {
		case <-c.StdIn:
			fmt.Print(c.StdIn)
		}
	}
	// cmdIn.Write([]byte("hello grep\ngoodbye grep"))
	cmdIn.Close()
	// grepBytes, _ := io.ReadAll(grepOut)
	cmd.Wait()

}
