@echo off
cd /d "%~dp0android"
python -m http.server 8090 --bind 0.0.0.0
pause
