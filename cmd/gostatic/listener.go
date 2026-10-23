package main

import (
	"fmt"
	"net"
	"os"
)

const maxPort = 65535

func listenOnPort(port int, allowChange bool, listen func(string, string) (net.Listener, error)) (net.Listener, error) {
	if port < 0 || port > maxPort {
		return nil, fmt.Errorf("port must be between 0 and %d (got %d)", maxPort, port)
	}

	for candidate := port; ; candidate++ {
		listener, err := listen("tcp", fmt.Sprintf(":%d", candidate))
		if err == nil {
			return listener, nil
		}
		// Retry only a busy address; permission and other socket failures should
		// remain visible instead of silently changing the requested endpoint.
		if !allowChange || !isAddressInUse(err) {
			return nil, fmt.Errorf("listen on port %d: %v", candidate, err)
		}
		if candidate == maxPort {
			return nil, fmt.Errorf("no available port between %d and %d: %v", port, maxPort, err)
		}
	}
}

func isAddressInUse(err error) bool {
	// net.Listen wraps the native bind error in these two error types.
	// Unwrap explicitly to retain the module's Go 1.12 API compatibility.
	if operation, ok := err.(*net.OpError); ok {
		err = operation.Err
	}
	if call, ok := err.(*os.SyscallError); ok {
		err = call.Err
	}
	return err == addressInUse
}
