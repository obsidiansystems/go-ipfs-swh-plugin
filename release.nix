let
  pkgs = import ./dep/nixpkgs {};

  ipfs-swh = import ./. { inherit pkgs; };

  # CID representing the SWHID
  # https://archive.softwareheritage.org/browse/snapshot/c7c108084bc0bf3d81436bf980b46e98bd338453/directory/
  # (no particular attachment to darkroom, it was just the example
  # snapshot from the SWHID docs)
  swhid_cid = "znDfqECWqk8VqwNL2ayyYnvJdwDN9qYqa2";

  # Test data in dag-json format
  test_data_json = pkgs.writeText "test_data.json" ''
  {
    "Data": "This is a test"
  }
  '';
in
  pkgs.nixosTest {
    nodes = {
      bridge = { config, pkgs, ... }: {
        networking.extraHosts = ''
          0.0.0.0 archive.softwareheritage.org
        '';

        services.ipfs = {
          enable = true;
          package = ipfs-swh;
          emptyRepo = true;
          #profile = "swhbridge";
        };
        networking.firewall.allowedTCPPorts = [ 4001 ];
        networking.firewall.allowedUDPPorts = [ 4001 ];
      };
      client = { config, pkgs, ... }: {
        services.ipfs = {
          enable = true;
          package = ipfs-swh;
        };
        networking.firewall.allowedTCPPorts = [ 4001 ];
        networking.firewall.allowedUDPPorts = [ 4001 ];
      };
    };

    testScript = ''
      # SWH Archive servers are inaccessible from this machine
      bridge.fail(
        'curl https://archive.softwareheritage.org'
      )

      # Skip once we can set profile in NixOS Config.
      dir = bridge.succeed('mktemp -d').strip()
      # Complains some things can't be fetched, I think, but this is harmless.
      bridge.fail(f'IPFS_PATH={dir} ipfs -L --offline init -e -p swhbridge')
      bridge.succeed(f'grep swhbridge {dir}/config')

      # Skipping until we work on snapshot decoding more
      #
      # # Test storing some data with our codec (Fails with a specific
      # # error because encode is not implemented)
      # output = machine.fail(
      #   'ipfs -L --offline dag put --store-codec swhid-1-snp < ${test_data_json} 2>&1'
      # ).strip()

      # if "test error (encode)" not in output:
      #   raise ValueError("Test encoding error did not fire (plugin not loaded?)")
    '';
  }
