package tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/BurntSushi/toml"
	pp "github.com/mpmcintyre/process-party/internal"
	"gopkg.in/yaml.v3"
)

var tpExit pp.ExitCommand = "wait"
var tpColour pp.ColourCode = "green"
var tpDelays int = 1
var tpPID string = "123"
var tpRestartAttempts int = 1
var startStream string = "start"

// Creates a process with non-default values
func createRunTask(increment int, nameStamp string) pp.Process {
	return pp.Process{
		OnFailure:       tpExit,
		OnComplete:      tpExit,
		RestartAttempts: tpRestartAttempts,
		RestartDelay:    tpDelays,
		Delay:           tpDelays,
		Name:            nameStamp + fmt.Sprintf("%d", increment),
		Command:         nameStamp + fmt.Sprintf("%d", increment),
		Args:            []string{nameStamp + fmt.Sprintf("%d", increment)},
		Prefix:          nameStamp + fmt.Sprintf("%d", increment),
		Color:           tpColour,
		DisplayPid:      true,
		StartStream:     startStream,
		Pid:             tpPID,
		Silent:          true,
		// These must be set by the config file not the process
		ShowTimestamp: false,
		Trigger: pp.Trigger{
			RunOnStart: true,
			EndOnNew:   true,
			FileSystem: pp.FileSystemTrigger{
				DebounceTime:   50,
				NonRecursive:   true,
				Watch:          []string{"test"},
				Ignore:         []string{"test"},
				ContainFilters: []string{"test"},
			},
			Process: pp.ProcessTrigger{
				OnStart:    []string{"test"},
				OnComplete: []string{"test"},
				OnError:    []string{"test"},
			},
		},
	}
}

// Recursively searches maps (config) to ensure that there is not one value equal to one another
func recursiveSearchForEq(itemA map[string]interface{}, itemB map[string]interface{}) bool {
	// Handle the case where itemB is nil
	if itemB == nil {
		return false
	}

	for key, valueA := range itemA {
		// Check if the key exists in itemB
		valueB, exists := itemB[key]
		if !exists {
			continue
		}

		// Handle nested maps
		if mapA, isMapA := valueA.(map[string]interface{}); isMapA {
			if mapB, isMapB := valueB.(map[string]interface{}); isMapB {
				if recursiveSearchForEq(mapA, mapB) {
					// fmt.Printf("Map %s contains a default value\n", key)
					fmt.Printf("<--%s", key)
					return true
				}
			}
			continue
		}

		// Compare non-map values
		if reflect.DeepEqual(valueA, valueB) {
			fmt.Printf("Key %s is equal to the default value, Value: %v\n", key, valueA)
			fmt.Printf("%v<--%s", valueA, key)
			return true
		}
	}

	return false
}

// Returns true if a default value is found
func containsDefaultValues(process pp.Process) bool {
	dp := pp.Process{}
	dpString, err := json.Marshal(dp)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return true
	}
	pString, err := json.Marshal(process)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return true
	}
	var resultA map[string]interface{}
	err = json.Unmarshal([]byte(dpString), &resultA)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return true
	}
	var resultB map[string]interface{}
	err = json.Unmarshal([]byte(pString), &resultB)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return true
	}

	if recursiveSearchForEq(resultA, resultB) {
		fmt.Printf("<--config\n")
		fmt.Println("The test configuration contains a default value, this usually implies that the config was modified and the parsing is not tested properly")
		return true
	}

	return false
}

// Exists returns whether the given file or directory exists
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

// Utility function to write file
func writeFile(data []byte, dir string, filename string) error {
	file, err := os.Create(dir + filename)
	if err != nil {
		return err
	}
	file.Write(data)
	file.Close()
	return nil
}

// Creates a configuration where no value is equal to the default value (this ensures that new items are accounted for)
func createNonDefaultConfig(numberOfProcesses int, nameStamp string, tempDir string, testName string) error {

	if !dirExists(tempDir) {
		os.MkdirAll(tempDir, 0755)
	}

	defaultProcess := pp.Process{}

	config := pp.CreateConfig()

	if len(config.Processes) > 0 {
		return errors.New("non empty processes in config")
	}

	// Create x processes
	for index := range numberOfProcesses {
		p := createRunTask(index, nameStamp)
		config.Processes = append(config.Processes, p)
	}

	for _, process := range config.Processes {
		if reflect.DeepEqual(process, defaultProcess) {
			return errors.New("default processes in config")
		}
	}

	// Set the global settings in the config to non default values
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
	t.Parallel()
	tempDir := "./.tmp/config/"
	testName := "test"
	numberOfTestProcesses := 50
	err := createNonDefaultConfig(numberOfTestProcesses, "tmp", tempDir, testName)
	if err != nil {
		t.Fatal(err.Error())
	}

	jsonConfig := pp.CreateConfig()
	ymlConfig := pp.CreateConfig()
	yamlConfig := pp.CreateConfig()
	tomlConfig := pp.CreateConfig()

	jsonConfig.ParseFile(tempDir+testName+".json", true)
	ymlConfig.ParseFile(tempDir+testName+".yml", true)
	yamlConfig.ParseFile(tempDir+testName+".yaml", true)
	tomlConfig.ParseFile(tempDir+testName+".toml", true)

	// Default config timestamp should be true
	if !jsonConfig.ShowTimestamp {
		t.Fatalf("config contains default value")
	}

	for index := range numberOfTestProcesses {
		if containsDefaultValues(jsonConfig.Processes[index]) {
			t.Fatalf("process config contains default value")
		}
		if containsDefaultValues(ymlConfig.Processes[index]) {
			t.Fatalf("process config contains default value")
		}
		if containsDefaultValues(yamlConfig.Processes[index]) {
			t.Fatalf("process config contains default value")
		}
		if containsDefaultValues(tomlConfig.Processes[index]) {
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
			err := os.Remove(tempDir + file.Name())
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
