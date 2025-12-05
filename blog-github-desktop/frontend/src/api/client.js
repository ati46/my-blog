// frontend/src/api/client.js

// 去掉末尾多余的 /，方便后面拼接
const RAW_API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "";
const API_BASE_URL = RAW_API_BASE_URL.replace(/\/+$/, "");

async function handleResp(res) {
  if (!res.ok) {
    const t = await res.text();
    throw new Error(t || ("HTTP " + res.status));
  }
  const ct = res.headers.get("content-type") || "";
  if (ct.includes("application/json")) {
    return res.json();
  }
  return res.text();
}

// 通用构建 URL 函数：
// - 如果设置了 API_BASE_URL：用绝对地址（例如 http://127.0.0.1:8080/api/posts?page=1）
// - 如果没设置：用相对地址（例如 /api/posts?page=1）
function buildUrl(path, params) {
  let url = API_BASE_URL ? API_BASE_URL + path : path;

  if (params && Object.keys(params).length > 0) {
    const usp = new URLSearchParams();
    for (const [k, v] of Object.entries(params)) {
      if (v === undefined || v === null || v === "") continue;
      usp.set(k, String(v));
    }
    const qs = usp.toString();
    if (qs) {
      url += (url.includes("?") ? "&" : "?") + qs;
    }
  }

  return url;
}

// 列表：分页加载
export async function fetchPosts(page = 1) {
  const url = buildUrl("/api/posts", { page });
  const res = await fetch(url);
  return handleResp(res); // {items, page, page_size, total, has_more}
}

// 详情
export async function fetchPostDetail(slug) {
  const url = buildUrl(`/api/posts/${encodeURIComponent(slug)}`);
  const res = await fetch(url);
  return handleResp(res);
}

// 搜索接口：GET /api/search?q=xxx&limit=20
export async function searchPosts(keyword, limit = 20) {
  const url = buildUrl("/api/search", { q: keyword, limit });
  const res = await fetch(url);
  return handleResp(res); // {items: [...]}
}
