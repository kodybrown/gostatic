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
| `--api-command` | `API_COMMAND` | `GOSTATIC_API_COMMAND` | empty | Backend executable and arguments; empty disables launching |
| `--api-cwd` | `API_CWD` | `GOSTATIC_API_CWD` | current directory | Working directory for the backend process |
| `--api-forward-to` | `API_FORWARD_TO` | `GOSTATIC_API_FORWARD_TO` | empty | Backend HTTP(S) URL; empty disables forwarding |
| `--api-prefix` | `API_PREFIX` | `GOSTATIC_API_PREFIX` | `/api` | Request path prefix to forward, preserved in backend requests |

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

## Static frontend with a local API

Gostatic can launch one backend alongside a static preview and forward API
requests to it. It runs the command in the background, showing its output in
the same console. This is a developer-machine convenience; building and
publishing the backend remain separate steps.

Copyable configuration examples:

- [Static site only](examples/static-only.gostatic).
- [Static site with a published .NET API](examples/static-dotnet-api.gostatic).

Copy the chosen file into your preview directory as `.gostatic`, adjust its
paths and DLL name, then run `gostatic` from that directory. Both examples use
`./website` for the frontend; the API example uses `./published/api` for
published backend files. Relative paths resolve from where gostatic starts.
The example files are templates; they do not include a website or API.

For an already-published ASP.NET Core API, put this `.gostatic` in the directory
where you will run `gostatic`:

```ini
PATH=./website
PORT=1110
ALLOW_PORT_CHANGE=true
SECONDS=0

API_COMMAND=dotnet MyProject.dll --urls http://127.0.0.1:1111
API_CWD=./published/api
API_FORWARD_TO=http://127.0.0.1:1111
API_PREFIX=/api
```

Here `./website` contains the frontend and `./published/api/MyProject.dll`
is the published API. Start both with:

```sh
gostatic
```

The matching CLI invocation is:

```powershell
gostatic --path ./website --port 1110 --allow-port-change --api-command 'dotnet MyProject.dll --urls http://127.0.0.1:1111' --api-cwd ./published/api --api-forward-to http://127.0.0.1:1111 --api-prefix /api
```

The API needs its appropriate .NET runtime installed. Its launch settings must
allow HTTP on the supplied URL; explicit Kestrel endpoint configuration can
override `--urls`. The example uses a published DLL, not `dotnet run` or
`launchSettings.json`.

Requests to `/api` and `/api/...` are forwarded with their method, body,
query, and full path preserved. For example,
`http://localhost:1110/api/customers?active=true` reaches
`http://127.0.0.1:1111/api/customers?active=true`. The backend therefore needs
routes that include `/api`. Other paths, including `/apiculture`, use the
static file server. The frontend can call relative `/api/...` URLs, including
when the site port changes. `API_PREFIX=/` forwards every request instead.

`API_CWD` is relative to the directory where gostatic starts. Command arguments
are relative to that backend working directory. The executable is looked up on
PATH, or can be given as an explicit path. A missing `API_CWD` uses gostatic's
working directory.

`API_COMMAND` is an executable followed by arguments, not a shell script.
Single or double quotes group spaces, and an empty quoted argument is preserved.
Backslashes are literal, including in Windows paths. Use the other quote style
to include a literal quote. For example:

```ini
API_COMMAND="C:\Program Files\dotnet\dotnet.exe" "My Project.dll" --urls http://127.0.0.1:1111
API_CWD=./published/my api
```

Do not surround the entire command with configuration-file quotes. Shell
operators, wildcard expansion, and environment-variable expansion are not
performed. The child inherits gostatic's environment.

When launching with forwarding, gostatic rejects an already-listening backend
endpoint and waits up to ten seconds for the API's TCP listener before announcing
the site. Readiness confirms a listening socket, not application health.
`ALLOW_PORT_CHANGE` affects only the frontend; the API URL is never changed.
A startup failure, readiness timeout, or unexpected backend exit stops gostatic
with an error. An unavailable forwarded endpoint produces HTTP 502.

Ctrl+C, SIGTERM where supported, and `SECONDS` expiration stop the HTTP server
and the API process tree. Gostatic first requests an interrupt where supported,
allows up to three seconds for the backend to exit, then terminates remaining
processes. The timer begins after the frontend listener is acquired and includes
API startup time; process cleanup can extend the total elapsed time.

Launching and forwarding are independent: omit `API_FORWARD_TO` to launch an
API that the frontend calls directly on its own port (configure API CORS as
needed). Omit `API_COMMAND` and `API_CWD` to proxy an API you started yourself;
gostatic does not stop externally started APIs.

## Development

Run these commands from the repository root. The executable's Go source and unit
tests live together under `cmd/gostatic/`; `go.mod` stays at the root.
`version/` contains the existing separate version-metadata package, and
`examples/` contains copyable site configuration files. The installer tasks
live under `.tasks/`.

The working directory still determines which `.gostatic` file is loaded and
how relative site/API paths resolve. Running `go run ./cmd/gostatic` from the
repository root does not change the application's working directory.

Build, verify, and run from the source checkout:

```sh
go test ./...
go vet ./...
go run ./cmd/gostatic --path . --port 8080 --allow-port-change --seconds 0
```

To build a standalone executable on Windows without putting output in the source
tree:

```powershell
go build -trimpath -o "$env:TEMP\gostatic.exe" ./cmd/gostatic
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
its version before packaging. It uses the existing `buildVersionStr` in
`cmd/gostatic/version.go`; installer creation never increments or rewrites
version metadata.

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

The native Rust `test-installer` task exercises a built installer using temporary
roots, disables PATH changes, checks the installed payload/version, reinstalls,
and uninstalls:

```powershell
dotask test-installer --installer '<installer path printed above>'
```

The task preserves failed fixtures for diagnosis and cleans up successful runs.
No PowerShell test runner is required. Creation and acceptance are separate
operations: acceptance does not build an installer or change an active
installation.

This recipe and its installer acceptance are verified on Windows x64.
