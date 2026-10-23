package main

import (
	"flag"
	"testing"
)

func TestLoadOptionsPrecedence(t *testing.T) {
	allConfig := "PATH=./configured site\nPORT=9000\nSECONDS=60\nALLOW_PORT_CHANGE=true\n"
	allEnvironment := map[string]string{
		"GOSTATIC_PATH":              "./environment site",
		"GOSTATIC_PORT":              "9001",
		"GOSTATIC_SECONDS":           "30",
		"GOSTATIC_ALLOW_PORT_CHANGE": "false",
	}
	tests := []struct {
		name        string
		config      string
		environment map[string]string
		args        []string
		port        int
		seconds     int
		path        string
		allowChange bool
	}{
		{name: "defaults", port: 8080, path: "."},
		{
			name: "config overrides defaults", config: allConfig,
			port: 9000, seconds: 60, path: "./configured site", allowChange: true,
		},
		{
			name: "environment overrides config", config: allConfig, environment: allEnvironment,
			port: 9001, seconds: 30, path: "./environment site",
		},
		{
			name: "CLI overrides environment and config", config: allConfig, environment: allEnvironment,
			args: []string{"--port=9002", "--seconds=1", "--path=./CLI site", "--allow-port-change"},
			port: 9002, seconds: 1, path: "./CLI site", allowChange: true,
		},
		{
			name: "explicit default CLI values override other sources", config: allConfig,
			environment: map[string]string{
				"GOSTATIC_PORT": "9001", "GOSTATIC_SECONDS": "30",
				"GOSTATIC_PATH": "./environment site", "GOSTATIC_ALLOW_PORT_CHANGE": "true",
			},
			args: []string{"--port=0", "--seconds=0", "--path=.", "--allow-port-change=false"},
			path: ".",
		},
		{
			name: "empty environment values are ignored", config: allConfig,
			environment: map[string]string{
				"GOSTATIC_PORT": "", "GOSTATIC_SECONDS": "",
				"GOSTATIC_PATH": "", "GOSTATIC_ALLOW_PORT_CHANGE": "",
			},
			port: 9000, seconds: 60, path: "./configured site", allowChange: true,
		},
		{
			name: "invalid environment values preserve config", config: allConfig,
			environment: map[string]string{
				"GOSTATIC_PORT": "invalid", "GOSTATIC_SECONDS": "invalid",
				"GOSTATIC_ALLOW_PORT_CHANGE": "invalid",
			},
			port: 9000, seconds: 60, path: "./configured site", allowChange: true,
		},
		{
			name:   "invalid config boolean preserves earlier value",
			config: "ALLOW-PORT-CHANGE=true\nallow_port_change=invalid\n",
			port:   8080, path: ".", allowChange: true,
		},
		{
			name:        "mixed case environment boolean",
			environment: map[string]string{"GOSTATIC_ALLOW_PORT_CHANGE": " TrUe "},
			port:        8080, path: ".", allowChange: true,
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
			gotAllowChange := flags.Bool("allow-port-change", false, "")
			lookup := func(key string) (string, bool) {
				value, ok := test.environment[key]
				return value, ok
			}
			if err := loadOptions(configPath, flags, lookup, test.args); err != nil {
				t.Fatal(err)
			}
			if *gotPort != test.port || *gotSeconds != test.seconds || *gotPath != test.path || *gotAllowChange != test.allowChange {
				t.Fatalf("got port=%d seconds=%d path=%q allow-port-change=%t; want port=%d seconds=%d path=%q allow-port-change=%t",
					*gotPort, *gotSeconds, *gotPath, *gotAllowChange, test.port, test.seconds, test.path, test.allowChange)
			}
		})
	}
}

func TestEnvironmentUsesRegisteredFlags(t *testing.T) {
	flags := flag.NewFlagSet("gostatic-test", flag.ContinueOnError)
	value := flags.String("new-option", "", "")
	var requestedKey string
	err := readEnvironment(flags, func(key string) (string, bool) {
		requestedKey = key
		return "new value", true
	})
	if err != nil {
		t.Fatal(err)
	}
	if requestedKey != "GOSTATIC_NEW_OPTION" || *value != "new value" {
		t.Fatalf("got key=%q value=%q", requestedKey, *value)
	}
}

func TestBooleanConfigAndEnvironmentSpellings(t *testing.T) {
	for _, test := range []struct {
		value string
		want  bool
	}{
		{"true", true}, {"TrUe", true}, {"T", true}, {"1", true}, {"yes", true}, {"Y", true},
		{"false", false}, {"FaLsE", false}, {"F", false}, {"0", false}, {"no", false}, {"N", false},
	} {
		t.Run(test.value, func(t *testing.T) {
			flags := flag.NewFlagSet("gostatic-test", flag.ContinueOnError)
			value := flags.Bool("allow-port-change", !test.want, "")
			configPath, cleanup := writeTestConfig(t, "ALLOW_PORT_CHANGE="+test.value+"\n")
			defer cleanup()
			if err := readConfig(configPath, flags); err != nil {
				t.Fatal(err)
			}
			if *value != test.want {
				t.Fatalf("config boolean %q = %t; want %t", test.value, *value, test.want)
			}
			*value = !test.want
			if err := readEnvironment(flags, func(string) (string, bool) { return test.value, true }); err != nil {
				t.Fatal(err)
			}
			if *value != test.want {
				t.Fatalf("environment boolean %q = %t; want %t", test.value, *value, test.want)
			}
		})
	}
}
