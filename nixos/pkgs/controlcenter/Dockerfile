FROM nixos/nix

RUN nix-channel --add https://nixos.org/channels/nixpkgs-unstable nixpkgs
RUN nix-channel --update

COPY . .

RUN nix-build -E 'with import <nixpkgs> { };  callPackage ./default.nix {}' -v
