# go-ipfs-swh-plugin

IPFS plugin for bridging requests to the Software Heritage API,
implemented as a datastore.

Using the `makecid.py` script included in the repository, or by pasting
the magic `f01551114` bytes in front of an identifier hash, you can
convert a SWHID to a CID which the bridge can handle.

```
swh:1:cnt:94a9ed024d3859793618152ea559a168bbcbb5e2
          ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ copy this part

f0155111494a9ed024d3859793618152ea559a168bbcbb5e2
^^^^^^^^^ paste it after these bytes
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
$ result/bin/ipfs init -e -p swhbridge
$ result/bin/ipfs cat f0155111494a9ed024d3859793618152ea559a168bbcbb5e2
#                     ^^^ CID corresponding to GPLv3 text from SWH
```

Nodes connecting to a bridge node do not need to have the plugin to
fetch data from the SWH archive. In this way, it's entirely transparent.