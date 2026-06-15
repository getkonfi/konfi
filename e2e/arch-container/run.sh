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
	--user "$uid:$gid" \
	-e HOME=/tmp/konfi-home \
	-e GOCACHE=/tmp/konfi-go-cache \
	-e GOMODCACHE=/tmp/konfi-go-mod-cache \
	-e XDG_CONFIG_HOME=/tmp/konfi-xdg \
	-v "$root:/work:ro" \
	-w /work/src \
	"$image" \
	sh -lc 'mkdir -p "$HOME" "$GOCACHE" "$GOMODCACHE" "$XDG_CONFIG_HOME" && go test -count=1 -tags=container_e2e -run TestContainerE2E ./konfables'
