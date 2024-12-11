package tests

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

// Exported functions
func CreateSleepTask(sleepDurationSeconds int) *exec.Cmd {
	return exec.Command("go", "run", "./mocks/fake_process.go", "sleep", fmt.Sprintf("%d", sleepDurationSeconds))
}

func CreateTouchTask(filename string) *exec.Cmd {
	return exec.Command("go", "run", "./mocks/fake_process.go", "touch", filename)
}

func CreateMkdirTask(dirname string) *exec.Cmd {
	return exec.Command("go", "run", "./mocks/fake_process.go", "mkdir", dirname)
}

func DirectoryExists(dirname string, pathToDir string) (bool, error) {
	dirs, err := os.ReadDir(pathToDir)
	if err != nil {
		return false, err
	}

	for _, dir := range dirs {
		if dir.Name() == dirname && dir.Type().IsDir() {
			return true, nil
		}
	}

	return false, nil
}

func FileExists(filename string, path string) (bool, error) {
	dirs, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}

	for _, dir := range dirs {
		if dir.Name() == filename && dir.Type().IsRegular() {
			return true, nil
		}
	}

	return false, nil
}

// Tests
func TestTouch(t *testing.T) {
	filename := "test.file"
	dirFound, err := FileExists(filename, "./")
	if err != nil {
		t.Fatal(err)
	}

	if dirFound {
		t.Logf("Found %s file for testing in the current directory, removing it\n", filename)
		os.RemoveAll(filename)
	}

	cmd := CreateTouchTask(filename)
	x, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(x))
	dirFound, err = FileExists(filename, "./")
	if err != nil {
		t.Fatal(err)
	}

	if !dirFound {
		t.Fatal(errors.New("directory not created"))
	}
	os.RemoveAll(filename)
}

func TestMkdir(t *testing.T) {
	dirName := "test_dir"
	dirFound, err := DirectoryExists(dirName, "./")
	if err != nil {
		t.Fatal(err)
	}

	if dirFound {
		t.Logf("Found %s dir for testing in the current directory, removing it\n", dirName)
		os.RemoveAll(dirName)
	}

	cmd := CreateMkdirTask(dirName)
	x, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(x))
	dirFound, err = DirectoryExists(dirName, "./")
	if err != nil {
		t.Fatal(err)
	}

	if !dirFound {
		t.Fatal(errors.New("directory not created"))
	}
	os.RemoveAll(dirName)
}

func TestSleep(t *testing.T) {
	sleepTime := 1
	cmd := CreateSleepTask(sleepTime)
	t1 := time.Now()
	x, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}

	if time.Since(t1) <= time.Duration(sleepTime)*time.Second {
		t.Fatal(errors.New("Sleep did not sleep correctly"))
	}
	t.Log(string(x))
}
