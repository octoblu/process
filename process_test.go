package process

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestAlreadyExecutingFails(t *testing.T) {
	t.Parallel()
	what := "/bin/true"
	cmd := exec.Command(what)
	cmd.Process = new(os.Process)
	_, err := Background(cmd)
	wantErr := fmt.Errorf("process: command already executing: %q", what)
	if err == nil || err.Error() != wantErr.Error() {
		t.Fatalf("got %v, expected error %v", err, wantErr)
	}
}

func TestAlreadyExecutedFails(t *testing.T) {
	t.Parallel()
	what := "/bin/true"
	cmd := exec.Command(what)
	cmd.ProcessState = new(os.ProcessState)
	_, err := Background(cmd)
	wantErr := fmt.Errorf("process: command already executed: %q", what)
	if err == nil || err.Error() != wantErr.Error() {
		t.Fatalf("got %v, expected error %v", err, wantErr)
	}
}

func TestStartingNonExistingFailsRightAway(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("/var/run/nonexistant")
	_, err := Background(cmd)
	if err == nil {
		t.Fatalf("got %v, expected error", err)
	}
}

func TestBackgroundingWorks(t *testing.T) {
	t.Parallel()
	what := "true"
	cmd := exec.Command(what)
	g, err := Background(cmd)
	if err != nil {
		t.Fatalf("got unexpected error %v", err)
	}
	err = g.Terminate(time.Second)
	if err != nil && err != syscall.ESRCH && err.Error() != errors.New("os: process already finished").Error() {
		t.Fatalf("cannot terminate: %v", err)
	}
}

func TestSoftKillWorks(t *testing.T) {
	// t.Parallel() // Cannot be in parallel on Windows. Why?
	what := "sleep"
	cmd := exec.Command(what, "1")
	g, err := Background(cmd)
	if err != nil {
		t.Fatalf("got unexpected error %v", err)
	}
	err = g.Terminate(500 * time.Millisecond)
	if err != nil && err != syscall.ESRCH && err.Error() != errors.New("os: process already finished").Error() {
		t.Fatalf("cannot terminate: %v", err)
	}
}

func TestExitBeforeKill(t *testing.T) {
	// t.Parallel() // Cannot be in parallel on Windows. Why?
	what := "false"
	cmd := exec.Command(what)
	g, err := Background(cmd)
	if err != nil {
		t.Fatalf("got unexpected error %v", err)
	}
	time.Sleep(500 * time.Millisecond)
	err = g.Terminate(500 * time.Millisecond)

	if err != nil {
		t.Fatalf("got unexpected error %v", err)
	}
}

func TestWaitOnProgramExitClean(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("true")
	g, err := Background(cmd)
	if err != nil {
		t.Fatalf("got unexpected error %v", err)
	}

	err = g.Wait()
	want := 0
	if got := getExitCode(err); got != want {
		t.Fatalf("expected error code %d, but got %d, because %v", want, got, err)
	}
}

func TestWaitOnProgramExitDirty(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("false")
	g, err := Background(cmd)
	if err != nil {
		t.Fatalf("got unexpected error %v", err)
	}

	err = g.Wait()
	want := 1
	if got := getExitCode(err); got != want {
		t.Fatalf("expected error code %d, but got %d, because %v", want, got, err)
	}
}

func TestWaitOnSoftKill(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("sleep", "10")
	g, err := Background(cmd)
	if err != nil {
		t.Fatalf("got unexpected error %v", err)
	}

	go func() {
		g.Terminate(time.Second * 10)
	}()
	err = g.Wait()

	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	if _, isExitError := err.(*exec.ExitError); !isExitError {
		t.Fatalf("got '%v', expected error to be ExitError", err)
	}
}

func TestDoubleTerminate(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("sleep", "10")
	g, err := Background(cmd)
	if err != nil {
		t.Fatalf("got unexpected error %v", err)
	}

	err = g.Terminate(time.Second * 10)
	if err != nil {
		t.Fatalf("got unexpected error %v", err)
	}

	err = g.Terminate(time.Second * 10)
	if err != nil {
		t.Fatalf("got unexpected error %v", err)
	}
}

func getExitCode(err error) int {
	if err == nil {
		return 0
	}
	e, ok := err.(*exec.ExitError)
	if !ok {
		return -1
	}
	status := e.ProcessState.Sys().(syscall.WaitStatus)
	if !status.Exited() {
		return -2
	}
	return status.ExitStatus()
}
