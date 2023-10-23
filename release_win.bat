if exist releases\build rmdir releases\build /s /q
if not exist releases\build md releases\build

go build -ldflags -H=windowsgui -o lu-launcher.exe

copy lu-launcher.exe releases\build

7z a -r v0.1.0-win.zip releases\build 