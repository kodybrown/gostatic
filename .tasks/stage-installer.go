// dotask: 1
// description: Verify and stage the exact native gostatic payload for installer creation.
// options:
//   - {name: stage-dir, type: path, required: true, description: Private external installer staging directory.}
//   - {name: app-version, required: true, description: Expected version from cmd/gostatic/version.go.}
//
// requires: [{kind: task, value: _/go/stage-installer.go}]
// end-dotask
package main

import (
	"errors"

	dotask "dotask.local/sdk"
)

func main() {
	dotask.Run(func(c *dotask.BuildContext) error {
		if c.OS() != "windows" || c.Architecture() != "x64" {
			return errors.New("this installer recipe currently targets Windows x64")
		}
		stage, err := c.String("stage-dir")
		if err != nil {
			return err
		}
		version, err := c.String("app-version")
		if err != nil {
			return err
		}
		return c.ExecTarget("_/go/stage-installer.go", map[string]any{"stage-dir": stage, "app-version": version})
	})
}
