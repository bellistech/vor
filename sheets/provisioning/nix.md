# Nix (Package Manager & Build System)

Purely functional package manager with reproducible builds, atomic upgrades, and declarative system configuration.

## nix-env (Imperative Package Management)

### Search for packages

```bash
nix-env -qaP '.*firefox.*'                # search by regex
nix search nixpkgs firefox                 # new CLI (flakes-enabled)
```

### Install packages

```bash
nix-env -iA nixpkgs.htop                  # install by attribute
nix-env -iA nixpkgs.ripgrep nixpkgs.fd
```

### Remove packages

```bash
nix-env -e htop
```

### List installed packages

```bash
nix-env -q                                 # list installed
nix-env -q --installed
```

### Upgrade packages

```bash
nix-env -u                                 # upgrade all
nix-env -uA nixpkgs.htop                  # upgrade one
```

### Rollback

```bash
nix-env --rollback                         # previous generation
nix-env --list-generations                 # show all generations
nix-env --switch-generation 42             # switch to specific generation
```

## nix-shell (Ephemeral Environments)

### Ad-hoc shell with packages

```bash
nix-shell -p python3 nodejs                # drop into shell with packages
nix-shell -p python3 --run "python3 --version"  # run command and exit
```

### Project shell (shell.nix)

```bash
# { pkgs ? import <nixpkgs> {} }:
# pkgs.mkShell {
#   buildInputs = [
#     pkgs.go
#     pkgs.gopls
#     pkgs.sqlite
#     pkgs.pkg-config
#   ];
#   shellHook = ''
#     export GOPATH=$PWD/.go
#     echo "Dev shell ready"
#   '';
# }
```

```bash
nix-shell                                  # enter shell defined by shell.nix
nix-shell --pure                           # no host PATH leakage
```

## Flakes

### Enable flakes (in ~/.config/nix/nix.conf)

```bash
# experimental-features = nix-command flakes
```

### Initialize a flake

```bash
nix flake init
nix flake init -t templates#go             # from template
```

### flake.nix structure

```bash
# {
#   inputs = {
#     nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.05";
#     flake-utils.url = "github:numtide/flake-utils";
#   };
#   outputs = { self, nixpkgs, flake-utils }:
#     flake-utils.lib.eachDefaultSystem (system:
#       let pkgs = nixpkgs.legacyPackages.${system}; in {
#         devShells.default = pkgs.mkShell {
#           buildInputs = [ pkgs.go pkgs.gopls ];
#         };
#         packages.default = pkgs.buildGoModule {
#           pname = "myapp";
#           version = "0.1.0";
#           src = ./.;
#           vendorHash = "sha256-AAAA...";
#         };
#       });
# }
```

### Flake commands

```bash
nix develop                                # enter devShell
nix build                                  # build default package
nix run                                    # build and run
nix flake update                           # update flake.lock
nix flake show                             # show outputs
nix flake metadata                         # show inputs and lock info
```

### Run a package from a flake

```bash
nix run nixpkgs#htop
nix shell nixpkgs#ripgrep nixpkgs#fd       # ad-hoc shell (flake style)
```

## nix-build

### Build a derivation

```bash
nix-build                                  # build default.nix
nix-build -A mypackage                     # build specific attribute
nix-build '<nixpkgs>' -A htop              # build from nixpkgs
```

### Result

```bash
ls -la result                              # symlink to /nix/store/...
./result/bin/myapp                          # run the built binary
```

## NixOS Configuration

### /etc/nixos/configuration.nix

```bash
# { config, pkgs, ... }:
# {
#   boot.loader.grub.enable = true;
#   networking.hostName = "myhost";
#   time.timeZone = "America/Los_Angeles";
#
#   users.users.alice = {
#     isNormalUser = true;
#     extraGroups = [ "wheel" "docker" ];
#     shell = pkgs.zsh;
#   };
#
#   environment.systemPackages = with pkgs; [
#     vim git htop tmux
#   ];
#
#   services.openssh.enable = true;
#   services.nginx.enable = true;
#   services.postgresql.enable = true;
#
#   networking.firewall.allowedTCPPorts = [ 22 80 443 ];
#
#   system.stateVersion = "24.05";
# }
```

