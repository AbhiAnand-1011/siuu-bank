package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAccount(t *testing.T) {
	password := "siuu"

	acc, err := NewAccount("abhi", "anand", password)
	assert.NoError(t, err)
	assert.NotNil(t, acc)

	assert.NotZero(t, acc.Number, "account number should be generated")
	assert.NotEmpty(t, acc.EncryptedPassword, "password should be encrypted")
	assert.NotEqual(t, password, acc.EncryptedPassword, "password must not be stored in plaintext")
	assert.False(t, acc.CreatedAt.IsZero(), "createdAt should be set")

	assert.True(t, acc.ValidPassword(password), "valid password should authenticate")
	assert.False(t, acc.ValidPassword("wrong-password"), "invalid password should not authenticate")
}
