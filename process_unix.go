//go:build !windows
// +build !windows

package main

import (
	"os/exec"
	"syscall"
)

type processTree struct{ pid int }

func newProcessTree(cmd *exec.Cmd) (*processTree, error) {
	// A separate group lets us interrupt and terminate wrapper children too.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return &processTree{}, nil
}

func (p *processTree) attach(cmd *exec.Cmd) error { p.pid = cmd.Process.Pid; return nil }
func (p *processTree) interrupt() error           { return p.signal(syscall.SIGINT) }
func (p *processTree) kill() error                { return p.signal(syscall.SIGKILL) }
func (p *processTree) close()                     {}

func (p *processTree) signal(signal syscall.Signal) error {
	err := syscall.Kill(-p.pid, signal)
	if err == syscall.ESRCH {
		return nil
	}
	return err
}
