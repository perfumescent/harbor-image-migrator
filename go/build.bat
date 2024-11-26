@echo off
setlocal

:: Set build environment
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0

:: Clean previous build
if exist "build" rd /s /q "build"
mkdir build

:: Build
echo [INFO] Starting build process...
go build -ldflags=" -w -s" -o build/migrator.exe
@REM go build -trimpath -ldflags="-H windowsgui -w -s -buildid= -extldflags=-static" -o build\migrator.exe
@REM 
@REM :: Check build result
@REM if %ERRORLEVEL% NEQ 0 (
@REM     echo [ERROR] Build failed!
@REM     exit /b %ERRORLEVEL%
@REM )
@REM 
@REM :: Compress with UPX if available
@REM where upx >nul 2>nul
@REM if %ERRORLEVEL% EQU 0 (
@REM     echo [INFO] Compressing with UPX...
@REM     upx --best --lzma build\migrator.exe
@REM     if %ERRORLEVEL% NEQ 0 (
@REM         echo [WARN] UPX compression failed, using uncompressed binary
@REM     )
@REM )

echo.
echo [SUCCESS] Build completed successfully!
echo [INFO] Output file: %CD%\build\migrator.exe
echo.

endlocal