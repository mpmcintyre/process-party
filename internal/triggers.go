package pp

import (
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
func (c *ExecutionContext) CreateFsTrigger(exitChannel chan bool) chan bool {

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
			c.errorWriter.Write([]byte("File/Directory does not exist"))
			watcher.Close()
			return nil
		}
	}

	trigger := make(chan bool)
	filter := c.fileFilter()

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
				c.infoWriter.Printf("FS trigger captured - %s\n", event.String())
				if filter(event.Name) {
					trigger <- true
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

func (c *ExecutionContext) WaitForTrigger() {

}
