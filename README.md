# go-ipfs-swh-plugin

IPFS plugin for bridging requests to the Software Heritage API,
implemented as a datastore.

Using the `makecid.py` script included in the repository, or by pasting
the magic `f01781114` bytes in front of an identifier hash, you can
convert a git-like SWHID to a CID which the bridge can handle.

```
swh:1:cnt:94a9ed024d3859793618152ea559a168bbcbb5e2
          ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ copy this part
      ^^^ must be one of "cnt", "dir", "rev", or "rel"

f0178111494a9ed024d3859793618152ea559a168bbcbb5e2
         ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ paste it here
^^^^^^^^^ prefix so CID is self-describing
          (means roughly CID v1, "git-raw" codec, SHA-1, base-16)
```

For snapshots, however the magic prefix is different, since they are not
also git objects.

```
swh:1:snp:9dcebebe2bb56cabdd536787886d582b762a0376
          ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ copy this part

      ^^^ must be "snp"

f01F00311149dcebebe2bb56cabdd536787886d582b762a0376
           ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ paste it here
^^^^^^^^^^^ prefix so CID is self-describing
		    (means roughly CID v1, "swh-1-snp" codec, SHA-1, base-16)
```

## Setting up a bridge

To set up a bridge, you must first have IPFS built with the plugin.
This repository includes a Nix derivation to compile IPFS with the
plugin baked in, which is the recommended deployment strategy for Go
plugins. Simply

```
$ nix-build
```

And use the resulting `result/bin/ipfs` binary. You can then use the
plugin's own configuration profile to set up a bridge:

```bash
$ unset IPFS_PATH
#       ^^^ To avoid /var/lib/ipfs if installed globally, at least on
#       ^^^ NixOS.
$ result/bin/ipfs init -e -p swhbridge
```

This may show `Error: ipld: could not find QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn`,
you can safely ignore this error.

If you have an SWH authentication token, you can add it to the config
file just created with `ipfs init`:

``` diff
--- a/config
+++ b/config
@@ -11,7 +11,8 @@
       "mounts": [
         {
           "child": {
-            "type": "swhbridge"
+            "type": "swhbridge",
+            "auth-token": "<paste token here>"
           },
           "mountpoint": "/blocks",
           "prefix": "swhbridge.datastore",
```

## Using the bridge

After setting up, try to test fetching using any of the standard ipfs
reading commands:

Fetch a file:
```bash
$ result/bin/ipfs dag get \
    --output-codec=git-raw \
       f0178111494a9ed024d3859793618152ea559a168bbcbb5e2
#   ^^^ render in human readable text
#      ^^^ CID corresponding to GPLv3 text from SWH
```

Fetch a directory (listing):
```bash
$ result/bin/ipfs dag get \
    f017811141ecc6062e9b02c2396a63d90dfac4d63690e488b | jq
#   ^^^ CID corresponding to Linux's root directory
```

Fetch a revision:
```bash
$ result/bin/ipfs dag get \
    f017811141a0dd0088247f9d4e403a460f0f6120184af3e15 | jq
#   ^^^ CID corresponding to a recent GHC commit
```

Fetch a snapshot:
```bash
$ result/bin/ipfs dag get \
    f01F00311149dcebebe2bb56cabdd536787886d582b762a0376 | jq
#   ^^^ CID corresponding to a recent github.com/reflex-frp/patch
#       snapshot
```

Fetch recursively (!):
```bash
$ result/bin/ipfs dag get \
    --output-codec=git-raw \
       f017811141a0dd0088247f9d4e403a460f0f6120184af3e15/tree/compiler/hash/GHC/hash/Core/hash/Type.hs/hash
#   ^^^ render in human readable text
#      ^^^ CID and path corresponding to file in that recent GHC commit
```

Nodes connecting to a bridge node do not need to have the plugin to
fetch data from the SWH archive. In this way, it's entirely transparent.

## Additional Logging

The bridge uses [`go-log`](https://github.com/ipfs/go-log) like most
IPFS software. Prefix you commands with e.g.
```bash
GOLOG_LOG_LEVEL="swh-bridge=info"
```
to see more information about what is going on. (One can do `=debug` too
for even more info, but requests the bridge skips knowing it cannot
handle add a fair bit of noise.)

## Browsing the archive

We recommend going to https://archive.softwareheritage.org/browse/,
finding a file, and then clicking the floating "Permalinks" on the right
side to get a `swh:1:cnt:...` SWHID. One can then follow the
instructions at the top of this readme to fetch it.

Once we can implement fetching the other types objects, however, one can
just pick a repo root hash and browse from there all with just the IPFS
API!
