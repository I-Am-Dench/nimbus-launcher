@echo off

cd %~dp0

if exist bin rmdir bin /s /q
md bin

if exist launcher rmdir launcher /s /q
md launcher

if exist releases rmdir releases /s /q
md releases

copy ..\LICENSE .\launcher
copy ..\README.md .\launcher

for %%s IN (
    windows:amd64
    linux:amd64
) do (
    for /F "tokens=1,2 delims=:" %%a in ("%%s") do (
        setlocal enabledelayedexpansion
        echo ===== Building %%a - %%b =====

        if %%a==windows (
            set bin_output=%%a-%%b.exe
            set dist_output=nimbus-launcher.exe
            set build_flags=-ldflags -H=windowsgui
        ) else (
            set bin_output=%%a-%%b
            set dist_output=nimbus-launcher
        )

        set tags=-tags release
        if "%1"=="standalone" (
            set tags=
        )

        go run ..\cmd\releaser %%a -arch=%%b !tags! !build_flags! -o .\bin\!bin_output! ..
        if %errorlevel% neq 0 (
            echo Failed to build release; GOOS=%%a GOARCH=%%b 1>&2
            exit
        )

        copy .\bin\!bin_output! .\launcher\!dist_output!
        7z a -mx9 -r .\releases\nimbus-launcher_%%a-%%b.zip launcher
        del .\launcher\!dist_output!

        endlocal
    )
)