package gitssh_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/andrewslotin/doppelganger/git/gitssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthorizedRSAKey(t *testing.T) {
	pkey, err := rsa.GenerateKey(rand.Reader, 512)
	require.NoError(t, err)

	authKey, err := gitssh.AuthorizedRSAKey(pkey)
	require.NoError(t, err)

	pubkey, _, _, _, err := ssh.ParseAuthorizedKey(authKey)
	require.NoError(t, err)

	ok, err := testValidRSAKeyPair(pkey, pubkey)
	require.NoError(t, err)
	assert.True(t, ok)
}
