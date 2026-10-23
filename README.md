# gostatic

A **very** simple static file HTTP server written in Go.

## Usage

Serve the current directory on port 8080:

```sh
gostatic --path . --port 8080
```

Both single and double hyphens are supported, for example `-port 8080` and
`--port 8080`. Use `gostatic --help` to list CLI options.

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
with hyphens replaced by underscores. Newly registered options use this same
mapping automatically. Use uppercase variable names for portability.

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
