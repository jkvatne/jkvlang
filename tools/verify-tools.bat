@echo off

:: === Build Configuration ===
:: Paths and toolchain settings for Win64 development
set "SOURCE_DIR=."
set "OBJECT_DIR=build"
set "BIN_DIR=build"
set "TARGET=hello"
set "ASM_FLAGS=-f win64"
set "LINK_FLAGS=/entry:_start /console"
set "TOOL_DIR=."

:: === Directory Preparation ===
:: Create build artifacts directories
if not exist "%OBJECT_DIR%" mkdir "%OBJECT_DIR%"
if not exist "%BIN_DIR%" mkdir "%BIN_DIR%"

:: === Compilation Process ===
:: Assemble using NASM with Win64 format
echo [*] Compiling %TARGET%.asm for x64...
%TOOL_DIR%\nasm %ASM_FLAGS% "%SOURCE_DIR%\%TARGET%.asm" -o "%OBJECT_DIR%\%TARGET%.obj" || (
    echo [!] Assembly failed for %TARGET%.asm
    exit /b 1
)

:: === Linking Process ===
:: Generate executable with GoLink
echo [*] Linking %TARGET%.exe...
%TOOL_DIR%\golink %LINK_FLAGS% "%OBJECT_DIR%\%TARGET%.obj" /fo "%BIN_DIR%\%TARGET%.exe" kernel32.dll || (
    echo [!] Linking failed for %TARGET%.obj
    exit /b 1
)

:: === Post-Build Operations ===
:: Execute and display results
echo -----------------------------------------------------
echo [*] x64 build successful!
for %%F in ("%BIN_DIR%\%TARGET%.exe") do echo Binary size: %%~zF bytes
echo -----------------------------------------------------
echo [*] Running %TARGET%.exe...
"%BIN_DIR%\%TARGET%.exe"
echo.

:: === Cleanup (optional) ===
:: Uncomment to remove intermediate object files
:: del "%OBJECT_DIR%\%TARGET%.obj"
