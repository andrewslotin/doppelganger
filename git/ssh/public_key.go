package ssh

import (
	"crypto/rsa"
	"fmt"

	"golang.org/x/crypto/ssh"
)

// AuthorizedRSAKey returns RSA public key for given private key in a format that is suitable
// for authorized_keys file.
func AuthorizedRSAKey(pkey *rsa.PrivateKey) ([]byte, error) {
	pubkey, err := ssh.NewPublicKey(&pkey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create public key: %s", err)
	}

	return ssh.MarshalAuthorizedKey(pubkey), nil
}
