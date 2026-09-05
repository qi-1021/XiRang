package gui

import (
	"net/http"
)

const IndexHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>息壤工坊 · 创作控制台 (XiRang Studio)</title>
    <style>
        :root {
            --bg-base: #0f141c;
            --bg-card: #18202c;
            --bg-input: #121822;
            --border: #2b384e;
            --border-hover: #486288;
            --accent: #10b981;
            --accent-hover: #059669;
            --accent-glow: rgba(16, 185, 129, 0.2);
            --gold: #f59e0b;
            --gold-hover: #d97706;
            --text-main: #f1f5f9;
            --text-sub: #94a3b8;
            --text-muted: #64748b;
            --danger: #ef4444;
            --tag-bg: #1e293b;
        }
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
            background: var(--bg-base);
            color: var(--text-main);
            min-height: 100vh;
            padding: 24px;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            margin-bottom: 24px;
            padding-bottom: 16px;
            border-bottom: 1px solid var(--border);
        }
        .logo-box {
            display: flex;
            align-items: center;
            gap: 12px;
        }
        .logo-icon {
            font-size: 28px;
            background: rgba(16, 185, 129, 0.15);
            border: 1px solid var(--accent);
            padding: 6px 12px;
            border-radius: 8px;
        }
        .logo-title h1 {
            font-size: 20px;
            font-weight: 700;
            color: #fff;
            letter-spacing: 0.5px;
        }
        .logo-title p {
            font-size: 13px;
            color: var(--text-sub);
            margin-top: 2px;
        }
        .badge {
            font-size: 12px;
            padding: 4px 10px;
            background: rgba(16, 185, 129, 0.1);
            color: var(--accent);
            border: 1px solid var(--accent);
            border-radius: 20px;
            font-weight: 500;
        }

        .grid-layout {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 20px;
        }
        @media (max-width: 900px) {
            .grid-layout { grid-template-columns: 1fr; }
        }

        .card {
            background: var(--bg-card);
            border: 1px solid var(--border);
            border-radius: 12px;
            padding: 20px;
            box-shadow: 0 4px 20px rgba(0, 0, 0, 0.25);
        }
        .card-header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            margin-bottom: 16px;
            font-size: 15px;
            font-weight: 600;
            color: #fff;
            border-left: 3px solid var(--accent);
            padding-left: 10px;
        }
        .form-group {
            margin-bottom: 16px;
        }
        label {
            display: block;
            font-size: 13px;
            font-weight: 500;
            color: var(--text-sub);
            margin-bottom: 6px;
        }
        input[type="text"], input[type="password"], textarea, select {
            width: 100%;
            background: var(--bg-input);
            border: 1px solid var(--border);
            border-radius: 8px;
            color: var(--text-main);
            padding: 10px 14px;
            font-size: 13px;
            outline: none;
            transition: border-color 0.2s, box-shadow 0.2s;
            font-family: inherit;
        }
        input[type="text"]:focus, input[type="password"]:focus, textarea:focus, select:focus {
            border-color: var(--accent);
            box-shadow: 0 0 0 2px var(--accent-glow);
        }
        textarea {
            resize: vertical;
            min-height: 80px;
            font-family: Menlo, Monaco, Consolas, monospace;
            font-size: 12px;
            line-height: 1.5;
        }
        .input-row {
            display: flex;
            gap: 10px;
        }
        button {
            cursor: pointer;
            border: none;
            border-radius: 8px;
            font-size: 13px;
            font-weight: 600;
            padding: 10px 18px;
            display: inline-flex;
            align-items: center;
            justify-content: center;
            gap: 6px;
            transition: all 0.2s;
        }
        .btn-accent {
            background: var(--accent);
            color: #0f172a;
        }
        .btn-accent:hover {
            background: var(--accent-hover);
        }
        .btn-gold {
            background: var(--gold);
            color: #0f172a;
        }
        .btn-gold:hover {
            background: var(--gold-hover);
        }
        .btn-secondary {
            background: #243144;
            color: var(--text-main);
            border: 1px solid var(--border);
        }
        .btn-secondary:hover {
            background: #31425c;
        }
        .tag-list {
            display: flex;
            flex-wrap: wrap;
            gap: 6px;
            margin-top: 8px;
        }
        .tag-item {
            font-size: 11px;
            background: var(--tag-bg);
            color: #38bdf8;
            border: 1px solid #0369a1;
            padding: 2px 8px;
            border-radius: 4px;
        }
        .snippet-box {
            background: var(--bg-input);
            border: 1px solid var(--border);
            border-radius: 8px;
            padding: 12px;
            max-height: 140px;
            overflow-y: auto;
            font-family: Menlo, Monaco, Consolas, monospace;
            font-size: 11px;
            color: var(--text-sub);
            white-space: pre-wrap;
            line-height: 1.4;
        }
        .notice-box {
            background: rgba(245, 158, 11, 0.08);
            border: 1px solid rgba(245, 158, 11, 0.3);
            border-radius: 8px;
            padding: 10px 14px;
            font-size: 12px;
            color: #fcd34d;
            margin-bottom: 16px;
            line-height: 1.5;
        }
        .status-msg {
            margin-top: 12px;
            padding: 10px;
            border-radius: 8px;
            font-size: 12px;
            display: none;
        }
        .status-success {
            display: block;
            background: rgba(16, 185, 129, 0.15);
            border: 1px solid var(--accent);
            color: #6ee7b7;
        }
        .status-error {
            display: block;
            background: rgba(239, 68, 68, 0.15);
            border: 1px solid var(--danger);
            color: #fca5a5;
        }
        .loading {
            opacity: 0.6;
            pointer-events: none;
        }
    </style>
