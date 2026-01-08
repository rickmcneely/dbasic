@echo off
setlocal enabledelayedexpansion

echo ==========================================
echo   DBasic Installer for Windows
echo   BASIC-to-Go Transpiler
echo ==========================================
echo.

:: Check for Go installation
echo Checking for Go installation...
where go >nul 2>&1
if %errorlevel% neq 0 (
    echo.
    echo ERROR: Go is not installed or not in PATH.
    echo.
    echo Please install Go from: https://go.dev/dl/
    echo.
    pause
    exit /b 1
)

for /f "tokens=3" %%v in ('go version') do set GO_VERSION=%%v
echo Found %GO_VERSION%

:: Get script directory
set "SCRIPT_DIR=%~dp0"
cd /d "%SCRIPT_DIR%"

:: Build DBasic
echo.
echo Building DBasic...
go build -o dbasic.exe ./cmd/dbasic
if %errorlevel% neq 0 (
    echo.
    echo ERROR: Build failed.
    pause
    exit /b 1
)
echo Build successful.

:: Determine install location
set "INSTALL_DIR=%USERPROFILE%\AppData\Local\DBasic"

:: Create install directory
echo.
echo Installing to %INSTALL_DIR%...
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"

:: Copy executable
copy /y dbasic.exe "%INSTALL_DIR%\dbasic.exe" >nul
if %errorlevel% neq 0 (
    echo ERROR: Could not copy to install directory.
    pause
    exit /b 1
)
echo Installed dbasic.exe

:: Check if already in PATH
echo %PATH% | findstr /i /c:"%INSTALL_DIR%" >nul
if %errorlevel% equ 0 (
    echo DBasic directory already in PATH.
) else (
    echo.
    echo Adding DBasic to user PATH...

    :: Add to user PATH using setx
    for /f "tokens=2*" %%a in ('reg query "HKCU\Environment" /v PATH 2^>nul') do set "USER_PATH=%%b"

    if defined USER_PATH (
        setx PATH "%USER_PATH%;%INSTALL_DIR%" >nul 2>&1
    ) else (
        setx PATH "%INSTALL_DIR%" >nul 2>&1
    )

    if %errorlevel% equ 0 (
        echo Added to PATH successfully.
        echo.
        echo NOTE: Please restart your command prompt for PATH changes to take effect.
    ) else (
        echo.
        echo WARNING: Could not add to PATH automatically.
        echo Please add the following directory to your PATH manually:
        echo   %INSTALL_DIR%
    )
)

:: Verify installation
echo.
echo Verifying installation...
"%INSTALL_DIR%\dbasic.exe" version >nul 2>&1
if %errorlevel% equ 0 (
    echo Installation verified.
) else (
    "%INSTALL_DIR%\dbasic.exe" help >nul 2>&1
    if %errorlevel% equ 0 (
        echo Installation verified.
    ) else (
        echo WARNING: Could not verify installation.
    )
)

echo.
echo ==========================================
echo   DBasic installed successfully!
echo ==========================================
echo.
echo Usage:
echo   dbasic check ^<file.dbas^>   - Check syntax
echo   dbasic emit ^<file.dbas^>    - Generate Go code
echo   dbasic build ^<file.dbas^>   - Build executable
echo   dbasic run ^<file.dbas^>     - Build and run
echo.
echo Example:
echo   dbasic run examples\hello.dbas
echo.

pause
