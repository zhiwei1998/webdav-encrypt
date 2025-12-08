package main

import (
	"context"
	"flag"
	"fmt"
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
	// 设置flag的用法
	flag.Usage = func() {
		fmt.Println("WebDAV加密代理使用方法:")
		fmt.Println()
		fmt.Println("  webdav-encrypt [OPTIONS]")
		fmt.Println()
		fmt.Println("配置选项:")
		// 自定义参数列表，将缩写和长参数合并显示，移除重复的默认值描述
		fmt.Printf("  --listen             监听地址，默认: :8080\n")
		fmt.Printf("  -p, --password       加密密码 (必填)\n")
		fmt.Printf("  -t, --algorithm      加密算法，可选值: mix, rc4, aesctr (默认: aesctr)\n")
		fmt.Printf("  --backend            后端WebDAV服务器URL (必填)\n")
		fmt.Printf("  --backend-user       后端WebDAV用户名\n")
		fmt.Printf("  --backend-pass       后端WebDAV密码\n")
		fmt.Printf("  --auth-user          代理认证用户名\n")
		fmt.Printf("  --auth-pass          代理认证密码\n")
		fmt.Printf("  -c, --config         配置文件路径 (YAML格式)\n")
		fmt.Printf("  --chunk-size         块大小(字节)，默认: 8192\n")
		fmt.Printf("  --debug              启用调试模式，默认: false\n")
		fmt.Printf("  -h, --help           显示帮助信息\n")

		fmt.Println()
		fmt.Println("注意事项:")
		fmt.Println("  - 只有明确指定-c或--config参数时，才会使用配置文件")
		fmt.Println("  - 同时提供命令行参数和配置文件时，命令行参数优先级更高")
		fmt.Println("  - 必须提供--backend和--password参数，或在配置文件中配置")
		fmt.Println("  - 如果传入了--backend-user和--backend-pass，将自动启用代理端基本认证")
		fmt.Println("  - 如果传入了--auth-user和--auth-pass，将启用代理端基本认证并使用这些凭据")
		fmt.Println()
	}

	// 命令行参数
	var (
		listenAddr   = flag.String("listen", ":8080", "监听地址，默认: :8080")
		backendURL   = flag.String("backend", "", "后端WebDAV服务器URL (必填)")
		password     = flag.String("password", "", "加密密码 (必填) (简写: -p)")
		algorithm    = flag.String("algorithm", "aesctr", "加密算法，可选值: mix, rc4, aesctr (默认: aesctr) (简写: -t)")
		chunkSizeStr = flag.String("chunk-size", "8192", "块大小(字节)，默认: 8192")
		debug        = flag.Bool("debug", false, "启用调试模式，默认: false")
		backendUser  = flag.String("backend-user", "", "后端WebDAV用户名")
		backendPass  = flag.String("backend-pass", "", "后端WebDAV密码")
		authUser     = flag.String("auth-user", "", "代理认证用户名")
		authPass     = flag.String("auth-pass", "", "代理认证密码")
		configFile   = flag.String("config", "", "配置文件路径 (YAML格式) (简写: -c)")
	)
	// 只添加缩写的变量映射，不显示在帮助信息中
	flag.Bool("h", false, "显示帮助信息 (简写: -h)")
	flag.StringVar(configFile, "c", *configFile, "")
	flag.StringVar(algorithm, "t", *algorithm, "")
	flag.StringVar(password, "p", *password, "")

	// 保存默认值，用于判断参数是否被用户显式指定
	defaultListenAddr := ":8080"
	defaultChunkSize := "8192"
	defaultDebug := false
	// 解析命令行参数
	flag.Parse()

	// 检查是否请求了帮助信息
	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" {
			flag.Usage()
			os.Exit(0)
		}
	}

	// 检查哪些参数被用户显式指定了
	listenAddrSpecified := *listenAddr != defaultListenAddr
	chunkSizeSpecified := *chunkSizeStr != defaultChunkSize
	debugSpecified := *debug != defaultDebug

	// 如果指定了配置文件，设置环境变量
	if *configFile != "" {
		os.Setenv("CONFIG_FILE", *configFile)
		// 检查配置文件是否存在，如果不存在则生成默认配置
		if _, err := os.Stat(*configFile); os.IsNotExist(err) {
			log.Printf("配置文件 %s 不存在，正在生成默认配置...", *configFile)
			if err := config.GenerateDefaultConfig(*configFile); err != nil {
				log.Printf("生成默认配置文件失败: %v", err)
			} else {
				// 检查是否只有配置文件参数被指定，没有其他必填参数
				// 如果是，生成配置后直接退出
				if *backendURL == "" && *password == "" && *listenAddr == ":8080" && *algorithm == "aesctr" && *chunkSizeStr == "8192" && *debug == false && *backendUser == "" && *backendPass == "" && *authUser == "" && *authPass == "" {
					log.Printf("默认配置文件已生成，请根据需要修改配置后重新启动程序")
					os.Exit(0)
				}
			}
		}
	} else {
		// 如果没有指定配置文件，取消环境变量，确保不加载默认配置文件
		os.Unsetenv("CONFIG_FILE")
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
		// 清空默认的auth配置
		cfg.AuthUser = ""
		cfg.AuthPass = ""
		cfg.EnableAuth = false
	} else {
		// 如果配置加载成功，保留配置文件中的auth设置
		// 但如果命令行参数提供了auth信息，则会在后面的逻辑中优先使用命令行参数
		// 这里只需要确保EnableAuth的默认值为false，避免配置文件中的enable_auth影响认证逻辑
		cfg.EnableAuth = false
	}

	// 命令行参数覆盖 - 只有当用户显式指定了参数时才覆盖配置文件的值
	if listenAddrSpecified {
		cfg.ListenAddr = *listenAddr
	}
	if *backendURL != "" {
		cfg.BackendURL = *backendURL
	}
	if *password != "" {
		cfg.Password = *password
	}
	// 加密算法参数总是覆盖配置文件，因为有缩写 -t
	cfg.Algorithm = *algorithm
	if chunkSizeSpecified {
		if size, err := strconv.Atoi(*chunkSizeStr); err == nil {
			cfg.ChunkSize = size
		} else {
			log.Printf("无效的块大小，使用默认值: %d", cfg.ChunkSize)
		}
	}
	if debugSpecified {
		cfg.Debug = *debug
	}
	if *backendUser != "" {
		cfg.BackendUser = *backendUser
	}
	if *backendPass != "" {
		cfg.BackendPass = *backendPass
	}
	if *authUser != "" {
		cfg.AuthUser = *authUser
	}
	if *authPass != "" {
		cfg.AuthPass = *authPass
	}

	// 处理auth逻辑：
	// 1. 如果提供了auth-user和auth-pass（命令行或配置文件），则启用auth并使用这些凭据
	// 2. 如果没有提供auth-user和auth-pass，但提供了backend-user和backend-pass（命令行或配置文件），则同步启用auth并使用backend的凭据
	// 3. 其他情况，禁用auth
	if (*authUser != "" && *authPass != "") || (cfg.AuthUser != "" && cfg.AuthPass != "") {
		// 有明确的auth参数（命令行或配置文件），启用auth
		cfg.EnableAuth = true
		// 如果命令行参数提供了auth信息，则优先使用命令行参数
		if *authUser != "" && *authPass != "" {
			cfg.AuthUser = *authUser
			cfg.AuthPass = *authPass
		}
	} else if (*backendUser != "" && *backendPass != "") || (cfg.BackendUser != "" && cfg.BackendPass != "") {
		// 没有明确的auth参数，但有backend认证参数（命令行或配置文件），同步启用auth并使用backend的凭据
		cfg.EnableAuth = true
		cfg.AuthUser = cfg.BackendUser
		cfg.AuthPass = cfg.BackendPass
	} else {
		// 没有auth相关参数，禁用auth
		cfg.EnableAuth = false
		cfg.AuthUser = ""
		cfg.AuthPass = ""
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
