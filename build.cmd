@echo off
chcp 65001 >nul
cd /d "%~dp0"

echo ========== Work-Watch 构建 ==========

echo 1. 清理旧构建...
if exist work_watch.exe (
    del work_watch.exe
    echo   已删除旧 work_watch.exe
)

echo 2. 编译...
go build -ldflags="-s -w" -o work_watch.exe .
if %errorlevel% neq 0 (
    echo ❌ 编译失败，错误码: %errorlevel%
    pause
    exit /b %errorlevel%
)

echo 3. 验证...
if exist work_watch.exe (
    for %%F in (work_watch.exe) do echo   ✓ work_watch.exe (%%~zF bytes)
) else (
    echo ❌ 未生成可执行文件
    pause
    exit /b 1
)

echo.
echo ✅ 构建完成
pause
