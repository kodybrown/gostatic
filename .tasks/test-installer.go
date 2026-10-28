// dotask: 1
// description: Exercise a Windows x64 gostatic installer in isolated temporary directories using shared Go acceptance.
// options:
//   - {name: installer, type: path, required: true, description: Self-extracting gostatic installer to test.}
//
// requires: [{kind: task, value: _/go/test-installer.go}]
// end-dotask
package main

import (
	"errors"

	dotask "dotask.local/sdk"
)

func main() {
	dotask.Run(func(c *dotask.BuildContext) error {
		if c.OS() != "windows" || c.Architecture() != "x64" {
			return errors.New("installer acceptance currently targets Windows x64")
		}
		installer, err := c.String("installer")
		if err != nil {
			return err
		}
		return c.ExecTarget("_/go/test-installer.go", map[string]any{"installer": installer})
	})
}
