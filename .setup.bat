@echo off

:pre_init
    if not "%~1"=="" (
        set "GOARCH="
        set "GOOS="
        REM Clear these out just in case (Go 1.2+ no longer require them)
        set "GOBIN="
        set "GOROOT="
        set "GOPATH="

        if /i "%~1"=="windows" call PrintOutput {Highlight} Setup env for Windows & set "GOOS=" & set "GOARCH="
        if /i "%~1"=="linux" call PrintOutput {Highlight} Setup env for Linux & set "GOOS=linux" & set "GOARCH=amd64"
        if /i "%~1"=="raspi" call PrintOutput {Highlight} Setup env for Raspberry Pi & set "GOOS=linux" & set "GOARCH=arm"
        if /i "%~1"=="darwin" call PrintOutput {Highlight} Setup env for Mac OSX/Darwin & set "GOOS=darwin" & set "GOARCH=amd64"

        if not "%~2"=="" set "GOARCH=%~2"

        set GO

        exit /B
    )

:init
    call pathx --cleanup
    call pathx --insert C:\Windows\System32
    call pathx --insert C:\Bin

    where /Q git.exe
    if %ERRORLEVEL% NEQ 0 (
        call setup Git --noversion >NUL
    )

:parse

:main
	call PrintOutput {Header} gostatic
	echo.
	echo   WRITTEN  in Go
	echo   RUNS     anywhere
	echo   PROVIDES static file webserver
    echo.

    call pathx --insert "%UserProfile%\Dropbox\Public\Device-Setup\setup\rtl-sdr-release\x64"

    call PrintOutput {Highlight} Setting up Go..
    call setup Go --noversion >NUL

    set "GOARCH="
    set "GOOS="
    REM Clear these out just in case (Go 1.2+ no longer require them)
    set "GOBIN="
    set "GOROOT="
    set "GOPATH="

    call go version
    echo.

    pushd "%~dp0..\..\..\.."
    set "GOPATH=%CD%"
    popd

    if not exist "%GOPATH%\bin" mkdir "%GOPATH%\bin"
    call pathx --insert "%GOPATH%\bin"


    :: Display current Go envars.
    call PrintOutput {Highlight} Go envars..
    set GO
    echo.


    rem Give some useful help..

    rem call PrintOutput {Highlight} Common Go commands..

    rem echo   go build
    rem echo   go clean ^&^& go build
    rem rem echo   gb build all
    echo.

    rem echo Transcompile to other platforms:
    rem echo.
    rem echo # Raspberry Pi
    rem echo   setup --arg raspi
    rem echo     set GOOS=linux
    rem echo     set GOARCH=arm
    rem echo.
    rem echo # Mac OSX
    rem echo   setup --arg darwin
    rem echo     set GOOS=darwin
    rem echo     set GOARCH=amd64
    rem echo.
    rem echo # Linux
    rem echo   setup --arg linux
    rem echo     set GOOS=linux
    rem echo     set GOARCH=amd64
    rem echo.
    rem echo # Windows
    rem echo   setup --arg windows
    rem echo     set GOOS=windows
    rem echo     set GOARCH=amd64
    rem echo       -OR-
    rem echo     set GOOS=
    rem echo     set GOARCH=
    rem echo.
    rem echo More details about building go cross-platform..
    rem echo https://golang.org/doc/install/source#environment
    rem echo.

