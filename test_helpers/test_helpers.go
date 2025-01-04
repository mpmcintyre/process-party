package testHelpers

import (
	"fmt"
	"runtime"
)

type CmdSettings struct {
	Cmd  string
	Args []string
}

// Exported functions
func CreateSleepCmdSettings(sleepDurationSeconds int) CmdSettings {
	currentOS := runtime.GOOS
	command := "mocks/build/fake_process"
	if currentOS == "windows" {
		command += ".exe"
	}
	return CmdSettings{
		Cmd:  command,
		Args: []string{"sleep", fmt.Sprintf("%d", sleepDurationSeconds)},
	}
}

func CreateTouchCmdSettings(filename string) CmdSettings {
	currentOS := runtime.GOOS
	command := "mocks/build/fake_process"
	if currentOS == "windows" {
		command += ".exe"
	}

	return CmdSettings{
		Cmd:  command,
		Args: []string{"touch", filename},
	}
}

func CreateMkdirCmdSettings(dirname string) CmdSettings {
	currentOS := runtime.GOOS
	command := "mocks/build/fake_process"
	if currentOS == "windows" {
		command += ".exe"
	}

	return CmdSettings{
		Cmd:  command,
		Args: []string{"mkdir", dirname},
	}
}

func CreateFailCmdSettings() CmdSettings {
	currentOS := runtime.GOOS
	command := "mocks/build/fake_process"
	if currentOS == "windows" {
		command += ".exe"
	}

	return CmdSettings{
		Cmd:  command,
		Args: []string{"fail"},
	}
}
