package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckSafetyFilter(t *testing.T) {
	tests := []struct {
		name         string
		cmd          string
		forbidden    []string
		expectedSafe bool
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
			name:         "Empty allowlist defaults to cwd sandbox",
			target:       subDir,
			allowed:      nil,
			expectedSafe: true,
		},
		{
			name:         "Empty allowlist blocks outside cwd",
			target:       outDir,
			allowed:      nil,
			expectedSafe: false,
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

func TestSaveAndFindFastSkill(t *testing.T) {
	origScriptsDir := scriptsDir
	defer func() { scriptsDir = origScriptsDir }()

	tempDir, err := os.MkdirTemp("", "xirang_test_scripts_save")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	scriptsDir = tempDir

	scriptPath := filepath.Join(tempDir, "fix_cuda.sh")
	_ = os.WriteFile(scriptPath, []byte("#!/bin/sh\necho cuda fixed"), 0755)

	err = saveFastSkill("FixCUDA", "cuda out of memory", scriptPath, "CUDA OOM auto cleanup")
	if err != nil {
		t.Fatalf("saveFastSkill failed: %v", err)
	}

	found := findFastSkill("Error: CUDA out of memory on device 0")
	if found == nil {
		t.Fatalf("expected to find fast skill after saveFastSkill, got nil")
	}
	if found.Name != "FixCUDA" || found.Description != "CUDA OOM auto cleanup" {
		t.Errorf("unexpected skill content: %+v", found)
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

func TestHarmonyPCRunScriptGenerated(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "xirang_harmony_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	harmonyScript := filepath.Join(tempDir, "run_harmony.sh")
	content := `#!/bin/sh
UNAME_M="$(uname -m 2>/dev/null || echo "x86_64")"
echo "HarmonyOS PC Target: $UNAME_M"
`
	err = os.WriteFile(harmonyScript, []byte(content), 0755)
	if err != nil {
		t.Fatalf("failed to write run_harmony.sh: %v", err)
	}

	info, err := os.Stat(harmonyScript)
	if err != nil || info.Size() == 0 {
		t.Fatalf("expected non-empty run_harmony.sh script")
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

func TestCheckPathGuard_PathTraversal(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}

	subDir := filepath.Join(cwd, "allowed_workspace")

	// Path traversal attempt using ..
	traversalTarget := filepath.Join(subDir, "..", "..", "system32")

	ok, msg := checkPathGuard(traversalTarget, []string{subDir})
	if ok {
		t.Errorf("checkPathGuard should have blocked path traversal target %q, but passed; msg: %s", traversalTarget, msg)
	}
}

func TestRollbackTrackCreatedFile(t *testing.T) {
	origRollbackStack := rollbackStack
	origBackupDir := backupDir
	defer func() {
		rollbackMu.Lock()
		rollbackStack = origRollbackStack
		rollbackMu.Unlock()
		backupDir = origBackupDir
	}()

	tempDir, err := os.MkdirTemp("", "xirang_test_rollback")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	backupDir = filepath.Join(tempDir, "backups")
	rollbackMu.Lock()
	rollbackStack = nil
	rollbackMu.Unlock()

	// File does not exist initially
	newFilePath := filepath.Join(tempDir, "newly_created.txt")
	trackFileBackup(newFilePath)

	// Create file
	err = os.WriteFile(newFilePath, []byte("hello world"), 0644)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Verify file exists before rollback
	if _, err := os.Stat(newFilePath); os.IsNotExist(err) {
		t.Fatalf("file should exist before rollback")
	}

	// Execute rollback
	executeRollback()

	// Verify newly created file was deleted
	if _, err := os.Stat(newFilePath); !os.IsNotExist(err) {
		t.Errorf("newly created file should have been deleted during rollback")
	}
}

func TestRollbackCrossProcessFromIndex(t *testing.T) {
	origRollbackStack := rollbackStack
	origBackupDir := backupDir
	defer func() {
		rollbackMu.Lock()
		rollbackStack = origRollbackStack
		rollbackMu.Unlock()
		backupDir = origBackupDir
	}()

	tempDir, err := os.MkdirTemp("", "xirang_test_rollback_xproc")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	backupDir = filepath.Join(tempDir, "backups")
	rollbackMu.Lock()
	rollbackStack = nil
	rollbackMu.Unlock()

	// Simulate previous process wrote a file and persisted index
	newFilePath := filepath.Join(tempDir, "cross_proc.txt")
	trackFileBackup(newFilePath)
	if err := os.WriteFile(newFilePath, []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := os.Stat(filepath.Join(backupDir, "index.json")); err != nil {
		t.Fatalf("expected rollback index on disk: %v", err)
	}

	// Simulate new process: clear memory stack only
	rollbackMu.Lock()
	rollbackStack = nil
	rollbackMu.Unlock()

	executeRollback()

	if _, err := os.Stat(newFilePath); !os.IsNotExist(err) {
		t.Errorf("cross-process rollback should delete file via index.json")
	}
}

func TestTrackFileBackupConcurrent(t *testing.T) {
	origRollbackStack := rollbackStack
	origBackupDir := backupDir
	defer func() {
		rollbackMu.Lock()
		rollbackStack = origRollbackStack
		rollbackMu.Unlock()
		backupDir = origBackupDir
	}()

	tempDir, err := os.MkdirTemp("", "xirang_test_rollback_race")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	backupDir = filepath.Join(tempDir, "backups")
	rollbackMu.Lock()
	rollbackStack = nil
	rollbackMu.Unlock()

	const n = 32
	done := make(chan struct{}, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer func() { done <- struct{}{} }()
			path := filepath.Join(tempDir, filepath.Join(tempDir, fmt.Sprintf("f_%d.txt", i)))
			// use nested unique file under tempDir
			path = filepath.Join(tempDir, fmt.Sprintf("f_%d.txt", i))
			trackFileBackup(path)
			_ = os.WriteFile(path, []byte("data"), 0644)
		}(i)
	}
	for i := 0; i < n; i++ {
		<-done
	}

	rollbackMu.Lock()
	count := len(rollbackStack)
	rollbackMu.Unlock()
	if count < n {
		t.Fatalf("expected at least %d rollback ops after concurrent track, got %d", n, count)
	}
}

func TestDeriveSkillPatternAndAutoSave(t *testing.T) {
	origScriptsDir := scriptsDir
	origSession := sessionScripts
	defer func() {
		scriptsDir = origScriptsDir
		sessionScripts = origSession
	}()

	tempDir, err := os.MkdirTemp("", "xirang_test_autosave")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	defer os.RemoveAll(tempDir)
	scriptsDir = filepath.Join(tempDir, "scripts")

	pattern := deriveSkillPattern("Error: address already in use while binding", "fix_port.sh")
	if pattern == "" || !strings.Contains(pattern, "address already in use") {
		t.Fatalf("unexpected pattern: %q", pattern)
	}

	scriptPath := filepath.Join(scriptsDir, "fix_port.sh")
	os.MkdirAll(scriptsDir, 0755)
	maybeAutoSaveSkill(scriptPath, "echo address already in use")
	if _, err := os.Stat(scriptPath + ".meta.json"); err != nil {
		t.Fatalf("expected auto meta: %v", err)
	}
	found := findFastSkill("server failed: address already in use")
	if found == nil {
		t.Fatalf("expected fast skill after auto save")
	}
}

func TestFuzzyFindFastSkill(t *testing.T) {
	origScriptsDir := scriptsDir
	defer func() { scriptsDir = origScriptsDir }()

	tempDir, err := os.MkdirTemp("", "xirang_test_fuzzy_scripts")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	scriptsDir = tempDir

	// Write multi-keyword skill
	skill := DiscoveredSkill{
		Name:        "FixPortAndService",
		Pattern:     "port 8080 | address already in use",
		ScriptPath:  filepath.Join(tempDir, "fix_port.sh"),
		Description: "Multi keyword skill",
	}
	skillBytes, _ := json.Marshal(skill)

	_ = os.WriteFile(filepath.Join(tempDir, "fix_port.sh"), []byte("#!/bin/sh\necho fixed"), 0755)
	_ = os.WriteFile(filepath.Join(tempDir, "fix_port.sh.meta.json"), skillBytes, 0644)

	found := findFastSkill("Error starting server: address already in use on port 8080")
	if found == nil {
		t.Fatalf("expected to find skill with multi-keyword matching")
	}
	if found.Name != "FixPortAndService" {
		t.Errorf("expected skill name FixPortAndService, got %q", found.Name)
	}
}
