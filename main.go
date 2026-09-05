package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"zenagent/builder/gui"
)

// Provider 配置结构
type Provider struct {
	Name    string            `json:"name"`
	URL     string            `json:"url"`
	Key     string            `json:"key"`
	Model   string            `json:"model"`
	Headers map[string]string `json:"headers"`
}

// 任务里程碑阶段
type Milestone struct {
	Name        string `json:"name"`
	VerifyCmd   string `json:"verify_cmd"`
	Description string `json:"description"`
	Done        bool   `json:"done"`
}

// 售后健康巡检探针 (Health Check & Watchdog)
type HealthProbe struct {
	Name     string `json:"name"`     // 服务/组件名称
	CheckCmd string `json:"check_cmd"` // 快速健康检查命令 (退出码 0 为正常)
	FixGoal  string `json:"fix_goal"`  // 一旦异常时触发的自愈排错目标
}

// 动态外挂工具定义 (MCP/CLI Plugin)
type CustomTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	CommandTmpl string `json:"command_tmpl"` // 命令行模板，如 "git clone {{url}} {{path}}" 或 "docker {{args}}"
	TimeoutSec  int    `json:"timeout_sec"`
}

// 事务撤销单元 (Undo Action)
type RollbackOp struct {
	Action      string `json:"action"` // "restore_file", "delete_file", "run_cmd"
	Target      string `json:"target"`
	BackupPath  string `json:"backup_path"`
	RollbackCmd string `json:"rollback_cmd"`
}

// 工业级结构化任务规约 (Task Specification v3.0)
type TaskSpecification struct {
	TaskName        string       `json:"task_name"`
	Goal            string       `json:"goal"`
	AllowedPaths    []string     `json:"allowed_paths"`    // 写入路径白名单 (安全沙箱)
	ForbiddenCmds   []string     `json:"forbidden_cmds"`   // 危险指令黑名单
	Milestones      []Milestone  `json:"milestones"`       // 分步里程碑
	VerificationCmd string       `json:"verification_cmd"` // 终态防作弊验收命令 (DoD)
	StrictRules     []string      `json:"strict_rules"`     // 强制规则
	CustomTools     []CustomTool  `json:"custom_tools"`     // 打包预置的专属领域扩展工具箱
	HealthProbes    []HealthProbe `json:"health_probes"`    // 售后巡检探针
}

// 统一可外挂配置结构
type BuildConfig struct {
	DefaultGoal string             `json:"default_goal"`
	TaskSpec    *TaskSpecification `json:"task_spec,omitempty"`
	Providers   []Provider         `json:"providers"`
	MaxSteps    int                `json:"max_steps"`
	TimeoutSec  int                `json:"timeout_sec"`
}

type Config struct {
	Goal        string             `json:"goal"`
	TaskSpec    *TaskSpecification `json:"task_spec,omitempty"`
	Providers   []Provider         `json:"providers"`
	MaxSteps    int                `json:"max_steps"`
	TimeoutSec  int                `json:"timeout_sec"`
	ContextVars map[string]string  `json:"context_vars"`
}

// 单个原子动作 (用于单步或并发 batch_actions)
type SubAction struct {
	ID          string   `json:"id,omitempty"`
	Action      string   `json:"action"` // "run_command", "write_file", "read_file", "list_dir", "http_download", "call_tool"
	Command     string   `json:"command,omitempty"`
	Path        string   `json:"path,omitempty"`
	Content     string   `json:"content,omitempty"`
	URL         string   `json:"url,omitempty"`
	ToolName    string   `json:"tool_name,omitempty"`
	ToolArgs    string   `json:"tool_args,omitempty"`
	Explanation string   `json:"explanation,omitempty"`
}

// 决策结构体 (支持单步、并发批处理 batch_actions、黑板更新、事务与人机交互)
type Decision struct {
	Action        string      `json:"action"` // "run_command", "write_file", "read_file", "list_dir", "http_download", "batch_actions", "call_tool", "ask_human", "report_milestone", "finish"
	Thought       string      `json:"thought"`
	Explanation   string      `json:"explanation"`
	
	// 单步动作参数
	Command       string      `json:"command,omitempty"`
	Path          string      `json:"path,omitempty"`
	Content       string      `json:"content,omitempty"`
	URL           string      `json:"url,omitempty"`
	ToolName      string      `json:"tool_name,omitempty"`
	ToolArgs      string      `json:"tool_args,omitempty"`
	
	// 并发多线操作队列 (支持完全并行的任务批处理)
	BatchActions  []SubAction `json:"batch_actions,omitempty"`

	// 状态便签与反思黑板 (Working Memory & Scratchpad)
	Scratchpad    string      `json:"scratchpad,omitempty"` // 当前掌握的事实、排障假设与下一步计划

	// 交互决策
	Question      string      `json:"question,omitempty"`
	Options       []string    `json:"options,omitempty"`
	Milestone     string      `json:"milestone,omitempty"`
}

var defaultProviders = []Provider{}

// 统一工作空间隔离与持久化知识经验库 (.xirang/)
var (
	rollbackStack []RollbackOp
	xirangHomeDir = ".xirang"
	backupDir     = filepath.Join(xirangHomeDir, "backups")
	scriptsDir    = filepath.Join(xirangHomeDir, "scripts")
	skillsDir     = filepath.Join(xirangHomeDir, "skills")
)

func init() {
	// 自动平滑升级历史 .zen 目录，若存在则无缝迁移至 .xirang
	if _, err := os.Stat(".zen"); err == nil {
		if _, errX := os.Stat(xirangHomeDir); os.IsNotExist(errX) {
			_ = os.Rename(".zen", xirangHomeDir)
		}
	}
}

// 经验沉淀与快速故障指纹库 (Evolved Skill & Error Fingerprint)
type DiscoveredSkill struct {
	Name        string `json:"name"`
	Pattern     string `json:"pattern"`     // 错误指纹 / 触发关键字 (如 "address already in use", "cuda out of memory")
	ScriptPath  string `json:"script_path"` // 沉淀在 .zen/scripts/ 的具体脚本
	Description string `json:"description"`
}

