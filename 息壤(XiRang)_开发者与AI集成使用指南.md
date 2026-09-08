# 🌱 息壤 (XiRang) v4.0：开发者与 AI 协作实战操作指南

> **定位**：面向 AI Agent、软件架构师与运维工程师的下一代通用轻量化程序安装器、环境配置基建与系统级终身售后自愈解决方案。
> **核心优势**：单文件零依赖（无需预装 Python/Node/Docker）、12 平台全兼容（含原生鸿蒙 PC）、多线并发编排、白名单防越界沙箱、硬性终态防作弊验收 (DoD)、经验自沉淀与 0.2 秒秒级自愈。

---

## 目录
1. [设计哲学与为什么需要「息壤」](#1-设计哲学与为什么需要息壤)
2. [CLI 核心命令与四大运行模式](#2-cli-核心命令与四大运行模式)
3. [结构化任务规约 (Task Specification) 规范](#3-结构化任务规约-task-specification-规范)
4. [息壤工坊图形控制台 (XiRang Studio GUI) 与 AES-256 加密打包](#4-息壤工坊图形控制台-xirang-studio-gui-与-aes-256-加密打包)
5. [四大核心典型实战用例](#5-四大核心典型实战用例)
   - 用例一：为客户打造商业大模型/本地推理一键安装包（含硬件自适配与防作弊验收）
   - 用例二：复杂企业级服务集群部署（多线程并发批处理与原子事务回滚）
   - 用例三：华为鸿蒙 PC (HarmonyOS PC) / 信创环境静默交付
   - 用例四：作为售后常驻守护进程 (Watchdog Heartbeat) 与急诊诊所 (Doctor Mode)
6. [经验资产化与本地自进化机制 (.xirang/)](#6-经验资产化与本地自进化机制-xirang)
7. [AI 智能体 (Subagent) 集成「息壤」的最佳实践 Prompt 范式](#7-ai-智能体-subagent-集成息壤的最佳实践-prompt-范式)

---

## 1. 设计哲学与为什么需要「息壤」

传统的安装程序（如 InnoSetup、NSIS、Shell/Bat 脚本）存在几大致命痛点：
1. **静态且脆弱**：遇到未知的硬件架构、驱动版本冲突、系统路径特异性（例如 Windows 11 24H2 移除 `wmic`，鸿蒙 PC 采用 `musl libc`）立即报错崩溃，无法自愈。
2. **缺乏售后感知**：安装完成后无法持续守护，遇到端口被占、依赖被误删，只能人工排障。
3. **制造磁盘垃圾**：临时生成的 bat/sh 脚本满处乱扔，严重污染用户工作目录。
4. **大模型作弊隐患**：让通用 LLM 写脚本执行，模型经常产生幻觉并假装“已经成功配置完成”，实际服务根本没跑起来。

**「息壤」的解法**：
* **单兵空投**：编译后是纯原生单文件（Windows 下 `xirang.exe` 仅 6MB，包含 34KB 纯 C 单文件 `xirang.c`），无需任何运行环境。
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
| `-verify` | `false` | **0-Token DoD 极速网关**：无需调用大模型，直接测试验证规约中的验收命令 |
| `-skills` | `false` | **经验指纹透视**：查看并管理当前已沉淀的本地自愈脚本资产 |
| `-doctor` | `false` | **售后急诊医生模式**：交互式问诊台，随时粘贴任何报错，秒级自愈 |
| `-watchdog` | `false` | **售后常驻巡检模式**：静默心跳探活，服务挂掉时自动唤醒排障修复 |
| `-interval <秒>` | `30` | 售后巡检心跳间隔秒数 |
| `--rollback` | `false` | **安全事务一键回滚**：逆序撤销所有由息壤生成或修改的文件与配置 |
| `-forge` / `-gui` | `false` | **息壤工坊控制台**：启动图形化 Web UI 文件夹透视与规约创作中心 |

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

## 4. 息壤工坊图形控制台 (XiRang Studio GUI) 与 AES-256 加密打包

运行 `xirang -forge` 可启动息壤工坊图形控制台 (XiRang Studio)：

### 功能亮点：
1. **文件夹深度透视**：一键感知任意目录（如 U 盘或项目工程）的代码语言、依赖项与关键配置文件。
2. **AI 智能起草规约**：自动分析项目画像并起草 `xirang_task_spec.json`。
3. **平台预设派发 (包含鸿蒙 PC)**：支持勾选派发目标系统的二进制与适配启动脚本（`使用息壤配置.bat` / `run_xirang.sh` / `run_harmony.sh`）。
4. **AES-256 出厂熔炼**：通过 `xirang_builder` 或 GUI 熔炼独立单文件安装体，密文保护 API Key 与核心逻辑。

---

## 5. 四大核心典型实战用例

### 用例一：为客户制作本地大模型/私有算力一键安装器
* **场景**：客户有一台配置了新显卡的裸机，需要安装推理服务，且不能污染系统盘。
* **执行方式**：
  ```cmd
  xirang.exe -spec xirang_task_spec.json
  ```
* **息壤行为**：
  1. 自动探测驱动器空间，通过 `ask_human` 询问用户选择存放分卷（如 `D:\ollama_models`）。
  2. 探测 GPU 环境，绕开失效的 `wmic`，使用 PowerShell 或 `nvidia-smi`。
  3. 执行 `batch_actions` 并行建立配置文件与软链接。
  4. 触发 DoD 终态验收网关，真实发起单次 Prompt 推理，测试通过后交付。

---

### 用例二：复杂企业级分布式服务编排（并发多任务）
* **场景**：短时间内部署 WebUI、FastAPI 与 Redis 缓存。
* **执行方式**：
  ```bash
  ./xirang -task "并行拉起 Redis、FastAPI 与前端静态服务，修改各配置文件中的端口映射，完成后进行全链路联通性测试。"
  ```
* **息壤行为**：
  * 利用 `batch_actions` 并发派发线程，多子任务并行完成。若中途失败，执行 `./xirang --rollback` 1秒恢复。

---

### 用例三：华为鸿蒙 PC (HarmonyOS PC) 静默交付
* **场景**：在信创与华为鸿蒙 PC 环境部署软件，底层为 `musl libc` 且无 Bash 环境。
* **执行方式**：
  将 `xirang_harmony_pc_x64` 和 `run_harmony.sh` 拷贝至目标机执行：
  ```sh
  ./run_harmony.sh
  ```
* **息壤行为**：
  * 使用 Go 静态无依赖二进制，100% 原生适配鸿蒙 PC 的 `musl libc` 与 `mksh` 命令栈，无缝完成安装自愈。

---

### 用例四：售后常驻守护进程 (Watchdog) 与急诊医生模式 (Doctor)
* **场景**：客户因电脑休眠或端口冲突导致服务中断。
* **执行方式**：
  * 后台守护：`xirang.exe -watchdog -interval 30`
  * 现场排障：`xirang.exe -doctor`
* **息壤行为**：
  * 发现异常时自动唤醒排障内核；
  * `-doctor` 模式下优先匹配 `.xirang/scripts/` 的本地经验指纹，0.2 秒完成秒级自愈。

---

## 6. 经验资产化与本地自进化机制 (.xirang/)

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

---

## 7. AI 智能体 (Subagent) 集成「息壤」的最佳实践 Prompt 范式

当上层 AI Agent 需要调用息壤来完成现场环境部署时，推荐在 Prompt 中如下配置：

```markdown
【任务指派】：调用「息壤 (XiRang)」完成底层环境配置与持久化交付

你不需要在宿主机上亲自编写零散的 bat/sh 脚本，请直接调用现场提供的原生单文件程序 `xirang` / `xirang.exe`。

【执行准则】：
1. 规约下发：通过编写一份 `xirang_task_spec.json` 明确允许写入的白名单目录和终态验收命令 (DoD)。
2. 调用命令：运行 `xirang -spec xirang_task_spec.json` 触发全自动并发部署。
3. 校验网关：可在部署前/后调用 `xirang -spec xirang_task_spec.json -verify` 进行 0-Token 快速验收。
4. 售后防线：安装完成后，在后台启动 `xirang -watchdog -interval 30` 形成长效售后自愈。
5. 遇到偶发排障问题：随时使用 `xirang -doctor` 传入错误堆栈进行自愈。
```

---
*文档版本：v4.0 | 状态：最新发布 | 归属工程：XiRang Core Matrix*
