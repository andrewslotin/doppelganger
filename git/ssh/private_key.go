package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
)

const (
	sshKeySize       = 1024
	pemRSAPKeyHeader = "RSA PRIVATE KEY"
)

// ReadPrivateRSAKey reads an existing RSA private key from PKCS#1 PEM file specified by `path`.
func ReadPrivateRSAKey(path string) (*rsa.PrivateKey, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %s", err)
	}
	defer fd.Close()

	pkeyPEM, err := ioutil.ReadAll(fd)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %s", err)
	}

	var b *pem.Block
	for pkeyPEM != nil {
		b, pkeyPEM = pem.Decode(pkeyPEM)
		if b == nil {
			break
		}

		if b.Type == pemRSAPKeyHeader {
			pkey, err := x509.ParsePKCS1PrivateKey(b.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse private key in %s: %s", path, err)
			}

			return pkey, nil
		}
	}

	return nil, errors.New("no private key")
}

// CreatePrivateRSAKey generates a 1024-bit RSA key and stores it to `path` in PEM PKCS#1 format.
func CreatePrivateRSAKey(path string) (*rsa.PrivateKey, error) {
	fd, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create private key file: %s", err)
	}
	defer fd.Close()

	pkey, err := rsa.GenerateKey(rand.Reader, sshKeySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %s", err)
	}

	if err := pem.Encode(fd, &pem.Block{Type: pemRSAPKeyHeader, Bytes: x509.MarshalPKCS1PrivateKey(pkey)}); err != nil {
		return nil, fmt.Errorf("failed to write generated private key to %s: %s", path, err)
	}

	return pkey, nil
}
