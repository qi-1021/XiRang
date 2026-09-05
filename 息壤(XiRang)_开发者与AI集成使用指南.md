# 🌱 息壤 (XiRang) v4.0：开发者与 AI 协作实战操作指南

> **定位**：面向 AI Agent、软件架构师与运维工程师的下一代通用轻量化程序安装器、环境配置基建与系统级终身售后自愈解决方案。
> **核心优势**：单文件零依赖（无需预装 Python/Node/Docker）、12 平台全兼容、多线并发编排、白名单防越界沙箱、硬性终态防作弊验收 (DoD)、经验自沉淀与 0.2 秒秒级自愈。

---

## 目录
1. [设计哲学与为什么需要「息壤」](#1-设计哲学与为什么需要息壤)
2. [CLI 核心命令与四大运行模式](#2-cli-核心命令与四大运行模式)
3. [结构化任务规约 (Task Specification) 规范](#3-结构化任务规约-task-specification-规范)
4. [出厂固化与加密打包套件 (xirang_builder)](#4-出厂固化与加密打包套件-xirang_builder)
5. [四大核心典型实战用例](#5-四大核心典型实战用例)
   - 用例一：为客户打造商业大模型/本地推理一键安装包（含硬件自适配与防作弊验收）
   - 用例二：复杂企业级服务集群部署（多线程并发批处理与原子事务回滚）
   - 用例三：作为售后常驻守护进程 (Watchdog Heartbeat)
   - 用例四：给人类程序员/测试当现场急诊医生 (Doctor Mode)
6. [经验资产化与本地自进化机制 (.xirang/)](#6-经验资产化与本地自进化机制-xirang)
7. [AI 智能体 (Subagent) 集成「息壤」的最佳实践 Prompt 范式](#7-ai-智能体-subagent-集成息壤的最佳实践-prompt-范式)

---

## 1. 设计哲学与为什么需要「息壤」

传统的安装程序（如 InnoSetup、NSIS、Shell/Bat 脚本）存在几大致命痛点：
1. **静态且脆弱**：遇到未知的硬件架构、驱动版本冲突、系统路径特异性（例如 Windows 11 24H2 移除 `wmic`）立即报错崩溃，无法自愈。
2. **缺乏售后感知**：安装完成后无法持续守护，遇到端口被占、依赖被误删，只能人工排障。
3. **制造磁盘垃圾**：临时生成的 bat/sh 脚本满处乱扔，严重污染用户工作目录。
4. **大模型作弊隐患**：让通用 LLM 写脚本执行，模型经常产生幻觉并假装“已经成功配置完成”，实际服务根本没跑起来。

**「息壤」的解法**：
* **单兵空投**：编译后是纯原生单文件（Windows 下 `xirang.exe` 仅 6MB，甚至提供 34KB 的纯 C 单文件 `xirang.c`），无需任何运行环境。
* **终态硬性防作弊 (DoD Gate)**：未跑通真实验收指令（如 `curl` 或驱动检测），绝对禁止结束。
* **干净隔离与经验复利**：所有过程脚本自动沉淀在 `.xirang/scripts/`，下次遇同类问题 0.2 秒 Fast-Path 解决。
* **AES-256 出厂加固**：商业分发时，开发者可将专有规约与 API Key 加密熔炼，客户端无法逆向明文密钥。

---

## 2. CLI 核心命令与四大运行模式

```bash
# 查看完整参数帮助
xirang.exe -h
```

| 参数 / 标志 | 默认值 | 说明 |
| :--- | :--- | :--- |
| `-task "..."` | `""` | 动态自然语言目标。下达具体的一次性配置/修复任务 |
| `-spec spec.json` | `""` | 挂载结构化任务规约（支持定义里程碑、沙箱、验收命令） |
| `-config cfg.json` | `xirang_config.json` | 外挂模型通道与网络配置文件（自动兼容 `zen_config.json`） |
| `-doctor` | `false` | **售后急诊医生模式**：交互式问诊台，随时粘贴任何报错，秒级自愈 |
| `-watchdog` | `false` | **售后常驻巡检模式**：静默心跳探活，服务挂掉时自动唤醒排障修复 |
| `-interval <秒>` | `30` | 售后巡检心跳间隔秒数 |
| `--rollback` | `false` | **安全事务一键回滚**：逆序撤销所有由息壤生成或修改的文件与配置 |

### 四大运行模式速查表

```
               ┌── [1] 任务部署模式 (默认 / -task / -spec)
               │    全自动环境探测 -> 多线并发编排 -> 终态DoD硬性验收 -> 交付
               │
               ├── [2] 售后急诊医生模式 (-doctor)
模式矩阵 ──────┤    交互式终端，遇错随呼随到，优先 Fast-Path 本地解决，免消耗 Token
               │
               ├── [3] 常驻守护自愈模式 (-watchdog -interval 30)
               │    作为系统后台守护进程，心跳探活，故障秒级自动拉起
               │
               └── [4] 逆向事务回滚模式 (--rollback)
                    基于 .xirang/backups/ 原子镜像，1秒瞬间恢复系统干净原貌
```

---

## 3. 结构化任务规约 (Task Specification) 规范

通过编写 `xirang_task_spec.json`，可以精确约束智能体的行为准则，构建工业级生产交付闭环：

```json
{
  "task_name": "本地大模型高性能推理环境构建",
  "goal": "自动检测当前机器 GPU 状态，部署 Ollama 服务，载入指定模型并验证推理通道。",
  "allowed_paths": [
    "D:\\Ollama_Service",
    "C:\\Users\\Public\\AI_Engine"
  ],
  "forbidden_cmds": [
    "format",
    "rmdir /s /q c:\\",
    "rm -rf /",
    "del /f /s /q c:\\windows"
  ],
  "milestones": [
    {
      "name": "显卡驱动与计算环境检查",
      "verify_cmd": "nvidia-smi",
      "description": "确认当前 GPU 存在且驱动支持 CUDA 12+"
    },
    {
      "name": "Ollama 服务进程探活",
      "verify_cmd": "curl -s http://127.0.0.1:11434/api/version",
      "description": "确认核心推理引擎已在后台启动监听"
    }
  ],
  "verification_cmd": "curl -s -X POST http://127.0.0.1:11434/api/generate -d \"{\\\"model\\\":\\\"qwythos:9b\\\",\\\"prompt\\\":\\\"ping\\\",\\\"stream\\\":false}\"",
  "strict_rules": [
    "严禁在母目录下留存任何调试或临时文件，所有脚本必须保存在 .xirang/scripts/ 目录。",
    "遇到端口占用时，必须先查出 PID 并上报，切忌暴力清空不相关的系统进程。"
  ],
  "custom_tools": [
    {
      "name": "check_gpu_arch",
      "description": "检测 GPU 显存与微架构",
      "command_tmpl": "powershell -Command \"Get-CimInstance Win32_VideoController | Select-Object Name, AdapterRAM\"",
      "timeout_sec": 30
    }
  ],
  "health_probes": [
    {
      "name": "Ollama 服务健康探活",
      "check_cmd": "curl -s http://127.0.0.1:11434/api/tags",
      "fix_goal": "检测到 Ollama 未响应，请检查端口占用或后台服务，重新拉起并确保健康。"
    }
  ]
}
```

---

## 4. 出厂固化与加密打包套件 (xirang_builder)

当你需要把息壤做成商业软件的“安装包”分发给普通客户，且不希望客户看到你的大模型 API Key 或专有部署逻辑时使用。

### 操作流程：
1. 编译并启动编译器：
   ```bash
   cd /Volumes/mac第三磁盘/codes/Projects/XiRang/builder
   go run main.go
   ```
2. 交互式输入：
   * **出厂默认使命**（如：“一键初始化本司 CAD 渲染集群”）
   * **安全白名单路径**（如：`C:\Program Files\MyCADApp`）
   * **DoD 验收命令**（如：`curl http://localhost:8080/health`）
   * **私有大模型提供商通道与 API Key**（支持注入小米 MiMo、DeepSeek、Claude 等）
3. 编译器自动完成：
   * 采用 **AES-256-GCM** 加密全部配置与 Key；
   * 自动生成 `embedded_config.go`；
   * 调用 `go build -ldflags="-s -w"` 剔除符号表，输出加固独立可执行程序 `xirang_protected.exe`。
4. **交付结果**：客户只需要双击 `xirang_protected.exe`，无需任何配置，无法反编译提取 API Key。

---

## 5. 四大核心典型实战用例

### 用例一：为客户制作本地大模型/私有算力一键安装器
* **场景**：客户有一台配置了新显卡（如 RTX 40/50 系列）的裸机，需要把 15GB+ 的模型和推理引擎安装好，且不能污染系统盘。
* **执行方式**：
  ```cmd
  xirang.exe -spec xirang_task_spec.json
  ```
* **息壤行为**：
  1. 自动探测驱动器剩余空间，自主决定将大模型挂载至 `D:\ollama_models`。
  2. 探测 GPU 环境，遇到 Win11 24H2 自动绕开失效的 `wmic`，使用 PowerShell 或 `nvidia-smi`。
  3. 执行 `batch_actions` 并行建立 Modelfile 与模型软链接。
  4. 触发 DoD 终态验收网关，真实发起单次 Prompt 推理，验收通过后才退出并向客户交付。

---

### 用例二：复杂企业级分布式服务编排（并发多任务）
* **场景**：需要在短时间内在机器上部署前端 WebUI、后端 FastAPI、Redis 缓存与 Prometheus 监控。
* **执行方式**：
  ```bash
  ./xirang -task "并行拉起 Redis、FastAPI 与前端静态服务，修改各配置文件中的端口映射，完成后进行全链路联通性测试。"
  ```
* **息壤行为**：
  * 大模型利用原生动作 `batch_actions` 并发派发：
    * 线程 1：生成 Redis 配置文件并后台启动
    * 线程 2：下载依赖包并校验 hash
    * 线程 3：探测可用网络端口
  * 多子任务并行完成，效率比传统单步 Agent 提升数倍。若中途崩溃，执行 `./xirang --rollback` 立即恢复干净现场。

---

### 用例三：作为售后常驻守护进程 (Watchdog Daemon)
* **场景**：软件交付给非技术客户后，客户经常因电脑休眠、误杀后台、端口冲突导致服务中断，产生大量售后工单。
* **执行方式**：
  在 Windows 计划任务或开机脚本中加入：
  ```cmd
  xirang.exe -watchdog -interval 30
  ```
* **息壤行为**：
  * 每 30 秒执行一次 `health_probes` 中的探活命令。
  * 发现服务中断时，**全自动唤醒内置的自愈内核**，分析端口占用与崩溃原因，自主执行抢修脚本将服务重新拉起，实现“零工单”免维护。

---

### 用例四：现场急诊医生模式 (Doctor Mode)
* **场景**：程序员或现场客户在使用软件时突然在控制台看到报错红字（如 `CUDA driver version is insufficient for CUDA runtime version` 或 `bind: address already in use`）。
* **执行方式**：
  ```cmd
  xirang.exe -doctor
  ```
* **交互体验**：
  1. 控制台出现急诊医生问诊提示：`👉 请输入您遇到的错误或需要售后解决的问题:`
  2. 直接把整段终端报错粘贴进去，或者输入大白话：“帮我看下为什么 8080 端口启动失败”。
  3. **Fast-Path 机制生效**：若该机器之前处理过类似错误，息壤在 **0.2 秒内直接调用本地历史脚本解决**，不走云端大模型；若属于新问题，则连线大模型深度排查并把新脚本持久化到 `.xirang/scripts/`。

---

## 6. 经验资产化与本地自进化机制 (.xirang/)

息壤的工作目录规范是其与普通 Agent 最显著的区别：

```
目标机器项目目录/
├── 业务主程序/
└── .xirang/                          <-- 息壤专属隔离与经验沉淀资产库
    ├── backups/                      <-- 事务恢复快照 (用于一键回滚)
    │   └── 1741178920_config.yaml
    └── scripts/                      <-- 自愈脚本库 (永久留存，实现经验复利)
        ├── fix_cuda_env.bat          <-- 针对本机的修复脚本
        ├── fix_cuda_env.bat.meta.json<-- 触发指纹库 (记录匹配的错误关键字与说明)
        ├── clean_zombie_ports.ps1
        └── clean_zombie_ports.ps1.meta.json
```

### 元数据指纹格式 (`*.meta.json`)
```json
{
  "name": "清理孤儿端口脚本",
  "pattern": "address already in use",
  "script_path": ".xirang/scripts/clean_zombie_ports.ps1",
  "description": "当服务重启遇到端口被僵尸进程锁死时，精准定位并释放该端口"
}
```
**自进化收益**：机器运行时间越长，`.xirang/scripts/` 积累的工具越完备。未来遇到常见故障直接秒级自愈，无需漫长的网络推演，越用越快、越用越熟练！

---

## 7. AI 智能体 (Subagent) 集成「息壤」的最佳实践 Prompt 范式

当其他 AI（如 Claude、Cursor、Antigravity 等上层编排 Agent）需要调用息壤来完成繁琐的现场环境部署时，建议在子任务 Prompt 中这样写：

```markdown
【任务指派】：调用「息壤 (XiRang)」完成底层环境配置与持久化交付

你不需要在宿主机上亲自编写零散的 bat/sh 脚本，请直接调用现场提供的原生单文件程序 `xirang` / `xirang.exe`。

【执行准则】：
1. 规约下发：通过编写一份 `xirang_task_spec.json` 明确允许写入的白名单目录和终态验收命令 (DoD)。
2. 调用命令：运行 `xirang -spec xirang_task_spec.json` 触发全自动并发部署。
3. 售后防线：安装完成后，建议在后台为客户启动 `xirang -watchdog -interval 30` 形成长效售后自愈。
4. 遇到偶发排障问题：随时使用 `xirang -doctor` 传入错误堆栈进行自愈。
```

---
*文档版本：v4.0 | 状态：最新发布 | 归属工程：XiRang Core Matrix*
