// 回调模式：飞书把事件 POST 到我们的 HTTP 地址（生产使用）。
//
// 只做「读 body → 交给 SDK 分发器（解密 / URL 校验 / token 校验 / 分发）→ 回写响应」；
// 事件的实际处理在 dispatcher.go 里异步执行，保证 3 秒内响应。
package lark

import (
	"io"
	"net/http"

	larkevent "github.com/larksuite/oapi-sdk-go/v3/event"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"

	"bokeoncall/internal/conf"
	"bokeoncall/internal/model/dto"
)

// maxCallbackBody 回调体上限，正常请求远小于此。
const maxCallbackBody = 1 << 20

// CallbackHandler 飞书事件回调处理器（实现 http.Handler，可挂到任意 Web 框架）。
type CallbackHandler struct {
	dispatcher *dispatcher.EventDispatcher
}

// NewCallbackHandler 构造。
func NewCallbackHandler(cfg conf.LarkConfig, sink dto.Sink) *CallbackHandler {
	return &CallbackHandler{dispatcher: newDispatcher(cfg, sink)}
}

// ServeHTTP 处理飞书回调。
func (h *CallbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxCallbackBody))
	if err != nil {
		http.Error(w, "读取请求体失败", http.StatusBadRequest)
		return
	}

	resp := h.dispatcher.Handle(r.Context(), &larkevent.EventReq{
		Header:     r.Header,
		Body:       body,
		RequestURI: r.URL.RequestURI(),
	})
	if resp == nil {
		w.WriteHeader(http.StatusOK)
		return
	}
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(resp.Body)
}
