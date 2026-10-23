//go:build !windows
// +build !windows

package main

import (
	"os"
	"os/exec"
	"syscall"
)

func prepareConsoleTest(cmd *exec.Cmd)         {}
func interruptConsoleTest(cmd *exec.Cmd) error { return cmd.Process.Signal(os.Interrupt) }
func ignoreTestInterrupt()                     {}
func testProcessAlive(pid int) bool            { return syscall.Kill(pid, 0) == nil }
