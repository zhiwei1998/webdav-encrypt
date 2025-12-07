package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"webdav-proxy/encryption"
	"webdav-proxy/utils"
)

// BackendAuthConfig 后端认证配置
type BackendAuthConfig struct {
	Username string
	Password string
}

// ProxyAuthConfig 代理端认证配置
type ProxyAuthConfig struct {
	Enabled  bool
	Username string
	Password string
}

// ProxyHandler WebDAV代理处理器
type ProxyHandler struct {
	backend     *url.URL
	password    string
	algorithm   string
	chunkSize   int
	backendAuth *BackendAuthConfig
	proxyAuth   *ProxyAuthConfig
	logger      utils.Logger

	// 性能配置
	timeout             time.Duration
	maxIdleConns        int
	maxIdleConnsPerHost int
	idleConnTimeout     time.Duration

	// 反向代理
	reverseProxy *httputil.ReverseProxy

	// 缓存加密器（按文件大小）
	encryptorCache sync.Map

	// 加密器缓存清理定时器
	encryptorCleanupTicker *time.Ticker
	stopCleanupChan        chan struct{}
}

// NewProxyHandler 创建新的代理处理器
func NewProxyHandler(backend *url.URL, password, algorithm string, chunkSize int,
	backendAuth *BackendAuthConfig, proxyAuth *ProxyAuthConfig, logger utils.Logger,
	timeout time.Duration, maxIdleConns, maxIdleConnsPerHost int, idleConnTimeout time.Duration) (*ProxyHandler, error) {

	h := &ProxyHandler{
		backend:             backend,
		password:            password,
		algorithm:           algorithm,
		chunkSize:           chunkSize,
		backendAuth:         backendAuth,
		proxyAuth:           proxyAuth,
		logger:              logger,
		timeout:             timeout,
		maxIdleConns:        maxIdleConns,
		maxIdleConnsPerHost: maxIdleConnsPerHost,
		idleConnTimeout:     idleConnTimeout,
		stopCleanupChan:     make(chan struct{}),
	}

	// 创建传输层
	transport := h.createTransport()

	// 创建反向代理
	h.reverseProxy = &httputil.ReverseProxy{
		Director:       h.director,
		ModifyResponse: h.modifyResponse,
		ErrorHandler:   h.errorHandler,
		Transport:      transport,
	}

	// 启动加密器缓存清理协程
	h.startEncryptorCleanup()

	return h, nil
}

// ServeHTTP 处理所有HTTP请求
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 记录详细请求信息
	h.logger.Debug("[REQUEST] 收到请求: %s %s", r.Method, r.URL.Path)
	h.logger.Debug("[REQUEST] 客户端地址: %s", r.RemoteAddr)
	h.logger.Debug("[REQUEST] 请求头: %v", r.Header)

	// 处理WebDAV特殊方法
	switch r.Method {
	case "GET", "HEAD", "POST", "PUT", "DELETE",
		"PROPFIND", "PROPPATCH", "MKCOL", "COPY",
		"MOVE", "LOCK", "UNLOCK":
		// 直接使用反向代理处理请求
		h.reverseProxy.ServeHTTP(w, r)
	default:
		h.logger.Debug("[REQUEST] 不支持的请求方法: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// modifyResponse 修改后端响应
func (h *ProxyHandler) modifyResponse(resp *http.Response) error {
	// 移除后端认证相关的响应头，避免泄露信息
	resp.Header.Del("WWW-Authenticate")

	// 获取响应路径，处理302重定向的情况
	respPath := resp.Request.URL.Path

	// 检查是否是重定向后的请求（URL与后端地址不同）
	if resp.Request.URL.Host != h.backend.Host {
		// 如果是重定向请求，尝试从原始请求上下文中获取原始路径
		// 我们可以检查是否有自定义头或上下文字段
		// 这里我们使用一个简单的方法：如果路径不包含后端路径前缀，就显示为原始请求路径
		if !strings.HasPrefix(respPath, h.backend.Path) {
			// 这是一个重定向请求，显示为原始请求路径
			// 由于无法直接获取原始请求，我们使用日志标记
			respPath = "[REDIRECT] " + respPath
		}
	}

	h.logger.Debug("[RESPONSE] %s %d", respPath, resp.StatusCode)
	return nil
}

// errorHandler 错误处理器
func (h *ProxyHandler) errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	h.logger.Error("[ERROR] %s %s: %v", r.Method, r.URL.Path, err)

	// 如果后端认证失败，返回更友好的错误信息
	if strings.Contains(err.Error(), "401") {
		http.Error(w, "Backend authentication failed", http.StatusBadGateway)
		return
	}

	http.Error(w, "Gateway error", http.StatusBadGateway)
}

// createTransport 创建自定义传输层
func (h *ProxyHandler) createTransport() http.RoundTripper {
	// 创建基础传输层
	transport := &http.Transport{
		// 连接池配置
		MaxIdleConns:          h.maxIdleConns,
		MaxIdleConnsPerHost:   h.maxIdleConnsPerHost,
		IdleConnTimeout:       h.idleConnTimeout,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: h.timeout,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// 返回我们的加密传输层
	return &proxyTransport{
		handler: h,
		base:    transport,
	}
}

// getOrCreateEncryptor 获取或创建加密器
func (h *ProxyHandler) getOrCreateEncryptor(fileSize int64) (encryption.Encryptor, error) {
	// 创建缓存键
	cacheKey := fmt.Sprintf("%s:%d", h.algorithm, fileSize)

	// 尝试从缓存获取
	if enc, ok := h.encryptorCache.Load(cacheKey); ok {
		return enc.(encryption.Encryptor), nil
	}

	// 创建新的加密器
	enc, err := encryption.NewFlowEnc(h.password, h.algorithm, fileSize, func(msg string) {
		h.logger.Debug("[ENCRYPTION] %s", msg)
	})
	if err != nil {
		return nil, err
	}

	// 存入缓存
	h.encryptorCache.Store(cacheKey, enc)

	return enc, nil
}

// startEncryptorCleanup 启动加密器缓存清理协程
func (h *ProxyHandler) startEncryptorCleanup() {
	// 每小时清理一次缓存
	h.encryptorCleanupTicker = time.NewTicker(1 * time.Hour)

	go func() {
		for {
			select {
			case <-h.encryptorCleanupTicker.C:
				// 检查当前缓存大小
				var count int
				h.encryptorCache.Range(func(key, value interface{}) bool {
					count++
					return true
				})

				// 如果缓存项超过1000，清空缓存
				if count > 1000 {
					h.logger.Debug("加密器缓存已满 (%d项)，正在清空", count)
					h.encryptorCache = sync.Map{}
				}
			case <-h.stopCleanupChan:
				h.encryptorCleanupTicker.Stop()
				return
			}
		}
	}()
}

// Close 关闭资源
func (h *ProxyHandler) Close() {
	close(h.stopCleanupChan)
}
