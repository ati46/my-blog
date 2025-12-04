const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://127.0.0.1:8080";

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

export async function fetchPosts(page = 1) {
  const url = new URL(API_BASE_URL + "/api/posts");
  url.searchParams.set("page", String(page));
  const res = await fetch(url.toString());
  return handleResp(res);  // {items, page, page_size, total, has_more}
}

export async function fetchPostDetail(slug) {
  const res = await fetch(API_BASE_URL + "/api/posts/" + slug);
  return handleResp(res);
}

// 搜索文章：GET /api/search?q=xxx
export async function searchPosts(query) {
  const url = new URL(API_BASE_URL + "/api/search");
  url.searchParams.set("q", query);
  const res = await fetch(url.toString());
  return handleResp(res);  // {items: []}
}
