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

// Returns true if the trigger should run, false if it should not
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

// Recursivley watches directories if enabled
func (c *ExecutionContext) watch(path string, watcher *fsnotify.Watcher) error {

	err := watcher.Add(path)
	if err != nil {
		c.errorWriter.Write([]byte(fmt.Sprintf("Could not monitor file or folder, path: %s\nerror: %s", path, err.Error())))
		return err
	}

	// If the entry is a folder recursivley call watch if enabled
	dirs, err := os.ReadDir(path)
	if err == nil {
		for _, dir := range dirs {
			if dir.Type().IsDir() {
				if c.Process.Trigger.FileSystem.NonRecursive {
					return nil
				}
				err := c.watch(filepath.Join(path, dir.Name()), watcher)
				if err != nil {
					return nil
				}
			}
		}
	}

	return nil
}

// Checks if the fs event was a creation event and if the item is a directory. If it is it adds it to the watcher
func (c *ExecutionContext) recursivelyWatchCreatedEvent(event fsnotify.Event, watcher *fsnotify.Watcher) {
	if c.Process.Trigger.FileSystem.NonRecursive {
		return
	}

	if event.Op == fsnotify.Create {
		// Get the filetype of the created event
		parentPath := filepath.Dir(event.Name)
		dirs, err := os.ReadDir(parentPath)
		if err != nil {
			c.errorWriter.Write([]byte(fmt.Sprintf("Could not read directory %s, %s", parentPath, err.Error())))
		} else {
			for _, dir := range dirs {

				dirEntries := strings.Split(event.Name, string(os.PathSeparator))
				if dir.Name() == dirEntries[len(dirEntries)-1] && dir.Type().IsDir() {
					absPath, err := filepath.Abs(filepath.Clean(event.Name))
					if err != nil {
						c.errorWriter.Write([]byte("Invalid path: " + event.Name))
						return
					}
					c.infoWriter.Printf("A new subdirectory was created in FS watcher, monitoring %s", absPath)
					c.watch(absPath, watcher)
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

	addedPaths := []string{}

	for _, item := range c.Process.Trigger.FileSystem.Watch {
		absPath, err := filepath.Abs(filepath.Clean(item))
		if err != nil {
			c.errorWriter.Write([]byte("Invalid path: " + item))
			return nil, err
		}

		if contains(addedPaths, absPath) {
			c.errorWriter.Printf("Duplicate trigger path: \"%s\" - not monitoring twice", item)
			continue
		}

		c.infoWriter.Printf("Monitoring path: %s", absPath)

		err = c.watch(absPath, watcher)

		if err != nil {
			c.errorWriter.Write([]byte("File/Directory does not exist: " + item))
			watcher.Close()
			return nil, err
		}
		addedPaths = append(addedPaths, absPath)
	}

	trigger := make(chan string)
	filter := c.fileFilter()
	exitChannel := c.getInternalExitNotifier()

	// Start file watcher
	go func() {
		defer close(trigger)

		debounceTimer := time.Now()
		debounceTime := c.Process.Trigger.FileSystem.DebounceTime //  ms

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					c.errorWriter.Write([]byte("An unexpected error occured while watching"))
					watcher.Close()
					return
				}
				if filter(event.Name) {
					c.recursivelyWatchCreatedEvent(event, watcher)
					if time.Since(debounceTimer) > time.Duration(debounceTime)*time.Millisecond {
						filepath := strings.Split(event.Name, string(os.PathSeparator))
						trigger <- fmt.Sprintf("FS trigger captured - %s	%s", event.Op, filepath[len(filepath)-1])
						debounceTimer = time.Now()
					}
				}

			case err, ok := <-watcher.Errors:
				c.errorWriter.Write([]byte(fmt.Sprintf("An unexpected error occured, %s", err.Error())))
				if !ok {
					watcher.Close()
				}
			case <-exitChannel:
				c.infoWriter.Printf("Process exiting, closing FS watcher")
				watcher.Close()
				return
			}
		}
	}()
	return trigger, nil
}

// Creates a channel that runs when the contexts emits the listening signal
func (e *ExecutionContext) CreateProcessTrigger(signal ProcessStatus, message string) chan string {
	trigger := make(chan string)

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

// Utility function to check if a slice contains a string value
func contains(arr []string, target string) bool {
	for _, value := range arr {
		if value == target {
			return true
		}
	}
	return false
}

// Links process triggers together for a range of execution contexts
func LinkProcessTriggers(contexts []*ExecutionContext) error {
	// Filesystem triggers
	for _, context := range contexts {
		fsTrigger, err := context.CreateFsTrigger()
		if err != nil {
			return err
		}
		if fsTrigger != nil {
			if context.Process.RestartAttempts != 0 && (context.Process.OnComplete == ExitCommandRestart || context.Process.OnFailure == ExitCommandRestart) {
				context.errorWriter.Printf("Process contains a trigger and restarts on exit/failure with 1 or more restart attempts")
				return errors.New("Restarting triggered processes can lead to undesired behaviour. Remove triggers or restart attempts on process [" + context.Process.Name + "]")
			}
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
		monitoredProcesses := []string{}
		for _, process := range triggers {
			if process == context.Process.Name {
				return errors.New("Circular trigger detected: " + process + " canot depend on itself")
			}
			if contains(monitoredProcesses, process) {
				context.errorWriter.Printf("Duplicate trigger process: \"%s\" - not monitoring twice", process)
				continue
			}

			monitoredProcesses = append(monitoredProcesses, process)

			if value, exists := x[process]; exists {
				if contains(value.Process.Trigger.Process.OnComplete, process) {
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
		if len(triggers) > 0 {
			if context.Process.RestartAttempts != 0 && (context.Process.OnComplete == ExitCommandRestart || context.Process.OnFailure == ExitCommandRestart) {
				context.errorWriter.Printf("Process contains a trigger and restarts on exit/failure with 1 or more restart attempts")
				return errors.New("Restarting triggered processes can lead to undesired behaviour. Remove triggers or restart attempts on process [" + context.Process.Name + "]")
			}
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
		// On start
		err = applyTriggers(context.Process.Trigger.Process.OnStart, ProcessStatusRunning, context)
		if err != nil {
			return err
		}

	}

	return nil
}
