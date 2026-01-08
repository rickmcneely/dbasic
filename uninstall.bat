@echo off
setlocal enabledelayedexpansion

echo ==========================================
echo   DBasic Uninstaller for Windows
echo ==========================================
echo.

set "INSTALL_DIR=%USERPROFILE%\AppData\Local\DBasic"

:: Check if installed
if not exist "%INSTALL_DIR%\dbasic.exe" (
    echo DBasic does not appear to be installed.
    echo.
    pause
    exit /b 0
)

echo This will uninstall DBasic from:
echo   %INSTALL_DIR%
echo.
set /p CONFIRM="Continue? (Y/N): "
if /i not "%CONFIRM%"=="Y" (
    echo Cancelled.
    pause
    exit /b 0
)

echo.
echo Removing DBasic...

:: Remove executable
del /q "%INSTALL_DIR%\dbasic.exe" 2>nul

:: Remove directory if empty
rmdir "%INSTALL_DIR%" 2>nul

:: Remove from PATH
echo Removing from PATH...
for /f "tokens=2*" %%a in ('reg query "HKCU\Environment" /v PATH 2^>nul') do set "USER_PATH=%%b"

if defined USER_PATH (
    :: Remove the install dir from PATH
    set "NEW_PATH=!USER_PATH:%INSTALL_DIR%;=!"
    set "NEW_PATH=!NEW_PATH:;%INSTALL_DIR%=!"
    set "NEW_PATH=!NEW_PATH:%INSTALL_DIR%=!"

    if not "!NEW_PATH!"=="!USER_PATH!" (
        setx PATH "!NEW_PATH!" >nul 2>&1
        echo Removed from PATH.
    )
)

echo.
echo ==========================================
echo   DBasic has been uninstalled.
echo ==========================================
echo.
echo NOTE: Restart your command prompt for PATH changes to take effect.
echo.

pause
