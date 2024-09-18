package encryption

import (
	"crypto/sha256"
	"io"
	"log"
	"os"
)

func GetFingerprint(certPath string) string {
	
	file, err := os.Open(certPath)
	if err != nil {
		log.Fatal("Error opening tls certificate", err)
	}
	// buf, err := io.ReadAll(file)
	// if err != nil {
	// 	log.Fatalf("Error reading tls certificate from %s: %v", certPath, err)
	// }
	
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		log.Fatal(err)
	}
	fingerprint := h.Sum(nil)

	return string(fingerprint)
}

func checkFile(path string) bool {
	_, err := os.Open(path)
	if err != nil {
		log.Printf("Could not open %s: %v\n", path, err)
		return false
	}
	return true
}
// return true if the given paths point to 
func CheckTLS(certPath string, privKeyPath string) bool {
	return checkFile(certPath) && checkFile(privKeyPath)
}

func SetupTLSCerts(certPath string, privKeyPath string) {
	// TODO: make self signed tls cert setup a standalone module
}