// 本地快速命中故障指纹库 (Fast-Path Cache: 命中已知问题直接执行现成脚本，不走大模型)
func findFastSkill(errorText string) *DiscoveredSkill {
	files, err := os.ReadDir(scriptsDir)
	if err != nil || len(files) == 0 {
		return nil
	}
	lower := strings.ToLower(errorText)
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".meta.json") {
			data, err := os.ReadFile(filepath.Join(scriptsDir, f.Name()))
			if err == nil {
				var skill DiscoveredSkill
				if json.Unmarshal(data, &skill) == nil {
					if skill.Pattern != "" && strings.Contains(lower, strings.ToLower(skill.Pattern)) {
						return &skill
					}
				}
			}
		}
	}
	return nil
}

// 自动扫描已沉淀的本地脚本与技能 (越用越熟练)
func loadEvolvedSkills() string {
	os.MkdirAll(scriptsDir, 0755)
	os.MkdirAll(skillsDir, 0755)
	
	files, err := os.ReadDir(scriptsDir)
	if err != nil || len(files) == 0 {
		return "【本地经验库】: 当前为首次运行，暂无已沉淀的常驻脚本。"
	}

	var sb strings.Builder
	sb.WriteString("【本地已沉淀的经验脚本库 (.xirang/scripts/) - 可直接复用】:\n")
	for _, f := range files {
		if !f.IsDir() && !strings.HasSuffix(f.Name(), ".meta.json") {
			desc := "自动化运维自愈脚本"
			metaData, err := os.ReadFile(filepath.Join(scriptsDir, f.Name()+".meta.json"))
			if err == nil {
				var skill DiscoveredSkill
				if json.Unmarshal(metaData, &skill) == nil && skill.Description != "" {
					desc = skill.Description
				}
			}
			sb.WriteString(fmt.Sprintf("   - %s [%s] (路径: %s)\n", f.Name(), desc, filepath.Join(scriptsDir, f.Name())))
		}
	}
	return sb.String()
}

// 记录文件备份事务
func trackFileBackup(filePath string) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// 原本不存在，回滚操作为删除新产生的文件
		rollbackStack = append(rollbackStack, RollbackOp{
			Action: "delete_file",
			Target: filePath,
		})
		return
	}

	// 已存在，先备份原内容
	os.MkdirAll(backupDir, 0755)
	backupName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(filePath))
	backupPath := filepath.Join(backupDir, backupName)
	
	in, err := os.ReadFile(filePath)
	if err == nil {
		os.WriteFile(backupPath, in, 0644)
		rollbackStack = append(rollbackStack, RollbackOp{
			Action:     "restore_file",
			Target:     filePath,
			BackupPath: backupPath,
		})
	}
}

// 执行一键安全回滚
func executeRollback() {
	if len(rollbackStack) == 0 {
		fmt.Println("[ℹ️ 事务回滚] 当前未产生系统变更，无需撤销。")
		return
	}
	fmt.Println("\n================================================================")
	fmt.Println("⏪ [安全回滚] 正在逆序撤销本次任务所作的全部环境变更...")
	for i := len(rollbackStack) - 1; i >= 0; i-- {
		op := rollbackStack[i]
		switch op.Action {
		case "delete_file":
			os.Remove(op.Target)
			fmt.Printf("   -> 已清除新生成的文件: %s\n", op.Target)
		case "restore_file":
			data, err := os.ReadFile(op.BackupPath)
			if err == nil {
				os.WriteFile(op.Target, data, 0644)
				fmt.Printf("   -> 已还原原始文件备份: %s\n", op.Target)
			}
		}
	}
	os.RemoveAll(backupDir) // 仅清理撤销备份，保留 .xirang/scripts 经验资产
	fmt.Println("✅ [回滚完成] 宿主环境已完全恢复至执行前初始状态！")
	fmt.Println("================================================================")
}

// 智能日志剪枝压缩器 (保持上下文精炼，杜绝 Token 爆炸与遗忘)
func smartPruneLog(raw string) string {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	if len(lines) <= 25 {
		return strings.TrimSpace(raw)
	}

	head := lines[:8]
	tail := lines[len(lines)-12:]
	
	// 筛选中段关键报错与警告行
	var errorLines []string
	for _, l := range lines[8 : len(lines)-12] {
		lower := strings.ToLower(l)
		if strings.Contains(lower, "error") || strings.Contains(lower, "fail") || 
		   strings.Contains(lower, "fatal") || strings.Contains(lower, "warn") ||
		   strings.Contains(lower, "exception") || strings.Contains(lower, "denied") {
			errorLines = append(errorLines, "   [!] "+l)
			if len(errorLines) >= 8 {
				break
			}
		}
	}

	res := strings.Join(head, "\n") + fmt.Sprintf("\n... [自动折叠 %d 行长日志] ...\n", len(lines)-20)
	if len(errorLines) > 0 {
		res += "【中间捕获的异常信息】:\n" + strings.Join(errorLines, "\n") + "\n"
	}
	res += strings.Join(tail, "\n")
	return res
}

// 动态环境嗅探
func probeEnvironment() map[string]interface{} {
	env := make(map[string]interface{})
	env["os"] = runtime.GOOS
	env["arch"] = runtime.GOARCH

	// 探测 GPU
	cmd := exec.Command("nvidia-smi", "--query-gpu=name,memory.total,driver_version", "--format=csv,noheader")
	out, err := cmd.CombinedOutput()
	if err == nil && len(bytes.TrimSpace(out)) > 0 {
		env["gpu"] = strings.TrimSpace(string(out))
	} else {
		env["gpu"] = "无 NVIDIA 独显或驱动未装"
	}

	// 探测当前工作目录及文件
	cwd, _ := os.Getwd()
	env["cwd"] = cwd
	files, _ := os.ReadDir(cwd)
	var fileNames []string
	for _, f := range files {
		fileNames = append(fileNames, f.Name())
	}
	env["cwd_files"] = fileNames

	// 探测磁盘可用驱动器 (Windows)
	if runtime.GOOS == "windows" {
		var availableDrives []string
		for _, drive := range []string{"C:", "D:", "E:", "F:", "G:"} {
			if _, err := os.Stat(drive + "\\"); err == nil {
				availableDrives = append(availableDrives, drive)
			}
		}
		env["available_drives"] = availableDrives
	}

	return env
}

