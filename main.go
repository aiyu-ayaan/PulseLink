// Command pulselink is the repository-root placeholder entrypoint.
//
// Stage 1 (current): the backend runs headlessly from
// ./apps/desktop/cmd/pulselinkd. Go's internal-package rule stops this
// root-level file from importing apps/desktop/internal, so it only points at
// the daemon. Stage 2 replaces this file with the Wails desktop application,
// whose module root will live under apps/desktop and can import internal/app.
package main

import "fmt"

func main() {
	fmt.Println("PulseLink backend: run `go run ./apps/desktop/cmd/pulselinkd`")
}
