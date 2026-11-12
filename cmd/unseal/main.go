package main

import (
	"fmt"
	"os"

	"github.com/ngbcoder/Unseal/internal/ui"
	"github.com/ngbcoder/Unseal/internal/version"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "unseal: %v\n", err)
		os.Exit(1)
	}
}

// run dispatches to the command tree.
func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "run":
		return ui.Run()
	case "register":
		return notYetImplemented("register")
	case "library":
		return notYetImplemented("library")
	case "download":
		return notYetImplemented("download")
	case "export":
		return notYetImplemented("export")
	case "config":
		return notYetImplemented("config")
	case "profiles":
		return notYetImplemented("profiles")
	case "doctor":
		return doctor()
	case "version", "--version", "-v":
		fmt.Println(version.Current().Detail())
		return nil
	case "help", "--help", "-h":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

// notYetImplemented reports that a command exists but is not wired up yet.
func notYetImplemented(name string) error {
	return fmt.Errorf("%s is not implemented yet", name)
}

// doctor checks the environment and reports what it finds.
func doctor() error {
	fmt.Printf("Unseal %s\n", version.Current().Version)
	fmt.Printf("  go      %s\n", version.Current().GoVersion)
	fmt.Printf("  platform %s\n", version.Current().Platform)
	fmt.Println()
	fmt.Println("Dependencies:")
	fmt.Println("  ffmpeg: (not checked yet)")
	fmt.Println()
	fmt.Println("Directories:")
	fmt.Println("  (not resolved yet)")
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
	fmt.Println()
	fmt.Println("Global flags:")
	fmt.Println("  --log <n>     Set the log level (0-5)")
	fmt.Println("  --config <p>  Use an alternate configuration directory")
	fmt.Println("  --profile <a> Select an account profile by alias")
}
