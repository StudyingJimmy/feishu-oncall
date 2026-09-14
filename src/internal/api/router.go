// Package api 入口层：组装 HTTP 路由。
//
// 当前提供健康检查；回调模式下再把飞书事件回调地址挂上来（长连接模式 events 传 nil）。
// handler 只做「解析请求 → 调 service → 组装响应」，不写业务规则。
package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// New 组装路由。
// events 为飞书事件回调处理器；为 nil 时只提供健康检查（长连接模式）。
func New(events http.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health)
	if events != nil {
		mux.Handle("/lark/event", events)
	}
	return mux
}

// health 存活探针。就绪探针（探活 MySQL / ES / 对象存储）等依赖装配好后再加。
func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"time":   time.Now().UTC(),
	})
}
