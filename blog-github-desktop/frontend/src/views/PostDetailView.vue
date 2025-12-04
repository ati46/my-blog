<script setup>
import {
  ref,
  computed,
  onMounted,
  onBeforeUnmount,
  nextTick,
  watch,
} from "vue";
import { useRoute, useRouter } from "vue-router";
import { fetchPostDetail } from "../api/client";
import { renderMarkdown } from "../utils/markdown";

const route = useRoute();
const router = useRouter();

const slug = computed(() => route.params.slug);

const post = ref(null);
const loading = ref(false);
const error = ref("");
const progress = ref(0);

const contentRef = ref(null);

// 目录项：从 Markdown 文本解析
// { index: number, text: string, level: 'h1'|'h2'|'h3' }
const tocItems = ref([]);
const activeIndex = ref(-1);

// ===== 加载当前文章 =====
async function loadCurrentPost() {
  const s = slug.value;
  if (!s) return;

  loading.value = true;
  error.value = "";
  post.value = null;
  tocItems.value = [];
  activeIndex.value = -1;
  progress.value = 0;

  try {
    const data = await fetchPostDetail(s);
    post.value = data;

    // 先根据 Markdown 文本构建 TOC
    buildTocFromMarkdown();

    // 再等 DOM 渲染完做一次滚动状态更新
    await nextTick();
    updateScrollState();
    window.scrollTo({ top: 0, behavior: "auto" });
  } catch (e) {
    console.error("load post error", e);
    error.value = e.message || "加载文章失败";
  } finally {
    loading.value = false;
  }
}

// 路由参数变化时重新加载
watch(
  slug,
  async (n, o) => {
    if (!n || n === o) return;
    await loadCurrentPost();
  },
  { immediate: true }
);

// ===== Markdown 渲染 =====
const html = computed(() =>
  post.value ? renderMarkdown(post.value.content) : ""
);

// 返回列表
const goBack = () => {
  router.push({ path: "/" });
};

// ===== 从 Markdown 文本解析 TOC（不依赖 DOM） =====
function buildTocFromMarkdown() {
  const md = post.value?.content || "";
  const lines = md.split(/\r?\n/);

  const items = [];
  let idx = 0;

  // 行首 1~3 个 # + 空格，认为是标题
  const headingRE = /^(#{1,3})\s+(.+?)\s*$/;

  for (const line of lines) {
    const m = line.match(headingRE);
    if (!m) continue;

    const levelSharp = m[1].length; // 1,2,3
    const text = m[2].trim();
    const level = "h" + levelSharp;

    items.push({
      index: idx,
      text,
      level,
    });

    idx++;
  }

  tocItems.value = items;
}

// ===== 阅读进度 & 当前高亮 =====
function updateScrollState() {
  const doc = document.documentElement;
  const scrollTop = window.scrollY || doc.scrollTop || 0;
  const scrollHeight = doc.scrollHeight - doc.clientHeight;
  if (scrollHeight <= 0) {
    progress.value = 0;
  } else {
    progress.value = Math.min(
      100,
      Math.max(0, (scrollTop / scrollHeight) * 100)
    );
  }

  const root = contentRef.value;
  if (!root || tocItems.value.length === 0) {
    activeIndex.value = -1;
    return;
  }

  const headings = Array.from(root.querySelectorAll("h1, h2, h3"));
  if (!headings.length) {
    activeIndex.value = -1;
    return;
  }

  const offsetTop = 120;
  let current = -1;

  headings.forEach((el, i) => {
    const rect = el.getBoundingClientRect();
    if (rect.top - offsetTop <= 0) {
      current = i;
    }
  });

  activeIndex.value = current;
}

const onScroll = () => updateScrollState();

function scrollToTocItem(item) {
  const root = contentRef.value;
  if (!root) return;

  const headings = Array.from(root.querySelectorAll("h1, h2, h3"));
  if (!headings.length) return;

  const i = item.index;
  if (i < 0 || i >= headings.length) return;

  headings[i].scrollIntoView({ behavior: "smooth", block: "start" });
}

onMounted(() => {
  window.addEventListener("scroll", onScroll, { passive: true });
  window.addEventListener("resize", onScroll);
});

onBeforeUnmount(() => {
  window.removeEventListener("scroll", onScroll);
  window.removeEventListener("resize", onScroll);
});

// html 变化时，再更新一次滚动状态（DOM 位置可能变了）
watch(html, async () => {
  await nextTick();
  updateScrollState();
});
</script>

<template>
  <div class="detail-page">
    <!-- 顶部阅读进度条 -->
    <div class="reading-progress">
      <div class="reading-progress-bar" :style="{ width: progress + '%' }" />
    </div>

    <div class="detail-header">
      <button type="button" class="back-btn" @click="goBack">
        ← 返回
      </button>
    </div>

    <div v-if="loading" class="hint">正在加载正文…</div>
    <div v-else-if="error" class="error">{{ error }}</div>

    <!-- 一张阅读卡片，内部左正文 + 右 TOC -->
    <div v-else-if="post" class="detail-card">
      <div class="detail-card-inner">
        <!-- 左边正文 -->
        <div class="detail-main">
          <header class="detail-card-header">
            <h1 class="detail-title">
              {{ post.title }}
            </h1>
            <p class="detail-subtitle" v-if="post.path">
              <span class="chip">Markdown</span>
              <span class="path-text">{{ post.path }}</span>
            </p>
          </header>

          <section
            ref="contentRef"
            class="detail-body markdown-body"
            v-html="html"
          />
        </div>

        <!-- 右边 TOC：在阅读卡片内部右侧 -->
        <aside v-if="tocItems.length" class="detail-toc">
          <div class="toc-title">目录</div>
          <ul class="toc-list">
            <li
              v-for="(item, idx) in tocItems"
              :key="idx"
              :class="[
                'toc-item',
                'level-' + item.level,
                { active: idx === activeIndex },
              ]"
            >
              <button type="button" @click="scrollToTocItem(item)">
                {{ item.text }}
              </button>
            </li>
          </ul>
        </aside>
      </div>
    </div>
  </div>
