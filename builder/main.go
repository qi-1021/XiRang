package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"zenagent/builder/gui"
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

type HealthProbe struct {
	Name      string `json:"name"`
	Kind      string `json:"kind,omitempty"`
	Target    string `json:"target,omitempty"`
	CheckCmd  string `json:"check_cmd,omitempty"`
	FixGoal   string `json:"fix_goal,omitempty"`
	TimeoutSec int   `json:"timeout_sec,omitempty"`
}

type TaskSpecification struct {
	TaskName        string        `json:"task_name"`
	Goal            string        `json:"goal"`
	AllowedPaths    []string      `json:"allowed_paths"`
	ForbiddenCmds   []string      `json:"forbidden_cmds"`
	Milestones      []Milestone   `json:"milestones"`
	VerificationCmd string        `json:"verification_cmd"`
	StrictRules     []string      `json:"strict_rules"`
	CustomTools     []CustomTool  `json:"custom_tools"`
	HealthProbes    []HealthProbe `json:"health_probes"`
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
	guiFlag := flag.Bool("gui", false, "启动息壤工坊图形化控制台 (XiRang Studio Web GUI)")
	specPathFlag := flag.String("spec", "", "直接传入外部 task_spec.json 规范文件路径（免交互）")
	configPathFlag := flag.String("config", "", "直接传入包含模型密钥的 xirang_config.json 路径（免交互）")
	targetOSFlag := flag.String("os", "", "出厂目标操作系统 (windows / darwin / linux)")
	outNameFlag := flag.String("out", "", "输出二进制名称 (例如 xirang_custom.exe)")
	secretKeyFlag := flag.String("secret", "", "自定义混淆盐 (留空自动生成高熵随机盐)")
	externalSecretFlag := flag.Bool("external-secret", false, "不内嵌密钥：仅嵌入密文，运行时通过 XIRANG_BUILD_SECRET 注入")
	flag.Parse()

	projectRoot, _ := filepath.Abs("../")
	distDir := filepath.Join(projectRoot, "dist_protected")

	// 模式 0：图形化创作工坊 (GUI 模式)
	if *guiFlag {
		server := gui.NewForgeServer(projectRoot)
		if err := server.Start(true); err != nil {
			fmt.Printf("[X] 启动图形控制台失败: %v\n", err)
			os.Exit(1)
		}
		return
	}

	var cfg BuildConfig
	cfg.MaxSteps = 40
	cfg.TimeoutSec = 300

	// 模式 A：非交互式静默构建模式（专供人类脚本或上层 AI 自动化调用）
	if *specPathFlag != "" || *configPathFlag != "" {
		fmt.Println("================================================================")
		fmt.Println("    🛡️ 息壤 编译器 (XiRang Builder v4.0): 静默出厂熔炼构建模式")
		fmt.Println("================================================================")

		if *specPathFlag != "" {
			data, err := os.ReadFile(*specPathFlag)
			if err != nil {
				fmt.Printf("[X] 读取规范文件失败: %v\n", err)
				os.Exit(1)
			}
			var spec TaskSpecification
			if err := json.Unmarshal(data, &spec); err != nil {
				fmt.Printf("[X] 解析规范文件失败: %v\n", err)
				os.Exit(1)
			}
			cfg.TaskSpec = &spec
			cfg.DefaultGoal = spec.Goal
			fmt.Printf("[*] 已挂载任务规约: %s (阶段数: %d)\n", spec.TaskName, len(spec.Milestones))
		}

		if *configPathFlag != "" {
			data, err := os.ReadFile(*configPathFlag)
			if err != nil {
				fmt.Printf("[X] 读取模型配置文件失败: %v\n", err)
				os.Exit(1)
			}
			var fileCfg BuildConfig
			if err := json.Unmarshal(data, &fileCfg); err == nil && len(fileCfg.Providers) > 0 {
				cfg.Providers = fileCfg.Providers
			} else {
				// 尝试解析成标准 Config 格式
				var stdCfg struct {
					Providers []Provider `json:"providers"`
				}
				if err := json.Unmarshal(data, &stdCfg); err == nil && len(stdCfg.Providers) > 0 {
					cfg.Providers = stdCfg.Providers
				}
			}
			fmt.Printf("[*] 已挂载大模型通道: %d 项\n", len(cfg.Providers))
		}

		if len(cfg.Providers) == 0 {
			fmt.Println("[!] 警告: 未提供模型通道凭据，构建出的程序将依赖目标机提供显式 xirang_config.json。")
		}

		secretKey := *secretKeyFlag
		if secretKey == "" {
			randBytes := make([]byte, 16)
			rand.Read(randBytes)
			secretKey = hex.EncodeToString(randBytes)
		}

		cfgJSON, _ := json.Marshal(cfg)
		encryptedHex, err := encryptPayload(cfgJSON, secretKey)
		if err != nil {
			fmt.Printf("[X] 加密载荷失败: %v\n", err)
			os.Exit(1)
		}

		loaderPath := filepath.Join(projectRoot, "embedded_config.go")
		buildSecretEmbed := secretKey
		if *externalSecretFlag {
			buildSecretEmbed = ""
			secretPath := filepath.Join(distDir, ".xirang_build_secret")
			_ = os.MkdirAll(distDir, 0755)
			_ = os.WriteFile(secretPath, []byte(secretKey+"\n"), 0600)
			fmt.Printf("[*] 外置密钥已写入: %s\n", secretPath)
			fmt.Println("[*] 运行时请设置环境变量 XIRANG_BUILD_SECRET 后再执行受保护二进制。")
		} else {
			fmt.Println("[!] 安全说明：内嵌 BuildSecret 仅为混淆（可被逆向提取），不是分发级密钥保护。")
			fmt.Println("[!] 若需更强保护，请使用 -external-secret，密钥经 XIRANG_BUILD_SECRET 在运行时注入。")
		}
		code := fmt.Sprintf(`// Code generated by xirang-builder v4.0. DO NOT EDIT.
// 安全说明：若 BuildSecret 非空，仅为混淆而非真正保密；推荐 -external-secret + XIRANG_BUILD_SECRET。
package main

const (
	EncryptedPayload = "%s"
	BuildSecret      = "%s"
)
`, encryptedHex, buildSecretEmbed)
		os.WriteFile(loaderPath, []byte(code), 0644)
		fmt.Println("[*] 已成功生成 AES-256-GCM 混淆代码: embedded_config.go")

		targetOS := *targetOSFlag
		if targetOS == "" {
			targetOS = runtime.GOOS
		}
		targetArch := runtime.GOARCH
		outName := *outNameFlag
		if outName == "" {
			if targetOS == "windows" {
				outName = "xirang_protected.exe"
			} else {
				outName = "xirang_protected"
			}
		}

		os.MkdirAll(distDir, 0755)
		outPath := filepath.Join(distDir, outName)
		fmt.Printf("[*] 正在编译 -> %s/%s => %s ...\n", targetOS, targetArch, outPath)
		cmd := exec.Command("go", "build", "-ldflags=-s -w", "-o", outPath, "main.go", "embedded_config.go")
		cmd.Dir = projectRoot
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+targetOS, "GOARCH="+targetArch)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("[X] 编译失败: %v\n%s\n", err, string(output))
			os.Exit(1)
		}
		if targetOS == "darwin" && runtime.GOOS == "darwin" {
			_ = exec.Command("codesign", "--force", "--deep", "--sign", "-", outPath).Run()
		}
		fmt.Printf("✅ [出厂加固成功] 专有独立安装包已就绪:\n   %s\n", outPath)
		return
	}

	// 模式 B：交互式引导问答模式（面向人类运维工程师）
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("================================================================")
	fmt.Println("    🛡️ 息壤 编译器 (XiRang Builder v4.0): 工业级全并发·契约打包套件")
	fmt.Println("    (出厂预置多线并发能力、专属扩展工具箱、白名单沙箱、AES-256熔炼)")
	fmt.Println("================================================================")
	fmt.Println()

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
	loaderPath := filepath.Join(projectRoot, "embedded_config.go")

	// 安全说明：内嵌密钥可被提取，仅作混淆；强交付建议外置密钥
	fmt.Println("[!] 安全说明：BuildSecret 将内嵌进二进制，仅供混淆；请勿据此宣称“绝对安全”。")
	code := fmt.Sprintf(`// Code generated by xirang-builder v4.0. DO NOT EDIT.
// 安全说明：BuildSecret 内嵌仅为混淆；强交付请改用 -external-secret 与 XIRANG_BUILD_SECRET。
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

	distDir = filepath.Join(projectRoot, "dist_protected")
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
			if goos == "darwin" && runtime.GOOS == "darwin" {
				_ = exec.Command("codesign", "--force", "--deep", "--sign", "-", outPath).Run()
			}
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
