@echo off
cd /d "%~dp0admin"
if not exist node_modules call npm.cmd ci
call npm.cmd run dev
pause
