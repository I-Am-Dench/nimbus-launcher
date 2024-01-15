cd %~dp0

if exist launcher rmdir launcher /s /q
if not exist launcher md launcher

go build -tags release -ldflags -H=windowsgui -o lu-launcher.exe ..
move lu-launcher.exe launcher

7z a -r ../launcher-win.zip launcher