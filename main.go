// Command pulselink is the repository-root signpost entrypoint.
//
// Go's internal-package rule stops this root-level file from importing
// apps/desktop/internal, so the real entrypoints live under apps/desktop:
//
//	go build -tags wails -o pulselink.exe ./apps/desktop   # native Wails app
//	go run ./apps/desktop/cmd/pulselinkd                    # headless backend
package main

import "fmt"

func main() {
	fmt.Println("PulseLink native app:  go build -tags wails ./apps/desktop")
	fmt.Println("PulseLink backend only: go run ./apps/desktop/cmd/pulselinkd")
}
