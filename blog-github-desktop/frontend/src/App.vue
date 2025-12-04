<script setup>
import { ref, onMounted, onBeforeUnmount } from "vue";
import { useRouter } from "vue-router";
import SearchOverlay from "./components/SearchOverlay.vue";

const router = useRouter();

const theme = ref("light"); // "light" | "dark"
const searchOpen = ref(false);

const applyTheme = () => {
  const target = theme.value === "dark" ? "theme-dark" : "theme-light";
  const root = document.documentElement;
  if (!root) return;
  root.classList.remove("theme-light", "theme-dark");
  root.classList.add(target);
  root.dataset.theme = target;
};

const toggleTheme = () => {
  theme.value = theme.value === "dark" ? "light" : "dark";
  localStorage.setItem("blog-theme", theme.value);
  applyTheme();
};

// Cmd/Ctrl + K 打开/关闭搜索
const handleKeydown = (e) => {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
    e.preventDefault();
    searchOpen.value = !searchOpen.value;
  }
};

onMounted(() => {
  const saved = localStorage.getItem("blog-theme");
  if (saved === "light" || saved === "dark") {
    theme.value = saved;
  } else if (window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches) {
    theme.value = "dark";
  } else {
    theme.value = "light";
  }
  applyTheme();

  window.addEventListener("keydown", handleKeydown);
});

onBeforeUnmount(() => {
  window.removeEventListener("keydown", handleKeydown);
});

// 搜索结果点击打开文章
const openPostFromSearch = (slug) => {
  if (!slug) return;
  router.push({
    path: `/post/${encodeURIComponent(slug)}`,
  });
  searchOpen.value = false;
};
</script>

<template>
  <div class="app-bg">
    <div class="app-shell">
      <!-- 顶部右上角：搜索 + 主题切换 -->
      <div class="top-right">
        <button
          class="search-toggle"
          type="button"
          @click="searchOpen = true"
        >
          🔍 搜索
          <span class="shortcut">⌘ / Ctrl + K</span>
        </button>

        <button class="theme-toggle" type="button" @click="toggleTheme">
          <span class="icon-wrapper">
            <span :class="['icon', theme]" />
          </span>
          <span class="theme-label">
            {{ theme === "dark" ? "明亮模式" : "深色模式" }}
          </span>
        </button>
      </div>

      <main class="layout-main">
        <section class="layout-center">
          <router-view v-slot="{ Component }">
            <keep-alive include="PostListView">
              <component :is="Component" />
            </keep-alive>
          </router-view>
        </section>
      </main>

      <!-- 普通 footer：固定在视窗底部 -->
      <footer class="app-footer">
        © 2025 Typography · Powered by Wails + Vue
      </footer>

      <!-- 全局搜索 Overlay -->
      <SearchOverlay
        v-model="searchOpen"
        @openPost="openPostFromSearch"
      />
    </div>
  </div>
</template>

<style scoped>
/* 外层背景容器，居中内容 */
.app-bg {
  min-height: 100vh;
  display: flex;
  justify-content: center;
}

/* 主骨架：flex 列，footer 自然在底部 */
.app-shell {
  width: 100%;
  max-width: 1200px;
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  position: relative;
}

/* 主体布局：单列 */
.layout-main {
  flex: 1;
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 0;
  padding: 56px 40px 24px;
}

/* 中间阅读区域 */
.layout-center {
  min-width: 0;
}

/* 顶部右侧按钮区域 */
.top-right {
  position: absolute;
  top: 16px;
  right: 32px;
  display: flex;
  gap: 8px;
  z-index: 100;
}

/* 搜索按钮 */
.search-toggle {
  padding: 4px 10px;
  border-radius: 999px;
  border: 1px solid var(--card-border);
  background: var(--card-bg);
  font-size: 12px;
  color: var(--text-main);
  cursor: pointer;
  box-shadow: 0 1px 3px var(--card-shadow);
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.search-toggle .shortcut {
  font-size: 10px;
  color: var(--text-sub);
}

/* 主题切换按钮 */
.theme-toggle {
  padding: 4px 12px 4px 10px;
  border-radius: 999px;
  border: 1px solid var(--card-border);
  background: var(--card-bg);
  font-size: 12px;
  color: var(--text-main);
  cursor: pointer;
  box-shadow: 0 1px 3px var(--card-shadow);
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.icon-wrapper {
  width: 18px;
  height: 18px;
  position: relative;
}

.icon {
  position: absolute;
  inset: 0;
  border-radius: 50%;
  transition: all 0.3s ease;
}

/* 亮色：太阳 */
.icon.light {
  background: #fbbf24;
  transform: scale(1) rotate(0deg);
  box-shadow: 0 0 6px #fbbf24;
}

/* 暗色：月亮 */
.icon.dark {
  background: #f5f3f4;
  transform: scale(0.75) translateX(3px) rotate(25deg);
  box-shadow: 0 0 6px #e5e7eb;
}

.theme-label {
  white-space: nowrap;
}

/* 响应式：窄屏时右侧栏下移、按钮位置略调 */
@media (max-width: 1024px) {
  .layout-main {
    grid-template-columns: minmax(0, 1fr);
    gap: 24px;
    padding: 56px 16px 24px;
  }
}

@media (max-width: 640px) {
  .top-right {
    right: 16px;
  }

  .search-toggle .shortcut {
    display: none;
  }

  .theme-label {
    display: none;
  }
}

</style>
