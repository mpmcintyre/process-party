package pp

import (
	"os/exec"
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

	pid := c.Process.Pid

	c.infoWriter.Printf("Killing process - %s", pid)
	// On Windows, we need to kill the entire process group to ensure child processes are terminated
	// https://github.com/air-verse/air/blob/master/runner/util_windows.go
	kill := exec.Command("TASKKILL", "/T", "/F", "/PID", pid)
	if err := kill.Run(); err != nil {
		// Wait 100 miliseconds and try again
		time.Sleep(time.Duration(100 * time.Millisecond))
		kill := exec.Command("TASKKILL", "/T", "/F", "/PID", pid)
		if err := kill.Run(); err != nil {
			c.cmd.Process.Kill()
			return err
		}
	}
	c.cmd.Process.Kill()
	c.internalExit = true
	return nil
}
