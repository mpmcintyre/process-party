package testHelpers

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
)

var command = filepath.Join("mocks", "build", "fake_process")

type CmdSettings struct {
	Cmd  string
	Args []string
}

// Exported functions
func CreateSleepCmdSettings(sleepDurationSeconds int) CmdSettings {
	currentOS := runtime.GOOS
	local := command
	if currentOS == "windows" {
		local += ".exe"
	}
	return CmdSettings{
		Cmd:  local,
		Args: []string{"sleep", fmt.Sprintf("%d", sleepDurationSeconds)},
	}
}

func CreateTouchCmdSettings(filename string) CmdSettings {
	currentOS := runtime.GOOS
	local := command

	if currentOS == "windows" {
		local += ".exe"
	}

	return CmdSettings{
		Cmd:  local,
		Args: []string{"touch", filename},
	}
}

func CreateMkdirCmdSettings(dirname string) CmdSettings {
	currentOS := runtime.GOOS
	local := command

	if currentOS == "windows" {
		local += ".exe"
	}
	return CmdSettings{
		Cmd:  local,
		Args: []string{"mkdir", dirname},
	}
}

func CreateFailCmdSettings() CmdSettings {
	currentOS := runtime.GOOS
	local := command

	if currentOS == "windows" {
		local += ".exe"
	}
	return CmdSettings{
		Cmd:  local,
		Args: []string{"fail"},
	}
}

func Touch(path string) error {
	x := CreateTouchCmdSettings(path)
	cmd := exec.Command(x.Cmd, x.Args...)
	_, err := cmd.Output()
	return err
}

func Mkdir(path string) error {
	x := CreateMkdirCmdSettings(path)
	cmd := exec.Command(x.Cmd, x.Args...)
	_, err := cmd.Output()
	return err
}
