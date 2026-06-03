module github.com/eminert/konfi/tools/upstreamcheck

go 1.26

require (
	github.com/eminert/konfi v0.0.0
	golang.org/x/mod v0.36.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
)

replace github.com/eminert/konfi => ../../src
