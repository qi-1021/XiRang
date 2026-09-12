package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SessionState 支持进程崩溃后 resume
type SessionState struct {
	ID         string              `json:"id"`
	Goal       string              `json:"goal"`
	Mode       string              `json:"mode"`
	Scratchpad string              `json:"scratchpad"`
	Step       int                 `json:"step"`
	MaxSteps   int                 `json:"max_steps"`
	Messages   []map[string]string `json:"messages"`
	TaskSpec   *TaskSpecification  `json:"task_spec,omitempty"`
	StartedAt  string              `json:"started_at"`
	UpdatedAt  string              `json:"updated_at"`
	Finished   bool                `json:"finished"`
}

func sessionDir() string {
	return filepath.Join(xirangHomeDir, "sessions")
}

func newSessionID() string {
	return fmt.Sprintf("s_%d", time.Now().UnixNano())
}

func saveSession(st *SessionState) {
	if st == nil || st.ID == "" {
		return
	}
	dir := sessionDir()
	_ = os.MkdirAll(dir, 0755)
	st.UpdatedAt = time.Now().Format(time.RFC3339)
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dir, st.ID+".json"), data, 0644)
	// 更新 latest 指针
	_ = os.WriteFile(filepath.Join(dir, "latest.txt"), []byte(st.ID), 0644)
}

func loadSession(id string) *SessionState {
	if id == "" || id == "latest" {
		b, err := os.ReadFile(filepath.Join(sessionDir(), "latest.txt"))
		if err != nil {
			return nil
		}
		id = strings.TrimSpace(string(b))
	}
	if id == "" {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(sessionDir(), id+".json"))
	if err != nil {
		return nil
	}
	var st SessionState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil
	}
	return &st
}

func listSessions() []string {
	entries, err := os.ReadDir(sessionDir())
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			out = append(out, strings.TrimSuffix(e.Name(), ".json"))
		}
	}
	return out
}

// ---------- Spec 轻量校验（无外部依赖） ----------

func validateTaskSpec(spec *TaskSpecification) []string {
	var errs []string
	if spec == nil {
		return []string{"spec 为 nil"}
	}
	if strings.TrimSpace(spec.TaskName) == "" && strings.TrimSpace(spec.Goal) == "" {
		errs = append(errs, "task_name 与 goal 至少填写一项")
	}
	if spec.VerificationCmd != "" {
		// 验收命令不应明显危险
		if ok, reason := checkSafetyFilter(spec.VerificationCmd, spec.ForbiddenCmds); !ok {
			errs = append(errs, "verification_cmd 被安全过滤拒绝: "+reason)
		}
	}
	for i, m := range spec.Milestones {
		if strings.TrimSpace(m.Name) == "" {
			errs = append(errs, fmt.Sprintf("milestones[%d].name 为空", i))
		}
	}
	for i, t := range spec.CustomTools {
		if strings.TrimSpace(t.Name) == "" {
			errs = append(errs, fmt.Sprintf("custom_tools[%d].name 为空", i))
		}
		if strings.TrimSpace(t.CommandTmpl) == "" {
			errs = append(errs, fmt.Sprintf("custom_tools[%d].command_tmpl 为空", i))
		}
	}
	for i, p := range spec.HealthProbes {
		if strings.TrimSpace(p.Name) == "" {
			errs = append(errs, fmt.Sprintf("health_probes[%d].name 为空", i))
		}
		if strings.TrimSpace(p.CheckCmd) == "" {
			errs = append(errs, fmt.Sprintf("health_probes[%d].check_cmd 为空", i))
		}
	}
	for i, p := range spec.AllowedPaths {
		if strings.TrimSpace(p) == "" {
			errs = append(errs, fmt.Sprintf("allowed_paths[%d] 为空", i))
		}
	}
	return errs
}

func printSpecValidation(errs []string) {
	if len(errs) == 0 {
		fmt.Println("✅ [Spec 校验] 任务规约结构合法")
		return
	}
	fmt.Println("⚠️ [Spec 校验] 发现问题：")
	for _, e := range errs {
		fmt.Println("   -", e)
	}
}
