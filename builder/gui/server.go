package gui

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
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type ForgeServer struct {
	ProjectRoot string
	Port        int
}

type InspectRequest struct {
	Path string `json:"path"`
}

type FileItem struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
}

type InspectResponse struct {
	Path        string     `json:"path"`
	Exists      bool       `json:"exists"`
	TotalFiles  int        `json:"total_files"`
	ProjectTags []string   `json:"project_tags"`
	KeyFiles    []string   `json:"key_files"`
	TreeSnippet string     `json:"tree_snippet"`
	Items       []FileItem `json:"items"`
}

type AIAnalyzeRequest struct {
	FolderPath  string   `json:"folder_path"`
	ApiUrl      string   `json:"api_url"`
	ApiKey      string   `json:"api_key"`
	Model       string   `json:"model"`
	KeyFiles    []string `json:"key_files"`
	TreeSnippet string   `json:"tree_snippet"`
	UserIntent  string   `json:"user_intent"`
}

type TaskSpec struct {
	TaskName        string      `json:"task_name"`
	Goal            string      `json:"goal"`
	AllowedPaths    []string    `json:"allowed_paths"`
	ForbiddenCmds   []string    `json:"forbidden_cmds"`
	Milestones      []Milestone `json:"milestones"`
	VerificationCmd string      `json:"verification_cmd"`
	StrictRules     []string    `json:"strict_rules"`
}

type Milestone struct {
	Name        string `json:"name"`
	VerifyCmd   string `json:"verify_cmd"`
	Description string `json:"description"`
}

type InceptionRequest struct {
	TargetFolder string   `json:"target_folder"`
	TaskSpec     TaskSpec `json:"task_spec"`
	ModelConfig  struct {
		Url   string `json:"url"`
		Key   string `json:"key"`
		Model string `json:"model"`
	} `json:"model_config"`
}

type BuildProtectedRequest struct {
	TargetOS    string   `json:"target_os"`
	TargetArch  string   `json:"target_arch"`
	OutputName  string   `json:"output_name"`
	SecretKey   string   `json:"secret_key"`
	TaskSpec    TaskSpec `json:"task_spec"`
	ModelConfig struct {
		Url   string `json:"url"`
		Key   string `json:"key"`
		Model string `json:"model"`
	} `json:"model_config"`
}

func NewForgeServer(projectRoot string) *ForgeServer {
	return &ForgeServer{
		ProjectRoot: projectRoot,
		Port:        18888,
	}
}

func (s *ForgeServer) Start(openBrowser bool) error {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", s.Port))
	if err != nil {
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return err
		}
	}
	s.Port = listener.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/inspect", s.handleInspect)
	mux.HandleFunc("/api/ai_analyze", s.handleAIAnalyze)
	mux.HandleFunc("/api/incept", s.handleIncept)
	mux.HandleFunc("/api/build", s.handleBuild)

	url := fmt.Sprintf("http://127.0.0.1:%d", s.Port)
	fmt.Printf("\n================================================================\n")
	fmt.Printf("🌱 息壤图形创作工坊 (XiRang Studio) 已就绪！\n")
	fmt.Printf("🌐 本地图形控制台地址: %s\n", url)
	fmt.Printf("================================================================\n\n")

	if openBrowser {
		go func() {
			time.Sleep(400 * time.Millisecond)
			openBrowserURL(url)
		}()
	}

	return http.Serve(listener, mux)
}

func openBrowserURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

