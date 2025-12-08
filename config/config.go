package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"webdav-proxy/utils"

	"gopkg.in/yaml.v3"
)

// Config 配置结构
type Config struct {
	ListenAddr          string        `yaml:"listen_addr" env:"LISTEN_ADDR" default:":8080"`                      // 监听地址，格式为：:端口
	BackendURL          string        `yaml:"backend_url" env:"BACKEND_URL" default:""`                           // 后端WebDAV服务器URL
	Password            string        `yaml:"password" env:"PASSWORD" default:""`                                 // 加密密码
	Algorithm           string        `yaml:"algorithm" env:"ALGORITHM" default:"aesctr"`                         // 加密算法，可选值：mix, rc4, aesctr
	ChunkSize           int           `yaml:"chunk_size" env:"CHUNK_SIZE" default:"8192"`                         // 块大小（字节）
	Debug               bool          `yaml:"debug" env:"DEBUG" default:"false"`                                  // 是否启用调试模式（向后兼容，建议使用log_level）
	LogLevel            string        `yaml:"log_level" env:"LOG_LEVEL" default:"info"`                             // 日志级别：trace, debug, info, warn, error, fatal
	BackendUser         string        `yaml:"backend_user" env:"BACKEND_USER" default:""`                         // 后端WebDAV服务器用户名
	BackendPass         string        `yaml:"backend_pass" env:"BACKEND_PASS" default:""`                         // 后端WebDAV服务器密码
	EnableAuth          bool          `yaml:"enable_auth" env:"ENABLE_AUTH" default:"false"`                      // 是否启用代理端基本认证
	AuthUser            string        `yaml:"auth_user" env:"AUTH_USER" default:""`                               // 代理认证用户名
	AuthPass            string        `yaml:"auth_pass" env:"AUTH_PASS" default:""`                               // 代理认证密码
	Timeout             time.Duration `yaml:"timeout" env:"TIMEOUT" default:"300s"`                               // 请求超时时间
	MaxIdleConns        int           `yaml:"max_idle_conns" env:"MAX_IDLE_CONNS" default:"100"`                  // 最大空闲连接数
	MaxIdleConnsPerHost int           `yaml:"max_idle_conns_per_host" env:"MAX_IDLE_CONNS_PER_HOST" default:"10"` // 每个主机的最大空闲连接数
	IdleConnTimeout     time.Duration `yaml:"idle_conn_timeout" env:"IDLE_CONN_TIMEOUT" default:"90s"`            // 空闲连接超时时间
	DnsServers          []string      `yaml:"dns_servers" env:"DNS_SERVERS" default:"8.8.8.8:53,8.8.4.4:53"`      // 公共DNS服务器列表，格式为：IP:端口
	ConfigFile          string        `yaml:"-" env:"CONFIG_FILE" default:""`                                     // 配置文件路径
}

