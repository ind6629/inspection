@echo off
cd /d "%~dp0backend"
set ADDR=0.0.0.0:8080
go run .
pause
