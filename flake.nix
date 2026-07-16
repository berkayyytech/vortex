{
  description = "Vortex - a fast, agentless TUI for managing Linux VPS servers over SSH";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        # go.mod requires >=1.26.5; nixpkgs currently ships 1.26.4.
        go = pkgs.go.overrideAttrs (old: rec {
          version = "1.26.5";
          src = pkgs.fetchurl {
            url = "https://go.dev/dl/go${version}.src.tar.gz";
            hash = "sha256-SVvkvIcXasVnOS5bQRar2YRm0z17SdQedkzMaXay3EI=";
          };
        });
      in
      {
        packages.default = (pkgs.buildGoModule.override { inherit go; }) {
          pname = "vortex";
          version = "0.1.0";

          src = ./.;

          subPackages = [ "cmd/vps-manager" ];
          vendorHash = "sha256-JPX+AP6Xfn1ApBZ4TXurkaT0WR+XzKkQuYZt2ve6wTk=";

          env.GOTOOLCHAIN = "local";

          postInstall = ''
            mv $out/bin/vps-manager $out/bin/vortex
          '';

          meta = with pkgs.lib; {
            description = "A fast, agentless TUI for managing Linux VPS servers over SSH";
            homepage = "https://github.com/berkayyytech/vortex";
            license = licenses.mit;
            mainProgram = "vortex";
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = [
            go
            pkgs.gopls
            pkgs.golangci-lint
            pkgs.delve
          ];

          env.GOTOOLCHAIN = "local";
        };
      });
}
