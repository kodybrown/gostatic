package main

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestConfig(t *testing.T, contents string) (string, func()) {
	t.Helper()
	dir, err := ioutil.TempDir("", "gostatic-config-")
	if err != nil {
		t.Fatal(err)
	}
	cleanup := func() { os.RemoveAll(dir) }
	configPath := filepath.Join(dir, ".gostatic")
	if err := ioutil.WriteFile(configPath, []byte(contents), 0600); err != nil {
		cleanup()
		t.Fatal(err)
	}
	return configPath, cleanup
}

func TestReadConfigAndCLIOverrides(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		args    []string
		port    int
		seconds int
		path    string
	}{
		{name: "defaults", port: 8080, path: "."},
		{
			name:   "all CLI options in config",
			config: "PATH=./website\nPORT=9000\nSECONDS=60\n",
			port:   9000, seconds: 60, path: "./website",
		},
		{
			name:   "case whitespace comments and Windows line endings",
			config: "# comment\r\n ; another comment\r\n\r\n PaTh = ./my site \r\n pOrT = 9001 \r\n SeCoNdS = 2 \r\n",
			port:   9001, seconds: 2, path: "./my site",
		},
		{
			name:   "equals signs in path",
			config: "path=./site=preview=one\n",
			port:   8080, path: "./site=preview=one",
		},
		{
			name:   "CLI overrides every configured option",
			config: "PATH=./website\nPORT=9000\nSECONDS=60\n",
			args:   []string{"-path", ".", "-port", "0", "-seconds", "0"},
			path:   ".",
		},
		{
			name:   "CLI overrides only the requested option",
			config: "PATH=./website\nPORT=9000\nSECONDS=60\n",
			args:   []string{"-port=8081"},
			port:   8081, seconds: 60, path: "./website",
		},
		{
			name:   "invalid values preserve defaults",
			config: "PORT=invalid\nSECONDS=invalid\nPATH=./website\n",
			port:   8080, path: "./website",
		},
		{
			name:   "invalid values preserve earlier valid values",
			config: "PORT=9000\nPORT=invalid\nSECONDS=60\nSECONDS=invalid\n",
			port:   9000, seconds: 60, path: ".",
		},
		{
			name:   "unknown and malformed lines do not block valid options",
			config: "UNKNOWN=value\nmalformed\nPATH=./website\n",
			port:   8080, path: "./website",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configPath, cleanup := writeTestConfig(t, test.config)
			defer cleanup()
			flags := flag.NewFlagSet("gostatic-test", flag.ContinueOnError)
			gotPort := flags.Int("port", 8080, "")
			gotSeconds := flags.Int("seconds", 0, "")
			gotPath := flags.String("path", ".", "")
			if err := readConfig(configPath, flags); err != nil {
				t.Fatal(err)
			}
			if err := flags.Parse(test.args); err != nil {
				t.Fatal(err)
			}
			if *gotPort != test.port || *gotSeconds != test.seconds || *gotPath != test.path {
				t.Fatalf("got port=%d seconds=%d path=%q; want port=%d seconds=%d path=%q",
					*gotPort, *gotSeconds, *gotPath, test.port, test.seconds, test.path)
			}
		})
	}
}

func TestReadConfigUsesRegisteredFlags(t *testing.T) {
	// A newly registered flag needs no matching configuration switch or parser.
	flags := flag.NewFlagSet("gostatic-test", flag.ContinueOnError)
	enabled := flags.Bool("enabled", false, "")
	configPath, cleanup := writeTestConfig(t, "ENABLED=true\n")
	defer cleanup()
	if err := readConfig(configPath, flags); err != nil {
		t.Fatal(err)
	}
	if !*enabled {
		t.Fatal("registered boolean flag was not loaded from configuration")
	}
}

func TestReadConfigMissingFile(t *testing.T) {
	configPath, cleanup := writeTestConfig(t, "")
	defer cleanup()
	flags := flag.NewFlagSet("gostatic-test", flag.ContinueOnError)
	port := flags.Int("port", 8080, "")
	if err := readConfig(configPath+".missing", flags); err != nil {
		t.Fatal(err)
	}
	if *port != 8080 {
		t.Fatalf("missing config changed the default port to %d", *port)
	}
}

func TestReadConfigReportsIOErrors(t *testing.T) {
	configPath, cleanup := writeTestConfig(t, strings.Repeat("x", 70*1024))
	defer cleanup()
	flags := flag.NewFlagSet("gostatic-test", flag.ContinueOnError)
	if err := readConfig(configPath, flags); err == nil {
		t.Fatal("expected an error for an overlong configuration line")
	}
	if err := readConfig(filepath.Dir(configPath), flags); err == nil {
		t.Fatal("expected an error when the configuration path is a directory")
	}
}
