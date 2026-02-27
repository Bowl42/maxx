@echo off
setlocal EnableExtensions EnableDelayedExpansion

set "SCRIPT_DIR=%~dp0"
for %%I in ("%SCRIPT_DIR%.") do set "CURRENT_DIR=%%~fI"

:findRoot
if exist "!CURRENT_DIR!\go.mod" if exist "!CURRENT_DIR!\web\package.json" (
  set "REPO_ROOT=!CURRENT_DIR!"
  goto rootFound
)

for %%I in ("!CURRENT_DIR!\..") do set "PARENT_DIR=%%~fI"
if /I "!PARENT_DIR!"=="!CURRENT_DIR!" goto rootNotFound
set "CURRENT_DIR=!PARENT_DIR!"
goto findRoot

:rootNotFound
echo [ERROR] Could not locate repo root from "%SCRIPT_DIR%".
echo [ERROR] Expected files: go.mod and web\package.json
exit /b 1

:rootFound
set "WEB_DIR=!REPO_ROOT!\web"

echo [1/3] Build frontend ...
cd /d "!WEB_DIR!" || (
  echo [ERROR] Failed to enter web dir: "!WEB_DIR!"
  exit /b 1
)

call pnpm build
if errorlevel 1 (
  echo [ERROR] pnpm build failed. Abort restart.
  exit /b 1
)

echo [2/3] Stop processes on ports 9880 and 9881 ...
set "KILLED_ANY=0"
set "SEEN_PIDS=;"

call :killByPort 9880
call :killByPort 9881

if "!KILLED_ANY!"=="0" (
  echo [INFO] No running process found on ports 9880/9881.
)

echo [3/3] Start wails dev ...
cd /d "!REPO_ROOT!" || (
  echo [ERROR] Failed to enter repo root: "!REPO_ROOT!"
  exit /b 1
)

call wails dev %*
set "EXIT_CODE=%ERRORLEVEL%"
echo [INFO] wails dev exited with code %EXIT_CODE%.
exit /b %EXIT_CODE%

:killByPort
set "PORT=%~1"
set "FOUND_ON_PORT=0"

for /f "tokens=2,5" %%A in ('netstat -ano -p tcp ^| findstr /R /C:":[0-9][0-9]* " 2^>nul') do (
  set "LOCAL_ADDR=%%A"
  set "PID=%%B"
  echo !LOCAL_ADDR! | findstr /R /C:":%PORT%$" >nul
  if not errorlevel 1 (
    set "FOUND_ON_PORT=1"
    call :tryKillPid !PID! %PORT%
  )
)

if "!FOUND_ON_PORT!"=="0" (
  echo [INFO] Port %PORT% is free.
)
goto :eof

:tryKillPid
set "PID=%~1"
set "PORT=%~2"

if not defined PID (
  echo [WARN] Empty PID for port %PORT%, skip.
  goto :eof
)

echo %PID%| findstr /R "^[0-9][0-9]*$" >nul
if errorlevel 1 (
  echo [WARN] Invalid PID %PID% (port %PORT%), skip.
  goto :eof
)

if %PID% LEQ 4 (
  echo [WARN] Skip protected PID %PID% (port %PORT%).
  goto :eof
)

echo !SEEN_PIDS! | findstr /C:";%PID%;" >nul
if not errorlevel 1 (
  goto :eof
)

set "SEEN_PIDS=!SEEN_PIDS!%PID%;"
echo [INFO] Killing PID %PID% (port %PORT%)...
taskkill /PID %PID% /F >nul 2>&1
if errorlevel 1 (
  echo [WARN] Failed to kill PID %PID% (might have exited already).
) else (
  echo [OK] Killed PID %PID%.
  set "KILLED_ANY=1"
)
goto :eof
