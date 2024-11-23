package encryption

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"path/filepath"
	"github.com/atomic-7/gocalsend/internal/data"
	"io"
	"log/slog"
	"math/big"
	"os"
	"time"
)

func GetFingerPrint(paths *data.TLSPaths) (string, error) {

	file, err := os.Open(paths.Cert)
	if err != nil {
		slog.Error("failed to open cert", slog.Any("error", err))
		return "", err
	}
	contents, err := io.ReadAll(file)
	if err != nil {
		slog.Error("failed to read cert from disk", slog.Any("error", err))
		return "", err
	}
	fingerprint := sha256.Sum256(contents)
	return hex.EncodeToString(fingerprint[:]), nil
}

func checkFile(path string) bool {
	_, err := os.Open(path)
	if err != nil {
		slog.Debug("could not open file", slog.String("file", path), slog.Any("error", err))
		return false
	}
	return true
}

// return true if the given paths point to
func CheckTLSFiles(certPath string, privKeyPath string) bool {
	return checkFile(certPath) && checkFile(privKeyPath)
}

func SetupTLSCerts(alias string, paths *data.TLSPaths) error {

	if paths.Key == "" && paths.Cert == "" {
		paths.Key = filepath.Join(paths.Dir, "key.pem")
		paths.Cert = filepath.Join(paths.Dir, "cert.pem")
	}
	// TODO: Expand this to reuse an existing private key
	if !(checkFile(paths.Key) && checkFile(paths.Cert)) {
		slog.Debug("unable to find existing tls cert, generating new cert and key",
			slog.String("dir", paths.Dir),
			slog.String("cert", paths.Cert),
			slog.String("key", paths.Key),
		)
		os.MkdirAll(paths.Dir, 0700)
		cred, err := createCert(nil, alias, "localhost")
		if err != nil {
			return err
		}
		cred.WriteCredentials(paths)
	} else {
		slog.Debug("found existing cert and key")
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
func createCert(pk *rsa.PrivateKey, org string, dnsname string) (*Credentials, error) {

	var privateKey *rsa.PrivateKey
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
		privateKey, err = rsa.GenerateKey(rand.Reader, 2045)
		if err != nil {
			slog.Error("failed to generate private key", slog.Any("error", err))
			os.Exit(1)
		}
	}

	maxSerialNumber := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, maxSerialNumber)
	if err != nil {
		slog.Error("Failed to generate serial number", slog.Any("error", err))
		os.Exit(1)
	}

	// Use a certificate template to construct the cert
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{org},
		},
		// maybe consider specifying URIs here as well
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
		slog.Error("failed to create certificate", slog.Any("error", err))
		os.Exit(1)
	}

	// Certificate
	pemCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if pemCert == nil {
		slog.Error("failed to encode certificate to pem")
		os.Exit(1)
	}

	// Private Key
	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		slog.Error("unable to marshal private key", slog.Any("error", err))
		os.Exit(1)
	}
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	if pemKey == nil {
		slog.Error("failed to encode private key to pem", slog.Any("error", err))
		os.Exit(1)
	}

	return &Credentials{
		Cert: pemCert,
		Key:  pemKey,
	}, nil
}

// Write the credentials to the supplied paths on disk.
// Throws an error if writing any of the files failed
func (creds *Credentials) WriteCredentials(paths *data.TLSPaths) error {

	err := os.WriteFile(paths.Cert, creds.Cert, 0644)
	if err != nil {
		slog.Error("error writing certificate to disk", slog.Any("error", err))
		return err
	}
	// keep the private key private!
	err = os.WriteFile(paths.Key, creds.Key, 0600)
	if err != nil {
		slog.Error("error writing private key to disk", slog.Any("error", err))
		return err
	}

	return nil
}
