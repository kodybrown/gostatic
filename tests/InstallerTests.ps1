#Requires -Version 7.0
param(
    [Parameter(Mandatory = $true)][string] $Installer
)

$ErrorActionPreference = 'Stop'
$taskInstaller = (Resolve-Path -LiteralPath $Installer).Path
$taskTempRoot = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
$taskSandbox = Join-Path $taskTempRoot ('gostatic-installer-test-' + [Guid]::NewGuid().ToString('N') + ' space')
$taskApplication = Join-Path $taskSandbox 'application'
$taskUserPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$taskMachinePath = [Environment]::GetEnvironmentVariable('Path', 'Machine')
$taskSuccess = $false
New-Item -ItemType Directory -Path $taskSandbox | Out-Null

function Invoke-InstallerTestProcess([string] $taskProgram, [string[]] $taskArguments) {
    $taskInfo = [Diagnostics.ProcessStartInfo]::new()
    $taskInfo.FileName = $taskProgram
    $taskInfo.WorkingDirectory = $taskSandbox
    $taskInfo.UseShellExecute = $false
    $taskInfo.CreateNoWindow = $true
    $taskInfo.RedirectStandardOutput = $true
    $taskInfo.RedirectStandardError = $true
    foreach ($taskArgument in $taskArguments) { $taskInfo.ArgumentList.Add($taskArgument) }
    # Redirect only the test child; preserve the caller's environment and active installs.
    $taskInfo.Environment['APPDATA'] = Join-Path $taskSandbox 'roaming'
    $taskInfo.Environment['LOCALAPPDATA'] = Join-Path $taskSandbox 'local'
    foreach ($taskKey in @($taskInfo.Environment.Keys)) {
        if ($taskKey.StartsWith('GOSTATIC_', [StringComparison]::OrdinalIgnoreCase)) {
            $taskInfo.Environment.Remove($taskKey) | Out-Null
        }
    }
    $taskProcess = [Diagnostics.Process]::Start($taskInfo)
    try {
        $taskOutput = $taskProcess.StandardOutput.ReadToEndAsync()
        $taskError = $taskProcess.StandardError.ReadToEndAsync()
        if (-not $taskProcess.WaitForExit(30000)) { throw "Timed out: $taskProgram" }
        $taskText = $taskOutput.GetAwaiter().GetResult()
        $taskErrorText = $taskError.GetAwaiter().GetResult()
        if ($taskProcess.ExitCode -ne 0) {
            throw "Exit $($taskProcess.ExitCode) from $taskProgram : $taskErrorText $taskText"
        }
        return $taskText
    } finally {
        if (-not $taskProcess.HasExited) { $taskProcess.Kill(); $taskProcess.WaitForExit() }
        $taskProcess.Dispose()
    }
}

function Invoke-TestInstall([string] $taskExtractName, [switch] $Validate) {
    $taskArguments = @(
        '--extract-to', (Join-Path $taskSandbox $taskExtractName),
        '--non-interactive', '--install-dir', $taskApplication,
        '--set', 'add-to-path=false', '--set', 'additional-command=false'
    )
    if ($Validate) { $taskArguments += '--validate' }
    Invoke-InstallerTestProcess $taskInstaller $taskArguments | Out-Null
}

try {
    Invoke-TestInstall 'validation extraction' -Validate
    if (Test-Path -LiteralPath $taskApplication) { throw 'Validation created an installation.' }

    Invoke-TestInstall 'initial extraction'
    $taskCommand = Join-Path $taskApplication 'gostatic.exe'
    $taskVersion = (Invoke-InstallerTestProcess $taskCommand @('--version')).Trim()
    if ($taskVersion -notmatch '^gostatic v[0-9]') { throw "Unexpected installed version: $taskVersion" }
    $taskBuilds = @(Get-ChildItem -LiteralPath (Join-Path $taskApplication 'app') -Directory)
    if ($taskBuilds.Count -ne 1) { throw 'Expected one installed application build.' }
    # Doinstall adds its resolved manifest and receipt to the immutable build.
    $taskPayload = @(Get-ChildItem -LiteralPath $taskBuilds[0].FullName -File |
        Where-Object { $_.Name -notin @('installation.yaml', 'installer.yaml') } |
        Select-Object -ExpandProperty Name)
    if (Compare-Object @('gostatic.exe', 'README.md', 'LICENSE.md') $taskPayload) {
        throw "Unexpected installed payload: $($taskPayload -join ', ')"
    }
    if (Get-ChildItem -LiteralPath $taskApplication -Filter '.gostatic' -Recurse -File) {
        throw 'Installer copied a site-specific .gostatic file.'
    }

    Invoke-TestInstall 'reinstall extraction'
    if ((Invoke-InstallerTestProcess $taskCommand @('--version')).Trim() -ne $taskVersion) {
        throw 'Reinstall changed the application version.'
    }
    $taskUninstaller = Join-Path $taskApplication 'installer\installer.exe'
    Invoke-InstallerTestProcess $taskUninstaller @('uninstall', '--non-interactive', '--leave-settings') | Out-Null
    if (Test-Path -LiteralPath $taskCommand) { throw 'Uninstall left the application command behind.' }
    $taskSuccess = $true
    Write-Output "PASS: validation, install, exact payload, installed version ($taskVersion), reinstall, and uninstall."
} finally {
    if ([Environment]::GetEnvironmentVariable('Path', 'User') -ne $taskUserPath -or
        [Environment]::GetEnvironmentVariable('Path', 'Machine') -ne $taskMachinePath) {
        throw 'Installer acceptance changed persistent PATH.'
    }
    if ($taskSuccess) {
        $taskResolved = (Resolve-Path -LiteralPath $taskSandbox).Path
        $taskTempPrefix = $taskTempRoot.TrimEnd([char[]]"\/") + [IO.Path]::DirectorySeparatorChar
        if (-not $taskResolved.StartsWith($taskTempPrefix, [StringComparison]::OrdinalIgnoreCase) -or
            (Split-Path $taskResolved -Leaf) -notlike 'gostatic-installer-test-* space') {
            throw "Unexpected cleanup target: $taskResolved"
        }
        Remove-Item -LiteralPath $taskResolved -Recurse
        Write-Output 'PASS: user/machine PATH unchanged; temporary acceptance files cleaned up.'
    } else {
        Write-Output "Failed acceptance fixture retained: $taskSandbox"
    }
}
