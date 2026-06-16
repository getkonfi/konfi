module github.com/getkonfi/konfi/tools/upstreamcheck

go 1.26

require (
	github.com/getkonfi/konfi v0.0.0
	golang.org/x/mod v0.37.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
)

replace github.com/getkonfi/konfi => ../../src
