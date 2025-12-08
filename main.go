package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"webdav-proxy/config"
	"webdav-proxy/proxy"
	"webdav-proxy/utils"
)

func main() {
	// 命令行参数
	var (
		listenAddr     = flag.String("listen", ":8080", "监听地址")
		backendURL     = flag.String("backend", "", "后端WebDAV服务器URL")
		password       = flag.String("password", "", "加密密码")
		algorithm      = flag.String("algorithm", "aesctr", "加密算法 (mix, rc4, aesctr)")
		chunkSizeStr   = flag.String("chunk-size", "8192", "块大小(字节)")
		debug          = flag.Bool("debug", false, "启用调试模式")
		backendUser    = flag.String("backend-user", "", "后端WebDAV用户名")
		backendPass    = flag.String("backend-pass", "", "后端WebDAV密码")
		enableAuth     = flag.Bool("auth", false, "启用代理端基本认证")
		authUser       = flag.String("auth-user", "", "代理认证用户名")
		authPass       = flag.String("auth-pass", "", "代理认证密码")
		configFile     = flag.String("config", "config.yaml", "配置文件路径 (YAML格式)")
	)

	flag.Parse()

	// 如果指定了配置文件，设置环境变量
	if *configFile != "" {
		os.Setenv("CONFIG_FILE", *configFile)
		// 检查配置文件是否存在，如果不存在则生成默认配置
		if _, err := os.Stat(*configFile); os.IsNotExist(err) {
			log.Printf("配置文件 %s 不存在，正在生成默认配置...", *configFile)
			if err := config.GenerateDefaultConfig(*configFile); err != nil {
				log.Printf("生成默认配置文件失败: %v", err)
			}
		}
	}
	
	// 加载配置
	cfg, err := config.Load()
	// 如果配置加载失败，创建一个新的配置对象
	if err != nil {
		if !strings.Contains(err.Error(), "backend URL is required") && !strings.Contains(err.Error(), "encryption password is required") {
			log.Fatal(err)
		}
		// 创建一个新的配置对象
		cfg = &config.Config{}
		// 设置默认值
		cfg.ListenAddr = ":8080"
		cfg.Algorithm = "aesctr"
		cfg.ChunkSize = 8192
		cfg.Debug = false
		cfg.Timeout = 30 * time.Second
		cfg.MaxIdleConns = 100
		cfg.MaxIdleConnsPerHost = 10
		cfg.IdleConnTimeout = 90 * time.Second
	}

	// 命令行参数覆盖
	if *listenAddr != ":8080" {
		cfg.ListenAddr = *listenAddr
	}
	if *backendURL != "" {
		cfg.BackendURL = *backendURL
	}
	if *password != "" {
		cfg.Password = *password
	}
	if *algorithm != "aesctr" {
		cfg.Algorithm = *algorithm
	}
	if *chunkSizeStr != "8192" {
		if size, err := strconv.Atoi(*chunkSizeStr); err == nil {
			cfg.ChunkSize = size
		} else {
			log.Printf("无效的块大小，使用默认值: %d", cfg.ChunkSize)
		}
	}
	if *debug {
		cfg.Debug = *debug
	}
	if *backendUser != "" {
		cfg.BackendUser = *backendUser
	}
	if *backendPass != "" {
		cfg.BackendPass = *backendPass
	}
	if *enableAuth {
		cfg.EnableAuth = *enableAuth
	}
	if *authUser != "" {
		cfg.AuthUser = *authUser
	}
	if *authPass != "" {
		cfg.AuthPass = *authPass
	}
	if *configFile != "" {
		cfg.ConfigFile = *configFile
	}

	// 验证配置，确保所有必要的配置项都已设置
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}

	// 创建日志器
	logLevel := utils.GetLogLevel(cfg.LogLevel, cfg.Debug)
	logger := utils.NewLogger(logLevel)
	logger.Info("正在启动WebDAV加密代理...")

	// 解析后端URL
	backend, err := url.Parse(cfg.BackendURL)
	if err != nil {
		logger.Error("无效的后端URL: %v", err)
		os.Exit(1)
	}

	// 处理监听地址，如果只输入了端口号，自动添加冒号前缀
	if cfg.ListenAddr != "" && cfg.ListenAddr[0] != ':' && !strings.Contains(cfg.ListenAddr, ":") {
		cfg.ListenAddr = ":" + cfg.ListenAddr
	}

	// 创建后端认证配置
	backendAuthConfig := &proxy.BackendAuthConfig{
		Username: cfg.BackendUser,
		Password: cfg.BackendPass,
	}

	// 创建代理端认证配置
	var proxyAuthConfig *proxy.ProxyAuthConfig
	if cfg.EnableAuth {
		proxyAuthConfig = &proxy.ProxyAuthConfig{
			Enabled:  true,
			Username: cfg.AuthUser,
			Password: cfg.AuthPass,
		}
	}

	// 创建代理处理器
proxyHandler, err := proxy.NewProxyHandler(
	backend,
	cfg.Password,
	cfg.Algorithm,
	cfg.ChunkSize,
	backendAuthConfig,
	proxyAuthConfig,
	logger,
	cfg.Timeout,
	cfg.MaxIdleConns,
	cfg.MaxIdleConnsPerHost,
	cfg.IdleConnTimeout,
	cfg.DnsServers,
)
	if err != nil {
		logger.Error("创建代理处理器失败: %v", err)
		os.Exit(1)
	}

	// 应用代理认证中间件
	var handler http.Handler = proxyHandler
	if cfg.EnableAuth {
		handler = proxy.NewProxyAuthMiddleware(handler, proxyAuthConfig)
	}

	// 设置优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动服务器
	server := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      handler,
		ReadTimeout:  300 * time.Second,
		WriteTimeout: 300 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("启动WebDAV加密代理")
		logger.Info("监听地址: %s", cfg.ListenAddr)
		logger.Info("后端服务器: %s", backend.String())
		logger.Info("后端用户名: %s", cfg.BackendUser)
		logger.Info("加密算法: %s", cfg.Algorithm)
		logger.Info("块大小: %d 字节", cfg.ChunkSize)
		if cfg.EnableAuth {
			logger.Info("代理认证已启用，用户: %s", cfg.AuthUser)
		}

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("服务器启动失败: %v", err)
			os.Exit(1)
		}
	}()

	// 等待关闭信号
	sig := <-sigChan
	logger.Info("接收到信号: %v，正在关闭服务器...", sig)

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("服务器关闭失败: %v", err)
	}

	logger.Info("服务器已关闭")
}
