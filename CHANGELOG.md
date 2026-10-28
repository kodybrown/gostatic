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

### 2026-10-27

- Convert create-gostatic-installer and stage-installer to thin Go tasks backed
  by dotask's reusable go/create-installer.go and go/stage-installer.go catalog
  entries. Configure package/version/smoke-check/payload behavior in settings.go.
  Replace the managed Rust creator with the precompiled native helper adapter;
  keep version metadata and the exact gostatic/README/LICENSE payload unchanged.
- Installer creation now requires installed Go and prebuilt dotask/DoInstall,
  without Rust/Cargo or .NET. The existing separate test-installer acceptance
  task remains Rust and is documented accordingly.
- Verified Windows x64 creation with copied prebuilt dotask/helper/Go SDK and a
  restricted PATH trapping Rust/Cargo/.NET, including inherited hostile Go
  controls, Unicode/spaced output paths and unchanged module/version sources.
  Temporary-root installer validation/install/reinstall/uninstall, exact payload
  and installed version checks passed; persistent user/machine PATH was unchanged.
  Go race tests and vet passed. Managed task hashes match the canonical sources;
  synchronization preview reports zero project changes. Linux/macOS native
  installer acceptance was not run.

### 2026-10-27 Go acceptance and SemVer date patches

- Convert test-installer to a thin Go wrapper backed by the reusable shared
  go/test-installer.go task. All installer creation and acceptance now use Go
  plus precompiled tools, without Rust/Cargo or .NET. Keep exact payload/version,
  temporary install/reinstall/uninstall and persistent PATH checks, with retained
  failure logs, process timeouts and Windows descendant containment.
- Replace the old generated metadata package and .version file with one explicit
  Version constant, v1.5.261023, retaining the former date as a valid SemVer patch.
  Configure the shared go/update-version.go directly with its built-in Utah
  YYMMDD patch policy; no custom gostatic version task or build-time version
  generation exists. Ordinary builds/tests/packaging preserve the source value.
- Verified dry-run date preparation leaves the real version unchanged, the CLI
  reports gostatic v1.5.261023, and a freshly created installer passed temporary
  validation/install/exact payload/version/reinstall/uninstall with persistent
  user/machine PATH unchanged. Race tests/vet and exact managed-source hashes/
  zero-change sync preview passed. The complete workflow also passed with only
  copied prebuilt dotask/helper/Go SDK and installed Go, with Rust/Cargo/.NET
  logging traps never invoked. Windows x64 acceptance only.

### 2026-10-27 Test the recorded installer without arguments

- Remove the redundant local test-installer Go wrapper and sync the shared task
  with optional installer selection. Plain dotask test-installer now verifies
  the last successful installer record; --installer remains an explicit override.
  Keep generated build records separate from portable .dotasks.yaml settings.
- Verified the no-argument shared shortcut selects the recorded v1.5.261023
  installer and passes temporary validation/install/exact payload/version/
  reinstall/uninstall with persistent user/machine PATH unchanged.
  The same no-argument workflow passed with Rust/Cargo/.NET blocked; project
  YAML and the saved record hashes were unchanged. Exact managed sources and
  zero-change synchronization preview also passed.
