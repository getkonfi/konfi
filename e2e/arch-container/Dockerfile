FROM archlinux:base-devel

RUN pacman -Syu --noconfirm \
	&& pacman -S --needed --noconfirm ca-certificates git go make \
	&& pacman -Scc --noconfirm

WORKDIR /work/src
