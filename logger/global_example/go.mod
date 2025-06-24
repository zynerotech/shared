module global_example

go 1.24.2

require gitlab.com/zynero/shared/logger v0.1.8

require (
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
)

replace gitlab.com/zynero/shared/logger => ../
