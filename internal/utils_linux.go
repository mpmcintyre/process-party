package pp

import (
	"syscall"
)

func (c *ExecutionContext) killExecution() error {
	// Ensure we have the PID of the process
	c.executionMutex.Lock()
	defer c.executionMutex.Unlock()
	for {
		if c.cmd != nil {
			c.errorWriter.Printf("CMD is not nil")
			break
		}
	}
	if c.cmd.Process == nil {
		c.errorWriter.Printf("Process is nil")
		return nil
	}

	if c.cmd.ProcessState != nil && c.cmd.ProcessState.Exited() {
		c.errorWriter.Printf("Process already exited")
		return nil
	}

	pid := c.cmd.Process.Pid
	for {
		c.errorWriter.Printf("Process attepted kills: %d", c.killAttemptCounter)

		// https://github.com/air-verse/air/blob/master/runner/util_windows.go
		if err := syscall.Kill(pid, syscall.SIGINT); err != nil {

			c.errorWriter.Write([]byte(err.Error()))
			if c.killAttemptCounter >= 3 {
				break
			}
		} else {
			break
		}
		c.killAttemptCounter += 1
	}
	c.cmd.Process.Kill()
	c.killAttemptCounter = 0
	c.errorWriter.Printf("Process killed")

	return nil
}
