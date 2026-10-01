@echo off
setlocal
cd /d "%~dp0"
if not exist tmp mkdir tmp
if errorlevel 1 exit /b %errorlevel%
go build -o tmp\app.exe ./cmd/app
if errorlevel 1 exit /b %errorlevel%
tmp\app.exe %*
exit /b %errorlevel%