func (s *ForgeServer) handleInspect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var req InspectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	targetPath := strings.TrimSpace(req.Path)
	if targetPath == "" {
		targetPath = "."
	}
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fi, err := os.Stat(absPath)
	if err != nil || !fi.IsDir() {
		resp := InspectResponse{
			Path:   absPath,
			Exists: false,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var items []FileItem
	var keyFiles []string
	var tags []string
	var treeLines []string
	tagMap := make(map[string]bool)

	for _, e := range entries {
		info, _ := e.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		items = append(items, FileItem{
			Name:  e.Name(),
			IsDir: e.IsDir(),
			Size:  size,
		})

		nameLower := strings.ToLower(e.Name())
		if e.IsDir() {
			treeLines = append(treeLines, fmt.Sprintf("[目录] %s/", e.Name()))
			if nameLower == ".git" {
				tagMap["Git 仓库"] = true
			} else if nameLower == "node_modules" {
				tagMap["Node.js 已就绪"] = true
			} else if nameLower == ".venv" || nameLower == "venv" {
				tagMap["Python 虚拟环境"] = true
			}
		} else {
			treeLines = append(treeLines, fmt.Sprintf("[文件] %s (%d 字节)", e.Name(), size))
			switch nameLower {
			case "package.json":
				tagMap["Node.js / NPM"] = true
				keyFiles = append(keyFiles, e.Name())
			case "requirements.txt", "pyproject.toml", "setup.py":
				tagMap["Python 项目"] = true
				keyFiles = append(keyFiles, e.Name())
			case "go.mod":
				tagMap["Golang 工程"] = true
				keyFiles = append(keyFiles, e.Name())
			case "dockerfile", "docker-compose.yml", "docker-compose.yaml":
				tagMap["Docker 容器化"] = true
				keyFiles = append(keyFiles, e.Name())
			case "pom.xml", "build.gradle":
				tagMap["Java / JVM"] = true
				keyFiles = append(keyFiles, e.Name())
			case "cargo.toml":
				tagMap["Rust"] = true
				keyFiles = append(keyFiles, e.Name())
			case "makefile":
				tagMap["Makefile"] = true
				keyFiles = append(keyFiles, e.Name())
			}
			if strings.HasSuffix(nameLower, ".bat") || strings.HasSuffix(nameLower, ".ps1") {
				tagMap["Windows 批处理/PowerShell"] = true
				keyFiles = append(keyFiles, e.Name())
			}
			if strings.HasSuffix(nameLower, ".sh") {
				tagMap["Shell 脚本"] = true
				keyFiles = append(keyFiles, e.Name())
			}
		}
	}

	for k := range tagMap {
		tags = append(tags, k)
	}

	resp := InspectResponse{
		Path:        absPath,
		Exists:      true,
		TotalFiles:  len(items),
		ProjectTags: tags,
		KeyFiles:    keyFiles,
		TreeSnippet: strings.Join(treeLines, "\n"),
		Items:       items,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *ForgeServer) handleAIAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var req AIAnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.ApiKey == "" {
		http.Error(w, "缺少 API Key，请先配置模型凭据", http.StatusBadRequest)
		return
	}
	if req.ApiUrl == "" {
		req.ApiUrl = "https://api.openai.com/v1"
	}
	if req.Model == "" {
		req.Model = "gpt-4o"
	}

	prompt := fmt.Sprintf(`你是一个顶级全栈系统架构师与息壤（XiRang）守护设计专家。
人类用户选定了目标工作文件夹【%s】。
该文件夹包含以下关键文件与目录特征：
- 关键文件: %v
- 文件快照样例:
%s

人类对该任务的补充说明/诉求意图:
%s

【铁律】“帮人修复，决不替人决策”：息壤帮助人类解决依赖、自愈环境和验收，但严禁擅作主张替代人类核心决策。
请分析：
1. 这个文件夹装的是什么？（例如：AI 推理服务、Web 服务、离线迁移包等）
2. 息壤应该如何控制并保护它？
3. 如何设计严格、防作弊的终态验证命令 (VerificationCmd)？

请务必只返回合法的纯 JSON 字符串（不要附带任何 markdown 块或附加解释）：
{
  "task_name": "项目名称或任务标题",
  "goal": "详细的自主运维与自愈排错目标",
  "allowed_paths": ["目标路径白名单"],
  "forbidden_cmds": ["rm -rf /", "format", "del /f /s /q c:\\", "mkfs"],
  "verification_cmd": "硬性终态验收测试命令",
  "strict_rules": [
    "帮人修复，决不替人决策：重大改动与业务决策前必须调用 ask_human 征询人类确认",
    "严禁越界修改非白名单系统关键文件",
    "保证所有环境与脚本具有最大可移动性"
  ],
  "milestones": [
    {
      "name": "步骤一：探查底层运行环境与依赖",
      "verify_cmd": "探测命令",
      "description": "说明"
    },
    {
      "name": "步骤二：主程序或服务连通性与验收",
      "verify_cmd": "主体验收命令",
      "description": "说明"
    }
  ]
}`, req.FolderPath, req.KeyFiles, req.TreeSnippet, req.UserIntent)

	reqBody := map[string]interface{}{
		"model": req.Model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a professional system architect. You must respond with pure JSON only."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.2,
	}

	reqBytes, _ := json.Marshal(reqBody)
	url := strings.TrimRight(req.ApiUrl, "/") + "/chat/completions"
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBytes))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+req.ApiKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("API 请求失败: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("大模型接口返回异常 HTTP %d: %s", resp.StatusCode, string(bodyBytes)), http.StatusBadGateway)
		return
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(bodyBytes, &chatResp); err != nil || len(chatResp.Choices) == 0 {
		http.Error(w, "解析模型返回失败", http.StatusInternalServerError)
		return
	}

	rawContent := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	rawContent = strings.TrimPrefix(rawContent, "```json")
	rawContent = strings.TrimPrefix(rawContent, "```")
	rawContent = strings.TrimSuffix(rawContent, "```")
	rawContent = strings.TrimSpace(rawContent)

	var spec TaskSpec
	if err := json.Unmarshal([]byte(rawContent), &spec); err != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(rawContent))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(spec)
}

