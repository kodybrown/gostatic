package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

func serveApplication(ctx context.Context, listener net.Listener, root string, options apiOptions, ready func()) (result error) {
	defer listener.Close()
	// https://github.com/golang/go/wiki/HttpStaticFiles
	handler, target, err := apiHandler(http.FileServer(http.Dir(root)), options)
	if err != nil {
		return err
	}
	var backendDone <-chan struct{}
	var backend *apiProcess
	if options.command != "" {
		if target != nil {
			// Refuse an already-running endpoint so a failed launch cannot silently
			// connect the frontend to an unrelated API on the requested fixed port.
			connection, err := net.DialTimeout("tcp", apiAddress(target), 200*time.Millisecond)
			if err == nil {
				connection.Close()
				return fmt.Errorf("API endpoint %s is already in use", target)
			}
		}
		backend, err = startAPI(options)
		if err != nil {
			return err
		}
		defer func() {
			if err := backend.stop(); err != nil {
				if result == nil {
					result = err
				} else {
					result = fmt.Errorf("%v; %v", result, err)
				}
			}
		}()
		backendDone = backend.done
		if target != nil {
			if err := waitForAPI(ctx, backend, target); err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
		}
	}
	if ctx.Err() != nil {
		return nil
	}
	server := &http.Server{Handler: handler}
	defer server.Close()
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	ready()
	select {
	case <-ctx.Done():
		// Local previews can have open streams. Closing the HTTP server ensures
		// these cannot keep the backend alive after Ctrl+C or the timer expires.
		return nil
	case <-backendDone:
		return backend.exited()
	case err := <-served:
		return err
	}
}
