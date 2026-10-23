package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func helperArguments() []string {
	for i, arg := range os.Args {
		if arg == "--" {
			return os.Args[i+1:]
		}
	}
	return nil
}

// Real subprocesses exercise OS ownership and the executable's timer/signal
// paths. They also let us test wrappers whose children outlive the wrapper.
func TestAPIHelperProcess(t *testing.T) {
	args := helperArguments()
	if len(args) == 0 {
		return
	}
	address, mode, marker := args[0], args[1], args[2]
	if mode == "exit" {
		os.Exit(17)
	}
	signal.Ignore(os.Interrupt, syscall.SIGTERM)
	ignoreTestInterrupt()
	if mode == "tree" {
		child := exec.Command(os.Args[0], "-test.run=^TestAPIHelperProcess$", "--", args[3], "hold", marker+".child")
		if err := child.Start(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	cwd, _ := os.Getwd()
	if err := ioutil.WriteFile(marker, []byte(cwd), 0600); err != nil {
		os.Exit(1)
	}
	if err := ioutil.WriteFile(marker+".pid", []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
		os.Exit(1)
	}
	if mode == "no-listener" {
		for {
			time.Sleep(time.Hour)
		}
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/exit" {
			os.Exit(19)
		}
		fmt.Fprint(w, "backend "+r.RequestURI)
	})
	http.Serve(listener, handler)
	os.Exit(0)
}

func TestGostaticMainProcess(t *testing.T) {
	args := helperArguments()
	if len(args) == 0 {
		return
	}
	os.Args = append([]string{os.Args[0]}, args...)
	main()
	os.Exit(0)
}

func testDirectory(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := ioutil.TempDir("", "gostatic API space ")
	if err != nil {
		t.Fatal(err)
	}
	return dir, func() { os.RemoveAll(dir) }
}

func availableAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	return listener.Addr().String()
}

func helperCommand(address, mode, marker string, extra ...string) string {
	args := []string{os.Args[0], "-test.run=^TestAPIHelperProcess$", "--", address, mode, marker}
	args = append(args, extra...)
	for i := range args {
		args[i] = `"` + args[i] + `"`
	}
	return strings.Join(args, " ")
}

func waitHTTP(t *testing.T, address string) string {
	t.Helper()
	client := &http.Client{Timeout: 200 * time.Millisecond}
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		response, err := client.Get(address)
		if err == nil {
			body, _ := ioutil.ReadAll(response.Body)
			response.Body.Close()
			return string(body)
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("HTTP endpoint did not start: %s", address)
	return ""
}

func assertPortClosed(t *testing.T, address string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		connection, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
		if err != nil {
			return
		}
		connection.Close()
		time.Sleep(25 * time.Millisecond)
	}
	t.Errorf("API is still listening after gostatic stopped: %s", address)
}

func assertAPIProcessExited(t *testing.T, marker string) {
	t.Helper()
	contents, err := ioutil.ReadFile(marker + ".pid")
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(string(contents))
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !testProcessAlive(pid) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Errorf("API process %d survived shutdown", pid)
}

func TestServeApplicationCancelsAndStopsProcessTree(t *testing.T) {
	dir, cleanup := testDirectory(t)
	defer cleanup()
	if err := ioutil.WriteFile(filepath.Join(dir, "index.html"), []byte("frontend"), 0600); err != nil {
		t.Fatal(err)
	}
	address, childAddress := availableAddress(t), availableAddress(t)
	marker := filepath.Join(dir, "api cwd.txt")
	options := apiOptions{helperCommand(address, "tree", marker, childAddress), dir, "http://" + address, "/api"}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- serveApplication(ctx, listener, dir, options, func() {}) }()
	site := "http://" + listener.Addr().String()
	if body := waitHTTP(t, site+"/"); body != "frontend" {
		t.Fatalf("static body = %q", body)
	}
	if body := waitHTTP(t, site+"/api/customers?q=1"); body != "backend /api/customers?q=1" {
		t.Fatalf("API body = %q", body)
	}
	waitHTTP(t, "http://"+childAddress+"/")
	actualCwd, err := ioutil.ReadFile(marker)
	if err != nil || !strings.EqualFold(filepath.Clean(string(actualCwd)), filepath.Clean(dir)) {
		t.Fatalf("API cwd = %q, %v; want %q", actualCwd, err, dir)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(8 * time.Second):
		t.Fatal("shutdown did not finish")
	}
	assertPortClosed(t, address)
	assertPortClosed(t, childAddress)
	assertAPIProcessExited(t, marker)
	assertAPIProcessExited(t, marker+".child")
}

