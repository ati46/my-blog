<script setup>
import { computed } from "vue";

const props = defineProps({
  post: { type: Object, required: true },
});

const emit = defineEmits(["open"]);

/**
 * 简单清理 Markdown 摘要：
 */
const cleanSummary = (text) => {
  if (!text) return "";

  let s = text;

  const prefixes = ["> ", "# ", "## ", "### ", "- ", "* ", "+ "];
  for (const p of prefixes) {
    if (s.startsWith(p)) {
      s = s.slice(p.length);
      break;
    }
  }

  s = s.split("```").join("");
  s = s.split("`").join("");
  s = s.replace(/\s+/g, " ").trim();

  return s;
};

const displaySummary = computed(() => {
  const raw = props.post.summary || "";
  const cleaned = cleanSummary(raw);
  if (!cleaned) return "";
  return cleaned.length > 160 ? cleaned.slice(0, 160) + "…" : cleaned;
});

/**
 * 目录标签：尽量简短
 * - 如果路径类似 posts/2025/01/xxx.md -> 显示 2025/01
 * - 如果只有一层目录 blog/xxx.md -> 显示 blog
 * - 根目录 -> “未分组”
 */
const folderLabel = computed(() => {
  const p = props.post.path || "";
  if (!p) return "未分组";
  const parts = p.split("/").filter(Boolean);
  if (parts.length <= 1) return "未分组";

  // 去掉文件名
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

  // 否则只取最后一层目录
  return dirs[dirs.length - 1];
});

/**
 * 日期格式：YYYY.MM.DD
 */
const dateLabel = computed(() => {
  const raw = props.post.updatedAt || props.post.updated_at;
  if (!raw) return "";
  const d = new Date(raw);
  if (Number.isNaN(d.getTime())) return "";
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}.${m}.${day}`;
});

const onOpen = () => {
  emit("open", props.post.slug);
};
</script>

<template>
  <article class="preview-row" @click="onOpen">
    <div class="preview-main">
      <h2 class="preview-title">
        {{ post.title }}
      </h2>
      <p v-if="displaySummary" class="preview-summary">
        {{ displaySummary }}
      </p>
    </div>

    <div class="preview-meta">
      <span class="folder-chip">{{ folderLabel }}</span>
      <span v-if="dateLabel" class="date-text">{{ dateLabel }}</span>
    </div>
  </article>
</template>
<style scoped>
.preview-row {
  padding: 12px 4px;
  border-bottom: 1px solid var(--card-border);
  cursor: pointer;
  display: flex;
  align-items: flex-start;
  gap: 16px;

  transition:
    background 0.22s ease,
    transform 0.18s ease,
    box-shadow 0.18s ease,
    border-color 0.18s ease;
}

.preview-row:last-of-type {
  border-bottom: none;
}

/* 默认 hover：亮色主题下，从中间向两边渐变变浅 */
.preview-row:hover {
  background: linear-gradient(
    90deg,
    rgba(255, 255, 255, 0) 0%,
    rgba(255, 255, 255, 0.7) 50%,
    rgba(255, 255, 255, 0) 100%
  );
  transform: translateY(-1px);
}

/* 左：标题+摘要 */
.preview-main {
  flex: 1;
  min-width: 0;
}

.preview-title {
  margin: 0 0 4px;
  font-size: 18px;
  font-weight: 700;
  color: var(--text-main);
  letter-spacing: 0.01em;
}

.preview-summary {
  margin: 0;
  font-size: 14px;
  line-height: 1.7;
  color: var(--text-sub);
  max-width: 64ch;
}

/* 右：目录+日期 */
.preview-meta {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
  font-size: 11px;
  white-space: nowrap;
}

.folder-chip {
  padding: 2px 8px;
  border-radius: 999px;
  border: 1px solid var(--chip-border);
  background: var(--chip-bg);
  color: var(--text-sub);
}

.date-text {
  color: #9ca3af;
}

/* 深色模式：高亮从中间变亮，边缘融回暗背景，不再整块灰条 */
:global(.theme-dark) .preview-row:hover {
  background: linear-gradient(
    90deg,
    rgba(15, 23, 42, 0) 0%,
    rgba(148, 163, 184, 0.35) 50%,
    rgba(15, 23, 42, 0) 100%
  );
  transform: translateY(-1px);
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.45);
}
</style>
