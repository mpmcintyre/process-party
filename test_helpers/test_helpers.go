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
// Create a basic command with sleep settings
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

// Create a basic command with touch settings
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

// Create a basic command with makedir settings
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

// Create a basic command with fail settings
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

// Run the custom touch command
func Touch(path string) error {
	x := CreateTouchCmdSettings(path)
	cmd := exec.Command(x.Cmd, x.Args...)
	_, err := cmd.Output()
	return err
}

// Run the custom mkdir command
func Mkdir(path string) error {
	x := CreateMkdirCmdSettings(path)
	cmd := exec.Command(x.Cmd, x.Args...)
	_, err := cmd.Output()
	return err
}
