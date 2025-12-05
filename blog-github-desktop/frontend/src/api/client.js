// frontend/src/api/client.js

// 优先用环境变量，不写死域名；没有就用相对路径
const rawBase = import.meta.env.VITE_API_BASE_URL || "";
const API_BASE_URL = rawBase.trim().replace(/\/+$/, "");

// 在控制台打印，方便在 Wails / 浏览器里确认
console.log("[API] VITE_API_BASE_URL =", import.meta.env.VITE_API_BASE_URL);
console.log("[API] API_BASE_URL     =", API_BASE_URL);

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

function buildUrl(path, params) {
  // 有 BASE 就绝对地址，否则相对路径
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

// 列表
export async function fetchPosts(page = 1) {
  const url = buildUrl("/api/posts", { page });
  console.log("[API] fetchPosts ->", url);
  const res = await fetch(url);
  return handleResp(res);
}

// 详情
export async function fetchPostDetail(slug) {
  const url = buildUrl(`/api/posts/${encodeURIComponent(slug)}`);
  console.log("[API] fetchPostDetail ->", url);
  const res = await fetch(url);
  return handleResp(res);
}

// 搜索
export async function searchPosts(keyword, limit = 20) {
  const url = buildUrl("/api/search", { q: keyword, limit });
  console.log("[API] searchPosts ->", url);
  const res = await fetch(url);
  return handleResp(res);
}
