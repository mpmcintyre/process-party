package pp

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/fsnotify/fsnotify"
)

func (c *WatchTaskContext) callback() {
	// Create command
	c.cmd = exec.Command(c.process.Command, c.process.Args...)
	c.cmd.Env = os.Environ() // Set the full environment, including PATH
	// Create IO
	r, w := io.Pipe()
	c.readPipe = r
	c.writePipe = w
	// Write into the command
	c.infoWriter = &customWriter{w: os.Stdout, severity: "info", process: c.process}   // Write info out
	c.errorWriter = &customWriter{w: os.Stdout, severity: "error", process: c.process} // Write errors out
	// Set IO
	c.cmd.Stdout = c.infoWriter
	c.cmd.Stderr = c.errorWriter
	c.cmd.Stdin = r
	c.cmd.Run()
	// Go wait somewhere else lamo (*insert you cant sit with us meme*)
	go c.cmd.Wait()
}

func (c *WatchTaskContext) Watch() {

	c.Task.Status = ExitStatusRunning

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		c.errorWriter.Write([]byte("File/Directory does not exist"))
		return
	}

	// Wait for the start delay
	c.infoWriter.Write([]byte(fmt.Sprintf("Starting watcher - watched: [%s], ignored: [%s]", strings.Join(c.Task.Watch, ","), strings.Join(c.Task.Ingore, ","))))

	for _, item := range c.Task.Watch {
		err := watcher.Add(item)
		if err != nil {
			c.errorWriter.Write([]byte("File/Directory does not exist"))
			return
		}
	}

	for _, item := range c.Task.Ingore {
		err := watcher.Remove(item)
		if err != nil {
			c.errorWriter.Write([]byte("File/Directory does not exist"))
			return
		}
	}

	// Start file watcher
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				go c.callback()

				if strings.HasSuffix(event.Name, ".html") || strings.HasSuffix(event.Name, ".tmpl") {
					log.Println("Template modified:", event.Name)
				} else if strings.HasSuffix(event.Name, ".css") {
					log.Println("Styles modified:", event.Name)
				} else {
					// startErr := c.cmd.Start()("File modified:", event.Name)
				}

			case err, ok := <-watcher.Errors:
				log.Print(err)
				if !ok {
					return
				}
				// startErr := c.cmd.Start()("Watcher error:", err)
			}
		}
	}()

}
