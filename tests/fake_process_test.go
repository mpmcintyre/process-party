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
	return exec.Command("go", "run", "./deps/fake_process.go", "sleep", fmt.Sprintf("%d", sleepDurationSeconds))
}

func CreateTouchTask(filename string) *exec.Cmd {
	return exec.Command("go", "run", "./deps/fake_process.go", "touch", filename)
}

func CreateMkdirTask(dirname string) *exec.Cmd {
	return exec.Command("go", "run", "./deps/fake_process.go", "mkdir", dirname)
}

// Tests
func TestTouch(t *testing.T) {

}

func TestMkdir(t *testing.T) {
	dirName := "test_dir"
	dirs, err := os.ReadDir("./")
	if err != nil {

	}
	cmd := CreateMkdirTask(dirName)
	x, err := cmd.Output()

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
	t.Log(x)
}
