module github.com/witsaba/paca/apps/cli

go 1.26.3

require (
	github.com/spf13/cobra v1.10.2
	github.com/witsaba/paca/packages/core v0.0.0
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
)

// The shared core lives in the repository, next to this module. The replace
// directive keeps this module buildable on its own, outside the workspace.
replace github.com/witsaba/paca/packages/core => ../../packages/core
