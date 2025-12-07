package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadFromEnv(t *testing.T) {
	// 设置测试环境变量
	err := os.Setenv("LISTEN_ADDR", "127.0.0.1:8080")
	if err != nil {
		t.Fatalf("设置环境变量失败: %v", err)
	}
	err = os.Setenv("BACKEND_URL", "http://example.com/webdav/")
	if err != nil {
		t.Fatalf("设置环境变量失败: %v", err)
	}
	err = os.Setenv("PASSWORD", "testpassword")
	if err != nil {
		t.Fatalf("设置环境变量失败: %v", err)
	}
	err = os.Setenv("ALGORITHM", "aesctr")
	if err != nil {
		t.Fatalf("设置环境变量失败: %v", err)
	}
	err = os.Setenv("CHUNK_SIZE", "4096")
	if err != nil {
		t.Fatalf("设置环境变量失败: %v", err)
	}
	err = os.Setenv("BACKEND_USER", "backenduser")
	if err != nil {
		t.Fatalf("设置环境变量失败: %v", err)
	}
	err = os.Setenv("BACKEND_PASS", "backendpass")
	if err != nil {
		t.Fatalf("设置环境变量失败: %v", err)
	}
	err = os.Setenv("AUTH_USER", "proxyuser")
	if err != nil {
		t.Fatalf("设置环境变量失败: %v", err)
	}
	err = os.Setenv("AUTH_PASS", "proxypass")
	if err != nil {
		t.Fatalf("设置环境变量失败: %v", err)
	}
	err = os.Setenv("TIMEOUT", "60s")
	if err != nil {
		t.Fatalf("设置环境变量失败: %v", err)
	}
	err = os.Setenv("MAX_IDLE_CONNS", "200")
	if err != nil {
		t.Fatalf("设置环境变量失败: %v", err)
	}
	err = os.Setenv("MAX_IDLE_CONNS_PER_HOST", "20")
	if err != nil {
		t.Fatalf("设置环境变量失败: %v", err)
	}
	err = os.Setenv("IDLE_CONN_TIMEOUT", "180s")
	if err != nil {
		t.Fatalf("设置环境变量失败: %v", err)
	}

	// 加载配置
	cfg, err := Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证配置
	if cfg.ListenAddr != "127.0.0.1:8080" {
		t.Errorf("期望LISTEN_ADDR为127.0.0.1:8080，实际为%v", cfg.ListenAddr)
	}
	if cfg.BackendURL != "http://example.com/webdav/" {
		t.Errorf("期望BACKEND_URL为http://example.com/webdav/，实际为%v", cfg.BackendURL)
	}
	if cfg.Password != "testpassword" {
		t.Errorf("期望PASSWORD为testpassword，实际为%v", cfg.Password)
	}
	if cfg.Algorithm != "aesctr" {
		t.Errorf("期望ALGORITHM为aesctr，实际为%v", cfg.Algorithm)
	}
	if cfg.ChunkSize != 4096 {
		t.Errorf("期望CHUNK_SIZE为4096，实际为%v", cfg.ChunkSize)
	}
	if cfg.BackendUser != "backenduser" {
		t.Errorf("期望BACKEND_USER为backenduser，实际为%v", cfg.BackendUser)
	}
	if cfg.BackendPass != "backendpass" {
		t.Errorf("期望BACKEND_PASS为backendpass，实际为%v", cfg.BackendPass)
	}
	if cfg.AuthUser != "proxyuser" {
		t.Errorf("期望AUTH_USER为proxyuser，实际为%v", cfg.AuthUser)
	}
	if cfg.AuthPass != "proxypass" {
		t.Errorf("期望AUTH_PASS为proxypass，实际为%v", cfg.AuthPass)
	}
	if cfg.Timeout != 60*time.Second {
		t.Errorf("期望TIMEOUT为60s，实际为%v", cfg.Timeout)
	}
	if cfg.MaxIdleConns != 200 {
		t.Errorf("期望MAX_IDLE_CONNS为200，实际为%v", cfg.MaxIdleConns)
	}
	if cfg.MaxIdleConnsPerHost != 20 {
		t.Errorf("期望MAX_IDLE_CONNS_PER_HOST为20，实际为%v", cfg.MaxIdleConnsPerHost)
	}
	if cfg.IdleConnTimeout != 180*time.Second {
		t.Errorf("期望IDLE_CONN_TIMEOUT为180s，实际为%v", cfg.IdleConnTimeout)
	}

	// 清理环境变量
	err = os.Unsetenv("LISTEN_ADDR")
	if err != nil {
		t.Fatalf("清理环境变量失败: %v", err)
	}
	err = os.Unsetenv("BACKEND_URL")
	if err != nil {
		t.Fatalf("清理环境变量失败: %v", err)
	}
	err = os.Unsetenv("PASSWORD")
	if err != nil {
		t.Fatalf("清理环境变量失败: %v", err)
	}
	err = os.Unsetenv("ALGORITHM")
	if err != nil {
		t.Fatalf("清理环境变量失败: %v", err)
	}
	err = os.Unsetenv("CHUNK_SIZE")
	if err != nil {
		t.Fatalf("清理环境变量失败: %v", err)
	}
	err = os.Unsetenv("BACKEND_USER")
	if err != nil {
		t.Fatalf("清理环境变量失败: %v", err)
	}
	err = os.Unsetenv("BACKEND_PASS")
	if err != nil {
		t.Fatalf("清理环境变量失败: %v", err)
	}
	err = os.Unsetenv("AUTH_USER")
	if err != nil {
		t.Fatalf("清理环境变量失败: %v", err)
	}
	err = os.Unsetenv("AUTH_PASS")
	if err != nil {
		t.Fatalf("清理环境变量失败: %v", err)
	}
	err = os.Unsetenv("TIMEOUT")
	if err != nil {
		t.Fatalf("清理环境变量失败: %v", err)
	}
	err = os.Unsetenv("MAX_IDLE_CONNS")
	if err != nil {
		t.Fatalf("清理环境变量失败: %v", err)
	}
	err = os.Unsetenv("MAX_IDLE_CONNS_PER_HOST")
	if err != nil {
		t.Fatalf("清理环境变量失败: %v", err)
	}
	err = os.Unsetenv("IDLE_CONN_TIMEOUT")
	if err != nil {
		t.Fatalf("清理环境变量失败: %v", err)
	}
}

