# 🌱 息壤 (XiRang) v4.0

> **“息壤者，言土自长息无限也。” ——《山海经》**
>
> 息壤是一款面向现代全平台异构计算环境（Windows / macOS / Linux / BSD / **华为鸿蒙 PC** / 嵌入式工控机）的**超轻量、单文件、强针对性、生生不息操作系统伴生自愈智能体**。

---

## 📚 项目核心文档与演进路线图

* 🏗️ **[架构全景图谱](README_架构全景.md)**：分层架构、核心机制、全平台产物矩阵
* 📖 **[开发者与 AI 集成指南](息壤(XiRang)_开发者与AI集成使用指南.md)**：CLI 核心命令、任务规约 Spec 规范、四大用例与 Subagent 集成范式
* 🚀 **[愿景差距与架构改进规划](息壤(XiRang)_愿景差距与架构改进规划.md)**：愿景差距深度剖析、核心模块短板改进建议与分期演进路线图

---


在复杂的操作系统与工程部署现场，传统安装包或 Shell 脚本面临**易崩溃、难自愈、制造垃圾文件、过度越权决策**等痛点。

息壤（XiRang）致力于提供工业级全场景解决方案：

1. **极致克制，原生单文件空投**：
   * 坚决摒弃 Electron/Chromium 及庞大的 Python 虚拟环境依赖；
   * 编译产物仅数兆，无任何外部运行环境依赖，随时随地在 U 盘、老旧裸机或异构架构机型上空投即用。
2. **帮人自愈，决不替人决策（人类决策主权）**：
   * 息壤负责现场底层探查、环境适配与故障修复；
   * 涉及核心业务决策（如存储选址、端口分配、文件覆盖）时，永远通过 `ask_human` 交互把决定权交还给人类。
3. **终态硬性防作弊网关 (DoD Gate)**：
   * 以真实业务测试指令（如 `curl` 端点探活、真实模型推理测试）为唯一准绳，未跑通测试绝不宣告完成。
4. **0.2 秒 Fast-Path 秒级自愈**：
   * 过程脚本与排障经验自动沉淀至 `.xirang/scripts/` 中；
   * 遇到同类问题自动匹配本地指纹库秒级修复，0 Token 消耗，不依赖云端响应。
5. **多线并发编排 (Parallel Batch Actions)**：
   * 原生支持多命令、多文件并发处理，多线程并行执行并安全聚合回报，效率提升 300%+。
6. **全平台全架构全覆盖 (含原生鸿蒙 PC)**：
   * 覆盖 12+ 种操作系统与 CPU 架构，针对**华为鸿蒙 PC (HarmonyOS PC / OpenHarmony PC)** 提供基于 `musl libc` 的无依赖静态编译标靶。

---

## 🛠️ 二、 息壤要怎么操作？(How to Operate XiRang?)

息壤支持命令行 CLI、GUI 创作工坊与后台守护等多模态操作方式：

### 1. 命令行常用参数指令表

```bash
# 1. 挂载结构化任务规约执行全自动安装与配置
xirang -spec xirang_task_spec.json

# 2. 自然语言一次性部署与定向排障
xirang -task "检查当前环境并配置本地推理服务"

# 3. 0-Token 极速 DoD 终态验收校验 (不消耗 Token，直接测试 VerificationCmd)
xirang -spec xirang_task_spec.json -verify

# 4. 查看本地已沉淀的经验脚本与故障指纹资产库
xirang -skills

# 5. 现场售后急诊医生模式（遇错随呼随到）
xirang -doctor

# 6. 常驻后台售后心跳巡检守护（30秒心跳自动探活自愈）
xirang -watchdog -interval 30

# 7. 一键安全事务回滚（逆序撤销修改，恢复纯净系统）
xirang --rollback

# 8. 启动息壤工坊可视化创作控制台 (Web GUI)
xirang -forge  # 或 xirang -gui
```

### 2. 模型通道配置原则 (安全边界守卫)

> [!IMPORTANT]
> **安全设计准则**：息壤**绝不在宿主操作系统的全局环境变量中盲目嗅探 API 密钥**，防止密钥盗用。模型通道由以下方式提供：
> 1. **出厂混淆熔炼（交付态，注意边界）**：使用 `builder` 将模型通道与规约 AES-256-GCM 加密后嵌入二进制。**内嵌 `BuildSecret` 仅是混淆，不是分发级密钥保护**；更强方案请用 `builder -external-secret`，运行时注入 `XIRANG_BUILD_SECRET`。
> 2. **显式配置文件注入**：在执行目录下放置受控的 `xirang_config.json` 文件。

**安全环境变量（默认更严）：**

