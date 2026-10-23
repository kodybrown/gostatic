# Changelog

## Unreleased

### 2026-10-23

- Added optional `--api-command`, `--api-cwd`, `--api-forward-to`, and
  `--api-prefix` settings, including matching `.gostatic` keys and environment
  variables. One invocation can serve static files, launch a backend, and proxy
  API requests with their prefix, method, body, and query preserved.
- Kept `--path` and frontend-only port fallback. Added backend startup checks,
  a bounded listener-readiness wait, and diagnostics for unexpected API exits.
- Replaced direct timed process exit with coordinated shutdown. Ctrl+C and timer
  expiration request API shutdown, then terminate any remaining backend children;
  Windows Job Objects and Unix process groups keep cleanup scoped to the API.
- Documented published .NET API configuration, command quoting, direct API access,
  and forwarding to independently started backends.
- Verified Go tests with the race detector and vet on Windows x64; native Windows
  x86 tests; Linux/macOS amd64 test compilation and vet (not runtime execution).
  Process tests cover console interruption, timer expiry during startup and
  serving, forced termination, backend crashes, busy API ports, and child cleanup.
  A published .NET 10 API smoke test verified configuration loading, static files,
  proxied POST/query handling, graceful timer shutdown, and released API process
  and port. Its build/intermediate/publish paths followed the user output policy.
- Added copyable, commented static-only and static-plus-published-.NET-API
  configuration templates under `examples/`, with README links and setup notes.
  Verified that both templates load through gostatic's actual configuration
  parser without diagnostics.
- Organized all executable Go source and unit tests under `cmd/gostatic/`,
  preserving their contents. Updated installer build/version paths, the VS Code
  launch configuration, and documented build/run commands from the module root.
- Replaced the standalone PowerShell installer acceptance script with native
  `dotask test-installer --installer <path>` Rust checks. Acceptance validates,
  installs, checks the exact payload and version, reinstalls, and uninstalls
  inside temporary roots; verifies persistent user/machine PATH is unchanged;
  retains failed fixtures and safely cleans successful ones.
- Verified relocated Go race tests and vet, source version/timed-preview runs,
  local Windows x64 installer creation and native acceptance, and missing/invalid
  installer rejection with retained failure logs. Linux/macOS amd64 test
  compilation and vet passed; their runtime execution was not performed.
