package testHelpers

import "fmt"

type CmdSettings struct {
	Cmd  string
	Args []string
}

// Exported functions
func CreateSleepCmdSettings(sleepDurationSeconds int) CmdSettings {
	return CmdSettings{
		Cmd:  "go",
		Args: []string{"run", "./mocks/fake_process.go", "sleep", fmt.Sprintf("%d", sleepDurationSeconds)},
	}
}

func CreateTouchCmdSettings(filename string) CmdSettings {
	return CmdSettings{
		Cmd:  "go",
		Args: []string{"run", "./mocks/fake_process.go", "touch", filename},
	}
}

func CreateMkdirCmdSettings(dirname string) CmdSettings {
	return CmdSettings{
		Cmd:  "go",
		Args: []string{"run", "./mocks/fake_process.go", "mkdir", dirname},
	}
}

func CreateFailCmdSettings() CmdSettings {
	return CmdSettings{
		Cmd:  "go",
		Args: []string{"run", "./mocks/fake_process.go", "fail"},
	}
}
