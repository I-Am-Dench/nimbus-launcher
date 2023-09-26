# Nimbus Launcher

The Nimbus Launcher helps players to quickly add, swap, and run variable server configurations for the game LEGO® Universe, which was discontinued as of January 2012. Per-server patch configurations are also available as a strictly experimental feature. More information is can be found [below](#patches).

This program DOES NOT include a LEGO® Universe client and/or its contents. Players must already have a client located on their system and configure the launcher to point to the client's directory.

Due to the LEGO Group's wishes, LEGO® Universe servers ARE NOT (and should not) be publicly available. The launcher is NOT a server browser. All server configurations managed by the launcher should be sent to players individually.

## Installation

Binaries for the current version of the launcher will be available in the [Releases]() section of this repo. Releases will be labeled with the current launcher version followed by the target platform (i.e. `v1.0.0-win.zip`). The `zip` file will include an `assets` folder and a copy of the compiled executable. The structure of the `zip` file's contents should be as follows:

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

If you would like to build or run the launcher from the source code, you will need both `go` and `gcc` installed on your system. While this program does not directly use `gcc`, its dependency, [fyne.io](https://github.com/fyne-io/fyne), uses it for compiling OpenGL. After these tools have been set up, you can use either the `go run` or `go build` commands to run or compile the launcher.

```bash
go run .
```

Or:

```bash
go build .
./lu-launcher
```

### Building or Running for MacOSX

If you build or run the launcher from source on MacOSX, you may run into a compiler issue along the lines of:

```bash
error: function does not return NSString
```

If this is the case, you can use the `mac_run_fix.sh` or `mac_build_fix.sh` scripts instead of the `go run` or `go build` commands, respectively.

## Setup

When you first run the launcher, ensure that your client configurations are properly set. Open up the settings window and click on `Launcher`. Make sure your Client Directory is configured to your client's folder (the folder that contains the `res/` directory and the `exe` file), and your Client Name is configured to the name of the client executable (this will most likely be `legouniverse.exe` and will probably never change).

When you close the settings window, ensure that the `Play` button is enabled. If it is, the launcher is properly configured.

You can then open the settings window again, and click on the `Servers` tab (this should be selected by default). Add or edit any server configurations that you need and then close settings window.

Once you are happy with your configurations, close the settings window and use the server selector to choose which server you would like to boot into. Server IP info for your currently selected server will be clearly labeled within the launcher. When you are ready, you can press the `Play` button.

## On Play

Two main phases occur when you press the `Play` button:

1. Client Preparation
2. Client Startup

### 1. Client Preparation

- Any cached client resources (original client resources that were replaced through patches) are copied over to their original locations. The copying happens for each resource stored in the `settings/client_cache.sqlite` database, regardless if the replaced resources are original or not.
- If the currently selected server is not the same as the previously run server, the `boot.cfg` file for the currently selected server is copied over into the client directory.
- Any resources that are a part of the current patch for the selected server are copied over into the client, caching the resources already there, ONLY IF the resource does not yet exist in the `settings/client_cache.sqlite` database.

### 2. Client Startup

#### Windows

- The client is run as `./legouniverse.exe` where the current directory is the configured `Client Directory`.

#### MacOSX

- This features has not been fully implemented. 
- I have been unable to get the client to run using `wine` or `wine-crossover`. If you are able to get this working, please create a pull request.

#### Linux

- This features has not been fully implemented.
- I *have* been able to get the client running through `wine`, however, I have not been able to get it running through the launcher. If you are able to get this working, please create a pull request.

## Patches

If, as a server owner, you decide to use the patch server capabilities, DO NOT distribute any resources that were used by, or packaged by, the LEGO® Universe client while it was in operation.

Using the patch server functionality allows automatic updating of both launcher configurations and client resources on a server-configuration by server-configuration basis. For example, a local server and a friend's server can be both run with different applied client resources, i.e., having a custom grass texture on the local server and the normal texture on the friend's server. Or for instance, if the AUTHSERVERIP changes for a given server, the launcher can detect a change in patch versions, and then pull and update the `boot.cfg` file for the out of date server.

For non server owners, always approach patches with EXTREME CAUTION. Never accept an update from a server you do not trust. By default, the `Review Patch Before Update` setting is enabled. While on, this settings will display the fetched `patch.json` file in a separate window with options to **Accept** the update, **Cancel** the update, or **Reject** the update. **If accepted**, the patch contents will be downloaded and updated as normal. **If cancelled**, the patch contents will NOT be downloaded nor updated, and the patch will simply be ignored until the next time the updates are refreshed. **If rejected**, the patch version will be blacklisted and will always be ignored on update refreshes or if it appears as a patch dependency.

### Patch Server Configuration

Configuring the launcher to point to a patch server is done through the `boot.cfg` file. Update the following fields:

- `PATCHSERVERIP`: Configured server IP
- `PATCHSERVERPORT`: Configured server port
- `PATCHSERVERDIR`: The patch server directory where patch resources are located
  - If the patch server host is `http://127.0.0.1:1000` and `PATCHSERVERDIR` is `patches`, the launcher will make requests to `http://127.0.0.1:1000/patches`

### Patch Server Setup

To set up a patch server, you need an HTTP/HTTPS server. Whenever the launcher searches for updates, it makes a request to the patch server directory expecting a `json` file (referred to as `patches.json`). This file contains the server's current patch version and a list of available versions. Currently, the list of versions is unused.

Example: `http://127.0.0.1:1000/patches/`

```json
{
    "currentVersion": "1.0.0",
    "versions": [
        "1.0.0"
    ]
}
```

Once the launcher has determined that the configured server has an available patch, the launcher makes a request to the version expecting a `json` file (referred to as `patch.json`).

Example: `http://127.0.0.1:1000/patches/1.0.0/`

```json
{
    "downloads": [
        {
            "path": "/1.0.0/logo.dds",
            "name": "logo.dds"
        }
    ],
    "transfer": {
        "logo.dds": "res/ui/ingame/passport_i90.dds"
    }
}
```

The above example will download `http://127.0.0.1:1000/patches/1.0.0/logo.dds` and save it to a file called `logo.dds`. During the [Client Preparation](#1-client-preparation) phase, the `logo.dds` file will replace the `res/ui/ingame/passport_i90.dds` file within the client directory.

The patch server's contents may look something like this:

```
patches/
|-- 1.0.0/
|   |-- logo.dds
|   |-- patch.json
|-- patches.json
```

Let's say we need another patch. We update the `patches.json` file as such:

```json
{
    "currentVersion": "1.1.0",
    "versions": [
        "1.0.0",
        "1.1.0"
    ]
}
```

And then we add another `patch.json` file as such:

```json
{
    "depend": ["1.0.0*"],
    "downloads": [
        {
            "path": "/1.1.0/boot.cfg",
            "name": "boot.cfg"
        }
    ],
    "updates": {
        "boot": "boot.cfg"
    }
}
```

The above patch will download `http://127.0.0.1:1000/patches/1.1.0/boot.cfg` and save it to a file called `boot.cfg`. The `updates` directive makes changes that may be more complicated than just moving resources into the client. The `boot` field says that it should update the server's `boot.cfg` file.

The `depend` directive will download and run the specified versions (unless that version is blacklisted), WITHOUT the versions' dependencies. If the version is appended by a `*`, then the dependency is recursive, and should download and run that version WITH dependencies. In this example, the patch will download and run patch version `1.0.0`, and if it has dependencies, download and run those too.

Our final patch server contents may be similar to the below example:

```
patches/
|-- 1.0.0/
|   |-- logo.dds
|   |-- patch.json
|-- 1.1.0/
|   |-- boot.cfg
|   |-- patch.json
|-- patches.json
```