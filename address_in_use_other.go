//go:build !windows
// +build !windows

package main

import "syscall"

const addressInUse = syscall.EADDRINUSE
