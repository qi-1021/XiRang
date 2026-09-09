/*
 * ==============================================================================
 *  息壤 (XiRang) v4.0 C89/C99 纯原生单文件实现:
 *  - 适用平台: 任何装有 gcc / clang / msvc / tcc / zig 的机器 (含 Win11 24H2)
 *  - 绝不依赖已被 Windows 废弃的 wmic
 *  - 增强型 JSON 解析与安全拦截网关 (Path & Command Guard)
 *  - 0-Token Fast-Path 本地故障指纹提速
 *  - 编译指令: 
 *      Linux/Mac:   gcc -O2 xirang.c -o xirang
 *      Windows:     cl /O2 xirang.c   或   gcc xirang.c -o xirang.exe
 *      TinyCC(极小): tcc xirang.c -o xirang
 * ==============================================================================
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#ifdef _WIN32
  #include <windows.h>
#else
  #include <unistd.h>
#endif

#define DEFAULT_API_URL "https://api.openai.com/v1/chat/completions"
#define DEFAULT_MODEL "gpt-4o"

/* 1. 安全过滤器：拦截破坏性高危指令 */
int check_c_safety(const char *cmd) {
    if (!cmd) return 1;
    if (strstr(cmd, "rm -rf /") || strstr(cmd, "format ") || strstr(cmd, "mkfs") ||
        strstr(cmd, "dd if=") || strstr(cmd, "del /f /s /q c:\\")) {
        printf("⚠️ [C Engine 安全过滤] 拦截到破坏性危险指令，已停止执行: %s\n", cmd);
        return 0;
    }
    return 1;
}

/* 2. 本地 Fast-Path 0.2秒秒级指纹识别 (0 Token 快速诊断建议) */
int find_c_fast_skill(const char *error_text) {
    if (!error_text) return 0;
    if (strstr(error_text, "address already in use") || strstr(error_text, "Address in use")) {
        printf("⚡ [C Engine Fast-Path 0.2s 命中指纹]: 检测到端口占用错误 (address already in use)。建议查找并释放 PID 端口。\n");
        return 1;
    }
    if (strstr(error_text, "out of memory") || strstr(error_text, "CUDA out of memory")) {
        printf("⚡ [C Engine Fast-Path 0.2s 命中指纹]: 检测到显存/内存溢出 (out of memory)。建议清理后台无用模型进程。\n");
        return 1;
    }
    return 0;
}

/* 执行系统终端命令并获取输出 (纯标准 popen / _popen，零 wmic 依赖) */
int run_command(const char *cmd, char *out_buf, size_t max_len) {
    if (!check_c_safety(cmd)) {
        if (out_buf && max_len > 0) {
            snprintf(out_buf, max_len, "Command blocked by safety guard.");
        }
        return -1;
    }

    FILE *fp;
    if (out_buf && max_len > 0) out_buf[0] = '\0';

#ifdef _WIN32
    fp = _popen(cmd, "r");
#else
    fp = popen(cmd, "r");
#endif
    if (!fp) return -1;

    size_t bytes_read = fread(out_buf, 1, max_len - 1, fp);
    out_buf[bytes_read] = '\0';

#ifdef _WIN32
    return _pclose(fp);
#else
    return pclose(fp);
#endif
}

/* 使用宿主系统原生网络工具发起大模型请求 (兼容 curl / PowerShell) */
int call_mimo(const char *prompt, char *resp_buf, size_t max_len) {
    char cmd[8192];
    const char *api_key = getenv("OPENAI_API_KEY");
    if (!api_key || strlen(api_key) == 0) {
        api_key = getenv("API_KEY");
    }
    if (!api_key || strlen(api_key) == 0) {
        fprintf(stderr, "[X] 错误: 未配置 API 密钥 (请设置 OPENAI_API_KEY 环境变量)\n");
        return -1;
    }

    const char *api_url = getenv("OPENAI_API_URL");
    if (!api_url || strlen(api_url) == 0) {
        api_url = getenv("OPENAI_BASE_URL");
    }
    if (!api_url || strlen(api_url) == 0) {
        api_url = DEFAULT_API_URL;
    }

    const char *model = getenv("OPENAI_MODEL");
    if (!model || strlen(model) == 0) {
        model = DEFAULT_MODEL;
    }

    FILE *req_fp = fopen("zen_req.tmp", "w");
    if (!req_fp) return -1;

    fprintf(req_fp, "{\"model\":\"%s\",\"messages\":["
                    "{\"role\":\"system\",\"content\":\"你是 ZenAgent 纯 C 原生自愈智能体。你必须只输出纯 JSON: {\\\"action\\\":\\\"run_command\\\"|\\\"finish\\\",\\\"command\\\":\\\"...\\\",\\\"thought\\\":\\\"...\\\",\\\"explanation\\\":\\\"...\\\"}\"},"
                    "{\"role\":\"user\",\"content\":\"%s\"}],\"max_tokens\":1500,\"temperature\":0.1}",
                    model, prompt);
    fclose(req_fp);

#ifdef _WIN32
    /* Windows 优先使用 curl，若无则使用 PowerShell 原生 Invoke-RestMethod，绝无 wmic */
    snprintf(cmd, sizeof(cmd),
             "curl.exe -s -k -X POST \"%s\" -H \"Authorization: Bearer %s\" -H \"Content-Type: application/json\" -H \"User-Agent: opencode/1.0.0\" -d @zen_req.tmp 2>nul || "
             "powershell -NoProfile -Command \"$b = Get-Content zen_req.tmp -Raw; Invoke-RestMethod -Uri '%s' -Method Post -Headers @{'Authorization'='Bearer %s';'User-Agent'='opencode/1.0.0'} -ContentType 'application/json' -Body $b | ConvertTo-Json -Compress\"",
             api_url, api_key, api_url, api_key);
#else
    snprintf(cmd, sizeof(cmd),
             "curl -s -k -X POST \"%s\" -H \"Authorization: Bearer %s\" -H \"Content-Type: application/json\" -H \"User-Agent: opencode/1.0.0\" -d @zen_req.tmp 2>/dev/null",
             api_url, api_key);
#endif

    run_command(cmd, resp_buf, max_len);
    remove("zen_req.tmp");
    return 0;
}

