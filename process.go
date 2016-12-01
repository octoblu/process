//+build !plan9

// Package process manages the lifecyle of processes and process groups
package process

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// TODO(nightlyone) remove me, when everything is implemented
var errUnimplemented = errors.New("process: function not implemented")

// Group is a process group we manage.
type Group struct {
	pid                int
	onExitForTerminate <-chan error
	onExitForWait      <-chan error
	onExitMux          <-chan error
}

// Background runs a command which can fork other commands (e.g. a shell script) building a tree of processes.
// This tree of processes are managed in a their own process group.
// NOTE: It overwrites cmd.SysProcAttr
func Background(cmd *exec.Cmd) (*Group, error) {
	if cmd.ProcessState != nil {
		return nil, fmt.Errorf("process: command already executed: %q", cmd.Path)
	}
	if cmd.Process != nil {
		return nil, fmt.Errorf("process: command already executing: %q", cmd.Path)
	}

	startc := make(chan startResult)
	onExitMux := make(chan error, 1)
	onExitForTerminate := make(chan error, 1)
	onExitForWait := make(chan error, 1)

	// Try to start process
	go startProcess(cmd, startc, onExitMux)
	// Start muxing the onExit
	go muxOnExit(onExitMux, onExitForTerminate, onExitForWait)

	res := <-startc

	if res.err != nil {
		return nil, res.err
	}

	// Now running in the background
	return &Group{
		pid:                res.pid,
		onExitForTerminate: onExitForTerminate,
		onExitForWait:      onExitForWait,
	}, nil
}

// ErrNotLeader is returned when we request actions for a process group, but are not their process group leader
var ErrNotLeader = errors.New("process is not process group leader")

// Signal sends POSIX signal sig to this process group
func (g *Group) Signal(sig os.Signal) error {
	if g == nil || g.onExitForTerminate == nil {
		return syscall.ESRCH
	}

	// This just creates a process object from a Pid in Unix
	// instead of actually searching it.
	grp, _ := os.FindProcess(g.pid)
	if grp == nil {
		return fmt.Errorf("could not find process for", g.pid)
	}
	return grp.Signal(sig)
}

// Terminate first tries to gracefully terminate the process, waits patience time, then does final termination and waits for it to exit.
func (g *Group) Terminate(patience time.Duration) error {
	// did we Terminate previously?
	if g.onExitForTerminate == nil {
		return nil
	}

	// did we exit outside of a Terminate call?
	select {
	case <-g.onExitForTerminate:
		g.onExitForTerminate = nil
		return nil
	default:
	}

	// try to be soft
	if err := g.Signal(softSignal()); err != nil {
		return err
	}

	// wait at most patience time for exit
	select {
	case <-g.onExitForTerminate:
		g.onExitForTerminate = nil
		return nil
	case <-time.After(patience):
	}

	// do it the hard way
	if err := g.Signal(syscall.SIGKILL); err != nil {
		return err
	}

	// But we need to wait on the result now
	<-g.onExitForTerminate
	g.onExitForTerminate = nil
	return nil
}

// Wait blocks until the backgrounded process has died, then returns
// the error as if from cmd.Wait()
func (g *Group) Wait() error {
	return <-g.onExitForWait
}

func muxOnExit(onExit chan error, otherOnExits ...chan error) {
	for err := range onExit {
		for _, otherOnExit := range otherOnExits {
			select {
			case otherOnExit <- err:
			default:
			}
		}
	}

	for _, otherOnExit := range otherOnExits {
		close(otherOnExit)
	}
}

type startResult struct {
	pid int
	err error
}

// startProcess tries to a new process. Start result in startc, exit result in waitc.
func startProcess(cmd *exec.Cmd, startc chan<- startResult, waitc chan<- error) {
	var res startResult

	// Startup new process
	if err := cmd.Start(); err != nil {
		res.err = err
		startc <- res
		return
	}
	res.pid = cmd.Process.Pid
	startc <- res
	close(startc)

	// No wait until we finish or get killed
	err := cmd.Wait()
	waitc <- err

	close(waitc)
}
