package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
)

func main() {
	if runtime.GOARCH != "arm64" && runtime.GOARCH != "amd64" {
		log.Fatal("This tool only runs on macOS (Apple Silicon or Intel)")
	}

	flag.Parse()

	kind := flag.String("kind", "", "cookie, logindata, creditcard, history, or extension")
	profile := flag.String("profile", "Default", "(optional) Chrome profile name")
	listProfilesFlag := flag.Bool("list-profiles", false, "List available profiles")

	flag.Parse()

	if *listProfilesFlag {
		fmt.Println("Profiles mode - simple check")
		os.Exit(0)
	}

	if *kind == "" {
		flag.Usage()
		os.Exit(1)
	}

	fmt.Println("Kind:", *kind)
	fmt.Println("Profile:", *profile)
	fmt.Println("Architecture:", runtime.GOARCH)
	fmt.Println("Go version: go1.26.3 - macOS arm64/amd64 supported")
}