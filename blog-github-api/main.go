package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// 分页返回结构
type PagedPostsResponse struct {
	Items    []PostMeta `json:"items"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
	Total    int        `json:"total"`
	HasMore  bool       `json:"has_more"`
}

// 搜索返回结构
type SearchResponse struct {
	Items []PostMeta `json:"items"`
}

func main() {
	// 1. 加载配置
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("load config error: %v", err)
	}

	// 2. 初始化缓存
	cache := NewBlogCache(cfg.CacheFile)
	if err := cache.LoadFromDisk(); err != nil {
		log.Printf("load cache error: %v", err)
	}

	// 3. 初始化 GitHub 客户端
	ghClient := NewGitHubClient(cfg)

	// 4. 启动时先同步一次（非致命）
	func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := SyncOnce(ctx, ghClient, cache); err != nil {
			log.Printf("initial sync error: %v", err)
		} else {
			log.Printf("initial sync done, posts: %d", len(cache.ListPosts()))
		}
	}()

	// 5. 定时同步任务
	go func() {
		ticker := time.NewTicker(cfg.RefreshEvery)
		defer ticker.Stop()

		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := SyncOnce(ctx, ghClient, cache); err != nil {
				log.Printf("scheduled sync error: %v", err)
			} else {
				log.Printf("scheduled sync ok, posts: %d", len(cache.ListPosts()))
			}
			cancel()
		}
	}()

	// 6. HTTP 路由
	mux := http.NewServeMux()

	// 健康检查（无需鉴权）
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		resp := map[string]any{
			"status":      "ok",
			"github_repo": cfg.GitHubOwner + "/" + cfg.GitHubRepo,
			"updated_at":  cache.UpdatedAt(),
		}
		writeJSON(w, http.StatusOK, resp)
	})

	// 带分页的列表：GET /api/posts?page=1&page_size=10
	mux.Handle("/api/posts", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		all := cache.ListPosts()
		total := len(all)

		q := r.URL.Query()

		// 页码
		page := 1
		if v := q.Get("page"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				page = n
			}
		}

		// 每页数量：优先 query 参数，其次 cfg.PageSize
		pageSize := cfg.PageSize
		if pageSize <= 0 {
			pageSize = 10
		}
		if v := q.Get("page_size"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
				pageSize = n
			}
		}

		start := (page - 1) * pageSize
		if start > total {
			start = total
		}
		end := start + pageSize
		if end > total {
			end = total
		}

		items := all[start:end]
		hasMore := end < total

		resp := PagedPostsResponse{
			Items:    items,
			Page:     page,
			PageSize: pageSize,
			Total:    total,
			HasMore:  hasMore,
		}

		writeJSON(w, http.StatusOK, resp)
	}))

	// 详情：GET /api/posts/{slug}
	mux.Handle("/api/posts/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		slug := strings.TrimPrefix(r.URL.Path, "/api/posts/")
		if slug == "" {
			http.Error(w, "slug required", http.StatusBadRequest)
			return
		}

		post, ok := cache.GetPost(slug)
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		// post 一般是 CachedPost，里面包含 Meta + Content
		writeJSON(w, http.StatusOK, post)
	}))

	// 搜索：GET /api/search?q=xxx
	mux.Handle("/api/search", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if q == "" {
			writeJSON(w, http.StatusOK, SearchResponse{Items: []PostMeta{}})
			return
		}
		qLower := strings.ToLower(q)

		all := cache.ListPosts()

		var strong []PostMeta // 标题/路径命中优先
		var weak []PostMeta   // 摘要命中其后

		for _, pm := range all {
			title := strings.ToLower(pm.Title)
			summary := strings.ToLower(pm.Summary)
			path := strings.ToLower(pm.Path)

			inTitle := strings.Contains(title, qLower)
			inPath := strings.Contains(path, qLower)
			inSummary := strings.Contains(summary, qLower)

			if inTitle || inPath {
				strong = append(strong, pm)
			} else if inSummary {
				weak = append(weak, pm)
			}
		}

		results := append(strong, weak...)
		if len(results) > 50 {
			results = results[:50]
		}

		resp := SearchResponse{Items: results}
		writeJSON(w, http.StatusOK, resp)
	}))

	// 包一层日志 + CORS
	handler := corsMiddleware(cfg.AllowedOrigins, logMiddleware(mux))

	// 7. 启动 HTTP 服务
	srv := &http.Server{
		Addr:         cfg.BindAddr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if cfg.TLSCertFile != "" {
		log.Printf("listening with TLS on %s", cfg.BindAddr)
		if err := srv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile); err != nil {
			log.Fatalf("server error: %v", err)
		}
	} else {
		log.Printf("listening on %s (HTTP, no TLS)", cfg.BindAddr)
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}
}

// 辅助：统一写 JSON
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// 日志中间件
func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

// CORS：为了本地开发简单，直接允许所有来源
// 如果你以后上生产，可以改成只允许特定域名。
func corsMiddleware(allowedOrigins []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && originAllowed(origin, allowedOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		} else if origin != "" && len(allowedOrigins) > 0 {
			http.Error(w, "origin not allowed", http.StatusForbidden)
			return
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func originAllowed(origin string, allowed []string) bool {
	if len(allowed) == 0 {
		return false
	}
	for _, o := range allowed {
		if o == origin {
			return true
		}
	}
	return false
}
