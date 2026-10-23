package main

// gostatic
// Copyright 2017-2019 Kody Brown.
// A **very** simple static file HTTP server written in Go.

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	port            int
	seconds         int
	path            string
	allowPortChange bool
	showVersion     bool
	api             apiOptions
)

func init() {
	registerAPIFlags(flag.CommandLine, &api)
	flag.IntVar(&port, "port", 8080, "the port; 0 lets the OS choose an available port")
	flag.IntVar(&seconds, "seconds", 0, "will exit after X seconds; defaults to 0 (no exit)")
	flag.StringVar(&path, "path", ".", "the path to serve pages from; defaults to current directory")
	flag.BoolVar(&allowPortChange, "allow-port-change", false, "try successive ports through 65535 when the requested port is in use")
	flag.BoolVar(&showVersion, "version", false, "print the build version and exit (CLI only)")
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if err := loadOptions(".gostatic", flag.CommandLine, os.LookupEnv, os.Args[1:]); err != nil {
		return err
	}

	if showVersion {
		fmt.Println("gostatic", buildVersionStr)
		return nil
	}

	if path == "." {
		var err error
		path, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("get current directory: %v", err)
		}
	}

	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("access html root path: %v", err)
	}

	// Bind before announcing success, and keep this listener for the HTTP server.
	// A separate availability probe would race another process for the same port.
	listener, err := listenOnPort(port, allowPortChange, net.Listen)
	if err != nil {
		return err
	}
	defer listener.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	go func(signalContext context.Context) {
		select {
		case <-signals:
			cancel()
		case <-signalContext.Done():
		}
	}(ctx)

	if seconds > 0 {
		var stopTimer context.CancelFunc
		ctx, stopTimer = context.WithTimeout(ctx, time.Duration(seconds)*time.Second)
		defer stopTimer()
		fmt.Printf(" - will exit in %d seconds.\n", seconds)
	}
	actualPort := listener.Addr().(*net.TCPAddr).Port
	err = serveApplication(ctx, listener, path, api, func() {
		if port != 0 && actualPort != port {
			fmt.Printf(" - port %d is in use; using port %d.\n", port, actualPort)
		}
		fmt.Printf("gostatic is listening: http://localhost:%d/\n", actualPort)
	})
	if seconds > 0 && ctx.Err() == context.DeadlineExceeded {
		fmt.Printf(" - exited after %d seconds.\n", seconds)
	}
	return err
}

func loadOptions(confPath string, flags *flag.FlagSet, lookupEnv func(string) (string, bool), args []string) error {
	// Each source overrides the previous one: defaults, current-directory config,
	// environment variables, then explicit command-line arguments.
	if err := readConfig(confPath, flags); err != nil {
		return err
	}
	if err := readEnvironment(flags, lookupEnv); err != nil {
		return err
	}
	return flags.Parse(args)
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

		// The CLI registry supplies every supported option and its typed parser.
		// Accept underscores in config keys to match environment-variable spelling.
		option := flags.Lookup(strings.ReplaceAll(strings.ToLower(key), "_", "-"))
		if option == nil || isCLIOnlyOption(option.Name) {
			fmt.Println("**** ERROR: Unknown configuration key:", key, "****")
			continue
		}
		if err := setOptionValue(option, value, key); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read configuration %s: %v", confPath, err)
	}
	return nil
}

func readEnvironment(flags *flag.FlagSet, lookupEnv func(string) (string, bool)) error {
	var result error
	flags.VisitAll(func(option *flag.Flag) {
		if result != nil || isCLIOnlyOption(option.Name) {
			return
		}
		// Follow PowerCode's application prefix and hyphen-to-underscore naming.
		key := "GOSTATIC_" + strings.ToUpper(strings.ReplaceAll(option.Name, "-", "_"))
		if value, ok := lookupEnv(key); ok && value != "" {
			result = setOptionValue(option, value, key)
		}
	})
	return result
}

func isCLIOnlyOption(name string) bool {
	// Informational actions must not be enabled by persistent configuration.
	return name == "version"
}

func setOptionValue(option *flag.Flag, value, source string) error {
	// PowerCode also accepts yes/no and y/n for environment booleans. Keep config
	// booleans consistent while leaving invalid values to the CLI's typed parser.
	if getter, ok := option.Value.(flag.Getter); ok {
		if _, isBool := getter.Get().(bool); isBool {
			value = strings.ToLower(strings.TrimSpace(value))
			switch value {
			case "yes", "y":
				value = "true"
			case "no", "n":
				value = "false"
			}
		}
	}

	previousValue := option.Value.String()
	if err := option.Value.Set(value); err != nil {
		// Integer flag parsers can change their value even when parsing fails.
		// Preserve the last valid value, including values from lower-priority sources.
		if restoreErr := option.Value.Set(previousValue); restoreErr != nil {
			return fmt.Errorf("restore configuration value for %s: %v", source, restoreErr)
		}
		fmt.Printf("**** ERROR: Invalid configuration value for %s: %v ****\n", source, err)
	}
	return nil
}
