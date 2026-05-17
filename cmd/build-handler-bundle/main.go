// Command build-handler-bundle pre-builds the Apono Connect.app handler bundle
// for inclusion in the Homebrew tarball. Invoked from goreleaser's before
// hooks during release. The resulting bundle is shipped to users and
// self-installs into ~/Library/Application Support/apono-cli/ when brew's
// post_install opens it via `open -a`.
package main

import (
	"fmt"
	"os"

	"github.com/apono-io/apono-cli/pkg/urihandler"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: build-handler-bundle <output-path>")
		os.Exit(2)
	}
	if err := urihandler.BuildBundleAt(os.Args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "build-handler-bundle: %v\n", err)
		os.Exit(1)
	}
}