func TestValidate(t *testing.T) {
	// 测试有效配置
	validCfg := &Config{
		BackendURL:  "http://example.com/webdav/",
		Password:    "testpassword",
		Algorithm:   "aesctr",
		ChunkSize:   4096,
		BackendUser: "user",
		BackendPass: "pass",
		EnableAuth:  true,
		AuthUser:    "proxyuser",
		AuthPass:    "proxypass",
	}

	err := validCfg.Validate()
	if err != nil {
		t.Errorf("有效配置验证失败: %v", err)
	}

	// 测试无效配置
	invalidCfg := &Config{
		BackendURL: "", // 缺少后端URL
		Password:   "testpassword",
		Algorithm:  "aesctr",
	}

	err = invalidCfg.Validate()
	if err == nil {
		t.Error("期望无效配置验证失败，但验证通过")
	}

	// 测试无效算法
	invalidAlgCfg := &Config{
		BackendURL: "http://example.com/webdav/",
		Password:   "testpassword",
		Algorithm:  "invalid_algorithm",
	}

	err = invalidAlgCfg.Validate()
	if err == nil {
		t.Error("期望无效算法验证失败，但验证通过")
	}

	// 测试无效分块大小
	invalidChunkCfg := &Config{
		BackendURL: "http://example.com/webdav/",
		Password:   "testpassword",
		Algorithm:  "aesctr",
		ChunkSize:  0, // 无效分块大小
	}

	err = invalidChunkCfg.Validate()
	if err == nil {
		t.Error("期望无效分块大小验证失败，但验证通过")
	}
}