// Load 加载配置，支持从环境变量和配置文件
func Load() (*Config, error) {
	// 创建默认配置
	cfg := &Config{}
	if err := loadDefaults(cfg); err != nil {
		return nil, err
	}

	// 加载配置文件
	if cfgFile := os.Getenv("CONFIG_FILE"); cfgFile != "" {
		if err := loadFromFile(cfgFile, cfg); err != nil {
			return nil, err
		}
	}

	// 加载环境变量，覆盖配置文件
	if err := loadFromEnv(cfg); err != nil {
		return nil, err
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate 验证配置的有效性
func (c *Config) Validate() error {
	// 检查必要的配置项
	if c.BackendURL == "" {
		return fmt.Errorf("backend URL is required")
	}

	if c.Password == "" {
		return fmt.Errorf("encryption password is required")
	}

	// 验证算法
	validAlgorithms := []string{"mix", "rc4", "aesctr"}
	valid := false
	for _, alg := range validAlgorithms {
		if c.Algorithm == alg {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid algorithm: %s, supported: %v", c.Algorithm, validAlgorithms)
	}

	// 验证分块大小
	if c.ChunkSize <= 0 {
		return fmt.Errorf("chunk size must be positive")
	}

	// 验证认证配置
	if c.EnableAuth {
		if c.AuthUser == "" || c.AuthPass == "" {
			return fmt.Errorf("auth user and password are required when auth is enabled")
		}
	}

	return nil
}

// GetLogLevel 获取日志级别
func (c *Config) GetLogLevel() utils.LogLevel {
	// 如果Debug为true，则使用Debug级别（向后兼容）
	if c.Debug {
		return utils.LogLevelDebug
	}
	// 否则使用配置的日志级别
	return utils.ParseLogLevel(c.LogLevel)
}

// 加载默认值
func loadDefaults(cfg *Config) error {
	// 使用默认值初始化
	cfg.ListenAddr = ":8080"
	cfg.Algorithm = "aesctr"
	cfg.ChunkSize = 8192
	cfg.Debug = false
	cfg.LogLevel = "info"
	cfg.EnableAuth = false
	cfg.Timeout = 30 * time.Second
	cfg.MaxIdleConns = 100
	cfg.MaxIdleConnsPerHost = 10
	cfg.IdleConnTimeout = 90 * time.Second
	// 设置默认公共DNS服务器（Google DNS）
	cfg.DnsServers = []string{"8.8.8.8:53", "8.8.4.4:53"}
	return nil
}

// 从配置文件加载
func loadFromFile(filePath string, cfg *Config) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}

	return nil
}

// GenerateDefaultConfig 生成默认的YAML配置文件，包含中文解释
func GenerateDefaultConfig(filePath string) error {
	// 创建配置文件目录
	dir := filepath.Dir(filePath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create config directory: %w", err)
		}
	}

	// 使用硬编码的YAML内容，包含示例值和清晰的注释
	content := `# WebDAV Proxy 配置文件

## 服务端设置
# 后端WebDAV服务器URL (必填项，必须修改为实际的WebDAV服务器地址)
backend_url: "https://example.com/webdav/"
# 后端WebDAV用户名 (可选，如果后端服务器需要认证)
backend_user: "webdav-username"
# 后端WebDAV密码 (可选，如果后端服务器需要认证)
backend_pass: "webdav-password"


## 加密设置
# 加密算法 (可选，默认: aesctr，可选项: mix, rc4, aesctr)
algorithm: aesctr
# 加密密码 (可选，如果不设置则不进行加密)
password: "your-encryption-password"


## 代理端设置
# 监听地址 (默认: :8080)
listen_addr: ":8080"
# 启用代理端基本认证 (可选，默认: false)
enable_auth: false
# 代理认证用户名 (可选，当enable_auth为true时必填)
auth_user: "proxy-username"
# 代理认证密码 (可选，当enable_auth为true时必填)
auth_pass: "proxy-password"


## 日志设置
# 日志级别 (可选，默认: info，可选项: trace, debug, info, warn, error, fatal)
log_level: "info"
# 启用调试模式 (可选，默认: false，向后兼容，建议使用log_level)
debug: false


## 性能设置
# 块大小(字节) (可选，默认: 8192)
chunk_size: 8192
# 请求超时时间 (可选，默认: 30s)
timeout: 30s
# 最大空闲连接数 (可选，默认: 100)
max_idle_conns: 100
# 每个主机的最大空闲连接数 (可选，默认: 10)
max_idle_conns_per_host: 10
# 空闲连接超时时间 (可选，默认: 1m30s)
idle_conn_timeout: 1m30s

## DNS设置
# 公共DNS服务器列表 (可选，默认: Google DNS)
# 格式为：IP:端口,多个服务器用逗号分隔
dns_servers: ["8.8.8.8:53", "8.8.4.4:53"]
`

	// 写入文件
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

// 从环境变量加载
func loadFromEnv(cfg *Config) error {
	if addr := os.Getenv("LISTEN_ADDR"); addr != "" {
		cfg.ListenAddr = addr
	}

	if url := os.Getenv("BACKEND_URL"); url != "" {
		cfg.BackendURL = url
	}

	if password := os.Getenv("PASSWORD"); password != "" {
		cfg.Password = password
	}

	if alg := os.Getenv("ALGORITHM"); alg != "" {
		cfg.Algorithm = alg
	}

	if cs := os.Getenv("CHUNK_SIZE"); cs != "" {
		if size, err := strconv.Atoi(cs); err == nil {
			cfg.ChunkSize = size
		} else {
			return fmt.Errorf("invalid CHUNK_SIZE: %w", err)
		}
	}

	if debug := os.Getenv("DEBUG"); debug != "" {
		cfg.Debug = debug == "true" || debug == "1" || debug == "yes" || debug == "on"
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		cfg.LogLevel = logLevel
	}

	if user := os.Getenv("BACKEND_USER"); user != "" {
		cfg.BackendUser = user
	}

	if pass := os.Getenv("BACKEND_PASS"); pass != "" {
		cfg.BackendPass = pass
	}

	if enableAuth := os.Getenv("ENABLE_AUTH"); enableAuth != "" {
		cfg.EnableAuth = enableAuth == "true" || enableAuth == "1" || enableAuth == "yes" || enableAuth == "on"
	}

	if user := os.Getenv("AUTH_USER"); user != "" {
		cfg.AuthUser = user
	}

	if pass := os.Getenv("AUTH_PASS"); pass != "" {
		cfg.AuthPass = pass
	}

	if timeout := os.Getenv("TIMEOUT"); timeout != "" {
		if t, err := time.ParseDuration(timeout); err == nil {
			cfg.Timeout = t
		} else {
			return fmt.Errorf("invalid TIMEOUT: %w", err)
		}
	}

	if maxIdle := os.Getenv("MAX_IDLE_CONNS"); maxIdle != "" {
		if val, err := strconv.Atoi(maxIdle); err == nil {
			cfg.MaxIdleConns = val
		} else {
			return fmt.Errorf("invalid MAX_IDLE_CONNS: %w", err)
		}
	}

	if maxIdlePerHost := os.Getenv("MAX_IDLE_CONNS_PER_HOST"); maxIdlePerHost != "" {
		if val, err := strconv.Atoi(maxIdlePerHost); err == nil {
			cfg.MaxIdleConnsPerHost = val
		} else {
			return fmt.Errorf("invalid MAX_IDLE_CONNS_PER_HOST: %w", err)
		}
	}

	if idleTimeout := os.Getenv("IDLE_CONN_TIMEOUT"); idleTimeout != "" {
		if t, err := time.ParseDuration(idleTimeout); err == nil {
			cfg.IdleConnTimeout = t
		} else {
			return fmt.Errorf("invalid IDLE_CONN_TIMEOUT: %w", err)
		}
	}

	if dnsServers := os.Getenv("DNS_SERVERS"); dnsServers != "" {
		// 解析DNS服务器列表，格式为：IP:端口,IP:端口
		cfg.DnsServers = []string{}
		for _, server := range strings.Split(dnsServers, ",") {
			if server = strings.TrimSpace(server); server != "" {
				cfg.DnsServers = append(cfg.DnsServers, server)
			}
		}
		// 如果解析结果为空，使用默认值
		if len(cfg.DnsServers) == 0 {
			cfg.DnsServers = []string{"8.8.8.8:53", "8.8.4.4:53"}
		}
	}

	return nil
}
