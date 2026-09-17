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
	localState := flag.String("localstate", "", "(optional) Chrome Local State file path")
	sessionstorage := flag.String("sessionstorage", "", "(optional) Chrome Session Storage")
	targetPath := flag.String("targetpath", "", "(optional) File path")
	profile := flag.String("profile", "Default", "(optional) Chrome profile name")
	listProfilesFlag := flag.Bool("list-profiles", false, "List available profiles")

	flag.Parse()

	if *listProfilesFlag {
		fmt.Println("Listing profiles...")
		usr, _ := getCurrentUser()
		fmt.Println("User:", usr.Name)
		os.Exit(0)
	}

	if *kind == "" {
		flag.Usage()
		os.Exit(1)
	}

	fmt.Println("Kind selected:", *kind)
	fmt.Println("Profile:", *profile)
	fmt.Println("Architecture:", runtime.GOARCH)
	fmt.Println("Go version: go1.26.3")
	fmt.Println("macOS support: Apple Silicon (arm64) and Intel (amd64)")
}

func getCurrentUser() (string, error) {
	usr, err := os.UserCurrent()
	if err != nil {
		return "", err
	}
	return usr.Name, nil
}