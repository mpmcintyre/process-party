package pp

import (
	"syscall"
)

func (c *ExecutionContext) killExecution() error {
	// Ensure we have the PID of the process
	if c.cmd.Process == nil {
		return nil
	}

	if c.cmd.ProcessState != nil && c.cmd.ProcessState.Exited() {
		return nil
	}

	c.killAttemptCounter += 1
	// https://github.com/air-verse/air/blob/master/runner/util_windows.go
	if err := syscall.Kill(c.cmd.Process.Pid, syscall.SIGINT); err != nil {
		if c.killAttemptCounter >= 3 {
			return err
		}
		c.killExecution()
	}
	c.cmd.Process.Kill()
	c.killAttemptCounter = 0
	c.setProcessStatus(ProcessStatusExited)
	return nil
}
