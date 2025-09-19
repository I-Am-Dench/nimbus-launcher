# Nimbus Launcher

> [!WARNING]
> Nimbus Launcher is current in version 0. Please expect major changes to functionality and saved data schemas without regards for backwards compatibility.

The Nimbus Launcher helps players to quickly add, swap, and run variable client configurations for the game LEGO® Universe, which was discontinued as of January 2012. Patcher configurations are also available which enable server owners the ability to send out new content for the game. More information can be found [below](#patches).

This program DOES NOT include a LEGO® Universe client and/or its contents. Players must already have a client located on their system and configure the launcher to point to the client's directory.

Due to the LEGO Group's wishes, LEGO® Universe servers ARE NOT (and should not be) publicly available. The Nimbus Launcher is NOT a server browser. All server configurations managed by the launcher should be sent to players privately.

## Installation

Binaries for the current version of the Nimbus Launcher are available under the [Releases](https://github.com/I-Am-Dench/nimbus-launcher/releases) tab. Releases will be labeled with the current launcher version followed by the target platform (i.e. `v1.0.0-win.zip`). The structure of the zip should look something like this:

```
launcher/
|-- LICENSE
|-- README.md
|-- nimbus-launcher.exe
```

The executable is NOT signed. Your operating system may prompt you, letting you know that the application is blocked. If you are not comfortable overriding the block, you will need to [Build or Run](#building-or-running-from-source) the application from source.

Running the executable will generate a settings folder in your current working directory. Make sure to bring this folder with you if you move the launcher to another location.

Since [Startup](#startup) functionality has not been implemented for Mac, the only available releases are for Windows and Linux. Building and/or running the launcher from source, however, will still work as normal but with the missing functionality.

## Setup

While not required, it is recommended that all of your client files are within a folder called `client`. This fixes an issue where the client will fail to reload the `boot.cfg` file if you logout/return to the login screen. It should look something like this:

```
client/
|-- res/
    |-- ...
|-- boot.cfg
|-- legouniverse.exe
|-- ...
```

The "Client Name" setting, is configured to `client/legouniverse.exe` by default. It is recommended that you do not change this setting. The name is prefixed with `client` to make configuring the "Installation Directory" convenient.

The directory that contains the `client` folder, is call the "Installation Directory". The launcher labels this, "Directory", under your client settings. If you were to run a NetDevil style patch on your client, all resources would be downloaded relative to this path. There is no recommended location for this directory, but you may find the default settings useful:

- Windows: `%LOCALAPPDATA%\LEGO Software\LEGO Universe`
- Linux: `$HOME/games/LEGO Software/LEGO Universe`

These settings are not universal and may be set independently for each server profile you save.

## Startup

### Windows

- The client is run as `.\legouniverse.exe` where the current directory is the parent directory of the executable. For example, if your launcher is configured with:
  - Installation Directory: `.../MyGames/LEGO Universe`
  - Client Name: `client/legouniverse.exe`
- then the current directory would be `.../MyGames/LEGO Universe/client`

### Linux

- The launcher uses [https://github.com/ValveSoftware/Proton](https://github.com/ValveSoftware/Proton) to run the client and will return an error if it cannot find an installed Proton version.
- While it is possible to build/install Proton from source, it is recommended that you install it through Steam as the launcher will search through the Steam directories for the installed versions.
- The Proton version, the steam directory, and the `compatdata` path (effectively the wine prefix) can be configured under the `Launcher` settings tab.
- The client is run as `{selected Proton version}/proton run ./legouniverse.exe` where the current directory is the parent directory of the executable with these environment variables:
  - `WINEDLLOVERRIDES=dinput8.dll=n,b`
  - `PROTON_USE_WINED3D=1`
  - `STEAM_COMPAT_DATA_PATH={configured compatdata path}/.proton`
  - `STEAM_COMPAT_CLIENT_INSTALL_PATH={configured steam path}`
- See [`client/client_linux.go`](./client/client_linux.go) for more details. 

## Building or Running from Source

If you would like to build or run the launcher from the source code, you will need both `go` and `gcc` installed on your system. While this program does not directly use `gcc`, its dependecy, [fyne.io](https://github.com/fyne-io/fyne), uses it for compiling OpenGL. After these tools have been set up, you can use either the `go run` or `go build` commands to run or compile the launcher.

```bash
go run .
```

Or:

```bash
go build .
./nimbus-launcher
```

### Building or Running for MaxOSX

If you build or run the launcher from source on MacOSX, you may run into a compiler issue along the lines of:

```bash
error: function does not return NSString
```

If this is the case, you can use the `mac_run_fix.sh` or `mac_build_fix.sh` scripts in place of the `go run` or `go build` commands.

## Patches

> [!IMPORTANT]
> Patching is currently disabled.

If, as a server owner, you decide to use the patch server capabilities, DO NOT distribute any resources that were used by, or packaged by, the LEGO® Universe client while it was in operation.

Despite being disabled, patcher settings can still be configured and saved for server profiles. Attempting to launch the client with a patcher configured will stop the client from starting and the launcher will display an error.

## TODO

### Features required for a v1.0.0 release

- [ ] Functioning patcher implementation (See: [patching](https://github.com/I-Am-Dench/nimbus-launcher/tree/patching) branch)
- [ ] Automatic updater for launcher (Look for recent GitHub releases)

### Future features

- [ ] Launcher locales
- [ ] Launcher themes!
- [ ] Settings for `lwo_override.xml` configs
- [ ] Setting for [aspect ratio fix](https://github.com/I-Am-Dench/nimbus-launcher/tree/patching)?
- [ ] Client version detection (v1.10.64 vs Darkflame Client vs Alpha Client)