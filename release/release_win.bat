cd %~dp0

if exist launcher rmdir launcher /s /q
if not exist launcher md launcher

go build -ldflags -H=windowsgui -o lu-launcher.exe ..
copy lu-launcher.exe launcher

7z a -r ../v0.1.0-win.zip launcher