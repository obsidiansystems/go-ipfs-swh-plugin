let
  pkgs = import ./dep/nixpkgs {};

  ipfs = import ./. { inherit pkgs; };

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
    system = "x86_64-linux";
    machine = { config, pkgs, ... }: {
      networking.firewall.enable = false;
      networking.extraHosts = ''
        0.0.0.0 archive.softwareheritage.org
      '';

      environment.systemPackages = [ ipfs ];
    };

    # TODO: Add bridge node (when we have a bridge)
    testScript = ''
      machine.succeed('ipfs -L --offline init')

      # Skipping until we work on snapshot decoding more
      #
      # # Test storing some data with our codec (Fails with a specific
      # # error because encode is not implemented)
      # output = machine.fail(
      #   'ipfs -L --offline dag put --store-codec swhid-1-snp < ${test_data_json} 2>&1'
      # ).strip()

      # if "test error (encode)" not in output:
      #   raise ValueError("Test encoding error did not fire (plugin not loaded?)")

      # SWH Archive servers are inaccessible from this machine
      machine.fail(
        'curl https://archive.softwareheritage.org'
      )
    '';
  }
