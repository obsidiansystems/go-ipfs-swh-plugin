#!/usr/bin/env python3

# With nix, try:
# nix-shell -p 'python3.withPackages (s: with s; [ py-multicodec py-multihash py-cid ])'

from cid import make_cid
import sys
import multicodec
import multihash

if len(sys.argv) <= 1:
  print(f"usage: {sys.argv[0]} [list of swhids or hashes]...")
  print("Alternatively, copy the hash from the SWHID and paste f01781114 in front")

for arg in sys.argv[1:]:
  if arg.startswith("swh:"):
    splat = arg.split(";")[0].split(':')
    ty = splat[2]
    if ty in {'cnt', 'dir', 'rev', 'rel'}:
      codec = 'git-raw'
    elif ty == 'snp':
      codec = 'swhid-1-snp'
      sys.stderr.write(f"warning: {arg} encoding is not yet supported\n")
    else:
      sys.stderr.write(f"unknown SWHID object type: {ty}")
      sys.exit(1)
    arg = splat[3]

  it = bytes.fromhex(arg)
  hash = multihash.encode(it, multicodec.multicodec.NAME_TABLE['sha1'])
  print(make_cid(1, codec, hash).encode(encoding='base16').decode('utf-8'))
