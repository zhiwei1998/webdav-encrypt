package proxy

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"
	"time"
)

// director 修改请求以指向后端服务器
func (h *ProxyHandler) director(req *http.Request) {
	h.logger.Debug("[DIRECTOR] 开始处理请求转发: %s %s", req.Method, req.URL.Path)
	
	// 设置目标URL
	req.URL.Scheme = h.backend.Scheme
	req.URL.Host = h.backend.Host
	h.logger.Debug("[DIRECTOR] 后端服务器: %s://%s", req.URL.Scheme, req.URL.Host)
	
	// 处理路径拼接，避免重复添加后端路径
	reqPath := req.URL.Path
	backendPath := h.backend.Path
	h.logger.Debug("[DIRECTOR] 请求路径: %s, 后端路径: %s", reqPath, backendPath)
	
	// 如果客户端请求路径已经包含后端路径的前缀，直接使用客户端路径
	if strings.HasPrefix(reqPath, backendPath) {
		req.URL.Path = reqPath
		h.logger.Debug("[DIRECTOR] 路径已经包含后端前缀，使用请求路径: %s", req.URL.Path)
	} else {
		// 否则拼接后端路径和客户端路径
		req.URL.Path = singleJoiningSlash(backendPath, reqPath)
		h.logger.Debug("[DIRECTOR] 拼接路径: %s", req.URL.Path)
	}
	
	// 保留查询参数
	if h.backend.RawQuery == "" || req.URL.RawQuery == "" {
		req.URL.RawQuery = h.backend.RawQuery + req.URL.RawQuery
	} else {
		req.URL.RawQuery = h.backend.RawQuery + "&" + req.URL.RawQuery
	}
	h.logger.Debug("[DIRECTOR] 完整URL: %s", req.URL.String())
	
	// 修改Host头
	req.Host = h.backend.Host
	h.logger.Debug("[DIRECTOR] 设置Host头: %s", req.Host)
	
	// 添加后端认证头
	if h.backendAuth != nil {
		auth := h.backendAuth.Username + ":" + h.backendAuth.Password
		basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
		req.Header.Set("Authorization", basicAuth)
		h.logger.Debug("[DIRECTOR] 添加后端认证头")
	} else {
		h.logger.Debug("[DIRECTOR] 未设置后端认证")
	}
	
	// 移除Hop-by-hop头部
	removeHopHeaders(req.Header)
	h.logger.Debug("[DIRECTOR] 请求头: %v", req.Header)
	
	// 设置超时上下文
	ctx, cancel := context.WithTimeout(req.Context(), 300*time.Second)
	*req = *req.WithContext(ctx)
	// 将cancel函数保存到请求上下文中，以便在请求完成后调用
	// 注意：不要在这里立即调用cancel()，否则会立即取消请求
	req = req.WithContext(context.WithValue(req.Context(), "cancel", cancel))
	h.logger.Debug("[DIRECTOR] 设置请求超时: 300秒")
}
