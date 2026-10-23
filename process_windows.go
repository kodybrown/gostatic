package main

import (
	"encoding/binary"
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"
)

var (
	processKernel32      = syscall.NewLazyDLL("kernel32.dll")
	createJobObject      = processKernel32.NewProc("CreateJobObjectW")
	setJobInformation    = processKernel32.NewProc("SetInformationJobObject")
	assignProcessToJob   = processKernel32.NewProc("AssignProcessToJobObject")
	terminateJobObject   = processKernel32.NewProc("TerminateJobObject")
	generateConsoleEvent = processKernel32.NewProc("GenerateConsoleCtrlEvent")
	thread32First        = processKernel32.NewProc("Thread32First")
	thread32Next         = processKernel32.NewProc("Thread32Next")
	openProcessThread    = processKernel32.NewProc("OpenThread")
	resumeProcessThread  = processKernel32.NewProc("ResumeThread")
)

type processTree struct {
	job syscall.Handle
	pid int
}

func newProcessTree(cmd *exec.Cmd) (*processTree, error) {
	handle, _, err := createJobObject.Call(0, 0)
	if handle == 0 {
		return nil, fmt.Errorf("create API job: %v", err)
	}
	p := &processTree{job: syscall.Handle(handle)}
	// JOBOBJECT_EXTENDED_LIMIT_INFORMATION is 144 bytes on 64-bit Windows and
	// 112 on 32-bit Windows. LimitFlags follows two LARGE_INTEGERs at offset 16.
	// Use its native byte layout because Go's 32-bit int64 alignment differs.
	var limits [144]byte
	size := uintptr(len(limits))
	if unsafe.Sizeof(uintptr(0)) == 4 {
		size = 112
	}
	binary.LittleEndian.PutUint32(limits[16:20], 0x2000) // KILL_ON_JOB_CLOSE
	ok, _, err := setJobInformation.Call(handle, 9, uintptr(unsafe.Pointer(&limits[0])), size)
	if ok == 0 {
		p.close()
		return nil, fmt.Errorf("configure API job: %v", err)
	}
	// Keep the console for shared output, but target CTRL_BREAK at the backend
	// group only. No new console window or shell is created.
	// Suspend until job assignment is complete, otherwise a fast launcher could
	// create an unmanaged child (or exit) before attach gets to run.
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x200 | 0x4} // NEW_PROCESS_GROUP | SUSPENDED
	return p, nil
}

func (p *processTree) attach(cmd *exec.Cmd) error {
	p.pid = cmd.Process.Pid
	handle, err := syscall.OpenProcess(0x100|0x1, false, uint32(p.pid)) // SET_QUOTA | TERMINATE
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(handle)
	ok, _, err := assignProcessToJob.Call(uintptr(p.job), uintptr(handle))
	if ok == 0 {
		return err
	}
	return p.resume()
}

func (p *processTree) resume() error {
	// os/exec closes the primary thread handle. The suspended process has not
	// executed user code, so locate its initial thread through Tool Help.
	snapshot, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(snapshot)
	entry := struct {
		size, usage, threadID, ownerPID uint32
		basePriority, deltaPriority     int32
		flags                           uint32
	}{}
	entry.size = uint32(unsafe.Sizeof(entry))
	ok, _, err := thread32First.Call(uintptr(snapshot), uintptr(unsafe.Pointer(&entry)))
	for ok != 0 {
		if entry.ownerPID == uint32(p.pid) {
			thread, _, err := openProcessThread.Call(0x2, 0, uintptr(entry.threadID)) // THREAD_SUSPEND_RESUME
			if thread == 0 {
				return err
			}
			defer syscall.CloseHandle(syscall.Handle(thread))
			count, _, err := resumeProcessThread.Call(thread)
			if uint32(count) == 0xffffffff {
				return err
			}
			return nil
		}
		entry.size = uint32(unsafe.Sizeof(entry))
		ok, _, err = thread32Next.Call(uintptr(snapshot), uintptr(unsafe.Pointer(&entry)))
	}
	return fmt.Errorf("locate suspended API thread: %v", err)
}

func (p *processTree) interrupt() error {
	ok, _, err := generateConsoleEvent.Call(1, uintptr(p.pid)) // CTRL_BREAK_EVENT
	if ok == 0 {
		return err
	}
	return nil
}

func (p *processTree) kill() error {
	// Jobs retain children even if their original launcher has already exited.
	ok, _, err := terminateJobObject.Call(uintptr(p.job), 1)
	if ok == 0 {
		return err
	}
	return nil
}

func (p *processTree) close() { syscall.CloseHandle(p.job) }
