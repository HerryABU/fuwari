@echo off
chcp 65001 >nul
echo ============================================
echo   Fuwari - Build (Astro frontend + Go embed)
echo ============================================
echo.

cd /d "%~dp0"

REM 使用 corepack 提供的 pnpm（避免依赖全局安装）
set COREPACK_PNPM=node "%~dp0node_modules\corepack\dist\corepack.js" pnpm
if not exist "%~dp0node_modules\corepack" (
    set COREPACK_PNPM=pnpm
)

echo [1/4] Install frontend dependencies...
call %COREPACK_PNPM% install --frozen-lockfile
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] pnpm install failed!
    pause
    exit /b 1
)

echo.
echo [2/4] Build Astro frontend -^> project root dist\
call %COREPACK_PNPM% build
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Frontend build failed!
    pause
    exit /b 1
)

REM Verify frontend output exists
if not exist "dist\index.html" (
    echo [ERROR] dist\index.html not found!
    pause
    exit /b 1
)

echo.
echo [3/4] Sync dist -^> server\dist + Build Go...
if exist "server\dist" rmdir /S /Q "server\dist"
xcopy /E /I /Y "dist" "server\dist" >nul
cd server
go build -ldflags="-s -w" -trimpath -o fuwari-server.exe .
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Go build failed!
    cd ..
    pause
    exit /b 1
)
cd ..

echo.
echo [4/4] Copy exe to project root...
copy /Y "server\fuwari-server.exe" "fuwari-server.exe" >nul

echo.
echo ============================================
echo   Build succeeded!
echo   Output: fuwari-server.exe  (embedded frontend + backend)
echo   Assets: dist\             (frontend output)
echo   Run:    fuwari-server.exe
echo   Visit:  http://localhost:9000  (editor: /editor)
echo ============================================
pause