</head>
<body>
<div class="container">
    <header>
        <div class="logo-box">
            <div class="logo-icon">🌱</div>
            <div class="logo-title">
                <h1>息壤工坊 · XiRang Studio</h1>
                <p>轻量智能工坊 · 文件夹透视 · AI自主规约 · 单文件加密熔炼</p>
            </div>
        </div>
        <div class="badge">“帮人修复，决不替人决策”</div>
    </header>

    <div class="notice-box">
        <strong>💡 核心使命与人类主权：</strong>
        息壤可自动探查任意工作目录的内容与依赖结构，并由 AI 辅助起草自愈与验收规约；但最终的执行路径、验收指令与分发权限由人类用户完全决定。出厂分发时支持 AES-256-GCM 强加密防逆向。
    </div>

    <!-- 顶部凭据栏 -->
    <div class="card" style="margin-bottom: 20px;">
        <div class="card-header">
            <span>🔑 大模型凭据配置 (用于 AI 辅助透视与制作)</span>
            <span style="font-size: 12px; color: var(--text-muted); font-weight: normal;">零系统嗅探 · 独立安全凭据</span>
        </div>
        <div style="display: grid; grid-template-columns: 2fr 1fr 1.5fr; gap: 12px;">
            <div>
                <label>API 端点 URL</label>
                <input type="text" id="api_url" value="https://api.openai.com/v1" placeholder="例如 https://api.deepseek.com/v1 或官方 OpenAI 端点">
            </div>
            <div>
                <label>模型名称</label>
                <input type="text" id="model_name" value="gpt-4o" placeholder="如 gpt-4o, deepseek-chat">
            </div>
            <div>
                <label>API Key (密钥)</label>
                <input type="password" id="api_key" placeholder="sk-...">
            </div>
        </div>
    </div>

    <div class="grid-layout">
        <!-- 左侧：工作文件夹透视与 AI 分析 -->
        <div class="card">
            <div class="card-header">
                <span>📁 第一步：目标工作文件夹透视</span>
            </div>
            <div class="form-group">
                <label>目标工作文件夹路径 (绝对路径或相对路径)</label>
                <div class="input-row">
                    <input type="text" id="folder_path" placeholder="如: /Volumes/齐-U盘/Windows_Transfer_Pack 或 D:\MyProject" value=".">
                    <button class="btn-secondary" onclick="inspectFolder()">🔍 探测扫描</button>
                </div>
            </div>

            <div id="inspect_result" style="display: none;">
                <div class="form-group">
                    <label>扫描指纹特征标签</label>
                    <div id="tag_container" class="tag-list"></div>
                </div>
                <div class="form-group">
                    <label>检测到的关键配置/脚本文件</label>
                    <div id="key_files" style="font-size: 12px; color: var(--text-main); line-height: 1.6;"></div>
                </div>
                <div class="form-group">
                    <label>目录快照摘要</label>
                    <div id="tree_snippet" class="snippet-box"></div>
                </div>
            </div>

            <hr style="border: 0; border-top: 1px solid var(--border); margin: 20px 0;">

            <div class="card-header">
                <span>🤖 第二步：AI 参谋辅助设计任务</span>
            </div>
            <div class="form-group">
                <label>告诉 AI 你的具体诉求 / 人类期望 (可选补充)</label>
                <textarea id="user_intent" placeholder="例如：这是给某客户电脑安装本地知识库的环境，要求自愈排查 Python 虚拟环境与端口 8080 是否被占用，严禁动系统驱动，帮我写出严密的规约。"></textarea>
            </div>
            <button class="btn-accent" style="width: 100%;" onclick="askAIToDraftSpec()">
                ✨ 让 AI 深度分析此目录并自动起草完整任务规约
            </button>
            <div id="ai_status" class="status-msg"></div>
        </div>

        <!-- 右侧：规约编辑器与交付打包 -->
        <div class="card">
            <div class="card-header">
                <span>📝 第三步：审阅与定制息壤任务规约 (Task Spec)</span>
                <span style="font-size: 12px; color: var(--text-muted); font-weight: normal;">人类最终决策权</span>
            </div>

            <div class="form-group">
                <label>任务名称 (Task Name)</label>
                <input type="text" id="task_name" value="通用自愈与环境修复任务">
            </div>

            <div class="form-group">
                <label>核心运维目标 (Goal)</label>
                <textarea id="task_goal" style="min-height: 70px;" placeholder="明确息壤在此目录下的最终目标"></textarea>
            </div>

            <div class="form-group">
                <label>硬性终态验收命令 (Verification Cmd - 防作弊与真实交付红线)</label>
                <input type="text" id="verify_cmd" placeholder="例如: python -c 'import torch; print(torch.__version__)' 或 curl -s http://127.0.0.1:8080">
            </div>

            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 10px;">
                <div class="form-group">
                    <label>操作白名单路径 (逗号分隔)</label>
                    <input type="text" id="allowed_paths" value="., ./.xirang">
                </div>
                <div class="form-group">
                    <label>绝对禁止命令 (逗号分隔)</label>
                    <input type="text" id="forbidden_cmds" value="rm -rf /, format, mkfs, dd if=">
                </div>
            </div>

            <div class="form-group">
                <label>里程碑规划 (Milestones JSON 数组)</label>
                <textarea id="milestones_json" style="min-height: 90px;" placeholder="[{...}]"></textarea>
            </div>

            <hr style="border: 0; border-top: 1px solid var(--border); margin: 20px 0;">

            <div class="card-header">
                <span>🚀 第四步：交付与植入</span>
            </div>

            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px;">
                <button class="btn-secondary" style="height: 48px;" onclick="deployInception()">
                    🌱 就地植入副本<br><span style="font-size: 11px; font-weight: normal; color: var(--text-sub);">写出规约与双击启动脚本</span>
                </button>
                <button class="btn-gold" style="height: 48px;" onclick="buildEncryptedBinary()">
                    🔒 熔炼独立加密程序<br><span style="font-size: 11px; font-weight: normal; color: #0f172a;">AES-256 出厂防篡改打包</span>
                </button>
            </div>

            <div id="deploy_status" class="status-msg"></div>
        </div>
    </div>
