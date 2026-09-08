# 🌱 息壤 (XiRang) v4.0: 工业级通用自愈操作系统智能体全景架构图谱

> **命名寓意**：“息壤者，言土自长息无限也。” 息壤是一款面向全平台（Windows / macOS / Linux / BSD / **华为鸿蒙 PC** / 嵌入式工控机）的通用伴生自愈智能体。零外部依赖、极低资源占用，具备多线并发编排、经验自沉淀、生生不息的系统级自愈与终身售后服务能力。

---

## 🏗️ 系统分层架构 (Architecture Overview)

```
+-----------------------------------------------------------------------------------+
|                        人类/AI 交互控制层 (Interaction Layer)                       |
|  - CLI 命令 (-task, -spec, -doctor, -watchdog, --rollback, -verify, -skills)      |
|  - 息壤工坊可视化创作控制台 (XiRang Studio GUI Web Dashboard / -forge)               |
+-----------------------------------------------------------------------------------+
                                          |
+-----------------------------------------------------------------------------------+
|                        核心元认知与调度大脑 (Core Engine)                           |
|  - 多线并发批处理引擎 (Parallel Batch Actions Pipeline)                            |
|  - 记忆与黑板便签机制 (Working Memory & Scratchpad Pruning)                         |
|  - 人类决策主权守卫 (Human Authority Guard: ask_human)                              |
+-----------------------------------------------------------------------------------+
                                          |
+-----------------------------------------------------------------------------------+
|                        安全防线与终态验收网关 (Safety & DoD Gate)                    |
|  - 沙箱白名单守卫 (PathGuard) & 危险指令过滤器 (Safety Filter)                      |
|  - 终态硬性防作弊验收网关 (DoD Gate: VerificationCmd)                               |
|  - 事务级原子备份撤销栈 (Rollback Stack: .xirang/backups/)                          |
+-----------------------------------------------------------------------------------+
                                          |
+-----------------------------------------------------------------------------------+
|                    经验资产化与 Fast-Path 秒级自愈 (.xirang/scripts/)               |
|  - 故障特征指纹匹配 (*.meta.json Error Fingerprint)                                 |
|  - 0.2 秒免 Token 快速自愈执行器 (Fast-Path Cache Engine)                           |
+-----------------------------------------------------------------------------------+
                                          |
+-----------------------------------------------------------------------------------+
|                    全平台全架构跨平台标靶矩阵 (Target Matrix / dist/)               |
|  - Windows (x64 / ARM64 / 32位老旧工控机)                                         |
|  - macOS (Apple Silicon M1-M4 / Intel)                                            |
|  - Linux (x64 / ARM64 / ARMv7 树莓派 / RISC-V / 龙芯 LoongArch / MIPS)             |
|  - 华为鸿蒙 PC / 开源鸿蒙 (HarmonyOS PC / OpenHarmony musl libc 静态标靶)           |
|  - C 原生嵌入式内核 (c_engine/xirang.c ANSI C 34KB)                                |
+-----------------------------------------------------------------------------------+
```

---

## 🌟 核心机制全景解构

### 1. 终态硬性防作弊网关 (DoD Gate & `-verify`)
- 强制校验业务验收指令（如推理命令、服务探针），未真实通过严禁宣告完成，防止模型“虚假完成”。
- 支持通过 `-verify` 参数进行 0-Token 极速 DoD 校验，无需调用大模型即可完成快速验收。

### 2. 多线并发批处理 (Parallel Batch Actions)
- 打破传统 Agent 单步串行等待的低效模式。
- 通过 `{"action": "batch_actions", "batch_actions": [...]}` 原生支持并发多命令、多文件操作，多线程并行执行并安全聚合回报，效率提升 300% 以上。

### 3. 本地经验持续生长与资产化 (.xirang/scripts/)
- **绝对整洁的工作空间**：严禁在宿主母目录制造临时脚本垃圾，所有实操、排障脚本全部归拢至 `.xirang/scripts/`。
- **经验复利**：生成的每个有效脚本自动标注 `.meta.json` 触发指纹，再次遇到同类问题无需重新摸索。
- **Fast-Path 0.2秒秒级自愈**：售后排障时直接匹配本地脚本库解决已知问题（0 Token 消耗，不等待大模型响应）。

### 4. 终身售后保障体系 (Doctor & Watchdog)
- **急诊医生模式 (`-doctor`)**：交互式问诊台，随时粘贴控制台报错或自然语言描述，秒级接管排障。
- **常驻守护模式 (`-watchdog`)**：30秒心跳自适应探针，关键服务一旦异常自动唤醒修复内核。
- **事务级撤销 (`--rollback`)**：基于原子备份栈逆向还原所有修改过的文件，恢复纯净状态。

### 5. 开发者 API 混淆防逆向 (AES-256-GCM Stripped)
- 通过 `xirang_builder` 将私有 API Key 与业务规约熔炼为高强度密文，编译剔除符号表（`-s -w`），分发绝对安全。

---

## 📦 跨平台多架构全矩阵产物 (dist/)

| 平台分类 | 目标文件名 | 架构适用与特点 |
| :--- | :--- | :--- |
| **Windows** | `xirang_win_x64.exe` (别名 `xirang.exe`) | 现代 x86_64 PC (Win 10/11 24H2) |
| | `xirang_win_arm64.exe` | 骁龙 X Elite / Surface Pro 等 ARM PC |
| | `xirang_win_x86_legacy.exe` | 32位老旧工控机 / 奔腾 / Win 7 |
| **macOS** | `xirang_mac_apple_silicon` (别名 `xirang`) | Apple M1/M2/M3/M4 系列 |
| | `xirang_mac_intel` | 传统 Intel x86_64 Mac |
| **Linux** | `xirang_linux_x64` | Ubuntu/Debian/CentOS/Arch 64位服务器 |
| | `xirang_linux_arm64_aarch64` | 鲲鹏/飞腾/树莓派4/5/开源鸿蒙 PC |
| | `xirang_linux_armv7_raspberrypi`| 树莓派 2/3、ARM 32位工控机 |
| | `xirang_linux_riscv64` | 算能、平头哥 RISC-V 现代化架构板卡 |
| | `xirang_linux_loongarch64` | 龙芯国产自主架构 LoongArch 64 |
| | `xirang_linux_mips64le` | 传统 MIPS 架构芯片 |
| **鸿蒙 PC**| `xirang_harmony_pc_x64` / `_arm64` | 华为鸿蒙 PC / OpenHarmony PC 零依赖 `musl libc` 静态无依赖标靶 |
| **BSD** | `xirang_freebsd_x64` | FreeBSD 高安全性网络节点 |
| **极致极小**| `c_engine/xirang.c` | 34KB 单文件 ANSI C，任意 C 编译器 0.2s 编译 |