| 变量 | 默认 | 说明 |
|------|------|------|
| TLS 校验 | 开启 | 仅当 `XIRANG_INSECURE_TLS=1` 才跳过证书校验 |
| 写路径沙箱 | cwd + `.xirang` | 未配置 `allowed_paths` 时默认拒绝白名单外写入；`XIRANG_UNRESTRICTED_PATHS=1` 可放开（不推荐） |
| `XIRANG_BUILD_SECRET` | 空 | 与 `-external-secret` 配合，运行时解密嵌入配置 |

**事务回滚**：变更会写入 `.xirang/backups/index.json`，跨进程 `xirang --rollback` 可读取索引撤销（含新建文件/目录）。

---

## 💻 三、 在不同操作系统与具体项目中如何使用息壤？

### 1. 针对不同操作系统 (Operating Systems)

#### 🪟 Windows (Win 7 / Win 10 / Win 11 24H2 / ARM64)
* **操作系统特性**：Win11 24H2+ 已彻底弃用 `wmic`，普通 bat 脚本易失效。
* **息壤操作**：
  * 在当前目录放置 `xirang.exe` 或 32位老旧兼容版 `xirang_win_legacy.exe`；
  * 双击运行工坊生成的 `使用息壤配置.bat`，息壤会自动识别 64位/32位/ARM64 架构并启动。

#### 🍎 macOS (Apple Silicon M1~M4 / Intel)
* **操作系统特性**：存在 macOS Quarantine 隔离与签名保护。
* **息壤操作**：
  * 使用配套脚本 `run_xirang.sh`，自动进行 `xattr -d com.apple.quarantine` 签名解除；
  * 自动识别 M 系列 Apple Silicon (`xirang_mac_apple_silicon`) 或 Intel 芯片 (`xirang_mac_intel`) 启动。

#### 🐧 Linux & 📱 Android Termux
* **操作系统特性**：包含 Ubuntu/Debian 服务器、树莓派 ARMv7、Android Termux/chroot。
* **息壤操作**：
  * 执行 `chmod +x xirang_linux_x64`，运行 `./xirang_linux_x64 -spec xirang_task_spec.json`。

#### 🔴 华为鸿蒙 PC (HarmonyOS PC / OpenHarmony PC)
* **操作系统特性**：底层采用 `musl libc` 而非 GNU `glibc`，命令行环境为 `mksh` 与 `toybox`，存在星盾签名沙箱。
* **息壤操作**：
  * 息壤提供 `CGO_ENABLED=0` 静态编译的专有标靶 `xirang_harmony_pc_x64` / `xirang_harmony_pc_arm64`，**零依赖原生适配 musl libc**；
  * 使用工坊生成的 `run_harmony.sh` 脚本在鸿蒙终端或 `hdc` 通道下一键启动自愈。

---

### 2. 针对典型工程项目 (Project Scenarios)

#### 场景一：私有化本地 AI 大模型 / 推理引擎一键交付
* **需求**：将 15GB+ 模型与 Ollama/vLLM 部署到客户裸机，严禁占用 C 盘系统空间。
* **操作方式**：
  ```bash
  xirang -spec xirang_task_spec.json
  ```
* **工作流**：
  1. 息壤自动探查磁盘空间，通过 `ask_human` 询问用户选择存放分卷（如 `D:\ollama_models`）；
  2. 探测 GPU 驱动与 CUDA 计算环境；
  3. 执行 `batch_actions` 并发下载与启动服务；
  4. 触发 VerificationCmd 执行实际 Prompt 推理验证，成功后向客户交付。

#### 场景二：复杂 Web & 数据库微服务集群部署
* **需求**：并发部署 Nginx、FastAPI、Redis 与 PostgreSQL，并分配正确端口。
* **操作方式**：
  ```bash
  xirang -task "配置 Web 集群，分配可用端口并校验连通性"
  ```
* **工作流**：
  * 使用 `batch_actions` 多线程并行下载依赖、创建配置文件、测试端口，大幅缩短部署时间；
  * 若配置中断或需要撤销，执行 `xirang --rollback` 1 秒还原系统初始状态。

#### 场景三：长效售后心跳守护与急诊诊所
* **需求**：软件交付后，因电脑休眠或端口冲突导致服务挂掉，降低售后工单率。
* **操作方式**：
  * **日常守护**：后台运行 `xirang -watchdog -interval 30`，发现探活异常自动唤醒修复内核并拉起服务；
  * **现场急诊**：用户控制台报错时，运行 `xirang -doctor` 粘贴错误信息，息壤优先调取 `.xirang/scripts/` 下的本地经验秒级自愈。

---

## 🏗️ 四、 编译与构建 (Build)

```bash
# 1. 编译当前宿主本地二进制
go build -ldflags="-s -w" -o xirang main.go embedded_default.go

# 2. 运行跨平台多架构全矩阵自动化交叉编译 (产物输出至 dist/)
bash build_all.sh

# 3. 运行项目全套单元测试
go test -v ./...
```

---

## 📄 License & Roadmap

*开发与演进中... 欢迎提交 Issue 与 PR 参与建设！*
