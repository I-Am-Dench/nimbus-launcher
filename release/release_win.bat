cd %~dp0

if exist launcher rmdir launcher /s /q
if not exist launcher md launcher

go build -tags release -ldflags -H=windowsgui -o .\launcher\lu-launcher.exe ..
copy ..\LICENSE .\launcher

7z a -r ../launcher-win.zip launcher