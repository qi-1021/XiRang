package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PermissionMode 控制智能体可执行的动作半径
type PermissionMode string

const (
	PermReadonly PermissionMode = "readonly" // 只读探测
	PermAssist   PermissionMode = "assist"   // 可改白名单内文件，默认确认危险命令
	PermAuto     PermissionMode = "auto"     // 允许自动修复/下载（仍受 PathGuard/Safety 约束）
)

func parsePermissionMode(s string) PermissionMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "readonly", "read-only", "ro":
		return PermReadonly
	case "auto", "full":
		return PermAuto
	default:
		return PermAssist
	}
}

func (p PermissionMode) allowsWrite() bool    { return p == PermAssist || p == PermAuto }
func (p PermissionMode) allowsExec() bool     { return p != PermReadonly }
func (p PermissionMode) allowsDownload() bool { return p == PermAuto }
func (p PermissionMode) allowsTool() bool     { return p != PermReadonly }

// requirePermission 检查子动作是否被当前权限模式允许；返回 (ok, reason)
func requirePermission(mode PermissionMode, action string) (bool, string) {
	switch action {
	case "read_file", "list_dir":
		return true, ""
	case "write_file":
		if mode.allowsWrite() {
			return true, ""
		}
		return false, fmt.Sprintf("【权限模式 %s】禁止写文件，请切换 -mode assist/auto", mode)
	case "run_command":
		if mode.allowsExec() {
			return true, ""
		}
		return false, fmt.Sprintf("【权限模式 %s】禁止执行命令", mode)
	case "http_download":
		if mode.allowsDownload() {
			return true, ""
		}
		return false, fmt.Sprintf("【权限模式 %s】禁止下载（需 auto）", mode)
	case "call_tool":
		if mode.allowsTool() {
			return true, ""
		}
		return false, fmt.Sprintf("【权限模式 %s】禁止外挂工具", mode)
	case "batch_actions":
		return mode != PermReadonly, fmt.Sprintf("【权限模式 %s】批处理需 assist/auto", mode)
	default:
		return true, ""
	}
}

// ---------- 结构化审计日志 ----------

type AuditEvent struct {
	Time    string `json:"time"`
	Session string `json:"session_id,omitempty"`
	Step    int    `json:"step,omitempty"`
	Type    string `json:"type"`
	Action  string `json:"action,omitempty"`
	Target  string `json:"target,omitempty"`
	OK      bool   `json:"ok"`
	Detail  string `json:"detail,omitempty"`
	Mode    string `json:"mode,omitempty"`
}

type AuditLogger struct {
	path    string
	session string
	step    int
	mode    string
}

func NewAuditLogger(sessionID string, mode PermissionMode) *AuditLogger {
	dir := filepath.Join(xirangHomeDir, "logs")
	_ = os.MkdirAll(dir, 0755)
	name := sessionID
	if name == "" {
		name = "default"
	}
	return &AuditLogger{
		path:    filepath.Join(dir, name+".jsonl"),
		session: sessionID,
		mode:    string(mode),
	}
}

func (a *AuditLogger) SetStep(n int) {
	if a == nil {
		return
	}
	a.step = n
}

func (a *AuditLogger) Log(typ, action, target string, ok bool, detail string) {
	if a == nil {
		return
	}
	ev := AuditEvent{
		Time:    time.Now().Format(time.RFC3339),
		Session: a.session,
		Step:    a.step,
		Type:    typ,
		Action:  action,
		Target:  target,
		OK:      ok,
		Detail:  detail,
		Mode:    a.mode,
	}
	data, err := jsonMarshalCompact(ev)
	if err != nil {
		return
	}
	f, err := os.OpenFile(a.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(data, '\n'))
}

// ---------- 有界对话记忆 ----------

const (
	defaultMaxMessages   = 40
	defaultMaxMsgChars   = 8000
	maxKeepSystemAtStart = 1
)

// BoundMessages 裁剪消息历史：始终保留 system + 最近 user 窗口，并压缩过长 assistant/user 内容
func BoundMessages(msgs []map[string]string, maxMsgs, maxChars int) []map[string]string {
	if maxMsgs <= 0 {
		maxMsgs = defaultMaxMessages
	}
	if maxChars <= 0 {
		maxChars = defaultMaxMsgChars
	}

	out := make([]map[string]string, 0, len(msgs))
	var systemMsgs []map[string]string
	var rest []map[string]string
	for _, m := range msgs {
		if m["role"] == "system" {
			systemMsgs = append(systemMsgs, m)
		} else {
			rest = append(rest, m)
		}
	}
	// 只保留首条 system（多条少见）
	if len(systemMsgs) > maxKeepSystemAtStart {
		systemMsgs = systemMsgs[:maxKeepSystemAtStart]
	}
	for _, m := range systemMsgs {
		out = append(out, map[string]string{
			"role":    m["role"],
			"content": truncateRunes(m["content"], maxChars*2),
		})
	}

	if len(rest) > maxMsgs {
		// 保留最近 maxMsgs 条，并在中间插入压缩提示
		dropped := len(rest) - maxMsgs
		summary := map[string]string{
			"role":    "user",
			"content": fmt.Sprintf("[系统自动压缩] 已省略较早的 %d 条对话，当前继续基于最近上下文执行。", dropped),
		}
		rest = append([]map[string]string{summary}, rest[len(rest)-maxMsgs:]...)
	}

	for _, m := range rest {
		out = append(out, map[string]string{
			"role":    m["role"],
			"content": truncateRunes(m["content"], maxChars),
		})
	}
	return out
}

func truncateRunes(s string, max int) string {
	if max <= 0 || len(s) <= max {
		// fast path for ASCII; still safe for UTF-8 by rune count when needed
		if len(s) <= max {
			return s
		}
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "\n...[内容已截断]..."
}
