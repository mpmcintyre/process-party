package pp

import (
	"os/exec"
	"strconv"
)

func (c *ExecutionContext) killExecution() error {
	// Ensure we have the PID of the process
	if c.cmd.Process == nil {
		return nil
	}

	if c.cmd.ProcessState != nil && c.cmd.ProcessState.Exited() {
		return nil
	}

	// On Windows, we need to kill the entire process group to ensure child processes are terminated
	// https://github.com/air-verse/air/blob/master/runner/util_windows.go
	kill := exec.Command("TASKKILL", "/T", "/F", "/PID", strconv.Itoa(c.cmd.Process.Pid))
	err := kill.Run()
	c.killAttemptCounter += 1
	if err != nil {
		if c.killAttemptCounter > 3 {
			return err
		}
		c.killExecution()
	}
	c.cmd.Process.Kill()
	c.killAttemptCounter = 0

	c.setProcessStatus(ProcessStatusExited)
	return nil
}
