package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckSafetyFilter(t *testing.T) {
	tests := []struct {
		name          string
		cmd           string
		forbidden     []string
		expectedSafe  bool
	}{
		{
			name:         "Safe Command",
			cmd:          "echo hello world",
			forbidden:    nil,
			expectedSafe: true,
		},
		{
			name:         "Default Forbidden Command - rm -rf /",
			cmd:          "rm -rf /",
			forbidden:    nil,
			expectedSafe: false,
		},
		{
			name:         "Custom Forbidden Command",
			cmd:          "python -m http.server",
			forbidden:    []string{"http.server"},
			expectedSafe: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			safe, msg := checkSafetyFilter(tt.cmd, tt.forbidden)
			if safe != tt.expectedSafe {
				t.Errorf("checkSafetyFilter(%q) safe = %v, expected %v; msg = %q", tt.cmd, safe, tt.expectedSafe, msg)
			}
		})
	}
}

func TestCheckPathGuard(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}

	subDir := filepath.Join(cwd, "test_dir")
	outDir := filepath.Join(cwd, "..", "outside_dir")

	tests := []struct {
		name         string
		target       string
		allowed      []string
		expectedSafe bool
	}{
		{
			name:         "No restrictions",
			target:       subDir,
			allowed:      nil,
			expectedSafe: true,
		},
		{
			name:         "Inside allowed path",
			target:       subDir,
			allowed:      []string{cwd},
			expectedSafe: true,
		},
		{
			name:         "Outside allowed path",
			target:       outDir,
			allowed:      []string{subDir},
			expectedSafe: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, msg := checkPathGuard(tt.target, tt.allowed)
			if ok != tt.expectedSafe {
				t.Errorf("checkPathGuard(%q, %v) ok = %v, expected %v; msg = %q", tt.target, tt.allowed, ok, tt.expectedSafe, msg)
			}
		})
	}
}

func TestSmartPruneLog(t *testing.T) {
	shortLog := "Line 1\nLine 2\nLine 3"
	if pruned := smartPruneLog(shortLog); pruned != shortLog {
		t.Errorf("smartPruneLog short output modified: got %q, expected %q", pruned, shortLog)
	}

	var buf bytes.Buffer
	for i := 1; i <= 30; i++ {
		if i == 15 {
			buf.WriteString("Error: Failed to bind port\n")
		} else {
			buf.WriteString("Normal log line content\n")
		}
	}

	longLog := buf.String()
	pruned := smartPruneLog(longLog)
	if len(pruned) >= len(longLog) {
		t.Errorf("smartPruneLog did not reduce log length")
	}
	if !bytes.Contains([]byte(pruned), []byte("Failed to bind port")) {
		t.Errorf("smartPruneLog failed to retain captured error line")
	}
}

func TestFindFastSkill(t *testing.T) {
	// Create temporary scriptsDir
	origScriptsDir := scriptsDir
	defer func() { scriptsDir = origScriptsDir }()

	tempDir, err := os.MkdirTemp("", "xirang_test_scripts")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	scriptsDir = tempDir

	// Write dummy skill
	skill := DiscoveredSkill{
		Name:        "FixPort",
		Pattern:     "address already in use",
		ScriptPath:  filepath.Join(tempDir, "fix_port.sh"),
		Description: "Port cleanup script",
	}
	skillBytes, _ := json.Marshal(skill)

	_ = os.WriteFile(filepath.Join(tempDir, "fix_port.sh"), []byte("#!/bin/sh\necho fixed"), 0755)
	_ = os.WriteFile(filepath.Join(tempDir, "fix_port.sh.meta.json"), skillBytes, 0644)

	found := findFastSkill("Error: address already in use on port 8080")
	if found == nil {
		t.Fatalf("expected to find fast skill, got nil")
	}
	if found.Name != "FixPort" {
		t.Errorf("expected skill name FixPort, got %q", found.Name)
	}

	notFound := findFastSkill("out of memory")
	if notFound != nil {
		t.Errorf("expected nil for unmatched pattern, got %+v", notFound)
	}
}

func TestLoadEmbeddedProtectedConfig(t *testing.T) {
	// When empty
	if cfg := loadEmbeddedProtectedConfig(); cfg != nil {
		t.Errorf("expected nil for empty EncryptedPayload/BuildSecret, got %+v", cfg)
	}

	secret := "test-secret-key-123"
	testCfg := Config{
		Goal:     "Test Goal",
		MaxSteps: 10,
	}
	cfgBytes, _ := json.Marshal(testCfg)

	// Encrypt
	keyHash := sha256.Sum256([]byte(secret))
	block, _ := aes.NewCipher(keyHash[:])
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	_, _ = io.ReadFull(rand.Reader, nonce)
	cipherText := gcm.Seal(nonce, nonce, cfgBytes, nil)

	EncryptedPayload = hex.EncodeToString(cipherText)
	BuildSecret = secret

	defer func() {
		EncryptedPayload = ""
		BuildSecret = ""
	}()

	loaded := loadEmbeddedProtectedConfig()
	if loaded == nil {
		t.Fatalf("failed to load embedded protected config")
	}
	if loaded.Goal != "Test Goal" || loaded.MaxSteps != 10 {
		t.Errorf("loaded config mismatched: %+v", loaded)
	}
}
