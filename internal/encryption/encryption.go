package encryption

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"github.com/atomic-7/gocalsend/internal/data"
	"io"
	"log"
	"math/big"
	"os"
	"time"
)

func calcFingerPrint(reader io.Reader) string {

	// the hash does not seem to be correct
	h := sha256.New()
	if _, err := io.Copy(h, reader); err != nil {
		log.Fatal(err)
	}
	fingerprint := h.Sum(nil)

	return string(fingerprint)
}

func GetFingerPrint(paths *data.TLSPaths) (string, error) {

	file, err := os.Open(paths.CertPath)
	if err != nil {
		log.Println("Error opening tls certificate", err)
		return "", err
	}
	return calcFingerPrint(file), nil
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

func SetupTLSCerts(alias string, paths *data.TLSPaths) error {

	log.Println("Generating tls cert in %v", paths)
	// TODO: Expand this to reuse an existing private key
	if !(checkFile(paths.KeyPath) && checkFile(paths.CertPath)) {
		log.Println("Could not find existing certificate and private key, generating a new one")
		os.MkdirAll(paths.Dir, 0700)
		cred, err := createCert(nil, alias, "localhost")
		if err != nil {
			return err
		}
		cred.WriteCredentials(paths)
	} else {
		log.Println("Found existing key and certificate!")
	}

	return nil
}

type Credentials struct {
	Cert []byte
	Key  []byte
}

// Generate a self signed certificate with the given private key. If key is nil, a new one is generated
// based on https://eli.thegreenplace.net/2021/go-https-servers-with-tls/
// as its own module in github.com/atomic-7/goncert/goncert
func createCert(pk *ecdsa.PrivateKey, org string, dnsname string) (*Credentials, error) {

	var privateKey *ecdsa.PrivateKey
	var err error
	if org == "" {
		return nil, errors.New("Missing parameter: organization cannot be empty string")
	}
	if org == "" {
		return nil, errors.New("Missing parameter: dnsname cannot be empty string")
	}
	if pk != nil {
		privateKey = pk
	} else {
		// P256 is allowed for TLS1.3
		privateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			log.Fatalf("Failed to generate private key: %v", err)
		}
	}

	maxSerialNumber := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, maxSerialNumber)
	if err != nil {
		log.Fatalf("Failed to generate serial number: %v", err)
	}

	// Use a certificate template to construct the cert
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{org},
		},
		DNSNames:  []string{dnsname}, // this might be optional
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(24 * time.Hour),

		KeyUsage: x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth,
		}, // can also include ExtKeyUsageAny
		BasicConstraintsValid: true,
	}

	// the certificate in DER encoding
	// passing the same template for both the template and the parent makes this a self signed certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		log.Fatalf("Failed to create certificate: %v", err)
	}

	// Certificate
	pemCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if pemCert == nil {
		log.Fatal("Failed to encode certificate to PEM")
	}

	// Private Key
	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		log.Fatalf("Unable to marshal private key: %v")
	}
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	if pemKey == nil {
		log.Fatal("Failed to encode private key to PEM")
	}

	return &Credentials{
		Cert: pemCert,
		Key:  pemKey,
	}, nil
}

// Write the credentials to the supplied paths on disk.
// Throws an error if writing any of the files failed
func (creds *Credentials) WriteCredentials(paths *data.TLSPaths) error {

	err := os.WriteFile(paths.CertPath, creds.Cert, 0644)
	if err != nil {
		log.Printf("Error writing certificate to disk: %v", err)
		return err
	}
	// keep the private key private!
	err = os.WriteFile(paths.KeyPath, creds.Key, 0600)
	if err != nil {
		log.Printf("Error writing private key to disk: %v", err)
		return err
	}

	return nil
}
