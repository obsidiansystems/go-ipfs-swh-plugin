# go-ipfs-swh-plugin

IPFS plugin for bridging requests to the Software Heritage API,
implemented as a datastore.

Using the `makecid.py` script included in the repository, or by pasting
the magic `f01781114` bytes in front of an identifier hash, you can
convert a SWHID to a CID which the bridge can handle.

```
swh:1:cnt:94a9ed024d3859793618152ea559a168bbcbb5e2
          ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ copy this part

f0178111494a9ed024d3859793618152ea559a168bbcbb5e2
         ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ paste it here
^^^^^^^^^ prefix so CID is self-describing
          (means roughly CID v1, "git-raw" codec, SHA-128, base-16)
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
plugin's own configuration profile to set up a bridge, and test fetching
using any of the standard ipfs reading commands:


```bash
$ unset IPFS_PATH
#       ^^^ To avoid /var/lib/ipfs if installed globally, at least on
#       ^^^ NixOS.
$ result/bin/ipfs init -e -p swhbridge
$ result/bin/ipfs dag get \
    --output-codec=git-raw \
       f0178111494a9ed024d3859793618152ea559a168bbcbb5e2
#   ^^^ render in human readable text
#      ^^^ CID corresponding to GPLv3 text from SWH
```

Nodes connecting to a bridge node do not need to have the plugin to
fetch data from the SWH archive. In this way, it's entirely transparent.

## Browsing the archive

We recommend going to https://archive.softwareheritage.org/browse/,
finding a file, and then clicking the floating "Permalinks" on the right
side to get a `swh:1:cnt:...` SWHID. One can then follow the
instructions at the top of this readme to fetch it.

Once we can implement fetching the other types objects, however, one can
just pick a repo root hash and browse from there all with just the IPFS
API!
