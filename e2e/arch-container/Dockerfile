FROM archlinux:base-devel

RUN pacman -Syu --noconfirm \
	&& pacman -S --needed --noconfirm \
		alacritty \
		ca-certificates \
		dbus \
		dconf \
		fuzzel \
		git \
		glib2 \
		ghostty \
		go \
		gsettings-desktop-schemas \
		helix \
		make \
		openssh \
		otf-atkinsonhyperlegiblemono-nerd \
		pacman-contrib \
		starship \
		tmux \
		yazi \
		zsh \
	&& pacman -Scc --noconfirm

WORKDIR /work/src
