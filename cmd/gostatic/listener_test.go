package main

import (
	"io"
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

type testListener struct{}

func (*testListener) Accept() (net.Conn, error) { return nil, io.EOF }
func (*testListener) Close() error              { return nil }
func (*testListener) Addr() net.Addr            { return &net.TCPAddr{Port: 8080} }

func TestListenOnPort(t *testing.T) {
	busy := &net.OpError{Op: "listen", Net: "tcp", Err: os.NewSyscallError("bind", addressInUse)}
	denied := &net.OpError{Op: "listen", Net: "tcp", Err: os.NewSyscallError("bind", syscall.EACCES)}
	tests := []struct {
		name        string
		port        int
		allowChange bool
		failures    []error
		wantPorts   []int
		wantError   string
	}{
		{name: "requested port available", port: 8080, wantPorts: []int{8080}},
		{name: "OS selected port", port: 0, allowChange: true, wantPorts: []int{0}},
		{
			name: "busy with fallback disabled", port: 8080, failures: []error{busy},
			wantPorts: []int{8080}, wantError: "listen on port 8080",
		},
		{
			name: "multiple busy ports", port: 8080, allowChange: true, failures: []error{busy, busy},
			wantPorts: []int{8080, 8081, 8082},
		},
		{
			name: "permission error stops immediately", port: 8080, allowChange: true,
			failures: []error{denied}, wantPorts: []int{8080}, wantError: "listen on port 8080",
		},
		{
			name: "permission error after busy port", port: 8080, allowChange: true,
			failures: []error{busy, denied}, wantPorts: []int{8080, 8081}, wantError: "listen on port 8081",
		},
		{
			name: "unrelated error stops immediately", port: 8080, allowChange: true,
			failures: []error{io.EOF}, wantPorts: []int{8080}, wantError: "listen on port 8080",
		},
		{name: "maximum valid port", port: maxPort, allowChange: true, wantPorts: []int{maxPort}},
		{
			name: "last available port", port: maxPort - 1, allowChange: true, failures: []error{busy},
			wantPorts: []int{maxPort - 1, maxPort},
		},
		{
			name: "exhaustion does not wrap", port: maxPort - 1, allowChange: true,
			failures: []error{busy, busy}, wantPorts: []int{maxPort - 1, maxPort},
			wantError: "no available port between 65534 and 65535",
		},
		{
			name: "maximum port occupied", port: maxPort, allowChange: true,
			failures: []error{busy}, wantPorts: []int{maxPort},
			wantError: "no available port between 65535 and 65535",
		},
		{name: "negative port rejected", port: -1, allowChange: true, wantError: "port must be between"},
		{name: "out of range port rejected", port: maxPort + 1, allowChange: true, wantError: "port must be between"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var attempted []int
			expectedListener := &testListener{}
			listen := func(network, address string) (net.Listener, error) {
				if network != "tcp" {
					t.Fatalf("unexpected network %q", network)
				}
				candidate, err := strconv.Atoi(strings.TrimPrefix(address, ":"))
				if err != nil {
					t.Fatal(err)
				}
				attempted = append(attempted, candidate)
				if len(attempted) <= len(test.failures) {
					return nil, test.failures[len(attempted)-1]
				}
				return expectedListener, nil
			}
			listener, err := listenOnPort(test.port, test.allowChange, listen)
			if test.wantError == "" {
				if err != nil || listener != expectedListener {
					t.Fatalf("got listener=%v error=%v; want successful listener", listener, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), test.wantError) || listener != nil {
				t.Fatalf("got listener=%v error=%v; want error containing %q", listener, err, test.wantError)
			}
			if !reflect.DeepEqual(attempted, test.wantPorts) {
				t.Fatalf("tried %v; want %v", attempted, test.wantPorts)
			}
		})
	}
}

func TestListenOnPortRealBusyListener(t *testing.T) {
	occupied, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()
	port := occupied.Addr().(*net.TCPAddr).Port
	if port == maxPort {
		t.Skip("OS selected the maximum port, leaving no room to test fallback")
	}
	if listener, err := listenOnPort(port, false, net.Listen); err == nil {
		listener.Close()
		t.Fatal("occupied port succeeded with fallback disabled")
	}
	listener, err := listenOnPort(port, true, net.Listen)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if selected := listener.Addr().(*net.TCPAddr).Port; selected <= port || selected > maxPort {
		t.Fatalf("selected port %d after occupied port %d", selected, port)
	}
}

func TestListenOnPortRealOSSelectedPort(t *testing.T) {
	listener, err := listenOnPort(0, false, net.Listen)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if port := listener.Addr().(*net.TCPAddr).Port; port <= 0 || port > maxPort {
		t.Fatalf("invalid OS-selected port %d", port)
	}
}
