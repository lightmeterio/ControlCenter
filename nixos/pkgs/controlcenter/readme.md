# NixOs Package

If we publish a new release then we need to update all nix deps and change the 'rev' and 'version' inside of the default.nix

## requirements

* [docker](https://www.docker.com/) - Build, Manage and Secure Your Apps Anywhere. Your Way.
* [nixos](https://nixos.org/) -  Nix is a powerful package manager for Linux and other Unix systems that makes package management reliable and reproducible
* [vgo2nix](https://github.com/nix-community/vgo2nix) - Convert go.mod files to nixpkgs buildGoPackage compatible deps.nix files

## Versions requirements
* Nix **>=2.3.7**
* Docker **>=18.09.2**
* vgo2nix **9286b289764831bd40c2a82fe311caef019056f4**

# Usage

From the root directory execute

```bash
vgo2nix
```

Move dep.nix to /gitlab.com/lightmeter/controlcenter/nixos/pkgs/controlcenter

# Build pkg

```bash
nix-build -E 'with import <nixpkgs> { };  callPackage ./default.nix {}' -v
```

# docker build pkg

```bash
sudo docker build .
```

# Inclusion example

```bash
{pkgs, ...}:

let
  customPkgs = import /gitlab.com/lightmeter/controlcenter/nixos/pkgs/controlcenter/default.nix {};
{
  environment.systemPackages = [
    customPkgs.lightmeter
  ];
}
```