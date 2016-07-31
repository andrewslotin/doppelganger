package ssh

import (
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
