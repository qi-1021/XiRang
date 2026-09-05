package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// 默认未加密时的占位，由 builder 动态覆盖
var (
	EncryptedPayload = ""
	BuildSecret      = ""
)

// 解密并还原混淆配置
func loadEmbeddedProtectedConfig() *Config {
	if EncryptedPayload == "" || BuildSecret == "" {
		return nil
	}

	cipherBytes, err := hex.DecodeString(EncryptedPayload)
	if err != nil {
		return nil
	}

	keyHash := sha256.Sum256([]byte(BuildSecret))
	block, err := aes.NewCipher(keyHash[:])
	if err != nil {
		return nil
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil
	}

	nonceSize := gcm.NonceSize()
	if len(cipherBytes) < nonceSize {
		return nil
	}

	nonce, ciphertext := cipherBytes[:nonceSize], cipherBytes[nonceSize:]
	plainText, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil
	}

	var cfg Config
	if err := json.Unmarshal(plainText, &cfg); err != nil {
		return nil
	}
	return &cfg
}
