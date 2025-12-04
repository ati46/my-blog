<script setup>
import { ref } from "vue";
import { searchPosts } from "../api/client";

const props = defineProps({
  modelValue: {
    type: Boolean,
    required: true,
  },
});

const emit = defineEmits(["update:modelValue", "openPost"]);

const query = ref("");
const loading = ref(false);
const results = ref([]); // 标准化后的数组
const error = ref("");
const hasSearched = ref(false); // ✅ 是否已经执行过一次搜索

function close() {
  emit("update:modelValue", false);
  query.value = "";
  results.value = [];
  error.value = "";
  hasSearched.value = false;
}

// Esc 关闭
function onKeydown(e) {
  if (e.key === "Escape") {
    e.preventDefault();
    close();
  }
}

// 执行搜索（只在回车 / 点击按钮时触发）
async function doSearch() {
  const q = query.value.trim();
  if (!q) {
    results.value = [];
    error.value = "";
    hasSearched.value = false;
    return;
  }

  loading.value = true;
  error.value = "";
  hasSearched.value = false; // 搜索开始前先清掉状态

  try {
    const data = await searchPosts(q);

    // 兼容 {items: [...]} 和 [...]
    let items;
    if (Array.isArray(data)) {
      items = data;
    } else if (data && Array.isArray(data.items)) {
      items = data.items;
    } else {
      items = [];
    }

    results.value = items.filter(
      (it) => it && typeof it === "object"
    );
    hasSearched.value = true; // ✅ 搜索成功完成
  } catch (e) {
    console.error("search error", e);
    error.value = e.message || "搜索失败";
    results.value = [];
    hasSearched.value = true; // 也算完成一次搜索，只是失败了
  } finally {
    loading.value = false;
  }
}

function handleClickItem(item) {
  if (!item || !item.slug) {
    return;
  }
  emit("openPost", item.slug);
  close();
}
</script>

<template>
  <teleport to="body">
    <div
      v-if="modelValue"
      class="search-overlay"
      tabindex="-1"
      @keydown.stop="onKeydown"
    >
      <div class="backdrop" @click="close" />

      <div class="panel" @click.stop>
        <div class="input-row">
          <input
            v-model="query"
            class="search-input"
            type="text"
            placeholder="搜索文章标题或摘要…"
            @keyup.enter="doSearch"
            autofocus
          />
          <button class="search-btn" type="button" @click="doSearch">
            搜索
          </button>
        </div>

        <div class="hint-row">
          <span class="hint">回车搜索 · Esc 关闭</span>
        </div>

        <div class="result-area">
          <div v-if="loading" class="hint">搜索中…</div>
          <div v-else-if="error" class="error-block">
            <div class="error">{{ error }}</div>
            <button type="button" class="retry-btn" @click="doSearch">
              重试
            </button>
          </div>

          <ul v-else class="result-list">
            <li
              v-for="item in results"
              :key="item.slug || item.path || item.title"
              class="result-item"
            >
              <button type="button" @click="handleClickItem(item)">
                <div class="title">
                  {{ item.title || item.slug || "未命名" }}
                </div>
                <div v-if="item.summary" class="summary">
                  {{ item.summary }}
                </div>
                <div v-if="item.path" class="path">
                  {{ item.path }}
                </div>
              </button>
            </li>

            <!-- ✅ 只有在执行过搜索、没有结果且没有错误时才提示“没有找到” -->
            <li
              v-if="hasSearched && !loading && !error && !results.length && query.trim()"
              class="hint"
            >
              没有找到相关内容
            </li>
          </ul>
        </div>
      </div>
    </div>
  </teleport>
</template>

<style scoped>
.search-overlay {
  position: fixed;
  inset: 0;
  z-index: 2000;
}

.backdrop {
  position: absolute;
  inset: 0;
  background: rgba(15, 23, 42, 0.55);
}

.panel {
  position: relative;
  z-index: 1;
  max-width: 640px;
  margin: 80px auto 0;
  padding: 16px 18px 12px;
  border-radius: 16px;
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  box-shadow: 0 20px 40px var(--card-shadow);
}

.input-row {
  display: flex;
  gap: 8px;
}

.search-input {
  flex: 1;
  border-radius: 999px;
  border: 1px solid var(--card-border);
  padding: 8px 12px;
  font-size: 14px;
  outline: none;
  background: rgba(255, 255, 255, 0.85);
}

:global(.theme-dark) .search-input {
  background: #111827;
  color: #e5e7eb;
}

.search-btn {
  border-radius: 999px;
  border: 1px solid var(--chip-border);
  background: var(--chip-bg);
  padding: 6px 14px;
  cursor: pointer;
  font-size: 13px;
}

.hint-row {
  margin-top: 4px;
  font-size: 11px;
  color: var(--text-sub);
}

.result-area {
  margin-top: 10px;
  max-height: 320px;
  overflow-y: auto;
}

.result-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.result-item + .result-item {
  margin-top: 4px;
}

.result-item button {
  width: 100%;
  text-align: left;
  padding: 6px 8px;
  border-radius: 8px;
  border: none;
  background: transparent;
  color: inherit;
  cursor: pointer;
}

.result-item button:hover {
  background: rgba(255, 255, 255, 0.06);
}

.error-block {
  display: flex;
  align-items: center;
  gap: 8px;
}

.retry-btn {
  border-radius: 8px;
  border: 1px solid var(--card-border);
  background: var(--card-bg);
  padding: 4px 10px;
  cursor: pointer;
  color: var(--text-main);
}

.title {
  font-size: 14px;
  font-weight: 600;
}

.summary {
  font-size: 12px;
  color: var(--text-sub);
}

.path {
  font-size: 11px;
  color: #9ca3af;
}

.hint {
  font-size: 12px;
  color: #9ca3af;
  padding: 4px 0;
}

.error {
  font-size: 13px;
  color: #f97373;
}
</style>
