package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type BlogCache struct {
	mu        sync.RWMutex
	updatedAt time.Time
	posts     []CachedPost
	bySlug    map[string]CachedPost
	shaByPath map[string]string
	cacheFile string
}

// 新建空缓存
func NewBlogCache(cacheFile string) *BlogCache {
	return &BlogCache{
		bySlug:    make(map[string]CachedPost),
		shaByPath: make(map[string]string),
		cacheFile: cacheFile,
	}
}

// 从磁盘加载缓存（如果不存在就保持为空）
func (c *BlogCache) LoadFromDisk() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	f, err := os.Open(c.cacheFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	var cf cacheFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return err
	}

	c.updatedAt = cf.UpdatedAt
	c.posts = cf.Posts
	c.bySlug = make(map[string]CachedPost, len(cf.Posts))
	c.shaByPath = make(map[string]string, len(cf.Posts))

	for _, p := range cf.Posts {
		c.bySlug[p.Meta.Slug] = p
		c.shaByPath[p.Meta.Path] = p.Meta.SHA
	}

	return nil
}

// 保存到磁盘（先写临时文件再原子替换，避免损坏）
func (c *BlogCache) SaveToDisk() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cf := cacheFile{
		UpdatedAt: c.updatedAt,
		Posts:     c.posts,
	}

	data, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(c.cacheFile), 0o755); err != nil {
		return err
	}

	tmp := c.cacheFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}

	return os.Rename(tmp, c.cacheFile)
}

// 替换全部数据（同步任务完成后调用）
func (c *BlogCache) ReplaceAll(posts []CachedPost) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.updatedAt = time.Now().UTC()
	c.posts = posts
	c.bySlug = make(map[string]CachedPost, len(posts))
	c.shaByPath = make(map[string]string, len(posts))

	for _, p := range posts {
		c.bySlug[p.Meta.Slug] = p
		c.shaByPath[p.Meta.Path] = p.Meta.SHA
	}
}

// 获取当前 sha 映射（用来判断有无变更）
func (c *BlogCache) SnapshotSHA() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make(map[string]string, len(c.shaByPath))
	for k, v := range c.shaByPath {
		out[k] = v
	}
	return out
}

// 列表：只返回 Meta，避免把内容都传出去
func (c *BlogCache) ListPosts() []PostMeta {
	c.mu.RLock()
	defer c.mu.RUnlock()

	res := make([]PostMeta, 0, len(c.posts))
	for _, p := range c.posts {
		res = append(res, p.Meta)
	}
	return res
}

// 详情：根据 slug 获取
func (c *BlogCache) GetPost(slug string) (PostDetail, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	p, ok := c.bySlug[slug]
	if !ok {
		return PostDetail{}, false
	}
	return PostDetail{
		PostMeta: p.Meta,
		Content:  p.Content,
	}, true
}

// 获取最后更新时间
func (c *BlogCache) UpdatedAt() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.updatedAt
}
