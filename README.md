# gostatic

A **very** simple static file HTTP server written in Go.

## Usage

Serve the current directory on port 8080:

```sh
gostatic --path . --port 8080
```

Both single and double hyphens are supported, for example `-port 8080` and
`--port 8080`. Use `gostatic --help` to list CLI options.

Print the build version and exit without starting the server:

```sh
gostatic --version
```

`--version` is CLI-only. A `VERSION` config key is rejected as unsupported,
and `GOSTATIC_VERSION` is ignored.

## Options and configuration

Every serving option can be set on the CLI, in a `.gostatic` file in the current
working directory, or through an environment variable.

| CLI option | Configuration key | Environment variable | Default | Meaning |
| --- | --- | --- | --- | --- |
| `--path` | `PATH` | `GOSTATIC_PATH` | `.` | Directory containing the files to serve |
| `--port` | `PORT` | `GOSTATIC_PORT` | `8080` | HTTP port; 0 lets the OS choose an available port |
| `--seconds` | `SECONDS` | `GOSTATIC_SECONDS` | `0` | Exit after this many seconds; 0 disables the timer |
| `--allow-port-change` | `ALLOW_PORT_CHANGE` | `GOSTATIC_ALLOW_PORT_CHANGE` | `false` | Try successive ports when a port is already in use |

Precedence, from lowest to highest:

1. Built-in defaults.
2. `.gostatic` values.
3. Environment variables.
4. Explicit CLI arguments.

Explicit CLI values such as `0`, `.`, and `--allow-port-change=false` override
the other sources.

### Configuration file

For example:

```ini
# Serve a site for one minute, trying the next port if necessary.
PATH=./website
PORT=9000
SECONDS=60
ALLOW_PORT_CHANGE=true
```

Keys are case-insensitive; `ALLOW-PORT-CHANGE` also works. Spaces around keys
and values are ignored. Blank lines and lines beginning with `#` or `;` are
ignored. Write values without surrounding quotes; paths may contain spaces and
`=`. Relative paths are resolved from the current working directory.

A missing file is optional. Unknown keys and invalid values produce diagnostics;
invalid values leave the previous valid value in place.

### Environment variables

Environment names use `GOSTATIC_` followed by the uppercase CLI option name,
with hyphens replaced by underscores. Newly registered serving options use this
same mapping automatically. Use uppercase variable names for portability.

For example, in PowerShell:

```powershell
$env:GOSTATIC_PORT = '9000'
$env:GOSTATIC_ALLOW_PORT_CHANGE = 'true'
gostatic
```

Or for one invocation in a POSIX shell:

```sh
GOSTATIC_PORT=9000 GOSTATIC_ALLOW_PORT_CHANGE=true gostatic
```

Unset or empty environment values are ignored. Invalid values produce a
diagnostic and preserve the value from the config file or built-in default.
Boolean config and environment values accept `true`/`false`, `t`/`f`,
`1`/`0`, `yes`/`no`, and `y`/`n` (case-insensitive).

### Automatic port changes

```sh
gostatic --port 8080 --allow-port-change
```

If port 8080 is already in use, gostatic tries 8081, 8082, and so on through
65535, stopping at the first successful bind. It keeps that listener open and
prints the actual URL. The shutdown timer starts after a listener is acquired.

Port changes are disabled by default. Without the flag, a busy port is an error.
Other bind failures, including permission errors, stop immediately. If every
remaining port through 65535 is occupied, gostatic exits with an error without
wrapping around to a lower port. Valid port values are 0 through 65535;
`--port 0` asks the OS to choose a port directly and prints the chosen port.

To disable port changes enabled in a config file or environment variable:

```sh
gostatic --allow-port-change=false
```

## Development

Build, verify, and run from the source checkout:

```sh
go test ./...
go vet ./...
go run . --path . --port 8080 --allow-port-change --seconds 0
```

### Create a Windows installer

Install Go, dotask with its Rust task toolchain (Cargo/Rust 1.95 or newer), and a
matching native `doinstall` tool bundle. The installed `doinstall-builder` must be on
PATH; it locates its actual runtime and launcher companions.

Create a self-extracting Windows x64 installer:

```powershell
dotask create-gostatic-installer
```

The task runs Go tests and vet, builds with trimmed compiler paths in a temporary
directory, checks the executable for local checkout/profile paths, and verifies
its version before packaging. It uses the existing `buildVersionStr` in `version.go`;
installer creation never increments or rewrites version metadata.

The printed output path is under
`artifacts/installers/windows-x64/gostatic-<version>-<commit>[-dirty]/`.
The directory contains a single `gostatic-setup-v<version>-se.exe`. Existing build
directories are never overwritten. To create another copy of the same build,
choose a different output parent:

```powershell
dotask create-gostatic-installer --output 'C:\tmp\gostatic installers'
```

The installer contains `gostatic.exe`, `README.md`, and `LICENSE.md`. It does not
include the repository's `.gostatic` file: configure each served site separately.
Creation does not launch the installer or change an existing installation.
Launch the resulting executable when ready to choose installation and PATH
options.

Use `dotask help create-gostatic-installer` for explicit `--builder`, `--installer`,
and `--launcher-template` overrides. Supply matching prebuilt native doinstall
tools; this task never builds, downloads, or installs doinstall.

Installer acceptance uses temporary roots, disables PATH changes, checks the
installed payload/version, reinstalls, and uninstalls:

```powershell
pwsh -File .\tests\InstallerTests.ps1 -Installer '<installer path printed above>'
```

This recipe and its installer acceptance are verified on Windows x64.