### Rebuild system

```bash
sudo nixos-rebuild switch                  # apply and switch
sudo nixos-rebuild test                    # apply without adding to bootloader
sudo nixos-rebuild boot                    # apply on next boot
sudo nixos-rebuild build                   # build only, no activation
```

## Overlays

### Override a package

```bash
# final: prev: {
#   htop = prev.htop.overrideAttrs (old: {
#     patches = old.patches ++ [ ./my-patch.patch ];
#   });
# }
```

### Apply overlay in flake

```bash
# nixpkgs.overlays = [ (import ./overlay.nix) ];
```

## Derivations

### Simple derivation

```bash
# pkgs.stdenv.mkDerivation {
#   pname = "myapp";
#   version = "1.0";
#   src = ./.;
#   buildInputs = [ pkgs.gcc ];
#   buildPhase = "gcc -o myapp main.c";
#   installPhase = "mkdir -p $out/bin; cp myapp $out/bin/";
# }
```

### Go module

```bash
# pkgs.buildGoModule {
#   pname = "myapp";
#   version = "0.1.0";
#   src = ./.;
#   vendorHash = "sha256-...";         # or vendorHash = null for vendored
# }
```

## Home Manager

### Standalone installation

```bash
nix-channel --add https://github.com/nix-community/home-manager/archive/release-24.05.tar.gz home-manager
nix-channel --update
nix-shell '<home-manager>' -A install
```

### ~/.config/home-manager/home.nix

```bash
# { config, pkgs, ... }:
# {
#   home.username = "alice";
#   home.homeDirectory = "/home/alice";
#   home.stateVersion = "24.05";
#
#   home.packages = with pkgs; [
#     ripgrep fd bat jq
#   ];
#
#   programs.git = {
#     enable = true;
#     userName = "Alice";
#     userEmail = "alice@example.com";
#   };
#
#   programs.zsh.enable = true;
#   programs.tmux.enable = true;
#   programs.neovim.enable = true;
# }
```

### Apply home configuration

```bash
home-manager switch
home-manager generations
home-manager rollback
```

## Garbage Collection

```bash
nix-collect-garbage                        # remove unreachable store paths
nix-collect-garbage -d                     # also delete old generations
nix-collect-garbage --delete-older-than 30d
nix store gc                               # new CLI equivalent
nix store optimise                         # hard-link identical files
```

## Tips

- Everything in `/nix/store` is immutable and content-addressed. No dependency conflicts.
- `nix-shell -p` is the fastest way to try a tool without installing it permanently.
- Flakes are the modern way to manage Nix projects. They replace channels with pinned, reproducible inputs.
- `nix develop` replaces `nix-shell` for flake-based projects.
- `nix-collect-garbage -d` frees disk space by removing all old generations. Run it periodically.
- Pin `nixpkgs` to a specific commit or release branch for reproducible builds.
- `nix why-depends /nix/store/...-A /nix/store/...-B` shows why package A depends on B.
- NixOS rollback is instant: just select a previous generation from the bootloader.

## See Also

- ansible
- docker
- vagrant
- apt
- brew

## References

- [Nix Manual](https://nixos.org/manual/nix/stable/)
- [NixOS Manual](https://nixos.org/manual/nixos/stable/)
- [Nixpkgs Manual](https://nixos.org/manual/nixpkgs/stable/)
- [Nix Package Search](https://search.nixos.org/packages)
- [NixOS Options Search](https://search.nixos.org/options)
- [Nix Flakes Reference](https://nixos.org/manual/nix/stable/command-ref/new-cli/nix3-flake)
- [Nix Language Overview](https://nixos.org/manual/nix/stable/language/)
- [Nixpkgs GitHub Repository](https://github.com/NixOS/nixpkgs)
- [Nix GitHub Repository](https://github.com/NixOS/nix)
- [nix.dev — Community Tutorials](https://nix.dev/)
- [Home Manager — User Dotfile Management](https://github.com/nix-community/home-manager)
