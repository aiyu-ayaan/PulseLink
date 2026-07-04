//go:build !wails

// This stub stands in for the Wails desktop app (main.go, //go:build wails) so
// the module builds and tests cleanly without the Wails v3 dependency. Build
// the real native app with `-tags wails` after installing the toolchain — see
// docs/desktop-app.md.
package main

import "fmt"

func main() {
	fmt.Println("PulseLink desktop app is built with the `wails` tag:")
	fmt.Println("  go build -tags wails -o pulselink.exe ./apps/desktop")
	fmt.Println("Headless backend only: go run ./apps/desktop/cmd/pulselinkd")
}
