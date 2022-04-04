# go-ipfs-swh-plugin

IPFS plugin for bridging requests to the Software Heritage API,
implemented as a datastore.

Using the `makecid.py` script included in the repository, or by pasting
the magic `f01551114` bytes in front of an identifier hash, you can
convert a SWHID to a CID which the bridge can handle.

```
swh:1:cnt:94a9ed024d3859793618152ea559a168bbcbb5e2
          ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ copy this part

f0178111494a9ed024d3859793618152ea559a168bbcbb5e2
         ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ paste it here
^^^^^^^^^ (prefix means CID v1, "git-raw" codec, SHA-128)
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
#       ^^^ To avoid /var/lib/ipfs if installed globally, at least on NixOS
$ result/bin/ipfs init -e -p swhbridge
$ result/bin/ipfs dag get \
    --output-codec=git-raw \
       f0178111494a9ed024d3859793618152ea559a168bbcbb5e2
#   ^^^ render in human readable text
#      ^^^ CID corresponding to GPLv3 text from SWH
```

Nodes connecting to a bridge node do not need to have the plugin to
fetch data from the SWH archive. In this way, it's entirely transparent.