</div>

<script>
    let currentScanData = null;

    async function inspectFolder() {
        const path = document.getElementById("folder_path").value.trim();
        const btn = event.target;
        btn.classList.add("loading");
        try {
            const resp = await fetch("/api/inspect", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ path })
            });
            const data = await resp.json();
            currentScanData = data;
            if (!data.exists) {
                alert("未找到该目录或不是有效文件夹：" + data.path);
                return;
            }
            document.getElementById("folder_path").value = data.path;
            document.getElementById("inspect_result").style.display = "block";

            // 渲染标签
            const tagBox = document.getElementById("tag_container");
            tagBox.innerHTML = "";
            (data.project_tags || ["未分类项目"]).forEach(t => {
                const span = document.createElement("span");
                span.className = "tag-item";
                span.innerText = t;
                tagBox.appendChild(span);
            });

            // 渲染关键文件
            document.getElementById("key_files").innerText = (data.key_files && data.key_files.length > 0)
                ? data.key_files.join(", ")
                : "未检测到常见构建/脚本文件";

            // 渲染快照
            document.getElementById("tree_snippet").innerText = data.tree_snippet || "空目录";

            // 默认白名单填入当前绝对路径
            document.getElementById("allowed_paths").value = data.path + ", .";
        } catch (e) {
            alert("探测失败: " + e.message);
        } finally {
            btn.classList.remove("loading");
        }
    }

    async function askAIToDraftSpec() {
        const apiKey = document.getElementById("api_key").value.trim();
        if (!apiKey) {
            alert("请先在顶部填入您的 API Key 凭据！");
            document.getElementById("api_key").focus();
            return;
        }
        if (!currentScanData) {
            alert("请先点击【探测扫描】目标文件夹！");
            return;
        }

        const statusEl = document.getElementById("ai_status");
        statusEl.className = "status-msg";
        statusEl.innerText = "⏳ 正在连线大模型，深度理解此文件夹并起草严谨任务规约...";
        statusEl.style.display = "block";

        try {
            const req = {
                folder_path: currentScanData.path,
                api_url: document.getElementById("api_url").value.trim(),
                api_key: apiKey,
                model: document.getElementById("model_name").value.trim(),
                key_files: currentScanData.key_files,
                tree_snippet: currentScanData.tree_snippet,
                user_intent: document.getElementById("user_intent").value.trim()
            };

            const resp = await fetch("/api/ai_analyze", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(req)
            });

            if (!resp.ok) {
                const errText = await resp.text();
                throw new Error(errText);
            }

            const spec = await resp.json();
            document.getElementById("task_name").value = spec.task_name || "";
            document.getElementById("task_goal").value = spec.goal || "";
            document.getElementById("verify_cmd").value = spec.verification_cmd || "";
            if (spec.allowed_paths) document.getElementById("allowed_paths").value = spec.allowed_paths.join(", ");
            if (spec.forbidden_cmds) document.getElementById("forbidden_cmds").value = spec.forbidden_cmds.join(", ");
            document.getElementById("milestones_json").value = JSON.stringify(spec.milestones || [], null, 2);

            statusEl.className = "status-msg status-success";
            statusEl.innerText = "✅ AI 规约起草完成！您可在右侧进行人工微调和拍板确认。";
        } catch (e) {
            statusEl.className = "status-msg status-error";
            statusEl.innerText = "❌ AI 起草失败: " + e.message;
        }
    }

    function getFormTaskSpec() {
        let milestones = [];
        try {
            milestones = JSON.parse(document.getElementById("milestones_json").value || "[]");
        } catch (e) {
            milestones = [];
        }
        const allowed = document.getElementById("allowed_paths").value.split(",").map(s => s.trim()).filter(Boolean);
        const forbidden = document.getElementById("forbidden_cmds").value.split(",").map(s => s.trim()).filter(Boolean);

        return {
            task_name: document.getElementById("task_name").value.trim(),
            goal: document.getElementById("task_goal").value.trim(),
            allowed_paths: allowed,
            forbidden_cmds: forbidden,
            verification_cmd: document.getElementById("verify_cmd").value.trim(),
            milestones: milestones,
            strict_rules: ["帮人修复，决不替人决策", "禁止违规执行破坏性命令"]
        };
    }

    async function deployInception() {
        const folder = document.getElementById("folder_path").value.trim();
        const spec = getFormTaskSpec();
        const statusEl = document.getElementById("deploy_status");
        statusEl.className = "status-msg";
        statusEl.innerText = "正在向目标文件夹写入规约与双击启动器...";
        statusEl.style.display = "block";

        try {
            const resp = await fetch("/api/incept", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    target_folder: folder,
                    task_spec: spec,
                    model_config: {
                        url: document.getElementById("api_url").value.trim(),
                        key: document.getElementById("api_key").value.trim(),
                        model: document.getElementById("model_name").value.trim()
                    }
                })
            });
            const res = await resp.json();
            statusEl.className = "status-msg status-success";
            statusEl.innerText = "🎉 " + res.message;
        } catch (e) {
            statusEl.className = "status-msg status-error";
            statusEl.innerText = "❌ 植入失败: " + e.message;
        }
    }

    async function buildEncryptedBinary() {
        const apiKey = document.getElementById("api_key").value.trim();
        if (!apiKey) {
            alert("熔炼独立加密二进制需要内置模型凭据，请在顶部填入 API Key！");
            return;
        }
        const spec = getFormTaskSpec();
        const statusEl = document.getElementById("deploy_status");
        statusEl.className = "status-msg";
        statusEl.innerText = "⏳ 正在进行 AES-256-GCM 强加密并调用 Go 编译器交叉编译独立程序，请稍候...";
        statusEl.style.display = "block";

        try {
            const resp = await fetch("/api/build", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    target_os: "windows",
                    target_arch: "amd64",
                    output_name: (spec.task_name || "xirang_installer") + ".exe",
                    secret_key: "xirang-forge-" + Date.now(),
                    task_spec: spec,
                    model_config: {
                        url: document.getElementById("api_url").value.trim(),
                        key: apiKey,
                        model: document.getElementById("model_name").value.trim()
                    }
                })
            });
            const res = await resp.json();
            if (res.status === "ok") {
                statusEl.className = "status-msg status-success";
                statusEl.innerText = "🎉 " + res.message;
            } else {
                throw new Error(res.message || "未知错误");
            }
        } catch (e) {
            statusEl.className = "status-msg status-error";
            statusEl.innerText = "❌ 熔炼失败: " + e.message;
        }
    }

    // 页面加载完成自动探测当前目录
    window.onload = () => {
        inspectFolder();
    };
</script>
</body>
</html>`

func (s *ForgeServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(IndexHTML))
}
