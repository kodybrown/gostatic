package main

import (
	"flag"
	"testing"
)

func TestVersionIsCLIOnly(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		want bool
	}{
		{name: "config and environment cannot enable version"},
		{name: "double hyphen", args: []string{"--version"}, want: true},
		{name: "single hyphen", args: []string{"-version"}, want: true},
		{name: "explicit true", args: []string{"--version=true"}, want: true},
		{name: "explicit false", args: []string{"--version=false"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			configPath, cleanup := writeTestConfig(t, "VERSION=true\nPORT=9000\n")
			defer cleanup()
			flags := flag.NewFlagSet("gostatic-test", flag.ContinueOnError)
			version := flags.Bool("version", false, "")
			port := flags.Int("port", 8080, "")
			lookup := func(key string) (string, bool) {
				if key == "GOSTATIC_VERSION" {
					t.Fatal("CLI-only version option was looked up in the environment")
				}
				return "", false
			}
			if err := loadOptions(configPath, flags, lookup, test.args); err != nil {
				t.Fatal(err)
			}
			if *version != test.want {
				t.Fatalf("version = %t; want %t", *version, test.want)
			}
			if *port != 9000 {
				t.Fatalf("serving configuration was not applied: port = %d", *port)
			}
		})
	}
}
