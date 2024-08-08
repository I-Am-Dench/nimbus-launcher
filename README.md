# Nimbus Launcher

The Nimbus Launcher helps players to quickly add, swap, and run variable client configurations for the game LEGO® Universe, which was discontinued as of January 2012. Per-server patch configurations are also available to allow players to automatically update local client configurations. More information can be found [below](#patches).

This program DOES NOT include a LEGO® Universe client and/or its contents. Players must already have a client located on their system and configure the launcher to point to the client's directory.

Due to the LEGO Group's wishes, LEGO® Universe servers ARE NOT (and should not be) publicly available. The Nimbus Launcher is NOT a server browser. All server configurations managed by the launcher should be sent to players privately.

## Installation

Binaries for the current version of the launcher will be available under the [Releases]() tab. Releases will be labeled with the current launcher version followed by the target platform (i.e. `v1.0.0-win.zip`). The `zip` file will include an `assets` folder and a copy of the compiled executable. The structure of the `zip` file's contents should be as follows:

```
launcher/
|-- launcher.exe
```

The executable is NOT SIGNED. Your operating system may prompt you that the application is blocked. If you are not comfortable overriding the block, you will need to [Build or Run](#building-or-running-from-source) the application from source.

Running the executable will generate a settings folder. If you move the executable to a different folder, make sure to bring the settings folder with it. 

Since the [Client Startup](#2-client-startup) functionality has not yet been implemented for Mac and Linux, releases for each platform will not be available until their functions have been tested. Building and/or running the launcher from source, however, will still work as normal just with the missing functionality.

If you have Go installed on your system, you may follow the instructions the [Building or Running from Source](#building-or-running-from-source) section.

## Setup

When you first run the launcher, you may need to ensure that your client configurations are properly set. Clicking the gear icon will reveal the Settings window with two tabs: **Servers** and **Launcher**. The **Servers** tab is where you can Add, Edit, and Remove server configurations. The **Launcher** tab includes some launcher specific settings and patching settings, as well as client settings. If your launcher's play button is disabled, make sure that your Client Directory is configured to your client's folder (the folder that contains the `res/` directory and the `exe` file), and your Client Name is configured to the name of the client exectuable (this will most likely be `legouniverse.exe` and will probably never change).

Once you are happy with your configurations, close the settings window and use the server selector to choose which server you would like to boot into. Server IP info for your currently selected server will be clearly labeled within the launcher. When you are ready, you can press the `Play` button.

## On Play

Two main phases occur when you press the `Play` button:

1. Client Preparation
2. Client Startup

### 1. Client Preparation

1. The client is effectively reset to its original state before any patches were applied.
    - Original client resources (cached in the `settings/client_cache.sqlite` database) which were **replaced** through patches, are copied over to their original locations irrespective of whether the resources are original or not.
    - New client resources (cached in the `settings/client_cache.sqlite` database) which were **added** through patches, are removed.
2. If the currently selected server is not the same as the previously run server, the `boot.cfg` file for the currently selected server is copied over into the client directory.
3. Any resources that are a part of the current patch for the selected server are copied over into the client.
    - Replacement resources will cache the already existing resource ONLY IF the resource does not yet exist in the `settings/client_cache.sqlite` database.
    - Added resources will cache the path of the resources ONLY IF the resource does not yet exist in the `settings/client_cache.sqlite` database.

### 2. Client Startup

#### Windows

- The client is run as `./legouniverse.exe` where the current directory is the configured `Client Directory`.

#### MacOSX (NOT IMPLEMENTED)

- Intel (x86): Due to Apple dropping support for 32-bit programs, the client will NOT run through the native executable nor the windows executable through external programs such as [wine](https://www.winehq.org/). Playing the game on an Intel based Mac will require the use of an emulator or VM.
- M1 (ARM): The client may still be able to be launched through [wine](https://www.winehq.org/). This is currently a **work in progress**.

#### Linux (NOT IMPLEMENTED)

- The client is able to run through [wine](https://www.winehq.org/), but is currently a **work in progress**. 

## Building or Running from Source

If you would like to build or run the launcher from the source code, you will need both `go` and `gcc` installed on your system. While this program does not directly use `gcc`, its dependency, [fyne.io](https://github.com/fyne-io/fyne), uses it for compiling OpenGL. After these tools have been set up, you can use either the `go run` or `go build` commands to run or compile the launcher.

```bash
go run .
```

Or:

```bash
go build .
./nimbus-launcher
```

### Building or Running for MacOSX

If you build or run the launcher from source on MacOSX, you may run into a compiler issue along the lines of:

```bash
error: function does not return NSString
```

If this is the case, you can use the `mac_run_fix.sh` or `mac_build_fix.sh` scripts instead of the `go run` or `go build` commands, respectively.

## Patches

If, as a server owner, you decide to use the patch server capabilities, DO NOT distribute any resources that were used by, or packaged by, the LEGO® Universe client while it was in operation.

Using the patch server functionality allows automatic updating of both launcher configurations and client resources on a server-configuration by server-configuration basis. For example, a local server and a friend's server can be both run with different applied client resources, i.e., having a custom grass texture on the local server and the normal texture on the friend's server. Or for instance, if the AUTHSERVERIP changes for a given server, the launcher can detect a change in patch versions, and then pull and update the `boot.cfg` file for the out of date server.

For non server owners, always approach patches with EXTREME CAUTION. Never accept an update from a server you do not trust. By default, the `Review Patch Before Update` setting is enabled. While on, this settings will display the fetched `patch.json` file in a separate window with options to **Accept** the update, **Cancel** the update, or **Reject** the update.
  - **If accepted**, the patch contents will be downloaded and updated as normal.
  - **If cancelled**, the patch contents will NOT be downloaded nor updated, and the patch will simply be ignored until the next time the updates are refreshed.
  - **If rejected**, the patch version will be blacklisted and will always be ignored on update refreshes or if it appears as a patch dependency.

### Patch Server Configuration

Configuring the launcher to point to a patch server is done through both the `boot.cfg` file and the local server configuration.

When updating or creating a local server configuration within the settings window, select one of the options for the **Patch Protocol** field:

- (None)
- http
- https

The selected option will determine which protocol the launcher will make requests to the patch server with. Selecting (None) will disable all patch server configurations.

> Both http and https follow the TPP Protocol

For the `boot.cfg` file, modify the following fields:

- `PATCHSERVERIP`: Configured server IP
- `PATCHSERVERPORT`: Configured server port
- `PATCHSERVERDIR`: The patch server directory where patch resources are located
  - If the patch server host is `http://127.0.0.1:3000` and `PATCHSERVERDIR` is `patches`, the launcher will make requests to `http://127.0.0.1:3000/patches`

### Patch Server Setup

To set up a patch server, you need an HTTP/HTTPS server which complies with the custom [TPP Protocol](/PATCHING.md).

If you are not integrating the TPP Protocol onto your own server, a simple patch server can be found here: [nimbus-patcher](https://github.com/I-Am-Dench/nimbus-patcher).

### Patch Server Authentication (Optional)

Whenever the launcher makes a patch server request, if the `Patch Token` setting is not empty, it will include a custom header which complies with the TPP Protocol. The patch server should verify that the token is valid before sending any patch contents.

The patch token should be included within the exported `server.xml` file, but it can still be changed by editing the local server configuration through the settings window.
