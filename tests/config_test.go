package tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"testing"

	"github.com/BurntSushi/toml"
	runner "github.com/mpmcintyre/process-party/internal"
	"gopkg.in/yaml.v3"
)

var tpExit runner.ExitCommand = "wait"
var tpColour runner.ColourCode = "green"
var tpDelays int = 1
var tpPID string = "123"

// Creates a process with non-default values
func createProcess(increment int, nameStamp string) runner.Process {
	return runner.Process{
		Name:          nameStamp + fmt.Sprintf("%d", increment),
		Command:       nameStamp + fmt.Sprintf("%d", increment),
		Args:          []string{nameStamp + fmt.Sprintf("%d", increment)},
		Prefix:        nameStamp + fmt.Sprintf("%d", increment),
		Color:         tpColour,
		OnFailure:     tpExit,
		OnComplete:    tpExit,
		DisplayPid:    true,
		Delay:         tpDelays,
		TimeoutOnExit: tpDelays,
		RestartDelay:  tpDelays,
		Status:        runner.ExitStatusRunning,
		Pid:           tpPID,
		// These must be set by the config file not the process
		ShowTimestamp:    false,
		SeperateNewLines: false,
	}
}

// Returns true if a default value is found
func testForDefaultProcessValues(process runner.Process) bool {
	dp := runner.Process{}
	if process.Name == dp.Name {
		return false
	}
	if process.Command == dp.Command {
		return false
	}
	if reflect.DeepEqual(process.Args, dp.Args) {
		return false
	}
	if process.Prefix == dp.Prefix {
		return false
	}
	if process.Color == dp.Color {
		return false
	}
	if process.OnFailure == dp.OnFailure {
		return false
	}
	if process.OnComplete == dp.OnComplete {
		return false
	}
	if process.DisplayPid == dp.DisplayPid {
		return false
	}
	if process.Delay == dp.Delay {
		return false
	}
	if process.TimeoutOnExit == dp.TimeoutOnExit {
		return false
	}
	if process.RestartDelay == dp.RestartDelay {
		return false
	}
	if process.Status == dp.Status {
		return false
	}
	if process.Pid == dp.Pid {
		return false
	}
	// These must be set by the config file not the process
	if process.ShowTimestamp == dp.ShowTimestamp {
		return false
	}
	if process.SeperateNewLines == dp.SeperateNewLines {
		return false
	}

	return true
}

// exists returns whether the given file or directory exists
func dirExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func writeFile(data []byte, dir string, filename string) error {
	file, err := os.Create(dir + filename)
	if err != nil {
		return err
	}
	file.Write(data)
	file.Close()
	return nil
}

func createNonDefaultConfig(numberOfProcesses int, nameStamp string, tempDir string, testName string) error {

	if !dirExists(tempDir) {
		os.Mkdir(tempDir, fs.ModeDir)
	}

	defaultProcess := runner.Process{}

	config := runner.CreateConfig()

	if len(config.Processes) > 0 {
		return errors.New("non empty processes in config")
	}

	// Create x processes
	for index := range numberOfProcesses {
		p := createProcess(index, nameStamp)
		config.Processes = append(config.Processes, p)
	}

	for _, process := range config.Processes {
		if reflect.DeepEqual(process, defaultProcess) {
			return errors.New("default processes in config")
		}
	}

	// Set the global settings in the config to non default values
	config.SeperateNewLines = true
	config.ShowTimestamp = true

	jString, err := json.Marshal(config)
	if err != nil {
		return err
	}
	ymlString, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	tomlString, err := toml.Marshal(config)
	if err != nil {
		return err
	}

	err = writeFile(jString, tempDir, testName+".json")
	if err != nil {
		return err
	}
	err = writeFile(ymlString, tempDir, testName+".yml")
	if err != nil {
		return err
	}
	err = writeFile(ymlString, tempDir, testName+".yaml")
	if err != nil {
		return err
	}
	err = writeFile(tomlString, tempDir, testName+".toml")
	if err != nil {
		return err
	}

	return nil
}

// Create configs in every filetype with non-default values, write them to files, and parse the files
// - Make sure that all settings are non-default
func TestConfigParsing(t *testing.T) {
	t.Log("Testing config parsing")
	tempDir := "./.tmp/"
	testName := "test"
	numberOfTestProcesses := 50
	err := createNonDefaultConfig(numberOfTestProcesses, "tmp", tempDir, testName)
	if err != nil {
		t.Fatal(err.Error())
	}

	jsonConfig := runner.CreateConfig()
	ymlConfig := runner.CreateConfig()
	yamlConfig := runner.CreateConfig()
	tomlConfig := runner.CreateConfig()

	jsonConfig.ParseFile(tempDir + testName + ".json")
	ymlConfig.ParseFile(tempDir + testName + ".yml")
	yamlConfig.ParseFile(tempDir + testName + ".yaml")
	tomlConfig.ParseFile(tempDir + testName + ".toml")

	// Default config seperate new lines and show timestamp should be true
	if !jsonConfig.SeperateNewLines || !jsonConfig.ShowTimestamp {
		t.Fatalf("config contains default value")
	}

	for index := range numberOfTestProcesses {
		if testForDefaultProcessValues(jsonConfig.Processes[index]) {
			t.Fatalf("process config contains default value")
		}
		if testForDefaultProcessValues(ymlConfig.Processes[index]) {
			t.Fatalf("process config contains default value")
		}
		if testForDefaultProcessValues(yamlConfig.Processes[index]) {
			t.Fatalf("process config contains default value")
		}
		if testForDefaultProcessValues(tomlConfig.Processes[index]) {
			t.Fatalf("process config contains default value")
		}
	}

	t.Cleanup(func() {
		// Remove temp directory
		files, err := os.ReadDir(tempDir)
		if err != nil {
			t.Fatal(err.Error())
		}
		for _, file := range files {
			err := os.Remove("./.tmp/" + file.Name())
			if err != nil {
				t.Fatal(err.Error())
			}
		}
		err = os.Remove(tempDir)
		if err != nil {
			t.Fatal(err.Error())
		}

	})
}