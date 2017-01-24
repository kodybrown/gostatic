@setlocal EnableDelayedExpansion
@echo off

:init
    if not exist "%~dp0.name" echo the `.name` file is missing. & goto :end
    set /P _name=<"%~dp0.name"
    rem Trim the value read from the file
    for /f "tokens=* delims= " %%a in ("!_name!") do set "_name=%%a"
    for /l %%i in (1,1,31) do if "!_name:~-1!"==" " set "_name=!_name:~0,-1!"

:main
    rem set "_ext="
    rem if defined WINDIR set "_ext=.exe"
    call ..\..\..\..\bin\windows_amd64\!_name!.exe %*

:end
    endlocal
    exit /B
