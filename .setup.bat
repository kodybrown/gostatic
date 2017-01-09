@echo off

call PrintOutput {Header} gostatic
echo.
echo   WRITTEN  in Go
echo   RUNS     anywhere
echo   PROVIDES static file webserver
echo.

call PrintOutput {Highlight} Setting up Go..
call setup Go --noversion >NUL
echo.

call PrintOutput {Highlight} Build the gostatic web server..
echo   ^>go build
echo   ^>go clean ^& go build
