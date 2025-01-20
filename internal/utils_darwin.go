package pp

import (
	"syscall"
	"time"
)

func (c *ExecutionContext) killExecution() error {
	// Ensure we have the PID of the process
	c.executionMutex.Lock()
	defer c.executionMutex.Unlock()

	if c.cmd == nil {
		return nil
	}

	if c.cmd.Process == nil {
		return nil
	}

	if c.cmd.ProcessState != nil && c.cmd.ProcessState.Exited() {
		return nil
	}

	pid := c.cmd.Process.Pid
	c.infoWriter.Printf("Killing process - %d", pid)

	// Send SIGINT signal first
	if err := syscall.Kill(pid, syscall.SIGINT); err != nil {
		// Wait 100 milliseconds and try again
		time.Sleep(time.Duration(100 * time.Millisecond))
		if err := syscall.Kill(pid, syscall.SIGINT); err != nil {
			c.cmd.Process.Kill()
			return err
		}
	}
	c.cmd.Process.Kill()
	c.internalExit = true
	return nil
}
