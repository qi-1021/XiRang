package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// 扩展健康探针：在原有 CheckCmd 之外支持 type 字段（通过 map 或扩展结构）
// 为兼容 JSON，我们把扩展字段挂在 HealthProbe 上（json 可选）

// probeHealth 执行单个探针，返回 ok 与详情
func probeHealth(p HealthProbe) (bool, string) {
	kind := strings.ToLower(strings.TrimSpace(p.Kind))
	target := strings.TrimSpace(p.Target)
	timeout := p.TimeoutSec
	if timeout <= 0 {
		timeout = 10
	}

	switch kind {
	case "", "cmd", "command":
		code, out := executeCommand(p.CheckCmd, timeout)
		return code == 0, strings.TrimSpace(out)
	case "tcp":
		if target == "" {
			return false, "tcp 探针缺少 target (host:port)"
		}
		conn, err := net.DialTimeout("tcp", target, time.Duration(timeout)*time.Second)
		if err != nil {
			return false, err.Error()
		}
		_ = conn.Close()
		return true, "tcp ok: " + target
	case "http", "https":
		url := target
		if url == "" {
			url = p.CheckCmd
		}
		if url == "" {
			return false, "http 探针缺少 target URL"
		}
		client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			return false, err.Error()
		}
		defer resp.Body.Close()
		ok := resp.StatusCode >= 200 && resp.StatusCode < 400
		return ok, fmt.Sprintf("HTTP %d", resp.StatusCode)
	case "file":
		path := target
		if path == "" {
			path = p.CheckCmd
		}
		if _, err := os.Stat(path); err != nil {
			return false, err.Error()
		}
		return true, "file exists: " + path
	default:
		code, out := executeCommand(p.CheckCmd, timeout)
		return code == 0, strings.TrimSpace(out)
	}
}

// defaultRealProbes 生成一组「真实」默认探针（非假 echo）
func defaultRealProbes() []HealthProbe {
	return []HealthProbe{
		{
			Name:       "工作目录可读",
			Kind:       "file",
			Target:     ".",
			CheckCmd:   ".",
			FixGoal:    "确认当前工作目录存在且可访问",
			TimeoutSec: 5,
		},
		{
			Name:       "回环 TCP 连通性",
			Kind:       "tcp",
			Target:     "127.0.0.1:1",
			CheckCmd:   "",
			FixGoal:    "本机 TCP 栈基本可用（预期连接失败时记为软警可忽略，若目标服务应监听则修复）",
			TimeoutSec: 3,
		},
	}
}
