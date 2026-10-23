package main

import "syscall"

// Winsock's WSAEADDRINUSE is not exported by Go's syscall package.
// https://learn.microsoft.com/en-us/windows/win32/winsock/windows-sockets-error-codes-2
const addressInUse = syscall.Errno(10048)
