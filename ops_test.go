package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBoundMessages(t *testing.T) {
	var msgs []map[string]string
	msgs = append(msgs, map[string]string{"role": "system", "content": "sys"})
	for i := 0; i < 100; i++ {
		msgs = append(msgs, map[string]string{"role": "user", "content": "x"})
	}
	out := BoundMessages(msgs, 10, 100)
	if len(out) > 12 { // system + maybe summary + 10
		t.Fatalf("messages not bounded: %d", len(out))
	}
	if out[0]["role"] != "system" {
		t.Fatalf("system must be first")
	}
}

func TestPermissionModes(t *testing.T) {
	if ok, _ := requirePermission(PermReadonly, "write_file"); ok {
		t.Fatal("readonly should block write")
	}
	if ok, _ := requirePermission(PermAssist, "write_file"); !ok {
		t.Fatal("assist should allow write")
	}
	if ok, _ := requirePermission(PermAssist, "http_download"); ok {
		t.Fatal("assist should block download")
	}
	if ok, _ := requirePermission(PermAuto, "http_download"); !ok {
		t.Fatal("auto should allow download")
	}
}

func TestValidateTaskSpec(t *testing.T) {
	errs := validateTaskSpec(&TaskSpecification{Goal: "g"})
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %v", errs)
	}
	errs = validateTaskSpec(&TaskSpecification{})
	if len(errs) == 0 {
		t.Fatal("empty spec should error")
	}
}

func TestProbeFileAndTCP(t *testing.T) {
	ok, _ := probeHealth(HealthProbe{Kind: "file", Target: "."})
	if !ok {
		t.Fatal("file probe . should pass")
	}
	// closed port should fail quickly
	ok, _ = probeHealth(HealthProbe{Kind: "tcp", Target: "127.0.0.1:1", TimeoutSec: 1})
	if ok {
		t.Log("unexpected tcp ok on port 1 (maybe listening)")
	}
}

func TestSessionSaveLoad(t *testing.T) {
	origXH := xirangHomeDir
	// use real relative .xirang under temp by chdir
	origWD, _ := os.Getwd()
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(origWD)
		xirangHomeDir = origXH
		backupDir = filepath.Join(xirangHomeDir, "backups")
		scriptsDir = filepath.Join(xirangHomeDir, "scripts")
		skillsDir = filepath.Join(xirangHomeDir, "skills")
	}()

	id := newSessionID()
	saveSession(&SessionState{ID: id, Goal: "test-goal", Step: 3, MaxSteps: 10, Messages: []map[string]string{{"role": "user", "content": "hi"}}})
	got := loadSession(id)
	if got == nil || got.Goal != "test-goal" || got.Step != 3 {
		t.Fatalf("session roundtrip failed: %+v", got)
	}
	if got2 := loadSession("latest"); got2 == nil || got2.ID != id {
		t.Fatal("latest pointer broken")
	}
}

func TestBackoffDelay(t *testing.T) {
	d1 := backoffDelay(1, time.Second)
	d3 := backoffDelay(3, time.Second)
	if d3 <= d1 {
		t.Fatalf("backoff should grow: %v vs %v", d3, d1)
	}
	if d1 < time.Second {
		t.Fatalf("too small: %v", d1)
	}
}

func TestDefaultRealProbes(t *testing.T) {
	ps := defaultRealProbes()
	if len(ps) == 0 {
		t.Fatal("need default probes")
	}
	for _, p := range ps {
		if p.Kind == "" {
			t.Fatalf("probe %s missing kind", p.Name)
		}
		if p.Kind == "cmd" && p.CheckCmd == "" {
			t.Fatalf("cmd probe missing check_cmd")
		}
	}
}
