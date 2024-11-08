# NOTE TO READER

> [!IMPORTANT]
> This is an experimental package that will eventually replace the current patch implementation here: [resource/patch](/resource/patch). This rewrite aims to extend patcher configurability, create a more robust patcher implementation, and implement an additional patching protocol which closely aligns with the one used by the original patcher.

This experimental package includes: netdevil patcher implementation, nimbus launcher patcher implementation, and nlpatcher command.

## Patching Protocols

### `netdevil` - WORK IN PROGRESS

- Closely follows the patching protocol used by the original patcher during live. This implementation is NOT an exact replica, but follows many of the same steps. Documentation on this protocol can be found here: [LU PATCHING](https://docs.google.com/document/d/1vfcg-Wz4sNncWSQmNe_ucIZH0ayCexmW-Dg2016FRDU/edit?usp=sharing).
- Allows for locale based servers selection. Region based server selection may also be useful. 
- Properly sets `boot.cfg` files to enable in-game patching.
- Keeps track of disk space to clean up UserMade UGC data.

### `nimbus` - NOT IMPLEMENTED

- Nearly identical to the [Theo's Patching Protocol (TPP)](/PATCHING.md) specification.

## Commands

### `nlpatcher` - WORK IN PROGRESS

- This is NOT a command for generating patches. This command simply runs a specific patching protocol for testing that a patching service, either local or remote, patches the correct contents.

---

*Why have multiple patching protocols?*

1. To enable some amount of backwards compatibility for the early TPP protocol.
2. This gives server owners some amount of choice for how they can publish patches. Servers which only use a couple of additions (i.e. LCDR's Dark Souls "You Died" mod) may only need a lightweight patching protocol, like **nimbus**, that is easy to create and edit, while a server with many new zones and resources, may want to use something like the **netdevil** protocol.