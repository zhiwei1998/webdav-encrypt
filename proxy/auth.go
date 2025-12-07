package proxy

import (
	"net/http"
	"webdav-proxy/utils"
)

// proxyAuthMiddleware 代理端认证中间件
type proxyAuthMiddleware struct {
	handler    http.Handler
	authConfig *ProxyAuthConfig
	logger     utils.Logger
}

// NewProxyAuthMiddleware 创建代理端认证中间件
func NewProxyAuthMiddleware(handler http.Handler, authConfig *ProxyAuthConfig) http.Handler {
	if authConfig == nil || !authConfig.Enabled {
		return handler
	}
	
	// 从handler中获取logger
	proxyHandler, ok := handler.(*ProxyHandler)
	var logger utils.Logger
	if ok {
		logger = proxyHandler.logger
	} else {
		// 如果无法获取，创建一个默认的
		logger = utils.NewDefaultLogger(false)
	}
	
	return &proxyAuthMiddleware{
		handler:    handler,
		authConfig: authConfig,
		logger:     logger,
	}
}

// ServeHTTP 实现http.Handler接口
func (m *proxyAuthMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.logger.Debug("[AUTH] 收到请求: %s %s", r.Method, r.URL.Path)
	m.logger.Debug("[AUTH] 客户端地址: %s", r.RemoteAddr)
	m.logger.Debug("[AUTH] 请求头: %v", r.Header)
	
	if !m.checkAuth(r) {
		m.logger.Error("[AUTH] 认证失败: %s %s", r.Method, r.URL.Path)
		w.Header().Set("WWW-Authenticate", `Basic realm="WebDAV Proxy"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	m.logger.Debug("[AUTH] 认证成功: %s %s", r.Method, r.URL.Path)
	m.handler.ServeHTTP(w, r)
}

// checkAuth 检查基本认证
func (m *proxyAuthMiddleware) checkAuth(r *http.Request) bool {
	username, password, ok := r.BasicAuth()
	if !ok {
		m.logger.Debug("[AUTH] 无法解析认证信息")
		return false
	}
	
	m.logger.Debug("[AUTH] 尝试认证用户: %s", username)
	isValid := username == m.authConfig.Username && password == m.authConfig.Password
	if !isValid {
		m.logger.Debug("[AUTH] 用户名或密码不正确: %s", username)
	}
	return isValid
}