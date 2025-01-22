package pp

import (
	"strconv"
	"syscall"
	"time"
)

func (c *ExecutionContext) killExecution() error {
	c.internalExit.Store(true)
	c.executionMutex.Lock()
	defer c.executionMutex.Unlock()

	if c.cmd == nil {
		return nil
	}

	if c.cmd.Process == nil {
		return nil
	}

	if c.cmd.ProcessState != nil {
		return nil
	}

	if c.Process.Pid == "" {
		return nil
	}

	pid, err := strconv.Atoi(c.Process.Pid)
	if err != nil {
		return err
	}

	c.infoWriter.Printf("Killing process - %d", pid)
	// https://github.com/air-verse/air/blob/master/runner/util_windows.go
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
		// Wait 100 milliseconds and try again
		time.Sleep(time.Duration(100 * time.Millisecond))
		if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
			c.cmd.Process.Kill()
			return err
		}
	}
	c.cmd.Process.Kill()
	return nil
}