</template>

<style scoped>
.detail-page {
  padding-top: 8px;
  min-height: calc(100vh - 120px);
  display: flex;
  flex-direction: column;
}

/* 顶部阅读进度条 */
.reading-progress {
  position: sticky;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: transparent;
  z-index: 50;
}

.reading-progress-bar {
  height: 100%;
  border-radius: 999px;
  background: linear-gradient(90deg, #3b82f6, #22c55e);
  transition: width 0.15s ease-out;
}

.detail-header {
  margin: 8px 0 12px;
}

.back-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  border-radius: 999px;
  border: 1px solid var(--card-border);
  background: var(--card-bg);
  color: var(--text-main);
  cursor: pointer;
  box-shadow: 0 1px 2px var(--card-shadow);
  transition: all 0.22s ease;
}

.back-btn:hover {
  background: rgba(255, 255, 255, 0.15);
  transform: translateY(-1px);
}

/* 阅读卡片：整块阅读区 */
.detail-card {
  flex: 1;
  padding: 24px 26px 26px;
  border-radius: 18px;
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  box-shadow: 0 6px 18px var(--card-shadow);
  min-height: 60vh;
}

/* 卡片内部：左正文 + 右 TOC */
.detail-card-inner {
  display: flex;
  align-items: flex-start;
  gap: 24px;
}

/* 左侧正文 */
.detail-main {
  flex: 1;
  min-width: 0;
}

.detail-card-header {
  margin-bottom: 18px;
  border-bottom: 1px solid #e5e7eb33;
  padding-bottom: 10px;
}

.detail-title {
  margin: 0 0 8px;
  font-size: 24px;
  font-weight: 750;
  letter-spacing: 0.02em;
  color: var(--text-main);
}

.detail-subtitle {
  margin: 0;
  font-size: 12px;
  color: var(--text-sub);
  display: flex;
  align-items: center;
  gap: 8px;
}

.chip {
  display: inline-flex;
  align-items: center;
  padding: 0 8px;
  height: 20px;
  border-radius: 999px;
  border: 1px solid var(--chip-border);
  font-size: 11px;
  color: var(--text-sub);
  background: var(--chip-bg);
}

.path-text {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas,
    "Liberation Mono", "Courier New", monospace;
  font-size: 11px;
  color: #9ca3af;
}

.detail-body {
  margin-top: 8px;
  color: var(--text-main);
}

/* 右侧 TOC：在卡片内部右边，视觉更聚拢 */
.detail-toc {
  width: 200px;
  flex-shrink: 0;
  position: sticky;
  top: 72px;
  align-self: flex-start;
  font-size: 12px;
  color: var(--text-sub);
  border-left: 1px dashed rgba(148, 163, 184, 0.35);
  padding-left: 14px;
}

.toc-title {
  font-weight: 600;
  margin-bottom: 6px;
  color: var(--text-main);
}

.toc-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.toc-item {
  margin-bottom: 4px;
}

.toc-item button {
  padding: 2px 4px;
  border: none;
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
  width: 100%;
  border-radius: 4px;
}

/* hover/active 都比较柔和一点 */
.toc-item button:hover {
  background: rgba(148, 163, 184, 0.08);
}

.toc-item.level-h1 button {
  font-weight: 600;
}

.toc-item.level-h2 button {
  padding-left: 6px;
}

.toc-item.level-h3 button {
  padding-left: 14px;
  font-size: 11px;
}

.toc-item.active button {
  background: rgba(59, 130, 246, 0.14);
  color: #2563eb;
}

/* 窄屏时 TOC 收掉，只保留正文 */
@media (max-width: 900px) {
  .detail-card-inner {
    flex-direction: column;
  }
  .detail-toc {
    position: static;
    width: 100%;
    border-left: none;
    border-top: 1px dashed rgba(148, 163, 184, 0.35);
    padding-left: 0;
    padding-top: 8px;
    margin-top: 8px;
  }
}

.hint {
  font-size: 13px;
  color: #666;
}

.error {
  font-size: 13px;
  color: #c0392b;
}
</style>
