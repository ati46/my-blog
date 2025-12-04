package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"
)

// Git Tree API 返回
type githubTreeResponse struct {
	Tree []githubTreeItem `json:"tree"`
}

type githubTreeItem struct {
	Path string `json:"path"`
	Type string `json:"type"` // "blob" or "tree"
	SHA  string `json:"sha"`
}

// contents API 文件详情
type githubFileContent struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	SHA     string `json:"sha"`
	Content string `json:"content"` // base64
}

// GitHubClient 封装 GitHub API 调用
type GitHubClient struct {
	cfg    Config
	client *http.Client
}

func NewGitHubClient(cfg Config) *GitHubClient {
	return &GitHubClient{
		cfg: cfg,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (g *GitHubClient) apiRequest(ctx context.Context, method, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	if g.cfg.GitHubToken != "" {
		req.Header.Set("Authorization", "Bearer "+g.cfg.GitHubToken)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	return g.client.Do(req)
}

// listMarkdownFiles：使用 Git Trees API 递归列出所有 .md 文件（含子目录）
func (g *GitHubClient) listMarkdownFiles(ctx context.Context) ([]githubTreeItem, error) {
	// Git Trees：/git/trees/{branch}?recursive=1
	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1",
		g.cfg.GitHubOwner,
		g.cfg.GitHubRepo,
		g.cfg.GitHubBranch,
	)

	resp, err := g.apiRequest(ctx, http.MethodGet, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub trees status %d: %s", resp.StatusCode, string(body))
	}

	var tr githubTreeResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, err
	}

	var files []githubTreeItem
	baseDir := strings.TrimSpace(g.cfg.PostsDir) // 可以为空表示根目录
	for _, t := range tr.Tree {
		if t.Type != "blob" {
			continue
		}
		if !strings.HasSuffix(t.Path, ".md") {
			continue
		}

		if baseDir == "" {
			// 根目录 + 任意子目录都算
			files = append(files, t)
		} else {
			// 只要在 PostsDir 子树下
			if t.Path == baseDir || strings.HasPrefix(t.Path, baseDir+"/") {
				files = append(files, t)
			}
		}
	}
	return files, nil
}

// fetchFileContent：根据 path 拉取 markdown 内容
func (g *GitHubClient) fetchFileContent(ctx context.Context, filePath string) (githubFileContent, []byte, error) {
	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/contents/%s?ref=%s",
		g.cfg.GitHubOwner,
		g.cfg.GitHubRepo,
		filePath,
		g.cfg.GitHubBranch,
	)

	resp, err := g.apiRequest(ctx, http.MethodGet, url)
	if err != nil {
		return githubFileContent{}, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return githubFileContent{}, nil, fmt.Errorf("GitHub get file %s status %d: %s", filePath, resp.StatusCode, string(body))
	}

	var fc githubFileContent
	if err := json.NewDecoder(resp.Body).Decode(&fc); err != nil {
		return githubFileContent{}, nil, err
	}

	raw := strings.ReplaceAll(fc.Content, "\n", "")
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return githubFileContent{}, nil, fmt.Errorf("decode base64 content: %w", err)
	}

	return fc, decoded, nil
}

// extractTitleAndSummary：自动从 Markdown 提取标题 + 简介
func extractTitleAndSummary(slug string, md string) (title, summary string) {
	lines := strings.Split(md, "\n")

	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "# ") {
			title = strings.TrimSpace(strings.TrimPrefix(t, "# "))
			break
		}
		if strings.HasPrefix(t, "## ") && title == "" {
			title = strings.TrimSpace(strings.TrimPrefix(t, "## "))
		}
	}
	if title == "" {
		title = slug
	}

	foundTitle := false
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		if !foundTitle {
			if strings.HasPrefix(t, "#") {
				foundTitle = true
			}
			continue
		}
		if strings.HasPrefix(t, "#") {
			continue
		}
		summary = safeTruncate(t, 160)
		break
	}

	if summary == "" {
		for _, line := range lines {
			t := strings.TrimSpace(line)
			if t == "" {
				continue
			}
			if strings.HasPrefix(t, "#") {
				continue
			}
			summary = safeTruncate(t, 160)
			break
		}
	}

	return title, summary
}

func safeTruncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

// hasChanges：根据 path->sha 判断是否有变更
func hasChanges(oldMap map[string]string, files []githubTreeItem) bool {
	if len(oldMap) != len(files) {
		return true
	}
	for _, f := range files {
		if oldMap[f.Path] != f.SHA {
			return true
		}
	}
	return false
}

// SyncOnce：同步 GitHub -> 本地缓存（递归子目录 + 去重）
func SyncOnce(ctx context.Context, gh *GitHubClient, cache *BlogCache) error {
	files, err := gh.listMarkdownFiles(ctx)
	if err != nil {
		return err
	}

	old := cache.SnapshotSHA()
	if !hasChanges(old, files) {
		// 没变化，连拉内容都不用
		return nil
	}

	var posts []CachedPost

	for _, f := range files {
		_, contentBytes, err := gh.fetchFileContent(ctx, f.Path)
		if err != nil {
			return err
		}

		mdText := string(contentBytes)
		filename := path.Base(f.Path)
		slug := strings.TrimSuffix(filename, ".md")
		title, summary := extractTitleAndSummary(slug, mdText)

		meta := PostMeta{
			Slug:      slug,
			Title:     title,
			Summary:   summary,
			Path:      f.Path,
			SHA:       f.SHA,
			UpdatedAt: time.Now().UTC(),
		}

		posts = append(posts, CachedPost{
			Meta:    meta,
			Content: mdText,
		})
	}

	cache.ReplaceAll(posts)

	if err := cache.SaveToDisk(); err != nil {
		return fmt.Errorf("save cache: %w", err)
	}

	return nil
}
