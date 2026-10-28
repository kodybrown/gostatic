// dotask: 1
// description: Build and verify gostatic, then create a self-extracting Windows x64 installer.
// options:
//   - {name: output, type: path, description: Alternate installer output parent directory.}
//   - {name: builder, description: Alternate prebuilt doinstall-builder command or executable path.}
//   - {name: installer, type: path, description: Actual matching doinstall runtime executable.}
//   - {name: launcher-template, type: path, description: Actual matching doinstall launcher executable.}
//
// requires: [{kind: task, value: _/go/create-installer.go}]
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
		parameters := map[string]any{}
		for _, name := range []string{"output", "builder", "installer", "launcher-template"} {
			if value, err := c.Parameter(name); err == nil {
				parameters[name] = value
			}
		}
		artifact, err := c.CreateInstaller("_/go/create-installer.go", parameters)
		if err != nil {
			return err
		}
		return c.SetInstallerResult(artifact)
	})
}
