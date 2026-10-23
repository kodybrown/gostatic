package main

// gostatic
// Copyright 2017-2019 Kody Brown.
// A **very** simple static file HTTP server written in Go.

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
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
}

func main() {
	defer func() {
		// Error handling
		if r := recover(); r != nil {
			fmt.Printf(" - ERROR: Recovered main(): %v\n", r)
		}
	}()

	var err error

	// Parse the `.gostatic` file for additional configuration options.
	parseConfig()

	// Parse the command line arguments, AFTER the `.gostatic` file has been read.
	// This way, the command line arguments will override any `.gostatic` file values.
	flag.Parse()

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

func parseConfig() {
	// Configuration is read from the current directory before CLI overrides.
	if err := readConfig(".gostatic", flag.CommandLine); err != nil {
		log.Fatal(err)
	}
}

func readConfig(confPath string, flags *flag.FlagSet) error {
	file, err := os.Open(confPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open configuration %s: %v", confPath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Only the first equals sign separates the key, so paths may contain "=".
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Use the CLI registry and its typed parsers so new flags also work here.
		option := flags.Lookup(strings.ToLower(key))
		if option == nil {
			fmt.Println("**** ERROR: Unknown configuration key:", key, "****")
			continue
		}

		previousValue := option.Value.String()
		if err := option.Value.Set(value); err != nil {
			// Integer flag parsers can change their value even when parsing fails.
			// Keep the last valid value instead of accidentally replacing it with 0.
			if restoreErr := option.Value.Set(previousValue); restoreErr != nil {
				return fmt.Errorf("restore configuration value for %s: %v", key, restoreErr)
			}
			fmt.Printf("**** ERROR: Invalid configuration value for %s: %v ****\n", key, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read configuration %s: %v", confPath, err)
	}
	return nil
}
