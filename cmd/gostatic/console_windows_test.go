package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"testing"
)

func prepareConsoleTest(cmd *exec.Cmd) {
	// A private hidden console permits real Ctrl+Break testing even when the
	// parent test runner has no console (editors/CI), without touching its UI.
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x10, HideWindow: true} // CREATE_NEW_CONSOLE
}

func interruptConsoleTest(cmd *exec.Cmd) error {
	sender := exec.Command(os.Args[0], "-test.run=^TestSendConsoleInterrupt$", "--", strconv.Itoa(cmd.Process.Pid))
	output, err := sender.CombinedOutput()
	if err != nil {
		return fmt.Errorf("send console interrupt: %v: %s", err, output)
	}
	return nil
}

func TestSendConsoleInterrupt(t *testing.T) {
	args := helperArguments()
	if len(args) == 0 {
		return
	}
	pid, _ := strconv.Atoi(args[0])
	// The sender is a short-lived subprocess, so attaching/detaching never
	// changes the console belonging to the developer or the test runner.
	processKernel32.NewProc("FreeConsole").Call()
	ok, _, err := processKernel32.NewProc("AttachConsole").Call(uintptr(pid))
	if ok == 0 {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// The API is in its own group; the console's original group receives this.
	processKernel32.NewProc("SetConsoleCtrlHandler").Call(0, 1) // Ignore Ctrl+C in the sender itself.
	ok, _, err = generateConsoleEvent.Call(0, 0)                // CTRL_C_EVENT
	if ok == 0 {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	processKernel32.NewProc("FreeConsole").Call()
	os.Exit(0)
}

func ignoreTestInterrupt() {
	callback := syscall.NewCallback(func(event uint32) uintptr { return 1 })
	processKernel32.NewProc("SetConsoleCtrlHandler").Call(callback, 1)
}

func testProcessAlive(pid int) bool {
	handle, err := syscall.OpenProcess(0x1000, false, uint32(pid)) // QUERY_LIMITED_INFORMATION
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(handle)
	var status uint32
	return syscall.GetExitCodeProcess(handle, &status) == nil && status == 259 // STILL_ACTIVE
}
