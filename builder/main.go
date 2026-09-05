package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type Provider struct {
	Name    string            `json:"name"`
	URL     string            `json:"url"`
	Key     string            `json:"key"`
	Model   string            `json:"model"`
	Headers map[string]string `json:"headers"`
}

type Milestone struct {
	Name        string `json:"name"`
	VerifyCmd   string `json:"verify_cmd"`
	Description string `json:"description"`
	Done        bool   `json:"done"`
}

type CustomTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	CommandTmpl string `json:"command_tmpl"`
	TimeoutSec  int    `json:"timeout_sec"`
}

type TaskSpecification struct {
	TaskName        string       `json:"task_name"`
	Goal            string       `json:"goal"`
	AllowedPaths    []string     `json:"allowed_paths"`
	ForbiddenCmds   []string     `json:"forbidden_cmds"`
	Milestones      []Milestone  `json:"milestones"`
	VerificationCmd string       `json:"verification_cmd"`
	StrictRules     []string     `json:"strict_rules"`
	CustomTools     []CustomTool `json:"custom_tools"`
}

type BuildConfig struct {
	DefaultGoal string             `json:"default_goal"`
	TaskSpec    *TaskSpecification `json:"task_spec,omitempty"`
	Providers   []Provider         `json:"providers"`
	MaxSteps    int                `json:"max_steps"`
	TimeoutSec  int                `json:"timeout_sec"`
}

// AES-256-GCM 加密
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

