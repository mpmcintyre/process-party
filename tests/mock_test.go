package tests

import (
	"os"
	"os/exec"
	"testing"
	"time"

	testHelpers "github.com/mpmcintyre/process-party/test_helpers"
	"github.com/stretchr/testify/assert"
)

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
	t.Parallel()
	filename := "test.file"
	dirFound, err := FileExists(filename, "./")
	if err != nil {
		t.Fatal(err)
	}

	if dirFound {
		t.Logf("Found %s file for testing in the current directory, removing it\n", filename)
		os.RemoveAll(filename)
	}

	cmdSettings := testHelpers.CreateTouchCmdSettings(filename)
	cmd := exec.Command(cmdSettings.Cmd, cmdSettings.Args...)
	_, err = cmd.Output()
	if err != nil {
		t.Fatal(err)
	}

	dirFound, err = FileExists(filename, "./")
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, dirFound, "File not created")
	os.RemoveAll(filename)
}

func TestMkdir(t *testing.T) {
	t.Parallel()
	dirName := "test_dir"
	dirFound, err := DirectoryExists(dirName, "./")
	if err != nil {
		t.Fatal(err)
	}

	if dirFound {
		t.Logf("Found %s dir for testing in the current directory, removing it\n", dirName)
		os.RemoveAll(dirName)
	}

	cmdSettings := testHelpers.CreateMkdirCmdSettings(dirName)
	cmd := exec.Command(cmdSettings.Cmd, cmdSettings.Args...)
	_, err = cmd.Output()
	if err != nil {
		t.Fatal(err)
	}

	dirFound, err = DirectoryExists(dirName, "./")
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, dirFound, "Directory not created")
	os.RemoveAll(dirName)
}

func TestSleep(t *testing.T) {
	t.Parallel()
	sleepTime := 1
	cmdSettings := testHelpers.CreateSleepCmdSettings(sleepTime)
	cmd := exec.Command(cmdSettings.Cmd, cmdSettings.Args...)
	t1 := time.Now()

	_, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}

	assert.GreaterOrEqual(t, time.Since(t1), time.Duration(sleepTime)*time.Second, "Sleep did not sleep correctly")
}
