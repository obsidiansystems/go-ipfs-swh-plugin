from cid import make_cid
import sys
import multicodec
import multihash

codec = 'git-raw'

if len(sys.argv) <= 1:
  print(f"usage: {sys.argv[0]} [list of swhids or hashes]...")
  print("Alternatively, copy the hash from the SWHID and paste f01551114 in front")

for arg in sys.argv[1:]:
  if arg.startswith("swh:"):
    splat = arg.split(";")[0].split(':')
    if splat[2] == 'cnt':
      codec = 'raw'
    elif splat[2] == 'snp':
      codec = 'swhid-1-snp'
      sys.stderr.write(f"warning: {arg} encoding is not yet supported\n")
    arg = splat[3]

  it = bytes.fromhex(arg)
  hash = multihash.encode(it, multicodec.multicodec.NAME_TABLE['sha1'])
  print(make_cid(1, 'raw', hash).encode(encoding='base16').decode('utf-8'))