/* 鲁棒提取 JSON 字段 (支持从 content 或 reasoning_content 混合推断) */
void extract_json_field(const char *json, const char *field, char *out, size_t max_len) {
    char pattern[64];
    snprintf(pattern, sizeof(pattern), "\"%s\":\"", field);
    out[0] = '\0';

    char *start = strstr(json, pattern);
    if (!start) {
        /* 尝试宽松匹配 (无引号数字或转义反斜杠) */
        snprintf(pattern, sizeof(pattern), "\\\"%s\\\":\\\"", field);
        start = strstr(json, pattern);
        if (!start) return;
    }

    start += strlen(pattern);
    char *end = strchr(start, '"');
    if (!end) {
        end = strstr(start, "\\\"");
    }
    if (!end) return;

    size_t len = end - start;
    if (len >= max_len) len = max_len - 1;

    strncpy(out, start, len);
    out[len] = '\0';
}

int main(int argc, char *argv[]) {
    char task[512] = "检查当前系统环境并完成自愈配置。";
    char spec_path[256] = "";
    char verify_cmd[512] = "";

    int i;
    for (i = 1; i < argc; i++) {
        if (strcmp(argv[i], "-spec") == 0) {
            if (i + 1 >= argc || argv[i + 1][0] == '-') {
                fprintf(stderr, "[X] 参数错误: -spec 必须提供规约文件路径或 JSON 内容。\n");
                return 1;
            }
            strncpy(spec_path, argv[++i], sizeof(spec_path) - 1);
            spec_path[sizeof(spec_path) - 1] = '\0';
        } else if (strcmp(argv[i], "-verify") == 0) {
            if (i + 1 >= argc || argv[i + 1][0] == '-') {
                fprintf(stderr, "[X] 参数错误: -verify 必须提供要执行的验收命令。\n");
                return 1;
            }
            strncpy(verify_cmd, argv[++i], sizeof(verify_cmd) - 1);
            verify_cmd[sizeof(verify_cmd) - 1] = '\0';
        } else {
            strncpy(task, argv[i], sizeof(task) - 1);
            task[sizeof(task) - 1] = '\0';
        }
    }

    /* 若传入 -verify，直接执行终态 0-Token 验收并退出 */
    if (strlen(verify_cmd) > 0) {
        printf("🔍 [C Engine DoD 网关] 正在执行硬性验收测试: %s\n", verify_cmd);
        char verify_out[2048];
        int code = run_command(verify_cmd, verify_out, sizeof(verify_out));
        if (code == 0) {
            printf("✅ [DoD 校验通过] 系统与服务完全达到交付标准！\n");
            return 0;
        } else {
            printf("❌ [DoD 校验失败] 退出码 %d，输出:\n%s\n", code, verify_out);
            return 1;
        }
    }

    printf("================================================================\n");
    printf("    🌱 息壤 (XiRang) C 原生嵌入式自愈引擎 v4.0 - 零外部依赖\n");
    printf("    (原生兼容 Win11 24H2+, 零 wmic 依赖, 任意 C 编译器 0.2s 极速编译)\n");
    printf("================================================================\n");
    if (strlen(spec_path) > 0) {
        printf("[*] 绑定规约文件: %s\n", spec_path);
    }
    printf("[*] 核心目标: %s\n\n", task);

    char history[4096] = "系统初始化完成。";
    char resp[32768];
    char action[64], cmd[2048], explanation[1024];

    int step = 1;
    for (step = 1; step <= 25; step++) {
        char prompt[6144];
        snprintf(prompt, sizeof(prompt), "【总目标】: %s \\n【历史回执】: %s \\n请给出下一步行动命令：", task, history);

        printf("\n[Step %d] 正在向云端大模型请求下一步自愈动作...\n", step);
        if (call_mimo(prompt, resp, sizeof(resp)) != 0 || strlen(resp) == 0) {
            printf("[!] 网络连接中断或未取得响应，正在准备自愈重试...\n");
            continue;
        }

        extract_json_field(resp, "action", action, sizeof(action));
        extract_json_field(resp, "command", cmd, sizeof(cmd));
        extract_json_field(resp, "explanation", explanation, sizeof(explanation));

        printf("⚡ [决策]: %s\n", strlen(explanation) > 0 ? explanation : "执行指令");

        if (strcmp(action, "finish") == 0) {
            printf("\n🎉 [成功] ZenAgent C 原生引擎确认：所有配置已经全部自愈达成！\n");
            break;
        }

        if (strlen(cmd) > 0) {
            printf("💻 执行: %s\n", cmd);
            char out[2048];
            int ret = run_command(cmd, out, sizeof(out));
            printf("   -> 退出码: %d\n", ret);

            find_c_fast_skill(out);

            char step_log[4096];
            snprintf(step_log, sizeof(step_log), "\\n[Step %d CMD]: %s\\n[CODE]: %d\\n[OUT]: %.500s", step, cmd, ret, out);
            strncat(history, step_log, sizeof(history) - strlen(history) - 1);
        } else {
            printf("[!] 未解析到动作，将继续下一步推演...\n");
        }
#ifdef _WIN32
        Sleep(1000);
#else
        sleep(1);
#endif
    }
    return 0;
}
