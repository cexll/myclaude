@echo off
setlocal enabledelayedexpansion

echo Testing PATH update with long strings...
echo.

rem Create a very long PATH string (over 1024 characters)
set "LONG_PATH="
for /L %%i in (1,1,30) do (
    set "LONG_PATH=!LONG_PATH!C:\VeryLongDirectoryName%%i\SubDirectory\AnotherSubDirectory;"
)

echo Generated PATH length:
echo !LONG_PATH! > temp_path.txt
for %%A in (temp_path.txt) do set "PATH_LENGTH=%%~zA"
del temp_path.txt
echo !PATH_LENGTH! bytes

rem Test 1: Verify reg add can handle long strings
echo.
echo Test 1: Testing reg add with long PATH...
set "TEST_PATH=!LONG_PATH!%%USERPROFILE%%\bin"
reg add "HKCU\Environment" /v TestPath /t REG_EXPAND_SZ /d "!TEST_PATH!" /f >nul 2>nul
if errorlevel 1 (
    echo FAIL: reg add failed with long PATH
    goto :cleanup
) else (
    echo PASS: reg add succeeded with long PATH
)

rem Test 2: Verify the value was stored correctly
echo.
echo Test 2: Verifying stored value length...
for /f "tokens=2*" %%A in ('reg query "HKCU\Environment" /v TestPath 2^>nul ^| findstr /I "TestPath"') do set "STORED_PATH=%%B"
echo !STORED_PATH! > temp_stored.txt
for %%A in (temp_stored.txt) do set "STORED_LENGTH=%%~zA"
del temp_stored.txt
echo Stored PATH length: !STORED_LENGTH! bytes

if !STORED_LENGTH! LSS 1024 (
    echo FAIL: Stored PATH was truncated
    goto :cleanup
) else (
    echo PASS: Stored PATH was not truncated
)

rem Test 3: Verify %%USERPROFILE%%\bin is present
echo.
echo Test 3: Verifying %%USERPROFILE%%\bin is in stored PATH...
echo !STORED_PATH! | findstr /I "USERPROFILE" >nul
if errorlevel 1 (
    echo FAIL: %%USERPROFILE%%\bin not found in stored PATH
    goto :cleanup
) else (
    echo PASS: %%USERPROFILE%%\bin found in stored PATH
)

echo.
echo ========================================
echo All tests PASSED
echo ========================================

:cleanup
echo.
echo Cleaning up test registry key...
reg delete "HKCU\Environment" /v TestPath /f >nul 2>nul
endlocal
