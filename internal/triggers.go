package pp

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

func (c *ExecutionContext) fileFilter() func(string) bool {
	// Convert glob patterns to regular expressions
	convertGlobToRegex := func(pattern string) string {
		// Escape special regex characters except *
		special := []string{".", "+", "?", "^", "$", "[", "]", "(", ")", "{", "}", "\\", "|"}
		for _, ch := range special {
			pattern = strings.ReplaceAll(pattern, ch, "\\"+ch)
		}
		// Convert glob * to regex .*
		pattern = strings.ReplaceAll(pattern, "*", ".*")
		return "^" + pattern + "$"
	}

	// Precompile patterns for better performance
	includePatterns := make([]string, len(c.Process.Trigger.FileSystem.ContainFilters))
	for i, pattern := range c.Process.Trigger.FileSystem.ContainFilters {
		includePatterns[i] = convertGlobToRegex(pattern)
	}

	excludePatterns := make([]string, len(c.Process.Trigger.FileSystem.Ignore))
	for i, pattern := range c.Process.Trigger.FileSystem.Ignore {
		excludePatterns[i] = convertGlobToRegex(pattern)
	}

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
		for _, pattern := range excludePatterns {
			matched, err := filepath.Match(pattern, filepath.Base(path))
			if err == nil && matched {
				return false
			}
		}

		// Finally, check if path matches any of the include patterns
		for _, pattern := range includePatterns {
			matched, err := filepath.Match(pattern, filepath.Base(path))
			if err == nil && matched {
				return true
			}
		}

		return false
	}
}

func (c *ExecutionContext) CreateFsTrigger() chan string {

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		c.errorWriter.Write([]byte("File/Directory does not exist"))
		return nil
	}

	if len(c.Process.Trigger.FileSystem.Watch) == 0 {
		watcher.Close()
		return nil
	}

	for _, item := range c.Process.Trigger.FileSystem.Watch {
		err := watcher.Add(item)
		if err != nil {
			c.errorWriter.Write([]byte("File/Directory does not exist: " + item))
			watcher.Close()
			return nil
		}
	}

	trigger := make(chan string)
	filter := c.fileFilter()
	exitChannel := c.getInternalExitNotifier()

	// Start file watcher
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					c.errorWriter.Write([]byte("An unexpected error occured while watching"))
					watcher.Close()
					return
				}
				if filter(event.Name) {
					trigger <- fmt.Sprintf("FS trigger captured - %s\n", event.String())
				}

			case err, ok := <-watcher.Errors:
				c.errorWriter.Write([]byte(fmt.Sprintf("An unexpected error occured, %s", err.Error())))
				if !ok {
					watcher.Close()
					return
				}
			case <-exitChannel:
				watcher.Close()
				return
			}
		}
	}()

	return trigger
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
