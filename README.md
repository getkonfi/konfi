# konfi

TUI for editing dotfiles.

![konfi usage demo](assets/konfi-usage.webp)

## Install

### Install Script

```sh
curl -fsSL https://raw.githubusercontent.com/getkonfi/konfi/main/install.sh | sh
```

### Homebrew

```sh
brew tap getkonfi/tap
brew install konfi
```

### Nix

```sh
nix shell github:getkonfi/nur-packages#konfi
```

### Manual Packages

Debian / Ubuntu:

```sh
curl -LO https://github.com/getkonfi/konfi/releases/latest/download/konfi_<version>_amd64.deb
sudo apt install ./konfi_<version>_amd64.deb
```

Use the `arm64` package on ARM machines.

Fedora / RHEL:

```sh
curl -LO https://github.com/getkonfi/konfi/releases/latest/download/konfi-<version>-1.x86_64.rpm
sudo dnf install ./konfi-<version>-1.x86_64.rpm
```

Use the `aarch64` package on ARM machines.

Tarball:

```sh
curl -LO https://github.com/getkonfi/konfi/releases/latest/download/konfi_<version>_linux_amd64.tar.gz
tar -xzf konfi_<version>_linux_amd64.tar.gz
sudo install -m 0755 konfi /usr/local/bin/konfi
```

Available archive targets:

- `linux_amd64`
- `linux_arm64`
- `darwin_amd64`
- `darwin_arm64`

### From Source

```sh
git clone https://github.com/getkonfi/konfi.git
cd konfi
make build
sudo install -m 0755 konfi /usr/local/bin/konfi
```

Run without installing:

```sh
make run
```

## Development

```sh
make tools
make test
make lint
```

Local release build:

```sh
make release-snapshot
```
