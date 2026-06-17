#!/bin/sh
set -eu

repo="${KONFI_REPO:-getkonfi/konfi}"
bin="${KONFI_BIN:-konfi}"
version="${KONFI_VERSION:-}"
tmpdir=""
color_enabled=0
c_reset=""
c_dim=""
c_blue=""
c_green=""
c_yellow=""
c_red=""

setup_color() {
  case "${KONFI_COLOR:-auto}" in
    always|1|true|yes)
      color_enabled=1
      ;;
    never|0|false|no)
      color_enabled=0
      ;;
    *)
      if [ -t 2 ] && [ -z "${NO_COLOR:-}" ] && [ "${TERM:-}" != "dumb" ]; then
        color_enabled=1
      fi
      ;;
  esac

  if [ "$color_enabled" -eq 1 ]; then
    c_reset=$(printf '\033[0m')
    c_dim=$(printf '\033[2m')
    c_blue=$(printf '\033[34m')
    c_green=$(printf '\033[32m')
    c_yellow=$(printf '\033[33m')
    c_red=$(printf '\033[31m')
  fi
}

log_line() {
  color=$1
  label=$2
  shift 2

  if [ "$color_enabled" -eq 1 ]; then
    printf '%b%-7s%b %s\n' "$color" "$label" "$c_reset" "$*" >&2
  else
    printf '%-7s %s\n' "$label" "$*" >&2
  fi
}

detail() {
  if [ "$color_enabled" -eq 1 ]; then
    printf '        %b%s%b\n' "$c_dim" "$*" "$c_reset" >&2
  else
    printf '        %s\n' "$*" >&2
  fi
}

step() {
  log_line "$c_blue" "info" "$*"
}

success() {
  log_line "$c_green" "ok" "$*"
}

warn() {
  log_line "$c_yellow" "warn" "$*"
}

error() {
  log_line "$c_red" "error" "$*"
}

usage() {
  cat <<EOF
usage: install.sh [--version VERSION]

options:
  --version VERSION  install a specific release version or tag

environment:
  KONFI_VERSION      release version or tag, defaults to latest
  KONFI_REPO         GitHub repo, defaults to getkonfi/konfi
  KONFI_COLOR        auto, always, or never
  KONFI_SKIP_CHECKSUM  set to 1 to install without checksum verification
EOF
}

die() {
  error "$*"
  exit 1
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

release_asset_names() {
  release_tag=$1
  release_json=$(http_get "https://api.github.com/repos/${repo}/releases/tags/${release_tag}") || return 1

  printf '%s\n' "$release_json" \
    | sed -n 's|.*"browser_download_url"[[:space:]]*:[[:space:]]*"[^"]*/\([^/"]*\)".*|\1|p'
}

confirm_skip_checksum() {
  reason=$1

  warn "$reason"
  case "${KONFI_SKIP_CHECKSUM:-}" in
    1|true|TRUE|yes|YES)
      warn "continuing without checksum verification"
      return
      ;;
    0|false|FALSE|no|NO)
      die "checksum verification was not completed"
      ;;
  esac

  if [ ! -r /dev/tty ] || [ ! -w /dev/tty ]; then
    detail "set KONFI_SKIP_CHECKSUM=1 to install without checksum verification"
    die "checksum verification was not completed"
  fi

  if [ "$color_enabled" -eq 1 ]; then
    printf '%b%-7s%b continue without checksum verification? [y/N] ' "$c_yellow" "confirm" "$c_reset" >/dev/tty
  else
    printf '%-7s continue without checksum verification? [y/N] ' "confirm" >/dev/tty
  fi

  answer=""
  IFS= read -r answer </dev/tty || die "could not read confirmation"
  case "$answer" in
    y|Y|yes|YES)
      warn "continuing without checksum verification"
      ;;
    *)
      die "checksum verification was not completed"
      ;;
  esac
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
    confirm_skip_checksum "awk not found; checksum verification cannot run"
    return
  fi

  expected=$(awk -v file="$archive" '$2 == file { print $1 }' "$checksum_file" | head -n 1)
  if [ -z "$expected" ]; then
    confirm_skip_checksum "checksum for ${archive} not found"
    return
  fi

  if command -v sha256sum >/dev/null 2>&1; then
    actual=$(sha256sum "${tmpdir}/${archive}" | awk '{ print $1 }')
  elif command -v shasum >/dev/null 2>&1; then
    actual=$(shasum -a 256 "${tmpdir}/${archive}" | awk '{ print $1 }')
  else
    confirm_skip_checksum "sha256sum or shasum not found; checksum verification cannot run"
    return
  fi

  [ "$expected" = "$actual" ] || die "checksum mismatch for ${archive}"
  success "verified checksum"
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

  step "installing ${bin}"
  detail "$target"

  if mkdir -p "$dir" 2>/dev/null && [ -w "$dir" ]; then
    copy_binary "$src" "$target"
  else
    command -v sudo >/dev/null 2>&1 || die "cannot write to ${dir}"
    sudo mkdir -p "$dir"
    sudo_copy_binary "$src" "$target"
  fi

  success "installed ${bin}"

  case ":${PATH:-}:" in
    *":${dir}:"*) ;;
    *) warn "${dir} is not on PATH" ;;
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

setup_color

step "detecting platform"
need_os=$(detect_os)
need_arch=$(detect_arch)
detail "${need_os}/${need_arch}"

if [ -z "$version" ]; then
  step "resolving latest release"
else
  step "using requested release"
  detail "$version"
fi
tag=$(resolve_tag)
success "selected ${tag}"

artifact_version=${tag#v}
archive="${bin}_${artifact_version}_${need_os}_${need_arch}.tar.gz"
base_url="https://github.com/${repo}/releases/download/${tag}"

tmpdir=$(mktemp -d "${TMPDIR:-/tmp}/konfi-install.XXXXXX")
trap cleanup EXIT INT TERM

step "checking release assets"
if assets=$(release_asset_names "$tag"); then
  if ! printf '%s\n' "$assets" | grep -qx "$archive"; then
    if [ -z "$assets" ]; then
      error "release ${tag} has no downloadable assets"
    else
      error "release ${tag} does not contain ${archive}"
      detail "available assets:"
      printf '%s\n' "$assets" | sed 's/^/        /' >&2
    fi
    detail "expected ${archive}"
    detail "release https://github.com/${repo}/releases/tag/${tag}"
    die "release artifacts are missing; rerun the release upload or try again after it finishes"
  fi
  success "found ${archive}"
else
  warn "could not inspect release assets; trying direct download"
fi

step "downloading ${archive}"
detail "${base_url}/${archive}"
if http_download "${base_url}/${archive}" "${tmpdir}/${archive}"; then
  success "downloaded ${archive}"
else
  die "download failed for ${archive}"
fi

step "downloading checksums.txt"
if http_download "${base_url}/checksums.txt" "${tmpdir}/checksums.txt"; then
  step "verifying checksum"
  verify_checksum "$archive" "${tmpdir}/checksums.txt"
else
  confirm_skip_checksum "could not download checksums.txt"
fi

step "extracting ${archive}"
tar -xzf "${tmpdir}/${archive}" -C "$tmpdir"
[ -f "${tmpdir}/${bin}" ] || die "${bin} not found in ${archive}"
success "extracted ${bin}"

install_binary "${tmpdir}/${bin}"
