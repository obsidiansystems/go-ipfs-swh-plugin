# Nix expression for building IPFS + our plugin. Essentially the only
# way to compile plugins consistently is to sneak them into the IPFS
# source tree. In particular, none of the following sensible approaches
# work:
#
#   * Using IPFS_VERSION=${pkgs.ipfs.version} in the plugin Makefile
#   * Using IPFS_VERSION=v0.11.0 in the plugin Makefile
#   * Getting Nix to "build" a derivation of just the IPFS sources, and
#     building the plugin against that
#
# So what goes on here is that we use the Nixpkgs Go infrastructure to
# vendor IPFS dependencies, and use symlinkJoin to add our plugin to the
# generated vendor directory. Then, we can put the plugin into the IPFS
# preload list. That way, the plugin is baked into the binary, and
# guaranteed to build + run consistently.
{ pkgs ? 
  import (builtins.fetchTarball {
    name = "nixpkgs-release-21.11";
    url = "https://github.com/NixOS/nixpkgs/archive/release-21.11.tar.gz";
    sha256 = "1qgvkmjf69gqa1ddisgxl0g068q021f3ya7q9y4407pdrqcv61yq";
  }) {}
}:
let
  # Filter the plugin sources to just the skeleton + Go files
  filter = with pkgs.lib; path: type: 
    type == "directory" || hasSuffix ".go" (baseNameOf path);

  # This derivation copies the plugin *sources* to the same directory
  # where Go would put our package if it were a true dependency of ipfs.
  # Go vendoring works by source, so we do no compilation work here.
  go-ipfs-swh-plugin = pkgs.stdenv.mkDerivation rec {
    name = "go-ipfs-swh-plugin";
    version = "0.0.1";
    src = builtins.filterSource filter ./.;

    installPhase = ''
    mkdir -p $out/github.com/obsidiansystems/${name}
    cp * $out/github.com/obsidiansystems/${name} -r
    '';
  };

  # IPFS master as of 2022-02-03, 14:00 GMT-3
  ipfs-source = pkgs.fetchFromGitHub {
    owner = "ipfs";
    repo = "go-ipfs";
    rev = "cde79df1408c3bd518fec1622d97bf4a251af81e";
    sha256 = "1a9sylxv8ay6lvv1w3qhg29pyzk81szx3s20k00fa7k8bglwlw7j";
  };
  # The version that ^^^ reports itself as
  ipfs-version = "v0.13.0";

  ipfs-replacements = ''
  # Update our version of go-multicodec
  go mod edit -replace=github.com/multiformats/go-multicodec@v0.3.0=github.com/multiformats/go-multicodec@v0.4.0
  '';

  # Use the Nixpkgs Go build support to generate a fixed-output
  # derivation of the *dependencies* of the above IPFS package.
  ipfs-vendor = (pkgs.buildGoModule {
    name = "ipfs";
    version = ipfs-version;
    src = ipfs-source;

    vendorSha256 = "05f3f4ibsmzvlqmzv7g4xa867fsl38r03gzmidvaxq29grsjjhj2";
    overrideModAttrs = old: {
      postConfigure = ipfs-replacements;
    };
  }).go-modules;
  # ^^^^^^^^^^^^ This means that we don't build IPFS twice.

  # Join up the IPFS dependencies + our fake plugin dependency.
  go-modules = pkgs.symlinkJoin {
    name = "ipfs+swh-go-modules";
    paths = [ ipfs-vendor go-ipfs-swh-plugin ];
  };
in
pkgs.buildGoModule rec {
  name = "ipfs+swh";
  version = ipfs-version; 
  src = ipfs-source;

  buildInputs = [ go-modules ];
  vendorSha256 = null;
  # ^^^ Use a "pre-existing" vendor. Actually, we're going to smuggle in
  # our own vendor.

  subPackages = [ "cmd/ipfs" ];
  # ^^^ What Go packages will be built (cmd/ipfs is the IPFS CLI)

  patches = [ ./preload-plugin.patch ];
  #           ^^^^^^^^^^^^^^^^^^^^^^ Adds our plugin to the plugin
  #           preload list.

  passthru = {
    inherit go-ipfs-swh-plugin go-modules;
  };

  postConfigure = ''
  # Since we changed the plugin preload list, we re-run the script that
  # generates the Go file.
  bash plugin/loader/preload.sh > plugin/loader/preload.go

  # And copy our generated dependencies module to the vendor folder.
  cp -r --reflink=auto ${go-modules} vendor

  ${ipfs-replacements}
  '';

  postInstall = pkgs.ipfs.postInstall;
  doCheck = false;

  meta = with pkgs.lib; {
    description = "IPFS, built with the SWHID plugin";
    homepage = "https://ipfs.io/";
    license = licenses.mit;
    platforms = platforms.unix;
    maintainers = with maintainers; [ obsidian-systems-maintenance ];
  };
}