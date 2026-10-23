# gostatic

A **very** simple static file HTTP server written in Go.

## Configuration

All serving options can be set in a `.gostatic` file in the current working
directory. Command-line arguments override values from the file, including
explicit values of `0` and `.`.

| CLI option | Configuration key | Default | Meaning |
| --- | --- | --- | --- |
| `-path` | `PATH` | `.` | Directory containing the files to serve |
| `-port` | `PORT` | `8080` | HTTP port |
| `-seconds` | `SECONDS` | `0` | Exit after this many seconds; 0 disables the timer |

For example:

```ini
# Serve a site for one minute.
PATH=./website
PORT=9000
SECONDS=60
```

Keys are case-insensitive. Spaces around keys and values are ignored. Blank
lines and lines beginning with `#` or `;` are ignored. Write values without
surrounding quotes; paths may contain spaces and `=`. Relative paths are
resolved from the current working directory.

Unknown keys and invalid values produce diagnostics. Invalid values leave the
previous valid value in place. A missing file uses the CLI defaults.

Override only the port while keeping the configured path and timer:

```sh
gostatic -port 8081
```

Build, verify, and run from the source checkout:

```sh
go test ./...
go vet ./...
go run . -path . -port 8080 -seconds 0
```