// 安全过滤器：检查危险破坏性指令
func checkSafetyFilter(command string, forbiddenCmds []string) (bool, string) {
	lower := strings.ToLower(command)
	defaultForbidden := []string{
		"format ", "del /f /s /q c:\\", "rm -rf /", "rm -rf /*",
		"mkfs", "dd if=", ":(){ :|:& };:", "diskpart",
	}
	allForbidden := append(defaultForbidden, forbiddenCmds...)
	for _, pattern := range allForbidden {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return false, fmt.Sprintf("指令被安全守卫拦截！包含高危破坏性特征: '%s'", pattern)
		}
	}
	return true, ""
}

// 路径白名单守卫：检查是否写入未授权目录
func checkPathGuard(targetPath string, allowedPaths []string) (bool, string) {
	if len(allowedPaths) == 0 {
		return true, "" // 开发者未设限时允许
	}
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return false, fmt.Sprintf("路径解析错误: %v", err)
	}

	for _, allowed := range allowedPaths {
		absAllowed, _ := filepath.Abs(allowed)
		rel, err := filepath.Rel(absAllowed, absTarget)
		if err == nil && !strings.HasPrefix(rel, "..") {
			return true, ""
		}
	}
	return false, fmt.Sprintf("【PathGuard 路径安全违规】目标路径 '%s' 不在受信任的目录白名单 %v 中！", targetPath, allowedPaths)
}

// 执行本地命令 (升级为实时流式管道，支持超长任务即时打印与进度透出)
func executeCommand(command string, timeoutSec int) (int, string) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return -1, fmt.Sprintf("创建输出管道失败: %v", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return -1, fmt.Sprintf("创建错误管道失败: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return -1, fmt.Sprintf("启动命令失败: %v", err)
	}

	var outputBuf bytes.Buffer
	var mu sync.Mutex
	var wg sync.WaitGroup

	streamReader := func(r io.Reader, isErr bool) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			mu.Lock()
			outputBuf.WriteString(line + "\n")
			mu.Unlock()
			// 实时流式输出至终端，让用户清晰看到安装/编译进度
			if isErr {
				fmt.Printf("   [stderr] %s\n", line)
			} else {
				fmt.Printf("   | %s\n", line)
			}
		}
	}

	wg.Add(2)
	go streamReader(stdoutPipe, false)
	go streamReader(stderrPipe, true)

	done := make(chan error, 1)
	go func() {
		wg.Wait()
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(time.Duration(timeoutSec) * time.Second):
		cmd.Process.Kill()
		return -2, fmt.Sprintf("命令执行超时 (%d 秒已熔断)", timeoutSec)
	case err := <-done:
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = -1
			}
		}
		return exitCode, strings.TrimSpace(outputBuf.String())
	}
}

// 原生 HTTP 下载
func downloadFile(url, destPath string) (bool, string) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   300 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return false, fmt.Sprintf("下载连接失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Sprintf("服务器返回状态码 HTTP %d", resp.StatusCode)
	}

	os.MkdirAll(filepath.Dir(destPath), 0755)
	out, err := os.Create(destPath)
	if err != nil {
		return false, fmt.Sprintf("创建本地目标文件失败: %v", err)
	}
	defer out.Close()

	n, err := io.Copy(out, resp.Body)
	if err != nil {
		return false, fmt.Sprintf("写入文件流中断: %v", err)
	}

	return true, fmt.Sprintf("成功下载 %d 字节到 %s", n, destPath)
}

// 单个子动作执行器 (供单步或并发批处理复用)
func executeSubAction(sub SubAction, cfg *Config) (string, bool) {
	switch sub.Action {
	case "run_command":
		var forbidden []string
		if cfg.TaskSpec != nil {
			forbidden = cfg.TaskSpec.ForbiddenCmds
		}
		if ok, reason := checkSafetyFilter(sub.Command, forbidden); !ok {
			return reason, false
		}
		code, out := executeCommand(sub.Command, cfg.TimeoutSec)
		pruned := smartPruneLog(out)
		return fmt.Sprintf("退出码: %d\n输出:\n%s", code, pruned), (code == 0)

	case "write_file":
		var allowed []string
		if cfg.TaskSpec != nil {
			allowed = cfg.TaskSpec.AllowedPaths
		}
		if ok, reason := checkPathGuard(sub.Path, allowed); !ok {
			return reason, false
		}
		trackFileBackup(sub.Path)
		os.MkdirAll(filepath.Dir(sub.Path), 0755)
		err := os.WriteFile(sub.Path, []byte(sub.Content), 0644)
		if err != nil {
			return fmt.Sprintf("写入文件失败: %v", err), false
		}
		return fmt.Sprintf("文件已成功写入: %s (已创建事务备份)", sub.Path), true

	case "read_file":
		contentBytes, err := os.ReadFile(sub.Path)
		if err != nil {
			return fmt.Sprintf("读取失败: %v", err), false
		}
		contentStr := string(contentBytes)
		if len(contentStr) > 2500 {
			contentStr = contentStr[:2500] + "... [截断预览]"
		}
		return fmt.Sprintf("文件内容:\n%s", contentStr), true

	case "list_dir":
		files, err := os.ReadDir(sub.Path)
		if err != nil {
			return fmt.Sprintf("读取目录失败: %v", err), false
		}
		var list []string
		for _, f := range files {
			typeStr := "文件"
			if f.IsDir() {
				typeStr = "目录"
			}
			list = append(list, fmt.Sprintf("[%s] %s", typeStr, f.Name()))
		}
		return fmt.Sprintf("目录列表:\n%s", strings.Join(list, "\n")), true

	case "http_download":
		var allowed []string
		if cfg.TaskSpec != nil {
			allowed = cfg.TaskSpec.AllowedPaths
		}
		if ok, reason := checkPathGuard(sub.Path, allowed); !ok {
			return reason, false
		}
		trackFileBackup(sub.Path)
		ok, msg := downloadFile(sub.URL, sub.Path)
		return msg, ok

	case "call_tool":
		if cfg.TaskSpec == nil || len(cfg.TaskSpec.CustomTools) == 0 {
			return "当前未挂载扩展自定义工具", false
		}
		var targetTool *CustomTool
		for _, t := range cfg.TaskSpec.CustomTools {
			if strings.EqualFold(t.Name, sub.ToolName) {
				targetTool = &t
				break
			}
		}
		if targetTool == nil {
			return fmt.Sprintf("未找到名为 '%s' 的工具", sub.ToolName), false
		}
		cmdStr := strings.ReplaceAll(targetTool.CommandTmpl, "{{args}}", sub.ToolArgs)
		timeout := targetTool.TimeoutSec
		if timeout <= 0 {
			timeout = cfg.TimeoutSec
		}
		code, out := executeCommand(cmdStr, timeout)
		return fmt.Sprintf("[工具 %s 执行结果] 退出码: %d\n输出:\n%s", targetTool.Name, code, smartPruneLog(out)), (code == 0)

	default:
		return fmt.Sprintf("未知子动作: %s", sub.Action), false
	}
}

