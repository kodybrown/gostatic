// gostatic
// Copyright 2017-2019 Kody Brown.
//
// A **very** simple static file HTTP server written in Go.
//

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

var (
	port    int
	seconds int
	path    string
)

func init() {
	flag.IntVar(&port, "port", 8080, "the port; defaults to 8080")
	flag.IntVar(&seconds, "seconds", 0, "will exit after X seconds; defaults to 0 (no exit)")
	flag.StringVar(&path, "path", ".", "the path to server pages from; defaults to current directory")
	flag.Parse()
}

func main() {
	defer func() {
		// Error handling
		if r := recover(); r != nil {
			fmt.Printf(" - ERROR: Recovered main(): %v\n", r)
		}
	}()

	// TODO: Read additional config options from the .gostatic.conf file if it exists..

	var err error

	if path == "." {
		path, err = os.Getwd()
		if err != nil {
			fmt.Println(" - Failed to get current directory.")
			return
		}
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println(" - Cannot find (or can't access) html root path.")
		return
	}

	fmt.Printf("gostatic is listening: http://localhost:%d/\n", port)

	if seconds > 0 {
		go func() {
			fmt.Printf(" - will exit in %d seconds.\n", seconds)
			time.Sleep(time.Duration(seconds) * time.Second)
			fmt.Printf(" - exited after %d seconds.\n", seconds)
			os.Exit(0)
		}()
	}

	// https://github.com/golang/go/wiki/HttpStaticFiles

	panic(http.ListenAndServe(fmt.Sprintf(":%d", port), http.FileServer(http.Dir(path))))
}
