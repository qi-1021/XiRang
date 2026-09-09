package main

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestReadLine(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		prompt         string
		defaultValue   string
		expectedResult string
		expectedPrompt string
	}{
		{
			name:           "User provides custom input",
			input:          "custom_val\n",
			prompt:         "Enter value",
			defaultValue:   "default_val",
			expectedResult: "custom_val",
			expectedPrompt: "Enter value [default_val]: ",
		},
		{
			name:           "User provides empty input with default value",
			input:          "\n",
			prompt:         "Enter value",
			defaultValue:   "default_val",
			expectedResult: "default_val",
			expectedPrompt: "Enter value [default_val]: ",
		},
		{
			name:           "User provides empty input without default value",
			input:          "\n",
			prompt:         "Enter value",
			defaultValue:   "",
			expectedResult: "",
			expectedPrompt: "Enter value: ",
		},
		{
			name:           "User provides input with leading and trailing spaces",
			input:          "   trimmed_value   \n",
			prompt:         "Enter value",
			defaultValue:   "default_val",
			expectedResult: "trimmed_value",
			expectedPrompt: "Enter value [default_val]: ",
		},
		{
			name:           "EOF or no newline in input",
			input:          "no_newline",
			prompt:         "Enter value",
			defaultValue:   "default_val",
			expectedResult: "no_newline",
			expectedPrompt: "Enter value [default_val]: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout to verify prompt output
			oldStdout := os.Stdout
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("failed to create pipe: %v", err)
			}
			os.Stdout = w

			reader := bufio.NewReader(strings.NewReader(tt.input))
			result := readLine(reader, tt.prompt, tt.defaultValue)

			_ = w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			_ = r.Close()
			actualPrompt := buf.String()

			if result != tt.expectedResult {
				t.Errorf("readLine() result = %q, expected %q", result, tt.expectedResult)
			}

			if actualPrompt != tt.expectedPrompt {
				t.Errorf("readLine() prompt output = %q, expected %q", actualPrompt, tt.expectedPrompt)
			}
		})
	}
}

// decryptPayload is a helper function used to verify encryptPayload in tests
func decryptPayload(encryptedHex string, secretKey string) ([]byte, error) {
	cipherText, err := hex.DecodeString(encryptedHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex: %w", err)
	}

	keyHash := sha256.Sum256([]byte(secretKey))
	block, err := aes.NewCipher(keyHash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(cipherText) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextPayload := cipherText[:nonceSize], cipherText[nonceSize:]
	plainText, err := gcm.Open(nil, nonce, ciphertextPayload, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plainText, nil
}

func TestEncryptPayload(t *testing.T) {
	tests := []struct {
		name      string
		plainText []byte
		secretKey string
	}{
		{
			name:      "Normal String Payload",
			plainText: []byte("hello world"),
			secretKey: "supersecretkey",
		},
		{
			name:      "Empty Plaintext",
			plainText: []byte(""),
			secretKey: "secret",
		},
		{
			name:      "Empty Secret Key",
			plainText: []byte("some payload data"),
			secretKey: "",
		},
		{
			name:      "Binary Data Payload",
			plainText: []byte{0x00, 0xFF, 0xDE, 0xAD, 0xBE, 0xEF},
			secretKey: "key12345",
		},
		{
			name:      "Large Payload",
			plainText: []byte(fmt.Sprintf("%010000d", 42)),
			secretKey: "longsecretkeyvalue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encryptedHex, err := encryptPayload(tt.plainText, tt.secretKey)
			if err != nil {
				t.Fatalf("encryptPayload failed unexpectedly: %v", err)
			}

			if encryptedHex == "" {
				t.Fatalf("expected non-empty encrypted hex string")
			}

			decrypted, err := decryptPayload(encryptedHex, tt.secretKey)
			if err != nil {
				t.Fatalf("decryptPayload failed to decrypt encrypted payload: %v", err)
			}

			if string(decrypted) != string(tt.plainText) {
				t.Errorf("decrypted output mismatch: got %q, expected %q", string(decrypted), string(tt.plainText))
			}
		})
	}
}

func TestEncryptPayload_RandomNonce(t *testing.T) {
	plainText := []byte("identical payload")
	secretKey := "same-secret-key"

	enc1, err1 := encryptPayload(plainText, secretKey)
	enc2, err2 := encryptPayload(plainText, secretKey)

	if err1 != nil || err2 != nil {
		t.Fatalf("encryptPayload failed: err1=%v, err2=%v", err1, err2)
	}

	if enc1 == enc2 {
		t.Errorf("expected different ciphertexts for same input due to random nonce, but got identical strings")
	}

	dec1, err1 := decryptPayload(enc1, secretKey)
	dec2, err2 := decryptPayload(enc2, secretKey)

	if err1 != nil || err2 != nil {
		t.Fatalf("decryptPayload failed: err1=%v, err2=%v", err1, err2)
	}

	if string(dec1) != string(plainText) || string(dec2) != string(plainText) {
		t.Errorf("both decryptions should equal original plainText")
	}
}

func TestEncryptPayload_WrongKeyDecryptionFails(t *testing.T) {
	plainText := []byte("secret information")
	secretKey := "correct-key"
	wrongKey := "wrong-key"

	encryptedHex, err := encryptPayload(plainText, secretKey)
	if err != nil {
		t.Fatalf("encryptPayload failed: %v", err)
	}

	_, err = decryptPayload(encryptedHex, wrongKey)
	if err == nil {
		t.Errorf("expected decryption error when using wrong key, but succeeded")
	}
}