func TestAPICrashStopsSiteAndDescendants(t *testing.T) {
	dir, cleanup := testDirectory(t)
	defer cleanup()
	address, childAddress := availableAddress(t), availableAddress(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	options := apiOptions{helperCommand(address, "tree", filepath.Join(dir, "marker"), childAddress), dir, "http://" + address, "/api"}
	done := make(chan error, 1)
	go func() { done <- serveApplication(ctx, listener, dir, options, func() {}) }()
	waitHTTP(t, "http://"+listener.Addr().String()+"/api")
	waitHTTP(t, "http://"+childAddress+"/")
	client := &http.Client{Timeout: time.Second}
	response, _ := client.Get("http://" + address + "/exit")
	if response != nil {
		response.Body.Close()
	}
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "API exited unexpectedly") {
			t.Fatalf("expected API crash error, got %v", err)
		}
	case <-time.After(8 * time.Second):
		t.Fatal("backend crash did not stop gostatic")
	}
	assertPortClosed(t, listener.Addr().String())
	assertPortClosed(t, childAddress)
}

func TestAPIStartupFailures(t *testing.T) {
	dir, cleanup := testDirectory(t)
	defer cleanup()
	for _, options := range []apiOptions{
		{command: filepath.Join(dir, "missing.exe"), prefix: "/api"},
		{command: helperCommand(availableAddress(t), "exit", filepath.Join(dir, "marker")), forwardTo: "http://" + availableAddress(t), prefix: "/api"},
		{command: helperCommand(availableAddress(t), "hold", filepath.Join(dir, "marker")), cwd: filepath.Join(dir, "missing"), prefix: "/api"},
	} {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = serveApplication(ctx, listener, dir, options, func() { t.Error("announced ready after startup failure") })
		cancel()
		if err == nil {
			t.Fatalf("accepted failed backend %+v", options)
		}
		assertPortClosed(t, listener.Addr().String())
	}
}

func TestBusyAPIPortIsNotChanged(t *testing.T) {
	dir, cleanup := testDirectory(t)
	defer cleanup()
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()
	site, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dir, "should not exist")
	options := apiOptions{helperCommand(occupied.Addr().String(), "hold", marker), dir, "http://" + occupied.Addr().String(), "/api"}
	err = serveApplication(context.Background(), site, dir, options, func() { t.Error("announced ready with occupied API port") })
	if err == nil || !strings.Contains(err.Error(), "already in use") {
		t.Fatalf("expected busy API error, got %v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("started API despite occupied port")
	}
}

func TestGostaticSecondsAndInterruptCleanup(t *testing.T) {
	for _, mode := range []string{"seconds", "interrupt", "seconds-during-startup"} {
		t.Run(mode, func(t *testing.T) {
			dir, cleanup := testDirectory(t)
			defer cleanup()
			address, childAddress, site := availableAddress(t), availableAddress(t), availableAddress(t)
			_, port, _ := net.SplitHostPort(site)
			marker := filepath.Join(dir, "marker")
			apiMode := "tree"
			seconds := 0
			if mode != "interrupt" {
				seconds = 2
			}
			if mode == "seconds-during-startup" {
				apiMode = "no-listener"
			}
			args := []string{"-test.run=^TestGostaticMainProcess$", "--", "--path", dir, "--port", port,
				"--seconds", strconv.Itoa(seconds), "--api-command", helperCommand(address, apiMode, marker, childAddress),
				"--api-cwd", dir, "--api-forward-to", "http://" + address, "--api-prefix", "/api"}
			cmd := exec.Command(os.Args[0], args...)
			cmd.Dir = dir
			// Keep the subprocess independent of developer-specific .gostatic/env.
			for _, entry := range os.Environ() {
				if !strings.HasPrefix(strings.ToUpper(entry), "GOSTATIC_") {
					cmd.Env = append(cmd.Env, entry)
				}
			}
			prepareConsoleTest(cmd)
			var output bytes.Buffer
			cmd.Stdout, cmd.Stderr = &output, &output
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			defer cmd.Process.Kill()
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			if mode != "seconds-during-startup" {
				waitHTTP(t, "http://"+site+"/api")
				waitHTTP(t, "http://"+childAddress+"/")
			}
			if mode == "interrupt" {
				if err := interruptConsoleTest(cmd); err != nil {
					t.Fatal(err)
				}
			}
			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("gostatic failed: %v\n%s", err, output.String())
				}
			case <-time.After(10 * time.Second):
				t.Fatal("gostatic did not stop")
			}
			if _, err := os.Stat(marker); err != nil {
				t.Fatalf("API never started: %v\n%s", err, output.String())
			}
			assertPortClosed(t, address)
			assertPortClosed(t, childAddress)
			assertPortClosed(t, site)
			assertAPIProcessExited(t, marker)
			if mode != "seconds-during-startup" {
				assertAPIProcessExited(t, marker+".child")
			}
		})
	}
}
