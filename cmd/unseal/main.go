// Command unseal is the entry point of the Unseal desktop application.
//
// Unseal is a standalone audiobook downloader and DRM remover for a personal
// Audible library. It signs the user in through the browser, registers itself
// as an Audible device, lists the purchased library, downloads the encrypted
// payload, removes the DRM with the account-scoped keys Audible issued for
// that account, and writes plain .m4b files.
//
// Usage:
//
//	unseal [command] [flags]
//
// Run "unseal help" for the list of commands.
package main

import (
	"fmt"
	"os"

	"github.com/ngbcoder/Unseal/internal/version"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "unseal: %v\n", err)
		os.Exit(1)
	}
}

// run is the testable entry point. It dispatches to the CLI parser once the
// command tree exists (milestone M7); until then it only answers the
// informational commands.
func run(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "version", "--version", "-v":
			fmt.Println(version.Current().String())
			return nil
		case "help", "--help", "-h":
			printUsage()
			return nil
		}
	}

	printUsage()
	return nil
}

func printUsage() {
	fmt.Printf("Unseal %s\n\n", version.Current().Version)
	fmt.Println("A standalone audiobook downloader and DRM remover.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  unseal [command] [flags]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  run        Start the desktop application")
	fmt.Println("  register   Register a new Audible device profile")
	fmt.Println("  library    List the library of the selected profile")
	fmt.Println("  download   Download and decrypt selected titles")
	fmt.Println("  export     Export decrypted titles to .aax plus sidecars")
	fmt.Println("  config     Show or edit settings")
	fmt.Println("  profiles   Manage registered device profiles")
	fmt.Println("  doctor     Check the environment and dependencies")
	fmt.Println("  version    Print version information")
}
