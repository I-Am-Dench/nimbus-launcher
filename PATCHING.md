# Theo's Patching Protocol (TPP)

## Definitions

| Keyword                         | Definition                                                                              |
| ------------------------------- | --------------------------------------------------------------------------------------- |
| Patch                           | The union of the set of Patch Directives and its associated Patch Resources.            |
| Patch Runner                    | The process which implements and/or enforces the usage of the TPP.                      |
| Patch Version                   | The unique name given to a Patch.                                                       |
| Patch Resource                  | A file that is managed by the Patch Runner for use by Patch Directives.                 |
| Local Server Configuration      | The selectable configuration that is managed by the Patch Runner.                       |
| Local Server Boot Configuration | The `boot.cfg` file that is managed by the Local Server Configuration.                  |
| Local Patch Directory           | The directory for a specific Patch where Patch Resources are downloaded, or copied, to. |
| Patch Directory                 | The remote, or local, directory in which relative patch data is fetched from.           |
| versions.json                   | The JSON encoded data fetched from the Patch Directory which includes the server’s current Patch Version, and a list of previous, valid Patch Versions. |
| Version Directory               | The remote, or local, directory with a valid version name, that is a direct child of the Patch Directory.<br/></br>Example: `<Patch Directory>/v1.0.0/` |
| patch.json                      | The JSON encoded data fetched from the Version Directory containing a list of Patch Directives. |
| Patch Directive                 | An instruction within the patch.json which tells the Patch Runner what operations to perform. |
| Client Directory                | The directory where the client’s `res` directory is located.                            |

## Protocol

While this protocol could be broken down into a singular list of steps, this list has been split into two parts: [Fetch](#fetch) and [Update](#update). [Fetch](#fetch) dictates the steps in how the protocol determines if a patch is available, and [Update](#update) dictates how that patch is applied.

### Fetch

1. **Fetch the *versions.json*** from the *Patch Directory*.
2. **If the current *Patch Version*** listed in the *versions.json* matches the current version of the *Local Server Configuration*, the protocol should terminate.
3. **If the versions are different**, the protocol should continue on to step 3.
4. **Fetch the *patch.json*** from the *Version Directory*.

Servers should respond to valid requests with a `200 - OK` status code.

> The Nimbus Launcher treats any response status code >= `200` and \< `400` as a valid response.

### Update

PLEASE NOTE THE FOLLOWING WHEN IMPLEMENTING A PATCH RUNNER:

- *Patch* dependencies MUST NOT inherit the *Local Patch Directory* from its parent *Patch*.
- Dependencies MUST NOT run the **update** directive.
- *download objects* include a `path` relative to the *Version Directory* and a `name` which is relative to the *Local Patch Directory*.
- The Nimbus Launcher runs the **transfer** directive ONLY when the play button is pushed. The protocol DOES NOT enforce this as a standard.
- ALL PATCHES MUST verify that their *Patch Version* follows the [strict versioning conventions](#versioning) and should terminate if the version name does not match.

While all *Patch Directives* within the *patch.json* can be included in any order, the Patch Runner MUST run each directive in the following sequence:

1. Dependencies (**depend**) - For each of the *Patch Versions* listed as a dependency ->
    1. **If the current version name** iteration is INVALID ->
        1. The protocol MUST terminate.
    2. **If the current version name** iteration is VALID and is suffixed WITH `*` ->
        1. Fetch the *patch.json* for the current version name
        2. Recursively run the [Update](#update) section of the protocol.
    3. **If the current version name** iteration is VALID and is suffixed WITHOUT `*` ->
        1. Fetch the *patch.json* for the current version name
        2. Recursively run the [Update](#update) section of the protocol WITHOUT fetching that version’s dependencies.
2. Download (**download**) - contains a list of *download objects*: For each of the *download objects* ->
    1. **Fetch the Patch Resource** from the *Version Directory* located by the `path`.
    2. **Save the resource** within the *Local Patch Directory* with the specified `name`.
3. Transfer (**transfer**) - contains a mapping of *Patch Resource* names to a resource relative to the *Client Directory*. For each of the mapped pairs ->
    1. **If either of the resource names** are NONLOCAL (their resolved path is outside of their *Local Patch Directory* or the *Client Directory*) the protocol MUST terminate.
    2. **Copy the *Patch Resource*** to the resource relative to the *Client Directory*, ONLY IF that client resource already exists. If the client resource does not already exist, the transfer MUST be ignored and MAY terminate the protocol. This step may cache the client resources if necessary.
4. Update (**update**) - contains a set of sub-directives, completing operations that may be more complex than a simple copy. These sub-directive can be completed in any order. For each sub-directive ->
    - **boot** : the name of a *Patch Resource*
        - Update the *Local Server Boot Configuration* with the specified *Patch Resource*.
    - **protocol** : a protocol name
        - Update the *Local Server Configuration*’s protocol field with the specified protocol name.

## Authentication

If an authentication token is required to retrieve *Patch* content from a server, the token SHOULD be sent within the `TPP-Token` header. If the authentication token has been determined to be invalid, the server SHOULD respond with a `401 - Unauthorized` status code.

> The Nimbus Launcher treats any response status code >= `400` as an invalid response.

## Versioning

The **TPP** follows a strict version naming convention. Any *Patch Version* that does not follow the standard versioning pattern MUST incure an error.

*Patch Versions* MUST use 3 numeric version components: MAJOR, MINOR, and PATCH. Optionally, versions can be followed by any number of alpha numeric characters or a '_', '.', or '-'. An optional prefix, 'v', is also permitted.

The Nimbus Launcher checks *Patch Version* names with the following regular expression:

```
^(v|V)?[0-9]+\.[0-9]+\.[0-9]+([0-9a-zA-Z_.-]+)?$
```

### Valid Versions

- `1.0.0`
- `v1.0.0`
- `2.5.09-alpha2`
- `v3.0.1_experimental`

### Invalid Versions

- `1`
- `v2`
- `1.0`
- `version-1`

## Examples

### *versions.json*

```json
{
    "currentVersion": "v1.0.0",
    "previousVersions": [
        "v0.1.0",
        "v0.2.0",
        ...
    ]
}
```

### *patch.json*

```json
{
    "depend": [ "v0.5.1*", ... ],
    "download": [
        {
            "path": "/v1.0.0/boot.cfg",
            "name": "boot.cfg"
        }
    ],
    "update": {
        "boot": "boot.cfg"
    },
    "transfer": {
        "logo.dds": "res/ui/ingame/passport_i90.dds"
    }
}
```

### Server Patch Directory

```
patches/
|-- v0.1.0/
|   |-- ...
|   |-- patch.json
|-- v0.2.0/
|   |-- ...
|   |-- patch.json
|-- ...
|-- versions.json
```