# Nimbus Launcher

The Nimbus Launcher helps players to quickly add, swap, and run variable server configurations for the game LEGO® Universe, which was discontinued as of January 2012.

This program DOES NOT include a LEGO® Universe client and/or its contents. Players must already have a client located on their system and configure the launcher to point to the client's directory.

Due to the LEGO Group's wishes, LEGO® Universe servers ARE NOT (and should not) be publicly available. The launcher is NOT a server browser. All server configurations managed by the launcher should be sent to players individually.

## Installation

Binaries for the current version of the launcher will be available in the [Releases]() section of this repo. Releases will be labeled with the current launcher version followed by the target platform (i.e. `v1.0.0-win.zip`). The `zip` file will include an `assets` folder and a copy of the compiled executable. The `assets` folder and executable should be siblings withing the same directory.

```
launcher/
|-- assets/
|-- launcher.exe
```

If you have Go installed on your system, the launcher can be installed by using the `go install` command:

```bash
go install github.com/I-Am-Dench/lu-launcher@latest
```

## Building or Running from Source

If you would like to build or run the launcher from the source code, you will need both `go` and `gcc` installed on your system. While this program does not directly use `gcc`, its dependency, [`fyne.io`](https://github.com/fyne-io/fyne), does use it for compiling OpenGL. After these tools have been set up, you can use either the `go run` or `go build` commands to run or compile the launcher.

Running:

```bash
go run .
```

Building:

```bash
go build .
./lu-launcher
```

### Building or Running for MacOSX

If you build or run the launcher from source on MacOSX, you may run into a compiler issues along the lines of:

```bash
error: function does not return NSString
```

If this is the case, you can use the `mac_run_fix.sh` and `mac_build_fix.sh` scripts instead of the `go run` or `go build` commands, respectively.

## Setup

When you first run the launcher, ensure that your client configurations are properly set. Open up the settings window and click on `Launcher`. Make sure your Client Directory is configured your the folder your client exists in (which folder contains the `exe` and the `res/` directory), and your Client Name is configured to the name of your client executable (this will most likely be `legouniverse.exe` and will probably never change).

When you close the settings window, ensure that the `Play` button is enable. This means your client is properly configured.

You can then open the settings window again, and click on the `Servers` tab (this should be selected by default). Add or edit any server configurations that you need and then close settings window.

Once you are happy with your configurations, close the settings window and use the server selector to choose which server you would like to boot into. Server IP info for your currently selected server will be clearly labeled within the launcher. When you are ready, you can press the `Play` button.

## On Play

Two main phases occur when you press the `Play` button:

1. Client Preparation
2. Client Startup

### 1. Client Preparation

- Any cached client resources (original client resources that were replaced through patches) are copied over to their original locations. The copying happens for each resource stored in the `settings/client_cache.sqlite` database, regardless if the replaced resources is original or not.
- If the currently selected server is not the same as the previously run server, the `boot.cfg` file for the currently selected server is copied over into the client directory.
- Any resources that are a part of the current patch for the selected server are copied over into the client, caching the resources already there, ONLY IF the resource does not yet exist in the `settings/client_cache.sqlite` database.

### 2. Client Startup

#### Windows

- The client is simply run as `./legouniverse.exe` where the current directory is the configured `Client Directory`.

#### MacOSX

- This features has not been fully implemented. 
- I have been unable to get the client to run using `wine` or `wine-crossover`. If you are able to get this working, please create a pull request.

#### Linux

- This features has not been fully implemented.
- I *have* been able to get the client running through `wine`, however, I have not been able to get it running through the launcher. If you are able to get this working, please create a pull request.