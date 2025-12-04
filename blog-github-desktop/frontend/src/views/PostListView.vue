<script setup>
import { ref, onMounted, onActivated, onDeactivated, computed } from "vue";
import { useRouter } from "vue-router";
import { fetchPosts } from "../api/client";
import PostPreview from "../components/PostPreview.vue";

const posts = ref([]);
const loading = ref(false);
const error = ref("");

const loadingMore = ref(false);
const page = ref(1);
const hasMore = ref(true);

const router = useRouter();

// 记录离开列表页时滚动位置
let savedScrollY = 0;

const loadPage = async (p, append = false) => {
  if (p < 1) p = 1;

  if (!append) {
    loading.value = true;
    error.value = "";
  } else {
    loadingMore.value = true;
  }

  try {
    const data = await fetchPosts(p);
    if (!append) {
      posts.value = data.items || [];
    } else {
      posts.value = posts.value.concat(data.items || []);
    }
    page.value = data.page || p;
    hasMore.value = !!data.has_more;
  } catch (e) {
    error.value = e.message || "加载文章列表失败";
  } finally {
    loading.value = false;
    loadingMore.value = false;
  }
};

const loadMore = () => {
  if (!hasMore.value || loadingMore.value) return;
  loadPage(page.value + 1, true);
};

const openPost = (slug) => {
  router.push({ name: "post-detail", params: { slug } });
};

/**
 * 分组 key 算法：
 * - 如果 path 像 posts/2025/01/xxx.md -> 分组名 "2025/01"
 * - 如果 path 像 note/life/xxx.md -> 分组名 "life"
 * - 根目录 -> "未分组"
 */
const deriveGroupKey = (post) => {
  const p = post.path || "";
  if (!p) return "未分组";
  const parts = p.split("/").filter(Boolean);
  if (parts.length <= 1) return "未分组";
  const dirs = parts.slice(0, parts.length - 1);
  const n = dirs.length;

  if (n >= 2) {
    const a = dirs[n - 2];
    const b = dirs[n - 1];
    const isYear = /^\d{4}$/.test(a);
    const isMonth = /^\d{1,2}$/.test(b);
    if (isYear && isMonth) {
      return `${a}/${b.padStart(2, "0")}`;
    }
  }
  return dirs[dirs.length - 1];
};

const grouped = computed(() => {
  const map = new Map();
  for (const p of posts.value) {
    const key = deriveGroupKey(p);
    if (!map.has(key)) {
      map.set(key, []);
    }
    map.get(key).push(p);
  }

  // 保持插入顺序
  return Array.from(map.entries()).map(([key, items]) => ({ key, items }));
});

// 首次挂载时加载第一页；被 keep-alive 缓存后再次回来不会重新加载
onMounted(() => {
  if (posts.value.length === 0) {
    loadPage(1, false);
  }
});

const retryLoad = () => loadPage(1, false);

// 离开（被 keep-alive 缓存）时记录滚动位置
onDeactivated(() => {
  savedScrollY = window.scrollY || window.pageYOffset || 0;
});

// 回到列表时恢复滚动位置
onActivated(() => {
  if (savedScrollY > 0) {
    window.scrollTo({
      top: savedScrollY,
      behavior: "auto",
    });
  }
});
</script>

<template>
  <div class="list-page">
    <h1 class="page-title">文章</h1>

    <div v-if="loading && posts.length === 0" class="hint">
      加载文章列表中…
    </div>
    <div v-else-if="error" class="error-block">
      <div class="error">{{ error }}</div>
      <button type="button" class="retry-btn" @click="retryLoad">
        重试
      </button>
    </div>
    <div v-else-if="!loading && posts.length === 0" class="empty">
      暂无文章
      <button type="button" class="retry-btn" @click="retryLoad">
        刷新
      </button>
    </div>

    <div v-else class="list-card">
      <div class="list-inner">
        <section
          v-for="group in grouped"
          :key="group.key"
          class="group-section"
        >
          <h2 class="group-title">
            {{ group.key }}
          </h2>

          <PostPreview
            v-for="p in group.items"
            :key="p.slug"
            :post="p"
            @open="openPost"
          />
        </section>

        <div class="list-footer">
          <button
            v-if="hasMore"
            type="button"
            class="load-more-btn"
            :disabled="loadingMore"
            @click="loadMore"
          >
            <span v-if="!loadingMore">加载更多</span>
            <span v-else>加载中…</span>
          </button>

          <div
            v-else-if="posts.length > 0"
            class="end-text"
          >
            没有更多文章了
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.list-page {
  padding-top: 4px;
  min-height: calc(100vh - 120px);
  display: flex;
  flex-direction: column;
}

.page-title {
  margin-top: 0;
  margin-bottom: 16px;
  font-size: 26px;
  font-weight: 700;
  letter-spacing: 0.015em;
  color: var(--text-main);
}

/* 大卡片：现在使用 theme-light / theme-dark 的变量 */
.list-card {
  flex: 1;
  padding: 18px 22px;
  border-radius: 18px;

  background: var(--card-bg);
  border: 1px solid var(--card-border);
  box-shadow: 0 6px 18px var(--card-shadow);

  display: flex;
}

.list-inner {
  margin: 0;
  width: 100%;
  max-width: none;
  display: flex;
  flex-direction: column;
  min-height: 60vh;
}

.group-section + .group-section {
  margin-top: 12px;
  padding-top: 6px;
  border-top: 1px dashed var(--card-border);
}

.group-title {
  margin: 0 0 4px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-sub);
}

.error-block {
  display: flex;
  align-items: center;
  gap: 12px;
}

.retry-btn {
  padding: 6px 12px;
  border-radius: 999px;
  border: 1px solid var(--card-border);
  background: var(--card-bg);
  cursor: pointer;
  color: var(--text-main);
}

.empty {
  padding: 12px 0;
  color: var(--text-sub);
  display: flex;
  align-items: center;
  gap: 10px;
}

.list-footer {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--card-border);
  text-align: center;
}

.load-more-btn {
  padding: 6px 16px;
  border-radius: 999px;
  border: 1px solid var(--chip-border);
  background: var(--chip-bg);
  font-size: 13px;
  cursor: pointer;
  color: var(--text-main);
}

.end-text {
  font-size: 12px;
  color: var(--text-sub);
}
</style>
