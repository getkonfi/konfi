#!/bin/sh
set -eu

repo="${KONFI_REPO:-getkonfi/konfi}"
bin="${KONFI_BIN:-konfi}"
version="${KONFI_VERSION:-}"
tmpdir=""

die() {
  printf 'install.sh: %s\n' "$*" >&2
  exit 1
}

usage() {
  cat <<EOF
usage: install.sh [--version VERSION]

options:
  --version VERSION  install a specific release version or tag

environment:
  KONFI_VERSION      release version or tag, defaults to latest
  KONFI_REPO         GitHub repo, defaults to getkonfi/konfi
EOF
}

http_get() {
  url=$1

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url"
    return
  fi

  if command -v wget >/dev/null 2>&1; then
    wget -qO- "$url"
    return
  fi

  die "curl or wget is required"
}

http_download() {
  url=$1
  output=$2

  if command -v curl >/dev/null 2>&1; then
    curl -fL --progress-bar "$url" -o "$output"
    return
  fi

  if command -v wget >/dev/null 2>&1; then
    wget -q -O "$output" "$url"
    return
  fi

  die "curl or wget is required"
}

cleanup() {
  if [ -n "$tmpdir" ] && [ -d "$tmpdir" ]; then
    rm -rf "$tmpdir"
  fi
}

detect_os() {
  os=$(uname -s | tr '[:upper:]' '[:lower:]')

  case "$os" in
    linux|darwin)
      printf '%s\n' "$os"
      ;;
    *)
      die "unsupported operating system: $os"
      ;;
  esac
}

detect_arch() {
  arch=$(uname -m)

  case "$arch" in
    x86_64|amd64)
      printf 'amd64\n'
      ;;
    arm64|aarch64)
      printf 'arm64\n'
      ;;
    *)
      die "unsupported architecture: $arch"
      ;;
  esac
}

latest_tag() {
  http_get "https://api.github.com/repos/${repo}/releases/latest" \
    | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' \
    | head -n 1
}

resolve_tag() {
  if [ -z "$version" ]; then
    tag=$(latest_tag)
    [ -n "$tag" ] || die "could not determine latest release for ${repo}"
    printf '%s\n' "$tag"
    return
  fi

  case "$version" in
    v*)
      printf '%s\n' "$version"
      ;;
    *)
      printf 'v%s\n' "$version"
      ;;
  esac
}

verify_checksum() {
  archive=$1
  checksum_file=$2

  if ! command -v awk >/dev/null 2>&1; then
    printf 'install.sh: warning: awk not found, skipping checksum verification\n' >&2
    return
  fi

  expected=$(awk -v file="$archive" '$2 == file { print $1 }' "$checksum_file" | head -n 1)
  if [ -z "$expected" ]; then
    printf 'install.sh: warning: checksum for %s not found, skipping verification\n' "$archive" >&2
    return
  fi

  if command -v sha256sum >/dev/null 2>&1; then
    actual=$(sha256sum "${tmpdir}/${archive}" | awk '{ print $1 }')
  elif command -v shasum >/dev/null 2>&1; then
    actual=$(shasum -a 256 "${tmpdir}/${archive}" | awk '{ print $1 }')
  else
    printf 'install.sh: warning: sha256sum or shasum not found, skipping checksum verification\n' >&2
    return
  fi

  [ "$expected" = "$actual" ] || die "checksum mismatch for ${archive}"
}

copy_binary() {
  src=$1
  dst=$2

  if command -v install >/dev/null 2>&1; then
    install -m 0755 "$src" "$dst"
    return
  fi

  cp "$src" "$dst"
  chmod 0755 "$dst"
}

sudo_copy_binary() {
  src=$1
  dst=$2

  command -v sudo >/dev/null 2>&1 || die "cannot write to install directory and sudo is not available"

  if command -v install >/dev/null 2>&1; then
    sudo install -m 0755 "$src" "$dst"
    return
  fi

  sudo cp "$src" "$dst"
  sudo chmod 0755 "$dst"
}

default_install_dir() {
  if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
    printf '/usr/local/bin\n'
    return
  fi

  if command -v sudo >/dev/null 2>&1; then
    printf '/usr/local/bin\n'
    return
  fi

  [ -n "${HOME:-}" ] || die "HOME is required when sudo is unavailable"
  printf '%s/.local/bin\n' "$HOME"
}

install_binary() {
  src=$1
  dir=$(default_install_dir)
  target="${dir}/${bin}"

  if mkdir -p "$dir" 2>/dev/null && [ -w "$dir" ]; then
    copy_binary "$src" "$target"
  else
    command -v sudo >/dev/null 2>&1 || die "cannot write to ${dir}"
    sudo mkdir -p "$dir"
    sudo_copy_binary "$src" "$target"
  fi

  printf 'installed %s to %s\n' "$bin" "$target"

  case ":${PATH:-}:" in
    *":${dir}:"*) ;;
    *) printf 'note: %s is not on PATH\n' "$dir" ;;
  esac
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      shift
      [ "$#" -gt 0 ] || die "--version requires a value"
      version=$1
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "unknown argument: $1"
      ;;
  esac
  shift
done

need_os=$(detect_os)
need_arch=$(detect_arch)
tag=$(resolve_tag)
artifact_version=${tag#v}
archive="${bin}_${artifact_version}_${need_os}_${need_arch}.tar.gz"
base_url="https://github.com/${repo}/releases/download/${tag}"

tmpdir=$(mktemp -d "${TMPDIR:-/tmp}/konfi-install.XXXXXX")
trap cleanup EXIT INT TERM

printf 'downloading %s\n' "$archive"
http_download "${base_url}/${archive}" "${tmpdir}/${archive}"

if http_download "${base_url}/checksums.txt" "${tmpdir}/checksums.txt"; then
  verify_checksum "$archive" "${tmpdir}/checksums.txt"
else
  printf 'install.sh: warning: could not download checksums.txt, skipping checksum verification\n' >&2
fi

tar -xzf "${tmpdir}/${archive}" -C "$tmpdir"
[ -f "${tmpdir}/${bin}" ] || die "${bin} not found in ${archive}"

install_binary "${tmpdir}/${bin}"
