@echo off
chcp 65001 >nul
cd /d "%~dp0"

echo ========== Work-Watch Build ==========

echo 1. Cleaning old build...
if exist work_watch.exe (
    del work_watch.exe
    echo   Deleted old work_watch.exe
)

echo 2. Compiling...
go build -ldflags="-s -w" -o work_watch.exe .
if %errorlevel% neq 0 (
    echo [FAILED] Build failed with error code: %errorlevel%
    pause
    exit /b %errorlevel%
)

echo 3. Verifying...
if exist work_watch.exe (
    for %%F in (work_watch.exe) do echo   [OK] work_watch.exe (%%~zF bytes^)
) else (
    echo [FAILED] Executable not found
    pause
    exit /b 1
)

echo.
echo [DONE] Build complete
pause
