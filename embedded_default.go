package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
)

// 默认未加密时的占位，由 builder 动态覆盖
var (
	EncryptedPayload = ""
	BuildSecret      = ""
)

// resolveBuildSecret: 优先使用运行时环境变量，避免只能依赖二进制内嵌明文盐
// 说明：即使内嵌 BuildSecret，也只是混淆而非真正的分发级密钥保护。
func resolveBuildSecret() string {
	if env := os.Getenv("XIRANG_BUILD_SECRET"); env != "" {
		return env
	}
	return BuildSecret
}

// 解密并还原混淆配置
func loadEmbeddedProtectedConfig() *Config {
	secret := resolveBuildSecret()
	if EncryptedPayload == "" || secret == "" {
		return nil
	}

	cipherBytes, err := hex.DecodeString(EncryptedPayload)
	if err != nil {
		return nil
	}

	keyHash := sha256.Sum256([]byte(secret))
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
