package models

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type APIKey struct {
	gorm.Model `json:"-"`
	Name       string // Descriptive name like "merch-app-scanner-1"
	KeyHash    string `json:"-"` // bcrypt hash of the API key
	KeyID      string // UUID for identifying the key
	Active     bool
	Purpose    string // "scanner" or "ordering"
}

// GenerateAPIKey creates a new API key with a random 32-byte key
// Returns the plaintext key (only time it's visible) and the APIKey struct
func GenerateAPIKey(name, purpose string) (string, *APIKey) {
	// Generate random 32-byte key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		panic("unable to generate random key")
	}

	// Encode to base64 for transport
	plaintextKey := base64.URLEncoding.EncodeToString(keyBytes)

	// Hash the key for storage
	keyHash, err := bcrypt.GenerateFromPassword([]byte(plaintextKey), 14)
	if err != nil {
		panic("unable to hash API key")
	}

	apiKey := &APIKey{
		Name:    name,
		KeyHash: string(keyHash),
		KeyID:   uuid.New().String(),
		Active:  true,
		Purpose: purpose,
	}

	return plaintextKey, apiKey
}

// ValidateKey checks if the provided plaintext key matches the hash
func (k *APIKey) ValidateKey(plaintextKey string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(k.KeyHash), []byte(plaintextKey))
	return err == nil
}
