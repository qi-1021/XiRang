package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// TUI 操作台：纯终端 ANSI，无第三方依赖
// 命令: xirang -console

type consoleState struct {
	cfg       *Config
	sessionID string
	audit     *AuditLogger
	perm      PermissionMode
	quit      bool
}

func runConsole(cfg *Config, perm PermissionMode) {
	st := &consoleState{
		cfg:       cfg,
		sessionID: newSessionID(),
		perm:      perm,
	}
	st.audit = NewAuditLogger(st.sessionID, perm)

	reader := bufio.NewReader(os.Stdin)
	clearScreen()
	st.banner()

	for !st.quit {
		st.renderDashboard()
		fmt.Print("息壤操作台> ")
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		cmd := strings.ToLower(parts[0])
		args := parts[1:]

		switch cmd {
		case "help", "h", "?":
			st.printHelp()
		case "status", "st":
			st.renderDashboard()
		case "goal":
			if len(args) == 0 {
				fmt.Println("当前目标:", st.cfg.Goal)
				continue
			}
			st.cfg.Goal = strings.Join(args, " ")
			st.audit.Log("console", "set_goal", "", true, st.cfg.Goal)
			fmt.Println("已更新目标")
		case "mode":
			if len(args) == 0 {
				fmt.Println("当前权限:", st.perm)
				continue
			}
			st.perm = parsePermissionMode(args[0])
			st.audit.Log("console", "set_mode", string(st.perm), true, "")
			fmt.Println("权限已切换为", st.perm)
		case "probe", "health":
			st.runProbes()
		case "verify", "dod":
			st.runVerify()
		case "skills":
			runSkillsCLI(args)
		case "rollback":
			executeRollback()
			st.audit.Log("console", "rollback", "", true, "")
		case "audit":
			st.showAuditTail(20)
		case "sessions":
			for _, id := range listSessions() {
				fmt.Println(" -", id)
			}
		case "resume":
			id := "latest"
			if len(args) > 0 {
				id = args[0]
			}
			s := loadSession(id)
			if s == nil {
				fmt.Println("未找到会话:", id)
				continue
			}
			st.sessionID = s.ID
			st.cfg.Goal = s.Goal
			st.audit = NewAuditLogger(s.ID, st.perm)
			fmt.Printf("已恢复会话 %s step=%d goal=%s\n", s.ID, s.Step, s.Goal)
		case "run", "start":
			st.audit.Log("console", "run", st.cfg.Goal, true, "mode="+string(st.perm))
			// 会话落盘后进入主循环
			sid := st.sessionID
			runAgentLoopWithSession(cfg, perm, sid, nil)
			st.sessionID = newSessionID()
			st.audit = NewAuditLogger(st.sessionID, st.perm)
		case "doctor":
			runDoctorMode(cfg)
		case "watchdog":
			interval := 30
			if len(args) > 0 {
				fmt.Sscanf(args[0], "%d", &interval)
			}
			runWatchdogMode(cfg, interval)
		case "quit", "exit", "q":
			st.quit = true
			fmt.Println("再见。审计日志:", st.audit.path)
		default:
			fmt.Println("未知命令。输入 help 查看。")
		}
	}
}

func (st *consoleState) banner() {
	fmt.Println("================================================================")
	fmt.Println("  🌱 息壤操作台 (CLI/TUI) — 可控 · 可审计 · 可回滚")
	fmt.Println("  权限模式:", st.perm, " 会话:", st.sessionID)
	fmt.Println("================================================================")
	fmt.Println("  输入 help 查看命令；run 开始执行；quit 退出")
	fmt.Println("================================================================")
	fmt.Println()
}

func (st *consoleState) printHelp() {
	fmt.Print(`可用命令:
  status              查看状态仪表盘
  goal [文本...]      查看/设置目标
  mode readonly|assist|auto
  probe               执行健康探针
  verify              0-Token DoD 验收
  skills ...          技能库管理 (list/show/disable/enable/export/import)
  rollback            从 index.json 回滚
  audit               查看最近审计日志
  sessions            列出会话
  resume [id]         恢复会话
  run                 开始 Agent 执行
  doctor              急诊医生
  watchdog [秒]       守护巡检
  quit                退出操作台
`)
}

func (st *consoleState) renderDashboard() {
	fmt.Println("----------------------------------------------------------------")
	fmt.Printf("会话: %s | 权限: %s | 时间: %s\n", st.sessionID, st.perm, time.Now().Format("15:04:05"))
	fmt.Printf("目标: %s\n", st.cfg.Goal)
	if st.cfg.TaskSpec != nil {
		fmt.Printf("规约: %s | 验收: %s\n", st.cfg.TaskSpec.TaskName, boolOrDash(st.cfg.TaskSpec.VerificationCmd != "", st.cfg.TaskSpec.VerificationCmd, "无"))
		fmt.Printf("白名单: %v | 探针: %d\n", st.cfg.TaskSpec.AllowedPaths, len(st.cfg.TaskSpec.HealthProbes))
	} else {
		fmt.Println("规约: 未绑定 (默认沙箱 cwd+.xirang)")
	}
	// 回滚栈
	rollbackMu.Lock()
	n := len(rollbackStack)
	rollbackMu.Unlock()
	fmt.Printf("待回滚事务: %d | 技能数: %d | 审计: %s\n", n, len(loadAllSkills()), st.audit.path)
	fmt.Println("----------------------------------------------------------------")
}

func boolOrDash(ok bool, yes, no string) string {
	if ok {
		return yes
	}
	return no
}

func (st *consoleState) runProbes() {
	var probes []HealthProbe
	if st.cfg.TaskSpec != nil && len(st.cfg.TaskSpec.HealthProbes) > 0 {
		probes = st.cfg.TaskSpec.HealthProbes
	} else {
		probes = defaultRealProbes()
	}
	fmt.Printf("执行 %d 个探针...\n", len(probes))
	fails := 0
	for _, p := range probes {
		ok, detail := probeHealth(p)
		icon := "✅"
		if !ok {
			icon = "❌"
			fails++
		}
		fmt.Printf("  %s %-20s %s\n", icon, p.Name, detail)
		st.audit.Log("probe", p.Name, p.Target, ok, detail)
	}
	if fails > 0 {
		fmt.Printf("有 %d 个探针失败。可用 run / doctor 自愈。\n", fails)
	} else {
		fmt.Println("全部探针通过。")
	}
}

func (st *consoleState) runVerify() {
	if st.cfg.TaskSpec == nil || st.cfg.TaskSpec.VerificationCmd == "" {
		fmt.Println("未配置 VerificationCmd")
		return
	}
	ok, reason := checkSafetyFilter(st.cfg.TaskSpec.VerificationCmd, st.cfg.TaskSpec.ForbiddenCmds)
	if !ok {
		fmt.Println("拒绝执行:", reason)
		st.audit.Log("verify", "safety", st.cfg.TaskSpec.VerificationCmd, false, reason)
		return
	}
	code, out := executeCommand(st.cfg.TaskSpec.VerificationCmd, 60)
	success := code == 0
	fmt.Printf("DoD 退出码=%d\n%s\n", code, out)
	st.audit.Log("verify", "dod", st.cfg.TaskSpec.VerificationCmd, success, fmt.Sprintf("exit=%d", code))
}

func (st *consoleState) showAuditTail(n int) {
	data, err := os.ReadFile(st.audit.path)
	if err != nil {
		fmt.Println("暂无审计日志:", err)
		return
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	for _, l := range lines {
		if l != "" {
			fmt.Println(l)
		}
	}
}

func clearScreen() {
	fmt.Print("\033[2J\033[H")
}
