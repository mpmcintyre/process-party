package pp

import (
	"os/exec"
	"strconv"
)

func (c *ExecutionContext) killExecution() error {
	// Ensure we have the PID of the process
	c.executionMutex.Lock()
	defer c.executionMutex.Unlock()
	for {
		if c.cmd != nil {
			break
		}
	}
	if c.cmd.Process == nil {
		return nil
	}

	if c.cmd.ProcessState != nil && c.cmd.ProcessState.Exited() {
		return nil
	}

	pid := strconv.Itoa(c.cmd.Process.Pid)
	// On Windows, we need to kill the entire process group to ensure child processes are terminated
	// https://github.com/air-verse/air/blob/master/runner/util_windows.go
	for {
		kill := exec.Command("TASKKILL", "/T", "/F", "/PID", pid)
		if err := kill.Run(); err != nil {
			if c.killAttemptCounter >= 3 {
				break
			}
			c.killExecution()
		} else {
			break
		}
		c.killAttemptCounter += 1
	}
	c.cmd.Process.Kill()
	c.killAttemptCounter = 0
	return nil
}
