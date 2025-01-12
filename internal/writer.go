package pp

import (
	"fmt"
	"io"
	"strings"
	"time"

	color "github.com/fatih/color"
)

type customWriter struct {
	w        io.Writer
	severity string
	process  *Process
	prefix   string
}

// Returns true if the line is empty
func emptyMessage(s string) bool {
	if s == "" || s == " " {
		return true
	}
	// Remove all space characters to ensure there is something else
	x := strings.Replace(s, " ", "", -1) // Spaces
	x = strings.Replace(x, "	", "", -1)  // Tabs
	x = strings.Replace(x, "\t", "", -1) // Tabs
	return x == ""
}

// Creates the prefix so we dont need to do it on every message
func (c *customWriter) createPrefix() {
	if c.process.Prefix == "" && !c.process.DisplayPid {
		c.prefix = ""
		return
	}
	c.prefix = "[" + c.process.Prefix
	if c.process.DisplayPid {
		c.prefix = c.prefix + "-" + c.process.Pid

	}
	c.prefix = c.prefix + "]"
	colourFunc := c.process.GetFgColour()
	c.prefix = colourFunc(c.prefix)
	c.prefix += " "
}

// Utility function to simplyfy printing strings
func (c customWriter) Printf(format string, a ...any) {
	c.Write([]byte(fmt.Sprintf(format, a...)))
}

func (c customWriter) Write(p []byte) (int, error) {
	if c.process.Silent {
		return 0, nil
	}

	if c.prefix == "" && (c.process.Prefix != "" || c.process.DisplayPid) {
		c.createPrefix()
	}

	message := string(p)
	now := time.Now()

	// Format the time as HH:MM:SS:MS
	timeString := fmt.Sprintf("%02d:%02d:%02d:%03d",
		now.Hour(),
		now.Minute(),
		now.Second(),
		now.Nanosecond()/1e6) // Convert nanoseconds to milliseconds

	// Take out common line seperation
	message = strings.Replace(message, "\r", "", -1)
	x := strings.Split(message, "\n")

	if c.process.SeperateNewLines {
		for _, message := range x {
			if emptyMessage(message) {
				continue
			}

			if c.severity == "error" {
				message = color.RedString(message)
			}
			if c.process.ShowTimestamp {
				message = timeString + "	" + message
			}
			n, err := c.w.Write([]byte(c.prefix + message + "\n"))
			if err != nil {
				return n, err
			}

		}
	} else {
		if c.severity == "error" {
			message = color.RedString(message)
		}
		if c.process.ShowTimestamp {
			message = timeString + "	" + message
		}
		n, err := c.w.Write([]byte(c.prefix + message + "\n"))
		if err != nil {
			return n, err
		}
	}

	return len(p), nil
}
