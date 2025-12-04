package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	GitHubOwner    string        // 你的 GitHub 用户名 / 组织
	GitHubRepo     string        // 固定：my-blog-content
	GitHubBranch   string        // 分支，默认 main
	GitHubToken    string        // 私有仓库必须，放环境变量
	PostsDir       string        // posts 目录
	CacheFile      string        // 本地缓存文件路径
	RefreshEvery   time.Duration // 刷新间隔
	BindAddr       string        // HTTP 服务监听地址
	PageSize       int
	AllowedOrigins []string // 允许的 CORS 来源列表
	TLSCertFile    string   // 可选：TLS 证书
	TLSKeyFile     string   // 可选：TLS 私钥
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func LoadConfig() (Config, error) {
	cfg := Config{
		GitHubOwner:  os.Getenv("GITHUB_OWNER"),
		GitHubRepo:   getenvDefault("GITHUB_REPO", "my-blog-content"),
		GitHubBranch: getenvDefault("GITHUB_BRANCH", "main"),
		GitHubToken:  os.Getenv("GITHUB_TOKEN"),
		PostsDir:     os.Getenv("POSTS_DIR"),
		CacheFile:    getenvDefault("CACHE_FILE", "data/cache.json"),
		BindAddr:     getenvDefault("BIND_ADDR", "127.0.0.1:8080"),
	}
	pageSize := 3
	if v := os.Getenv("PAGE_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			pageSize = n
		}
	}
	cfg.PageSize = pageSize

	cfg.AllowedOrigins = parseOrigins(os.Getenv("ALLOWED_ORIGINS"))

	cfg.TLSCertFile = os.Getenv("TLS_CERT_FILE")
	cfg.TLSKeyFile = os.Getenv("TLS_KEY_FILE")
	if (cfg.TLSCertFile == "") != (cfg.TLSKeyFile == "") {
		return Config{}, fmt.Errorf("both TLS_CERT_FILE and TLS_KEY_FILE must be set together")
	}

	// 刷新间隔（分钟），默认 120 分钟 = 2 小时
	intervalStr := getenvDefault("REFRESH_INTERVAL_MINUTES", "120")
	mins, err := strconv.Atoi(intervalStr)
	if err != nil || mins <= 0 {
		return Config{}, fmt.Errorf("invalid REFRESH_INTERVAL_MINUTES: %q", intervalStr)
	}
	cfg.RefreshEvery = time.Duration(mins) * time.Minute

	if cfg.GitHubOwner == "" {
		return Config{}, fmt.Errorf("GITHUB_OWNER is required")
	}
	if cfg.GitHubToken == "" {
		return Config{}, fmt.Errorf("GITHUB_TOKEN is required for private repo my-blog-content")
	}

	return cfg, nil
}

func parseOrigins(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var origins []string
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			origins = append(origins, t)
		}
	}
	return origins
}
