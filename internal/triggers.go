package pp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

func (c *ExecutionContext) fileFilter() func(string) bool {
	return func(path string) bool {
		// First check exact matches in included items
		for _, item := range c.Process.Trigger.FileSystem.Watch {
			if item == path {
				return true
			}
		}

		// Then check exact matches in excluded items
		for _, item := range c.Process.Trigger.FileSystem.Ignore {
			if item == path {
				return false
			}
		}

		// Check if path matches any of the exclude patterns
		for _, pattern := range c.Process.Trigger.FileSystem.Ignore {
			matched, err := filepath.Match(pattern, filepath.Base(path))
			if err == nil && matched {
				return false
			}
		}

		// Finally, check if path matches any of the include patterns
		for _, pattern := range c.Process.Trigger.FileSystem.ContainFilters {
			matched, err := filepath.Match(pattern, filepath.Base(path))
			if err == nil && matched {
				return true
			}
		}

		// If we are looking for specific files and we reach here we did not find any, else we can use it
		return len(c.Process.Trigger.FileSystem.ContainFilters) == 0
	}
}

func (c *ExecutionContext) recursivelyWatch(event fsnotify.Event, watcher *fsnotify.Watcher) {
	if c.Process.Trigger.FileSystem.NonRecursive {
		return
	}

	if event.Op == fsnotify.Create {
		parentPath := filepath.Dir(event.Name)
		dirs, err := os.ReadDir(parentPath)
		if err != nil {
			c.errorWriter.Write([]byte(fmt.Sprintf("Could not read directory %s, %s", event.Name, err.Error())))
		} else {
			for _, dir := range dirs {
				dirEntries := strings.Split(event.Name, string(os.PathSeparator))
				if dir.Name() == dirEntries[len(dirEntries)-1] && dir.Type().IsDir() {
					c.infoWriter.Write([]byte(fmt.Sprintf("A new subdirectory was created in FS watcher, monitoring %s", event.Name)))
					err := watcher.Add(event.Name)
					if err != nil {
						c.errorWriter.Write([]byte(fmt.Sprintf("Could not monitor file, error: %s", err.Error())))
					}

				}
			}
		}
	}
}

// This creates a trigger that watches any directories and recursive subdirectories
func (c *ExecutionContext) CreateFsTrigger() (chan string, error) {

	if len(c.Process.Trigger.FileSystem.Watch) <= 0 {
		return nil, nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		c.errorWriter.Write([]byte("File/Directory does not exist"))
		return nil, err
	}

	if c.Process.RestartAttempts != 0 {
		c.errorWriter.Printf("Process contains a trigger and restart attempts")
		return nil, errors.New("Restarting triggered processes can lead to undesired behaviour. Remove triggers or restart attempts on process [" + c.Process.Name + "]")
	}

	for _, item := range c.Process.Trigger.FileSystem.Watch {
		err := watcher.Add(item)
		if err != nil {
			c.errorWriter.Write([]byte("File/Directory does not exist: " + item))
			watcher.Close()
			return nil, err
		}
	}

	trigger := make(chan string)
	filter := c.fileFilter()
	exitChannel := c.getInternalExitNotifier()

	// Start file watcher
	go func() {
		debounceTimer := time.Now()
		debounceTime := 5 // 5 ms

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					c.errorWriter.Write([]byte("An unexpected error occured while watching"))
					watcher.Close()
					return
				}

				if filter(event.Name) && time.Since(debounceTimer) > time.Duration(debounceTime)*time.Millisecond {
					c.recursivelyWatch(event, watcher)

					trigger <- fmt.Sprintf("FS trigger captured - %s\n", event.String())
					debounceTimer = time.Now()
				}

			case err, ok := <-watcher.Errors:
				c.errorWriter.Write([]byte(fmt.Sprintf("An unexpected error occured, %s", err.Error())))
				if !ok {
					watcher.Close()
					return
				}
			case _, ok := <-exitChannel:
				if !ok {
					watcher.Close()
				}
				return
			}
		}
	}()

	return trigger, nil
}

func (e *ExecutionContext) CreateProcessTrigger(signal ProcessStatus, message string) chan string {
	trigger := make(chan string, 10)

	go func() {
		exitChannel := e.getInternalExitNotifier()
		sigChannel := e.GetProcessNotificationChannel()
	monitorLoop:
		for {
			select {
			case sig := <-sigChannel:
				if signal == sig {
					if trigger != nil {
						trigger <- message
					}
				}
			case <-exitChannel:
				close(trigger)
				break monitorLoop
			}
		}
	}()

	return trigger
}

func contains(arr []string, target string) bool {
	for _, value := range arr {
		if value == target {
			return true
		}
	}
	return false
}

// Links process triggers together
func LinkProcessTriggers(contexts []*ExecutionContext) error {
	// Filesystem triggers
	for _, context := range contexts {
		fsTrigger, err := context.CreateFsTrigger()
		if err != nil {
			return err
		}
		if fsTrigger != nil {
			context.triggers = append(context.triggers, fsTrigger)
		}
	}

	// Process triggers

	// Create a map for quick access and checking circular triggers
	x := map[string]*ExecutionContext{}
	for _, context := range contexts {
		x[context.Process.Name] = context
	}

	applyTriggers := func(triggers []string, signal ProcessStatus, context *ExecutionContext) error {

		for _, process := range triggers {
			if value, exists := x[process]; exists {
				if contains(value.Process.Trigger.Process.OnComplete, process) {
					return errors.New("Circular trigger detected: " + value.Process.Name + " and " + process + " trigger each other")
				}
				if contains(value.Process.Trigger.Process.OnEnd, process) {
					return errors.New("Circular trigger detected: " + value.Process.Name + " and " + process + " trigger each other")
				}
				if contains(value.Process.Trigger.Process.OnError, process) {
					return errors.New("Circular trigger detected: " + value.Process.Name + " and " + process + " trigger each other")
				}
				if contains(value.Process.Trigger.Process.OnStart, process) {
					return errors.New("Circular trigger detected: " + value.Process.Name + " and " + process + " trigger each other")
				}
				trigger := value.CreateProcessTrigger(signal, fmt.Sprintf("[%s] triggered a run", process))
				context.triggers = append(context.triggers, trigger)
			} else {
				return errors.New("Specified target process for trigger does not exist on " + context.Process.Name + ", Non existant trigger = " + process)
			}
		}
		if len(triggers) > 0 && context.Process.RestartAttempts != 0 {
			context.errorWriter.Printf("Process contains a trigger and restart attempts")
			return errors.New("Restarting triggered processes can lead to undesired behaviour. Remove triggers or restart attempts on process [" + context.Process.Name + "]")
		}
		return nil
	}

	for _, context := range contexts {
		// On successfull completion
		err := applyTriggers(context.Process.Trigger.Process.OnComplete, ProcessStatusExited, context)
		if err != nil {
			return err
		}
		// On error
		err = applyTriggers(context.Process.Trigger.Process.OnError, ProcessStatusFailed, context)
		if err != nil {
			return err
		}
		// On any end
		err = applyTriggers(context.Process.Trigger.Process.OnEnd, ProcessStatusExited, context)
		if err != nil {
			return err
		}
		err = applyTriggers(context.Process.Trigger.Process.OnEnd, ProcessStatusFailed, context)
		if err != nil {
			return err
		}
		// On start
		err = applyTriggers(context.Process.Trigger.Process.OnStart, ProcessStatusRunning, context)
		if err != nil {
			return err
		}

	}

	return nil
}
