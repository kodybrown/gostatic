@setlocal EnableDelayedExpansion
@echo off

:init
    pushd "%~dp0"

    if not exist "%~dp0.version" echo the `.version` file is missing. & goto :end
    set /P _buildVersion=<"%~dp0.version"
    rem Trim the value read from the file
	for /f "tokens=* delims= " %%a in ("!_buildVersion!") do set "_buildVersion=%%a"
    for /l %%i in (1,1,31) do if "!_buildVersion:~-1!"==" " set "_buildVersion=!_buildVersion:~0,-1!"

    if not exist "%~dp0.name" echo the `.name` file is missing. & goto :end
    set /P _name=<"%~dp0.name"
    rem Trim the value read from the file
	for /f "tokens=* delims= " %%a in ("!_name!") do set "_name=%%a"
    for /l %%i in (1,1,31) do if "!_name:~-1!"==" " set "_name=!_name:~0,-1!"

    call PrintOutput {Header} Building %_name%

    set "_curfile=%~dpnx0"
    set "_specified="

    set "_publish="
    REM set "_destpath=%UserProfile%\Dropbox\Public\Device-Setup\%_name%"
    set "_saltpath=%UserProfile%\Source\Salt\srv\salt\%_name%"

    set "_time=%TIME%"
    if "%_time:~0,1%"==" " set "_time=0%_time:~1%"
    set "_buildDate=%date:~10,4%-%date:~4,2%-%date:~7,2% %_time:~0,2%:%_time:~3,2%:%_time:~6,2%"
    set "_buildString=v!_buildVersion!.%date:~12,4%%date:~4,2%%date:~7,2%.%_time:~0,2%%_time:~3,2%"
    set "_pubBuildString=v!_buildVersion!.%date:~12,4%%date:~4,2%%date:~7,2%"

    rem del /F /Q version.go
    echo.// Do not change anything in this file,>version.go
    echo.// as it is overwritten during each build.>>version.go
    echo.package main>>version.go
    echo.>>version.go
    echo.var ^(>>version.go
    echo.    buildVersion string = "!_buildVersion!">>version.go
    echo.    buildDate string = "%_buildDate%">>version.go
    echo.    buildVersionStr string = "%_buildString%">>version.go
    echo.^)>>version.go

    echo  buildDate: %_buildDate%
    echo  buildString: %_buildString%

    echo.

    rem GO flags:
    set "_rebuildPkgs="

:parse
    if "%~1"==""         goto :main

    if /i "%~1"=="-all"  goto :main
    if /i "%~1"=="--all" goto :main
    if /i "%~1"=="all"   goto :main

    if /i "%~1"=="-publish"  set "_publish=yes" & shift & goto :parse
    if /i "%~1"=="--publish" set "_publish=yes" & shift & goto :parse
    if /i "%~1"=="-p"        set "_publish=yes" & shift & goto :parse
    if /i "%~1"=="-pub"      set "_publish=yes" & shift & goto :parse

    rem GO flags
    if /i "%~1"=="-a"        set "_rebuildPkgs=yes" & shift & goto :parse
    if /i "%~1"=="-rebuild"  set "_rebuildPkgs=yes" & shift & goto :parse

    findstr /R /I /B /C:"^:do_%~1" "%_curfile%" >NUL 2>&1
    if %errorlevel% EQU 0 (
        set "_specified=yes"
        call :do_%~1
    ) else (
        echo Could not find target: %~1
        goto :end
    )

    shift
    goto :parse

:main
    if defined _specified goto :end

    call :do_windows
    call :do_raspi
    call :do_linux
    call :do_darwin

:end
    popd
    endlocal
    exit /B

:build_it
    set "_platform=%~1"
    set "_dir=..\..\..\..\bin\%~1"

    set "tmpfile=%TEMP%\%RANDOM%_build.log"

    set "_ext="
    if /i "%~1"=="windows_amd64" set "_ext=.exe"

    rem GO flags/options:
    set "opts="
    if defined _rebuildPkgs (
        echo  rebuilding all packages..
        set "opts=-a !opts!"
    )

    rem go build -ldflags "-X main.buildVersion=!_buildVersion! -X main.buildDate=%_buildDate%" all
    rem go build -ldflags "-X main.buildDate=%_buildDate%" all

    go build !opts! -o %_name%%_ext% >"%tmpfile%"

    if exist "%tmpfile%" (
        set /P build_result=<"%tmpfile%"
        if defined build_result if not "%build_result%"=="" (
            call PrintOutput.cmd "{Error}" "build failed"
            exit /B
        )
        del /F /Q "%tmpfile%"
    )
    set "tmpfile="

    echo  build succeeded..

    if not exist "%_dir%" mkdir "%_dir%"
    move /Y "%_name%%_ext%" "%_dir%" >NUL

    if defined _publish (
        echo  tagging file..
        copy /B /V /Y "%_dir%\%_name%%_ext%" "%_dir%\%_name%-%_buildString%%_ext%" >NUL

        rem if exist "%_destpath%" (
        rem     echo  publishing to dropbox..
        rem     if not exist "%_destpath%\%_platform%" mkdir "%_destpath%\%_platform%"
        rem     copy /B /V /Y /D "%_dir%\%_name%%_ext%" "%_destpath%\%_platform%\%_name%%_ext%" >NUL
        rem     copy /B /V /Y /D "%_dir%\%_name%%_ext%" "%_destpath%\%_platform%\%_name%-%_pubBuildString%%_ext%" >NUL
        rem )

        if exist "%_saltpath%" (
            echo  publishing to salt server..
            if not exist "%_saltpath%\%_platform%" mkdir "%_saltpath%\%_platform%"
            copy /B /V /Y /D "%_dir%\%_name%%_ext%" "%_saltpath%\%_platform%\%_name%%_ext%" >NUL
            copy /B /V /Y /D "%_dir%\%_name%%_ext%" "%_saltpath%\%_platform%\%_name%-%_pubBuildString%%_ext%" >NUL
        )
    )

    goto :eof

:do_windows
:do_windows_x64
:do_win
    call PrintOutput {Highlight} building windows:
    set "GOOS=windows"
    set "GOARCH=amd64"
    call :build_it "%GOOS%_%GOARCH%"
    goto :eof

:do_linux
:do_linux_x64
    call PrintOutput {Highlight} building linux:
    set "GOOS=linux"
    set "GOARCH=amd64"
    call :build_it "%GOOS%_%GOARCH%"
    goto :eof

:do_raspi
:do_raspi_x64
:do_rpi
    call PrintOutput {Highlight} building raspi:
    set "GOOS=linux"
    set "GOARCH=arm"
    call :build_it "%GOOS%_%GOARCH%"
    goto :eof

:do_darwin
:do_darwin_x64
:do_mac
:do_macos
:do_osx
    call PrintOutput {Highlight} building darwin:
    set "GOOS=darwin"
    set "GOARCH=amd64"
    call :build_it "%GOOS%_%GOARCH%"
    goto :eof
