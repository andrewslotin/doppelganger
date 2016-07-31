package ssh_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/andrewslotin/doppelganger/git/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	crypto_ssh "golang.org/x/crypto/ssh"
)

func TestAuthorizedRSAKey(t *testing.T) {
	pkey, err := rsa.GenerateKey(rand.Reader, 512)
	require.NoError(t, err)

	authKey, err := ssh.AuthorizedRSAKey(pkey)
	require.NoError(t, err)

	pubkey, _, _, _, err := crypto_ssh.ParseAuthorizedKey(authKey)
	require.NoError(t, err)

	ok, err := testValidRSAKeyPair(pkey, pubkey)
	require.NoError(t, err)
	assert.True(t, ok)
}
