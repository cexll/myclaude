@echo off
setlocal enabledelayedexpansion

set "EXIT_CODE=0"
set "REPO=cexll/myclaude"
set "VERSION=latest"
set "OS=windows"

call :detect_arch
if errorlevel 1 goto :fail

set "BINARY_NAME=codex-wrapper-%OS%-%ARCH%.exe"
set "URL=https://github.com/%REPO%/releases/%VERSION%/download/%BINARY_NAME%"
set "TEMP_FILE=%TEMP%\codex-wrapper-%ARCH%-%RANDOM%.exe"
set "DEST_DIR=%USERPROFILE%\bin"
set "DEST=%DEST_DIR%\codex-wrapper.exe"

echo Downloading codex-wrapper for %ARCH% ...
echo   %URL%
call :download
if errorlevel 1 goto :fail

if not exist "%TEMP_FILE%" (
    echo ERROR: download failed to produce "%TEMP_FILE%".
    goto :fail
)

echo Installing to "%DEST%" ...
if not exist "%DEST_DIR%" (
    mkdir "%DEST_DIR%" >nul 2>nul || goto :fail
)

move /y "%TEMP_FILE%" "%DEST%" >nul 2>nul
if errorlevel 1 (
    echo ERROR: unable to place file in "%DEST%".
    goto :fail
)

"%DEST%" --version >nul 2>nul
if errorlevel 1 (
    echo ERROR: installation verification failed.
    goto :fail
)

echo.
echo codex-wrapper installed successfully at:
echo   %DEST%

rem Automatically ensure %USERPROFILE%\bin is in the USER (HKCU) PATH
rem 1) Read current user PATH from registry (REG_SZ or REG_EXPAND_SZ)
set "USER_PATH_RAW="
set "USER_PATH_TYPE="
for /f "tokens=1,2,*" %%A in ('reg query "HKCU\Environment" /v Path 2^>nul ^| findstr /I /R "^ *Path  *REG_"') do (
    set "USER_PATH_TYPE=%%B"
    set "USER_PATH_RAW=%%C"
)
rem Trim leading spaces from USER_PATH_RAW
for /f "tokens=* delims= " %%D in ("!USER_PATH_RAW!") do set "USER_PATH_RAW=%%D"

rem Normalize DEST_DIR by removing a trailing backslash if present
if "!DEST_DIR:~-1!"=="\" set "DEST_DIR=!DEST_DIR:~0,-1!"

rem Build search tokens (expanded and literal)
set "PCT=%%"
set "SEARCH_EXP=;!DEST_DIR!;"
set "SEARCH_EXP2=;!DEST_DIR!\;"
set "SEARCH_LIT=;!PCT!USERPROFILE!PCT!\bin;"
set "SEARCH_LIT2=;!PCT!USERPROFILE!PCT!\bin\;"

rem Prepare user PATH variants for containment tests
set "CHECK_RAW=;!USER_PATH_RAW!;"
set "USER_PATH_EXP=!USER_PATH_RAW!"
if defined USER_PATH_EXP call set "USER_PATH_EXP=%%USER_PATH_EXP%%"
set "CHECK_EXP=;!USER_PATH_EXP!;"

rem Check if already present in user PATH (literal or expanded, with/without trailing backslash)
set "ALREADY_IN_USERPATH=0"
echo !CHECK_RAW! | findstr /I /C:"!SEARCH_LIT!" /C:"!SEARCH_LIT2!" >nul && set "ALREADY_IN_USERPATH=1"
if "!ALREADY_IN_USERPATH!"=="0" (
    echo !CHECK_EXP! | findstr /I /C:"!SEARCH_EXP!" /C:"!SEARCH_EXP2!" >nul && set "ALREADY_IN_USERPATH=1"
)

if "!ALREADY_IN_USERPATH!"=="1" (
    echo User PATH already includes %%USERPROFILE%%\bin.
) else (
    rem Not present: append to user PATH using setx without duplicating system PATH
    if defined USER_PATH_RAW (
        set "USER_PATH_NEW=!USER_PATH_RAW!"
        if not "!USER_PATH_NEW:~-1!"==";" set "USER_PATH_NEW=!USER_PATH_NEW!;"
        set "USER_PATH_NEW=!USER_PATH_NEW!!PCT!USERPROFILE!PCT!\bin"
    ) else (
        set "USER_PATH_NEW=!PCT!USERPROFILE!PCT!\bin"
    )
    rem Persist update to HKCU\Environment\Path (user scope)
    setx PATH "!USER_PATH_NEW!" >nul
    if errorlevel 1 (
        echo WARNING: Failed to append %%USERPROFILE%%\bin to your user PATH.
    ) else (
        echo Added %%USERPROFILE%%\bin to your user PATH.
    )
)

rem Update current session PATH so codex-wrapper is immediately available
set "CURPATH=;%PATH%;"
echo !CURPATH! | findstr /I /C:"!SEARCH_EXP!" /C:"!SEARCH_EXP2!" /C:"!SEARCH_LIT!" /C:"!SEARCH_LIT2!" >nul
if errorlevel 1 set "PATH=!DEST_DIR!;!PATH!"

goto :cleanup

:detect_arch
set "ARCH=%PROCESSOR_ARCHITECTURE%"
if defined PROCESSOR_ARCHITEW6432 set "ARCH=%PROCESSOR_ARCHITEW6432%"

if /I "%ARCH%"=="AMD64" (
    set "ARCH=amd64"
    exit /b 0
) else if /I "%ARCH%"=="ARM64" (
    set "ARCH=arm64"
    exit /b 0
) else (
    echo ERROR: unsupported architecture "%ARCH%". 64-bit Windows on AMD64 or ARM64 is required.
    set "EXIT_CODE=1"
    exit /b 1
)

:download
where curl >nul 2>nul
if %errorlevel%==0 (
    echo Using curl ...
    curl -fL --retry 3 --connect-timeout 10 "%URL%" -o "%TEMP_FILE%"
    if errorlevel 1 (
        echo ERROR: curl download failed.
        set "EXIT_CODE=1"
        exit /b 1
    )
    exit /b 0
)

where powershell >nul 2>nul
if %errorlevel%==0 (
    echo Using PowerShell ...
    powershell -NoLogo -NoProfile -Command " $ErrorActionPreference='Stop'; try { [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor 3072 -bor 768 -bor 192 } catch {} ; $wc = New-Object System.Net.WebClient; $wc.DownloadFile('%URL%','%TEMP_FILE%') "
    if errorlevel 1 (
        echo ERROR: PowerShell download failed.
        set "EXIT_CODE=1"
        exit /b 1
    )
    exit /b 0
)

echo ERROR: neither curl nor PowerShell is available to download the installer.
set "EXIT_CODE=1"
exit /b 1

:fail
echo Installation failed.
set "EXIT_CODE=1"
goto :cleanup

:cleanup
if exist "%TEMP_FILE%" del /f /q "%TEMP_FILE%" >nul 2>nul
set "CODE=%EXIT_CODE%"
endlocal & exit /b %CODE%