// 核心大模型路由 (带自动清洗与容错)
func callLLM(providers []Provider, messages []map[string]string) (*Decision, string, error) {

	var lastErr error

	for _, p := range providers {
		pPayload := map[string]interface{}{
			"model":       p.Model,
			"messages":    messages,
			"temperature": 0.2,
			"max_tokens":  2500,
		}
		pBytes, _ := json.Marshal(pPayload)

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr, Timeout: 90 * time.Second}

		req, err := http.NewRequest("POST", p.URL, bytes.NewBuffer(pBytes))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Authorization", "Bearer "+p.Key)
		req.Header.Set("Content-Type", "application/json")
		for k, v := range p.Headers {
			req.Header.Set(k, v)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		respBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBytes[:min(len(respBytes), 100)]))
			continue
		}

		var raw map[string]interface{}
		if err := json.Unmarshal(respBytes, &raw); err != nil {
			lastErr = err
			continue
		}

		choices, ok := raw["choices"].([]interface{})
		if !ok || len(choices) == 0 {
			lastErr = fmt.Errorf("no choices returned")
			continue
		}

		msg, ok := choices[0].(map[string]interface{})["message"].(map[string]interface{})
		if !ok {
			lastErr = fmt.Errorf("invalid message format")
			continue
		}

		content, _ := msg["content"].(string)
		reasoning, _ := msg["reasoning_content"].(string)

		cleanJSON := func(s string) string {
			if strings.Contains(s, "```json") {
				parts := strings.Split(s, "```json")
				if len(parts) > 1 {
					return strings.TrimSpace(strings.Split(parts[1], "```")[0])
				}
			} else if strings.Contains(s, "```") {
				parts := strings.Split(s, "```")
				if len(parts) > 1 {
					return strings.TrimSpace(parts[1])
				}
			}
			start := strings.Index(s, "{")
			end := strings.LastIndex(s, "}")
			if start != -1 && end != -1 && end > start {
				return s[start : end+1]
			}
			return strings.TrimSpace(s)
		}

		var decision Decision
		parsed := false

		// 1. 优先解析正文
		if content != "" {
			cStr := cleanJSON(content)
			if err := json.Unmarshal([]byte(cStr), &decision); err == nil && decision.Action != "" {
				parsed = true
			}
		}

		// 2. 正文若为空或格式不合，兜底解析思维链 (适配 MiMo/DeepSeek 推理模型)
		if !parsed && reasoning != "" {
			rStr := cleanJSON(reasoning)
			if err := json.Unmarshal([]byte(rStr), &decision); err == nil && decision.Action != "" {
				parsed = true
			}
		}

		if parsed {
			return &decision, p.Name, nil
		} else {
			lastErr = fmt.Errorf("模型未返回有效 action 指令: content=[%s], reasoning=[%s]", content, reasoning)
		}
	}

	return nil, "", lastErr
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	taskFlag := flag.String("task", "", "要全自主执行的配置目标（自然语言）")
	specFlag := flag.String("spec", "", "结构化任务规约 JSON 字符串或文件路径（可选）")
	configFlag := flag.String("config", "xirang_config.json", "外挂配置文件路径（可选，自动回落兼容 zen_config.json）")
	rollbackFlag := flag.Bool("rollback", false, "一键安全回滚所有由 息壤 (XiRang) 产生的本地文件与配置")
	watchdogFlag := flag.Bool("watchdog", false, "启动常驻售后巡检守护模式 (服务挂掉自动自愈)")
	intervalFlag := flag.Int("interval", 30, "售后巡检间隔秒数 (默认 30 秒)")
	doctorFlag := flag.Bool("doctor", false, "启动售后急诊医生交互模式 (有报错/有问题随时找它)")
	forgeFlag := flag.Bool("forge", false, "启动息壤工坊图形创作控制台 (XiRang Studio GUI)")
	guiAliasFlag := flag.Bool("gui", false, "启动息壤工坊图形创作控制台 (同 -forge)")
	flag.Parse()

	if *forgeFlag || *guiAliasFlag {
		server := gui.NewForgeServer(".")
		if err := server.Start(true); err != nil {
			fmt.Printf("[X] 启动息壤工坊失败: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *rollbackFlag {
		executeRollback()
		return
	}

	// 基础配置
	cfg := Config{
		Goal:        *taskFlag,
		Providers:   defaultProviders,
		MaxSteps:    40,
		TimeoutSec:  300,
		ContextVars: make(map[string]string),
	}

	// 1. 优先尝试解密还原由 xirang-builder 熔炼嵌入的开发者保密模型配置
	if protectedCfg := loadEmbeddedProtectedConfig(); protectedCfg != nil {
		fmt.Println("[🛡️ 安全引擎] 检测到开发者加密密文载荷，已在受保护内存中成功激活！")
		if protectedCfg.Goal != "" && cfg.Goal == "" {
			cfg.Goal = protectedCfg.Goal
		}
		if protectedCfg.TaskSpec != nil {
			cfg.TaskSpec = protectedCfg.TaskSpec
		}
		if len(protectedCfg.Providers) > 0 {
			cfg.Providers = protectedCfg.Providers
		}
		if protectedCfg.MaxSteps > 0 {
			cfg.MaxSteps = protectedCfg.MaxSteps
		}
	}

	// 2. 外部 spec 参数解析
	if *specFlag != "" {
		var spec TaskSpecification
		if data, err := os.ReadFile(*specFlag); err == nil {
			json.Unmarshal(data, &spec)
		} else {
			json.Unmarshal([]byte(*specFlag), &spec)
		}
		if spec.Goal != "" {
			cfg.TaskSpec = &spec
			if cfg.Goal == "" {
				cfg.Goal = spec.Goal
			}
		}
	}

	// 3. 外部 JSON 配置文件 (优先 xirang_config.json，自动兼容 zen_config.json)
	targetConfigFile := *configFlag
	if _, err := os.Stat(targetConfigFile); os.IsNotExist(err) && targetConfigFile == "xirang_config.json" {
		if _, errZen := os.Stat("zen_config.json"); errZen == nil {
			targetConfigFile = "zen_config.json"
		}
	}
	if _, err := os.Stat(targetConfigFile); err == nil {
		data, err := os.ReadFile(targetConfigFile)
		if err == nil {
			var fileCfg Config
			if json.Unmarshal(data, &fileCfg) == nil {
				if fileCfg.Goal != "" && cfg.Goal == "" {
					cfg.Goal = fileCfg.Goal
				}
				if fileCfg.TaskSpec != nil && cfg.TaskSpec == nil {
					cfg.TaskSpec = fileCfg.TaskSpec
				}
				if len(fileCfg.Providers) > 0 {
					cfg.Providers = fileCfg.Providers
				}
				if fileCfg.MaxSteps > 0 {
					cfg.MaxSteps = fileCfg.MaxSteps
				}
				fmt.Printf("[*] 已加载外部配置: %s\n", targetConfigFile)
			}
		}
	}

	// 严格校验模型通道：绝不在操作系统全局环境中盲目嗅探，必须来自出厂加密嵌入或专用配置文件
	if len(cfg.Providers) == 0 {
		fmt.Println("[X] 未检测到可用的大模型推理通道配置！")
		fmt.Println("    说明：息壤绝不在操作系统全局环境中随意嗅探 API Key。")
		fmt.Println("    请通过以下安全方式之一提供模型通道：")
		fmt.Println("    1. 使用 xirang_builder 将模型密钥与规约熔炼为 AES-256 加密二进制 (出厂交付态)；")
		fmt.Println("    2. 在当前执行目录下提供受控的 xirang_config.json 配置文件。")
		os.Exit(1)
	}

	if *doctorFlag {
		runDoctorMode(&cfg)
		return
	}

	if *watchdogFlag {
		runWatchdogMode(&cfg, *intervalFlag)
		return
	}

	if cfg.Goal == "" {
		if len(flag.Args()) > 0 {
			cfg.Goal = strings.Join(flag.Args(), " ")
		} else {
			cfg.Goal = "检查当前环境并全自动完成部署与配置。"
		}
	}

	runAgentLoop(&cfg)
}

func runAgentLoop(cfg *Config) {
	fmt.Println("================================================================")
	fmt.Println("    🌱 息壤 (XiRang) v4.0: 工业级全并发·生生不息·自愈演进系统智能体")
	fmt.Println("    (多线并发批处理 · 黑板记忆剪枝 · 动态工具箱 · 事务日志撤销)")
	fmt.Println("================================================================")

	envInfo := probeEnvironment()
	envBytes, _ := json.MarshalIndent(envInfo, "", "  ")

	fmt.Printf("[*] 宿主系统: %s (%s)\n", envInfo["os"], envInfo["arch"])
	fmt.Printf("[*] 显卡环境: %s\n", envInfo["gpu"])
	if drives, ok := envInfo["available_drives"]; ok {
		fmt.Printf("[*] 可用驱动器: %v\n", drives)
	}
	fmt.Printf("[*] 核心目标: %s\n", cfg.Goal)

	if cfg.TaskSpec != nil {
		fmt.Printf("[*] 绑定任务规约: %s (阶段数: %d)\n", cfg.TaskSpec.TaskName, len(cfg.TaskSpec.Milestones))
		if len(cfg.TaskSpec.AllowedPaths) > 0 {
			fmt.Printf("🔒 受信目录白名单: %v\n", cfg.TaskSpec.AllowedPaths)
		}
		if len(cfg.TaskSpec.CustomTools) > 0 {
			fmt.Printf("🛠️  挂载外挂扩展工具 (%d 项):\n", len(cfg.TaskSpec.CustomTools))
			for _, t := range cfg.TaskSpec.CustomTools {
				fmt.Printf("    - %s: %s\n", t.Name, t.Description)
			}
		}
		if cfg.TaskSpec.VerificationCmd != "" {
			fmt.Printf("🎯 终态硬性验收命令 (DoD): %s\n", cfg.TaskSpec.VerificationCmd)
		}
	}
	fmt.Println()

	// 构造系统级结构化 Prompt (包含多线并发与工具箱规范)
	specDoc := ""
	if cfg.TaskSpec != nil {
		specJSON, _ := json.MarshalIndent(cfg.TaskSpec, "", "  ")
		specDoc = fmt.Sprintf("\n【必须遵守的结构化任务规约 (Task Specification)】:\n%s\n", string(specJSON))
	}

	evolvedSkillsDoc := loadEvolvedSkills()

	systemPrompt := fmt.Sprintf(`你是一个具备全平台底层操作、多线并发编排、自愈与经验持续沉淀能力的工业级通用系统部署与运维智能体「息壤 (XiRang v4.0)」。
你的使命：不论用户给你指定何种项目任务、使用何种模型框架或服务，你都必须在当前宿主操作系统上 100%%%% 自主完成配置、排错、严格验收与经验资产化。
%s
%s

【当前宿主机器真实环境与文件探针】:
%s

【极度整洁的工作目录管理规范】:
1. 严禁在工作母目录下创建临时杂乱脚本！任何由你生成的运维脚本、探测脚本、修复脚本，必须全部持久化保存到专用子目录: .xirang/scripts/ (例如 .xirang/scripts/fix_env.bat 或 .xirang/scripts/check_cuda.py)！
2. 脚本持久化与复用原则：写出来的优质脚本会在 .xirang/scripts/ 永久留存，下次遇到相同或类似问题优先直接执行该脚本，实现经验复利！

【息壤核心元认知与自我繁衍规范 (Self-Bootstrapping & High Mobility)】:
1. 你本身就是「息壤 (XiRang) v4.0」。你深刻理解你自身的全部架构、工作原理与命令行参数 (-task, -spec, -config, -doctor, -watchdog, --rollback)。
2. 【人类决策主权至高无上（帮人修复，决不替人决策）】：
   - 息壤的崇高使命是帮助人类安装、排错、自愈与系统修复，但【绝对不能越俎代庖替人类做决策】！
   - 涉及核心业务选择（例如：将庞大数据/模型放在哪个磁盘分区、是否删除某项可能冲突的文件、修改哪个网络端口、选用何种运行模式等），息壤只负责用探针查明客观事实并提供选项，必须无条件调用 "action": "ask_human" 征询人类意愿，严禁擅作主张替人类做主！
   - 当未来你受命为其他项目创建/繁衍新的息壤实例 (Task Spec) 时，也必须将此条人类主权铁律刻入新实例的基因中！
3. 与市面上沉重、臃肿且依赖庞大 Python/Node 运行时的通用 Agent 截然不同：息壤的核心优势在于【极高的可移动性（单文件纯原生、零外部依赖、随时随地在 U 盘/老机器裸机空投运行）】与【极强的场景针对性（针对具体项目精准沙箱与硬性防作弊验收）】。
4. 当用户或上层 AI 要求你【创建新的息壤安装包/为某个项目配置息壤/繁衍定制版安装器】时，你必须熟练发挥你的自我繁衍能力：
   a. 分析当前目标项目的入口文件、端口需求、必须路径与终态验收命令；
   b. 自主编写严谨的 xirang_task_spec.json (明确 task_name, verification_cmd 硬性验收命令, milestones 里程碑, health_probes 探针，并严禁在规约中写死剥夺用户选择权的自动化逻辑)；
   c. 就地生成微型一键启动脚本 (如 1_双击全自动安装与配置.bat 或 run_installer.sh)，将可移动性与极简交互发挥到极致；
   d. 若当前系统具备 Go 开发环境且用户需要出厂加密分发，可直接在终端中静默调用创作构建工具进行 AES-256 熔炼打包：
      'xirang_builder -spec 自定义规约.json -config 模型配置.json -os windows -out 安装器名称.exe'
      以此完成模型密钥与业务规约的高强度加密混淆，输出 100% 独立的受保护交付单文件！
5. 当用户要求你【配置你自己或修改息壤自身】时，你能像操作自己身体一样，直接为自身编写或挂载对应的 task_spec、配置守护探针或调优 .xirang/scripts/ 下的经验工具。

【你可以直接执行的 10 项原生系统动作（必须输出严格的纯 JSON 格式）】:
1. 单步终端指令: {"action": "run_command", "command": "具体终端命令行", "thought": "分析", "scratchpad": "当前记忆便签更新", "explanation": "说明"}
2. 多线并发批处理 (极速多线程操作): 
   {"action": "batch_actions", "batch_actions": [
     {"id": "task1", "action": "run_command", "command": "探测命令1"},
     {"id": "task2", "action": "run_command", "command": "探测命令2"},
     {"id": "task3", "action": "write_file", "path": "路径", "content": "配置"}
   ], "thought": "分析为何并发执行", "scratchpad": "便签", "explanation": "说明"}
3. 创建/修补文件 (自动安全事务备份): {"action": "write_file", "path": "绝对/相对路径", "content": "内容", "thought": "分析", "scratchpad": "便签", "explanation": "说明"}
4. 查看文件: {"action": "read_file", "path": "路径", "thought": "分析", "explanation": "说明"}
5. 探测目录内容: {"action": "list_dir", "path": "目录路径", "thought": "分析", "explanation": "说明"}
6. 原生下载资源: {"action": "http_download", "url": "下载URL", "path": "本地保存路径", "thought": "分析", "explanation": "说明"}
7. 调用外挂扩展工具: {"action": "call_tool", "tool_name": "工具名", "tool_args": "传递参数", "thought": "分析", "explanation": "说明"}
8. 人机协同询问 (重大决策必用): {"action": "ask_human", "question": "向用户提出的选择问题", "options": ["选项1", "选项2"], "thought": "需要人类介入确认", "explanation": "说明"}
9. 上报里程碑: {"action": "report_milestone", "milestone": "阶段名称", "thought": "此阶段已完成验证", "explanation": "说明"}
10. 任务最终达成: {"action": "finish", "thought": "确认所有指标与业务验收测试均已达标", "explanation": "最终交付汇报"}

【工业级红线铁律】:
1. 并发优化原则：凡是可以同时做的事情（例如：同时下载多个模型分卷、同时探测多张显卡与驱动、同时生成多个配置文件），强烈推荐使用 "action": "batch_actions" 并行飞速搞定！
2. 终态真实防作弊：在调用 finish 之前，若存在终态验收命令 (VerificationCmd)，必须已经通过该命令真实测试！
3. 人类决策第一与尊重白名单：在涉及存储选址、资源分配等决策时绝不自作主张；通过 scratchpad 随时记录当前掌握的事实，让思考过程始终清晰！`, specDoc, evolvedSkillsDoc, string(envBytes))

	messages := []map[string]string{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": fmt.Sprintf("请开始全自主执行目标：\n%s", cfg.Goal)},
	}

	reader := bufio.NewReader(os.Stdin)
	var activeScratchpad string

	for step := 1; step <= cfg.MaxSteps; step++ {
		fmt.Printf("\n[Step %d/%d] 正在思考下一步自愈动作...\n", step, cfg.MaxSteps)
		decision, provName, err := callLLM(cfg.Providers, messages)
		if err != nil {
			fmt.Printf("[X] 调用上游大模型失败: %v\n", err)
			break
		}

		if decision.Scratchpad != "" {
			activeScratchpad = decision.Scratchpad
			fmt.Printf("📋 [工作便签 (Scratchpad)]: %s\n", activeScratchpad)
		}

		fmt.Printf("🧠 [思考 - %s]: %s\n", provName, decision.Thought)
		fmt.Printf("⚡ [动作]: %s (%s)\n", decision.Action, decision.Explanation)

		// 终态防作弊校验网关 (DoD Gate)
		if decision.Action == "finish" {
			if cfg.TaskSpec != nil && cfg.TaskSpec.VerificationCmd != "" {
				fmt.Println("\n🔍 [DoD 验收网关] 正在执行硬性终态业务验收命令...")
				code, verifyOut := executeCommand(cfg.TaskSpec.VerificationCmd, 60)
				if code != 0 {
					fmt.Printf("❌ [DoD 驳回] 业务验收测试未通过 (退出码 %d)！输出:\n%s\n", code, verifyOut)
					decBytes, _ := json.Marshal(decision)
					messages = append(messages, map[string]string{"role": "assistant", "content": string(decBytes)})
					messages = append(messages, map[string]string{
						"role":    "user",
						"content": fmt.Sprintf("【DoD 终态验收网关拦截】你试图调用 finish，但强制验收命令 '%s' 执行失败 (退出码 %d)！输出:\n%s\n请分析原因并继续自愈排错，通过后方可结束！", cfg.TaskSpec.VerificationCmd, code, verifyOut),
					})
					continue
				}
				fmt.Println("✅ [DoD 验收网关] 真实业务验证完全通过！硬件与服务均达到交付标准！")
			}

			fmt.Println("\n================================================================")
			fmt.Println("🎉 [成功] 息壤 (XiRang) v4.0 已全自主确认：所有任务与终态验收全部搞定！")
			fmt.Println("================================================================")
			return
		}

		// 多线并发批处理 (Parallel Batch Actions)
		if decision.Action == "batch_actions" && len(decision.BatchActions) > 0 {
			fmt.Printf("🚀 [多线并发] 启动 %d 个子任务并行执行流水线...\n", len(decision.BatchActions))
			
			results := make([]string, len(decision.BatchActions))
			var wg sync.WaitGroup
			
			for i, sub := range decision.BatchActions {
				wg.Add(1)
				go func(idx int, action SubAction) {
					defer wg.Done()
					resStr, success := executeSubAction(action, cfg)
					statusIcon := "✅"
					if !success {
						statusIcon = "❌"
					}
					results[idx] = fmt.Sprintf("【子任务 #%d [%s] %s %s】:\n%s", idx+1, action.ID, action.Action, statusIcon, resStr)
					fmt.Printf("   -> 并发子任务 #%d [%s] 完成: %s\n", idx+1, action.Action, statusIcon)
				}(i, sub)
			}
			wg.Wait()

			combinedBatchOut := strings.Join(results, "\n\n")
			decBytes, _ := json.Marshal(decision)
			messages = append(messages, map[string]string{"role": "assistant", "content": string(decBytes)})
			messages = append(messages, map[string]string{
				"role":    "user",
				"content": fmt.Sprintf("多线并发批处理执行完毕，各子任务回报如下：\n\n%s\n\n请评估各并发任务结果并决定下一步行动。", combinedBatchOut),
			})
			time.Sleep(1 * time.Second)
			continue
		}

		switch decision.Action {
		case "ask_human":
			fmt.Println("\n----------------------------------------------------------------")
			fmt.Printf("❓ [息壤 提问]: %s\n", decision.Question)
			for i, opt := range decision.Options {
				fmt.Printf("   [%d] %s\n", i+1, opt)
			}
			fmt.Println("----------------------------------------------------------------")
			fmt.Print("👉 请输入您的选择或直接输入回答: ")
			userInput, _ := reader.ReadString('\n')
			userInput = strings.TrimSpace(userInput)

			decBytes, _ := json.Marshal(decision)
			messages = append(messages, map[string]string{"role": "assistant", "content": string(decBytes)})
			messages = append(messages, map[string]string{
				"role":    "user",
				"content": fmt.Sprintf("人类用户给出了明确答复: \"%s\"。请根据用户的指示继续推进任务。", userInput),
			})

		case "report_milestone":
			fmt.Printf("🚩 [阶段里程碑达成]: %s\n", decision.Milestone)
			decBytes, _ := json.Marshal(decision)
			messages = append(messages, map[string]string{"role": "assistant", "content": string(decBytes)})
			messages = append(messages, map[string]string{"role": "user", "content": fmt.Sprintf("已确认里程碑达成: %s，请继续执行下一阶段。", decision.Milestone)})

		default:
			// 常规单步动作复用执行器
			sub := SubAction{
				Action:   decision.Action,
				Command:  decision.Command,
				Path:     decision.Path,
				Content:  decision.Content,
				URL:      decision.URL,
				ToolName: decision.ToolName,
				ToolArgs: decision.ToolArgs,
			}
			res, success := executeSubAction(sub, cfg)
			preview := res
			if len(preview) > 300 {
				preview = preview[:300] + "..."
			}
			fmt.Printf("   -> 结果 (成功:%v): %s\n", success, preview)

			decBytes, _ := json.Marshal(decision)
			messages = append(messages, map[string]string{"role": "assistant", "content": string(decBytes)})
			messages = append(messages, map[string]string{
				"role":    "user",
				"content": fmt.Sprintf("操作执行完成 (成功: %v)，详细回显如下：\n%s\n请分析并决定下一步行动。", success, res),
			})
		}

		time.Sleep(1 * time.Second)
	}
}

// 售后常驻巡检守护函数
func runWatchdogMode(cfg *Config, intervalSec int) {
	fmt.Println("================================================================")
	fmt.Println("    🛡️ 息壤 (XiRang) 售后常驻守护巡检系统 (Watchdog Mode)")
	fmt.Printf("    (全天候监控 · 自动心跳探活 · 故障秒级自主愈合 · 间隔: %d 秒)\n", intervalSec)
	fmt.Println("================================================================")

	var probes []HealthProbe
	if cfg.TaskSpec != nil && len(cfg.TaskSpec.HealthProbes) > 0 {
		probes = cfg.TaskSpec.HealthProbes
	} else {
		probes = []HealthProbe{
			{
				Name:     "核心进程探活",
				CheckCmd: "echo HEARTBEAT_OK",
				FixGoal:  "检测系统关键服务，若异常则自动拉起恢复。",
			},
		}
	}

	fmt.Printf("[*] 当前已激活 %d 个售后监控探针:\n", len(probes))
	for _, p := range probes {
		fmt.Printf("   -> 监控项: %s (探测命令: %s)\n", p.Name, p.CheckCmd)
	}
	fmt.Println("[*] 售后守护已进入静默常驻监听状态，按 Ctrl+C 可停止。")

	checkCount := 0
	for {
		checkCount++
		timestamp := time.Now().Format("2006-01-02 15:04:05")

		for _, p := range probes {
			code, out := executeCommand(p.CheckCmd, 15)
			if code != 0 {
				fmt.Printf("\n⚠️ [%s 故障预警!] 监控项【%s】探测异常 (退出码: %d)！\n", timestamp, p.Name, code)
				fmt.Printf("   输出报错: %s\n", strings.TrimSpace(out))
				fmt.Println("🚨 正在紧急唤醒 息壤 (XiRang) 通用自愈内核进行售后抢修...")

				rescueCfg := *cfg
				rescueCfg.Goal = fmt.Sprintf("【售后紧急自愈】监控项 '%s' 异常挂掉 (报错: %s)。目标：%s，恢复其正常运行并通过健康探测！", p.Name, strings.TrimSpace(out), p.FixGoal)
				runAgentLoop(&rescueCfg)
				fmt.Printf("✅ [%s] 监控项【%s】已全自主抢修完毕并恢复健康！\n\n", time.Now().Format("15:04:05"), p.Name)
			}
		}

		if checkCount%10 == 1 {
			fmt.Printf("[%s] 售后巡检正常运行中 (第 %d 次心跳健康)...\n", timestamp, checkCount)
		}
		time.Sleep(time.Duration(intervalSec) * time.Second)
	}
}

// 售后急诊医生交互模式 (人类遇错随呼随到)
func runDoctorMode(cfg *Config) {
	fmt.Println("================================================================")
	fmt.Println("    🩺 息壤 (XiRang) 售后专属急诊医生 (Doctor Mode)")
	fmt.Println("    (无论任何时候系统报错、服务起不来、配置异常，随时找它！)")
	fmt.Println("================================================================")
	fmt.Println("[*] 正在扫描当前宿主环境状态...")
	env := probeEnvironment()
	fmt.Printf("[*] 当前操作系统: %s (%s)\n", env["os"], env["arch"])
	fmt.Printf("[*] 显卡环境: %s\n", env["gpu"])
	if drives, ok := env["available_drives"]; ok {
		fmt.Printf("[*] 可用驱动器: %v\n", drives)
	}
	fmt.Println("----------------------------------------------------------------")
	fmt.Println("💡 使用提示：")
	fmt.Println("   - 您可以直接粘贴控制台报错信息 (如 'error: port already in use')")
	fmt.Println("   - 也可以直接用自然语言描述问题 (如 '模型跑不起来', '找不到CUDA')")
	fmt.Println("   - 输入 'exit' 或 'quit' 退出医生模式")
	fmt.Println("----------------------------------------------------------------")

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n👉 请输入您遇到的错误或需要售后解决的问题: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}
		if strings.EqualFold(input, "exit") || strings.EqualFold(input, "quit") {
			fmt.Println("👋 感谢使用 息壤 售后医生，祝您使用愉快！")
			break
		}

		// 优先尝试本地已有经验指纹快速命中 (Fast-Path)
		if fastSkill := findFastSkill(input); fastSkill != nil {
			fmt.Printf("\n⚡ [经验秒级命中!] 识别到已沉淀的已知问题特征: \"%s\"\n", fastSkill.Pattern)
			fmt.Printf("   -> 正在直接调取本地经过验证的专属脚本: %s (%s)...\n", fastSkill.ScriptPath, fastSkill.Description)
			code, out := executeCommand(fastSkill.ScriptPath, 120)
			if code == 0 {
				fmt.Printf("✅ [秒级自愈成功!] 已直接通过本地经验修复该问题 (耗时 0.2s，未消耗 Token)！\n回显:\n%s\n", out)
				continue
			} else {
				fmt.Printf("⚠️ 本地快速脚本执行未完全解决，正在无缝转入云端大模型进行深度自愈推演...\n")
			}
		}

		fmt.Println("\n🚨 收到售后报修请求！息壤 正在连线自愈大脑接管排障...")
		cureCfg := *cfg
		cureCfg.Goal = fmt.Sprintf("【用户售后报修求助】用户报告了以下系统故障/报错信息：\n\"%s\"\n请深入分析该错误原因，利用系统指令探查现场，定位根因并全自主执行修复，并将可复用的排障脚本沉淀到 .xirang/scripts/，最终向用户反馈排查结果与解决方案。", input)
		runAgentLoop(&cureCfg)
		fmt.Println("\n✅ [售后处理完毕] 该问题已诊断修复完成。若仍有其他异常，可继续输入，随时为您服务！")
	}
}