func readLine(reader *bufio.Reader, prompt, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	return input
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("================================================================")
	fmt.Println("    🛡️ 息壤 编译器 (XiRang Builder v4.0): 工业级全并发·契约打包套件")
	fmt.Println("    (出厂预置多线并发能力、专属扩展工具箱、白名单沙箱、AES-256熔炼)")
	fmt.Println("================================================================")
	fmt.Println()

	var cfg BuildConfig
	cfg.MaxSteps = 40
	cfg.TimeoutSec = 300

	fmt.Println("【第一步：定义出厂固定核心使命与规范契约】")
	cfg.DefaultGoal = readLine(reader, "输入该 Agent 出厂默认使命", "全自动完成当前环境安装、服务编排、多线并发配置与终态验收。")

	var spec TaskSpecification
	spec.TaskName = readLine(reader, "流水线项目名称", "工业级全自动环境编排与自愈流水线")
	spec.Goal = cfg.DefaultGoal

	allowed := readLine(reader, "受托写入路径白名单 (以分号;分割，留空不设限)", "")
	if allowed != "" {
		for _, p := range strings.Split(allowed, ";") {
			if t := strings.TrimSpace(p); t != "" {
				spec.AllowedPaths = append(spec.AllowedPaths, t)
			}
		}
	}

	dodCmd := readLine(reader, "终态硬性验收命令 (DoD，未跑通绝不退出，如 curl/nvidia-smi/服务探针)", "")
	spec.VerificationCmd = strings.TrimSpace(dodCmd)

	// 动态配置额外的专用领域工具 (Custom Tools)
	fmt.Println("\n【第二步：预置额外的专用领域扩展工具 (Custom Tools)】")
	fmt.Println("说明: 当流水线需要特殊 CLI 时（如 docker、git、kubectl、ffmpeg），可在此出厂硬编码预置。")
	for {
		addTool := readLine(reader, "是否为此打包版本挂载额外工具? (y/N)", "N")
		if strings.ToLower(addTool) != "y" {
			break
		}
		var tool CustomTool
		tool.Name = readLine(reader, "工具名称 (例如: git_sync / docker_run / port_kill)", "custom_tool")
		tool.Description = readLine(reader, "工具功能说明 (告诉大模型何时使用该工具)", "执行特定专用系统指令")
		tool.CommandTmpl = readLine(reader, "实际底层命令模板 (参数占位符用 {{args}})", "echo {{args}}")
		tool.TimeoutSec = 120
		spec.CustomTools = append(spec.CustomTools, tool)
		fmt.Printf("   -> 已挂载专用工具: %s\n", tool.Name)
	}

	cfg.TaskSpec = &spec

	fmt.Println("\n【第三步：配置出厂嵌入的大模型通道 (将被绝对保密)】")
	for {
		var p Provider
		p.Name = readLine(reader, "提供商名称", "私有高阶大模型通道")
		p.URL = readLine(reader, "API BaseURL", "https://opencode.ai/zen/go/v1/chat/completions")
		p.Key = readLine(reader, "私有 API Key (将被高强度加密混淆)", "")
		p.Model = readLine(reader, "模型标识 (Model ID)", "mimo-v2.5")

		p.Headers = make(map[string]string)
		ua := readLine(reader, "自定义 User-Agent (可选)", "opencode/1.0.0")
		if ua != "" {
			p.Headers["User-Agent"] = ua
		}

		cfg.Providers = append(cfg.Providers, p)

		more := readLine(reader, "\n是否继续添加备用熔断大模型通道? (y/N)", "N")
		if strings.ToLower(more) != "y" {
			break
		}
	}

	fmt.Println("\n【第四步：输入出厂混淆盐值 (Build Secret)】")
	secretKey := readLine(reader, "混淆加密秘钥 (直接回车使用内置高熵随机生成)", "")
	if secretKey == "" {
		randBytes := make([]byte, 16)
		rand.Read(randBytes)
		secretKey = hex.EncodeToString(randBytes)
		fmt.Printf("[*] 已自动生成高熵混淆盐: %s\n", secretKey)
	}

	// 序列化配置并执行 AES-GCM-256 加密
	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		fmt.Printf("[X] 配置序列化失败: %v\n", err)
		return
	}

	encryptedHex, err := encryptPayload(cfgJSON, secretKey)
	if err != nil {
		fmt.Printf("[X] 配置加密失败: %v\n", err)
		return
	}

	fmt.Println("\n================================================================")
	fmt.Println("🔒 全套出厂规约、多线配置与模型密钥已完成 AES-256-GCM 强加密！")
	fmt.Printf("%.60s... (完整密文已嵌入)\n", encryptedHex)
	fmt.Println("================================================================")

	// 注入代码生成专用的 embedded_config.go
	projectRoot, _ := filepath.Abs("../")
	loaderPath := filepath.Join(projectRoot, "embedded_config.go")

	code := fmt.Sprintf(`// Code generated by xirang-builder v4.0. DO NOT EDIT.
package main

const (
	EncryptedPayload = "%s"
	BuildSecret      = "%s"
)
`, encryptedHex, secretKey)

	if err := os.WriteFile(loaderPath, []byte(code), 0644); err != nil {
		fmt.Printf("[X] 写入 embedded_config.go 失败: %v\n", err)
		return
	}
	fmt.Printf("\n[*] 已成功生成嵌入式混淆载荷: %s\n", loaderPath)

	// 构建选项
	fmt.Println("\n【第五步：构建分发目标文件】")
	fmt.Println("1) 构建 Windows x64 独立受保护程序 (xirang_protected.exe)")
	fmt.Println("2) 构建 macOS 独立受保护二进制 (xirang_protected_mac)")
	fmt.Println("3) 构建 Linux 独立受保护二进制 (xirang_protected_linux)")
	fmt.Println("4) 仅生成加密配置，手动编译")

	choice := readLine(reader, "请选择编译目标 (1-4)", "1")

	distDir := filepath.Join(projectRoot, "dist_protected")
	os.MkdirAll(distDir, 0755)

	buildCmd := func(goos, goarch, outName string) {
		fmt.Printf("\n正在交叉编译防逆向二进制 -> %s/%s => %s ...\n", goos, goarch, outName)
		outPath := filepath.Join(distDir, outName)
		cmd := exec.Command("go", "build", "-ldflags=-s -w", "-o", outPath, "main.go", "embedded_config.go")
		cmd.Dir = projectRoot
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+goos, "GOARCH="+goarch)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("[X] 编译失败: %v\n%s\n", err, string(output))
		} else {
			fmt.Printf("✅ 编译成功！出厂绑定所有规约与工具的独立程序已输出到:\n   %s\n", outPath)
		}
	}

	switch choice {
	case "1":
		buildCmd("windows", "amd64", "xirang_protected.exe")
	case "2":
		arch := runtime.GOARCH
		buildCmd("darwin", arch, "xirang_protected_mac")
	case "3":
		buildCmd("linux", "amd64", "xirang_protected_linux")
	case "4":
		fmt.Println("[*] 已生成代码，请在工程根目录执行 go build 自行构建。")
	}
}
