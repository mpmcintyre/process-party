package runner

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
	process  Process
}

func emptyMessage(s string) bool {
	if s == "" || s == " " {
		return true
	}
	// Remove all space characters to ensure there is something else
	x := strings.Replace(s, " ", "", -1) // Spaces
	x = strings.Replace(x, "	", "", -1)  // Tabs
	return x == ""
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

	// Take out common line spacing
	message = strings.Replace(message, "\r", "", -1)
	message = strings.Replace(message, "\n\n", "", -1)
	message = strings.Replace(message, "\r\n", "", -1)
	x := strings.Split(message, "\n")

	if e.process.SeperateNewLines {
		for _, message := range x {
			if emptyMessage(message) {
				continue
			}
			colourFunc := e.process.GetFgColour()
			prefix = colourFunc(prefix)
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
		colourFunc := e.process.GetFgColour()
		prefix = colourFunc(prefix)
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