func (s *ForgeServer) handleIncept(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var req InceptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	targetDir := req.TargetFolder
	if targetDir == "" {
		http.Error(w, "未指定目标文件夹", http.StatusBadRequest)
		return
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("创建目录失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 1. 生成 xirang_task_spec.json
	specBytes, _ := json.MarshalIndent(req.TaskSpec, "", "  ")
	_ = os.WriteFile(filepath.Join(targetDir, "xirang_task_spec.json"), specBytes, 0644)

	// 2. 局部配置 xirang_config.json
	if req.ModelConfig.Key != "" {
		cfgData := map[string]interface{}{
			"providers": []map[string]interface{}{
				{
					"name":  "ForgeProvider",
					"url":   req.ModelConfig.Url,
					"key":   req.ModelConfig.Key,
					"model": req.ModelConfig.Model,
				},
			},
			"max_steps":   50,
			"timeout_sec": 300,
		}
		cfgBytes, _ := json.MarshalIndent(cfgData, "", "  ")
		_ = os.WriteFile(filepath.Join(targetDir, "xirang_config.json"), cfgBytes, 0600)
	}

	// 3. 复制预编译好的可执行体
	distDir := filepath.Join(s.ProjectRoot, "dist")
	_ = copyFile(filepath.Join(distDir, "xirang_win_x64.exe"), filepath.Join(targetDir, "xirang.exe"))
	_ = copyFile(filepath.Join(distDir, "xirang_mac_apple_silicon"), filepath.Join(targetDir, "xirang_mac"))
	_ = copyFile(filepath.Join(distDir, "xirang_linux_x64"), filepath.Join(targetDir, "xirang_linux"))

	// 4. 生成双击运行批处理与脚本
	batContent := `@echo off
chcp 65001 >nul
title 息壤 (XiRang) - 专属副本运行中
echo ================================================================
echo    🌱 息壤 (XiRang) 正在就地接管并自愈配置...
echo ================================================================
if exist xirang.exe (
    xirang.exe -spec xirang_task_spec.json
) else (
    echo [!] 未检测到 xirang.exe 执行文件，请将息壤放入本目录！
    pause
)
pause
`
	shContent := `#!/usr/bin/env bash
cd "$(dirname "$0")"
echo "🌱 息壤 (XiRang) 正在就地接管并自愈配置..."
if [ -f "./xirang_mac" ]; then
    chmod +x ./xirang_mac
    ./xirang_mac -spec xirang_task_spec.json
elif [ -f "./xirang_linux" ]; then
    chmod +x ./xirang_linux
    ./xirang_linux -spec xirang_task_spec.json
else
    echo "[!] 未找到 Linux/Mac 息壤二进制，请检查目录！"
fi
`
	_ = os.WriteFile(filepath.Join(targetDir, "1_双击启动息壤.bat"), []byte(batContent), 0755)
	_ = os.WriteFile(filepath.Join(targetDir, "run_xirang.sh"), []byte(shContent), 0755)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok","message":"已成功在目标文件夹就地植入息壤副本、定制规约与启动脚本！"}`))
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err == nil {
		_ = os.Chmod(dst, 0755)
	}
	return err
}

func (s *ForgeServer) handleBuild(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var req BuildProtectedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.SecretKey == "" {
		req.SecretKey = "xirang-forge-vault-secret"
	}
	if req.TargetOS == "" {
		req.TargetOS = "windows"
	}
	if req.TargetArch == "" {
		req.TargetArch = "amd64"
	}
	if req.OutputName == "" {
		if req.TargetOS == "windows" {
			req.OutputName = "xirang_custom.exe"
		} else {
			req.OutputName = "xirang_custom"
		}
	}

	buildConfig := map[string]interface{}{
		"default_goal": req.TaskSpec.Goal,
		"task_spec":    req.TaskSpec,
		"providers": []map[string]interface{}{
			{
				"name":  "ForgeEncryptedProvider",
				"url":   req.ModelConfig.Url,
				"key":   req.ModelConfig.Key,
				"model": req.ModelConfig.Model,
			},
		},
		"max_steps":   50,
		"timeout_sec": 300,
	}

	cfgBytes, _ := json.Marshal(buildConfig)
	encryptedHex, err := encryptPayload(cfgBytes, req.SecretKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("加密载荷失败: %v", err), http.StatusInternalServerError)
		return
	}

	loaderPath := filepath.Join(s.ProjectRoot, "embedded_config.go")
	code := fmt.Sprintf(`// Code generated by XiRang Studio Forge. DO NOT EDIT.
package main

const (
	EncryptedPayload = "%s"
	BuildSecret      = "%s"
)
`, encryptedHex, req.SecretKey)

	if err := os.WriteFile(loaderPath, []byte(code), 0644); err != nil {
		http.Error(w, fmt.Sprintf("写入嵌入代码失败: %v", err), http.StatusInternalServerError)
		return
	}

	distDir := filepath.Join(s.ProjectRoot, "dist_protected")
	_ = os.MkdirAll(distDir, 0755)
	outPath := filepath.Join(distDir, req.OutputName)

	cmd := exec.Command("go", "build", "-ldflags=-s -w", "-o", outPath, "main.go", "embedded_config.go")
	cmd.Dir = s.ProjectRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+req.TargetOS, "GOARCH="+req.TargetArch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		http.Error(w, fmt.Sprintf("交叉编译失败: %v\n%s", err, string(output)), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "ok",
		"output_path": outPath,
		"message":     fmt.Sprintf("恭喜！已成功熔炼 AES-256 高强度加密单文件独立安装体 -> %s", outPath),
	})
}

func encryptPayload(plainText []byte, secretKey string) (string, error) {
	keyHash := sha256.Sum256([]byte(secretKey))
	block, err := aes.NewCipher(keyHash[:])
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	cipherText := gcm.Seal(nonce, nonce, plainText, nil)
	return hex.EncodeToString(cipherText), nil
}
