package main

import "time"

// 对外返回的文章列表项
type PostMeta struct {
	Slug      string    `json:"slug"`       // 由文件名去掉 .md 得来
	Title     string    `json:"title"`      // 从 Markdown 第一行标题解析
	Summary   string    `json:"summary"`    // 自动截取的简介
	Path      string    `json:"path"`       // 仓库内路径：posts/xxx.md
	SHA       string    `json:"sha"`        // GitHub 文件 sha，用来判断是否有变更
	UpdatedAt time.Time `json:"updated_at"` // 本地缓存更新时间
}

// 文章详情（带正文）
type PostDetail struct {
	PostMeta
	Content string `json:"content"` // Markdown 原文
}

// 缓存文件格式
type cacheFile struct {
	UpdatedAt time.Time    `json:"updated_at"`
	Posts     []CachedPost `json:"posts"`
}

// 内部持久化结构
type CachedPost struct {
	Meta    PostMeta `json:"meta"`
	Content string   `json:"content"` // Markdown 原文
}
