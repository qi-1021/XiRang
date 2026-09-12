package main

import (
	"encoding/json"
	"fmt"
	"time"
)

// jsonMarshalCompact 供审计使用，避免与主包命名冲突的细微差异
func jsonMarshalCompact(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// backoffDelay 计算指数退避（带抖动上限保护）
func backoffDelay(attempt int, base time.Duration) time.Duration {
	if base <= 0 {
		base = time.Second
	}
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 6 {
		attempt = 6
	}
	d := base
	for i := 1; i < attempt; i++ {
		d *= 2
	}
	// 简单确定性抖动，避免 thundering herd 过重
	jitter := time.Duration(attempt*37) * time.Millisecond
	return d + jitter
}

// callLLMWithRetry 在 callLLM 之上增加：单 provider 多次退避 + 跨 provider failover
func callLLMWithRetry(providers []Provider, messages []map[string]string, maxRetriesPerProvider int) (*Decision, string, error) {
	if maxRetriesPerProvider <= 0 {
		maxRetriesPerProvider = 2
	}
	if len(providers) == 0 {
		return nil, "", fmt.Errorf("无可用模型通道")
	}

	var lastErr error
	for pi, p := range providers {
		for attempt := 1; attempt <= maxRetriesPerProvider; attempt++ {
			decision, provName, err := callLLM([]Provider{p}, messages)
			if err == nil {
				return decision, provName, nil
			}
			lastErr = fmt.Errorf("provider[%s] attempt %d: %w", p.Name, attempt, err)
			if attempt < maxRetriesPerProvider {
				time.Sleep(backoffDelay(attempt, time.Second))
			}
		}
		// 下一个 provider
		_ = pi
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("所有模型通道均失败")
	}
	return nil, "", lastErr
}
