#!/usr/bin/env sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
image=${KONFI_CONTAINER_E2E_IMAGE:-konfi-container-e2e:latest}
runtime=${CONTAINER_RUNTIME:-}

if [ -z "$runtime" ]; then
	if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
		runtime=docker
	elif command -v podman >/dev/null 2>&1 && podman info >/dev/null 2>&1; then
		runtime=podman
	else
		echo "a running docker or podman runtime is required for make e2e" >&2
		exit 127
	fi
fi

uid=${KONFI_CONTAINER_E2E_UID:-$(id -u)}
gid=${KONFI_CONTAINER_E2E_GID:-$(id -g)}

set --
if [ "$runtime" = "podman" ]; then
	set -- --network host
fi

"$runtime" build \
	-f "$root/e2e/arch-container/Dockerfile" \
	-t "$image" \
	"$root"

"$runtime" run --rm \
	"$@" \
	-e HOME=/tmp/konfi-home \
	-e GOCACHE=/tmp/konfi-go-cache \
	-e GOMODCACHE=/tmp/konfi-go-mod-cache \
	-e KONFI_CONTAINER_E2E_UID="$uid" \
	-e KONFI_CONTAINER_E2E_GID="$gid" \
	-e XDG_CONFIG_HOME=/tmp/konfi-xdg \
	-v "$root:/work:ro" \
	-w /work/src \
	"$image" \
	sh -lc '
set -eu

uid=${KONFI_CONTAINER_E2E_UID:-1000}
gid=${KONFI_CONTAINER_E2E_GID:-1000}

group_name=$(getent group "$gid" | cut -d: -f1 || true)
if [ -z "$group_name" ]; then
	group_name=konfi
	groupadd -g "$gid" "$group_name"
fi
if ! getent passwd "$uid" >/dev/null; then
	useradd -u "$uid" -g "$gid" -d "$HOME" -s /bin/sh konfi
fi

mkdir -p "$HOME" "$GOCACHE" "$GOMODCACHE" "$XDG_CONFIG_HOME"
chown -R "$uid:$gid" "$HOME" "$GOCACHE" "$GOMODCACHE" "$XDG_CONFIG_HOME"

exec setpriv --reuid "$uid" --regid "$gid" --init-groups \
	sh -lc "go test -count=1 -tags=container_e2e -run TestContainerE2E ./konfables"
'
