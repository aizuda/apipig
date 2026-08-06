<script setup lang="ts">
import { computed } from 'vue'
import DOMPurify from 'dompurify'
import { marked } from 'marked'

const props = withDefaults(
  defineProps<{ content: string; align?: 'left' | 'right' }>(),
  { align: 'left' },
)

const renderedContent = computed(() => {
  const html = marked.parse(props.content, { async: false, breaks: true, gfm: true }) as string
  return DOMPurify.sanitize(html, { USE_PROFILES: { html: true } })
})

function openLink(event: MouseEvent) {
  const element = event.target instanceof Element ? event.target.closest('a') : null
  if (!(element instanceof HTMLAnchorElement) || !element.href) return
  event.preventDefault()
  window.open(element.href, '_blank', 'noopener,noreferrer')
}
</script>

<template>
  <div
    class="markdown-content text-sm leading-6"
    :class="props.align === 'right' ? 'markdown-content--right' : 'markdown-content--left'"
    @click="openLink"
    v-html="renderedContent"
  />
</template>

<style scoped>
.markdown-content {
  overflow-wrap: anywhere;
  text-align: left;
}

.markdown-content--right {
  text-align: right;
}

.markdown-content :deep(> :first-child) {
  margin-top: 0;
}

.markdown-content :deep(> :last-child) {
  margin-bottom: 0;
}

.markdown-content :deep(p),
.markdown-content :deep(ul),
.markdown-content :deep(ol),
.markdown-content :deep(blockquote),
.markdown-content :deep(pre),
.markdown-content :deep(table) {
  margin: 0.5rem 0;
}

.markdown-content :deep(h1),
.markdown-content :deep(h2),
.markdown-content :deep(h3),
.markdown-content :deep(h4) {
  margin: 1rem 0 0.4rem;
  font-weight: 600;
  line-height: 1.35;
}

.markdown-content :deep(h1) {
  font-size: 1.25rem;
}

.markdown-content :deep(h2) {
  font-size: 1.125rem;
}

.markdown-content :deep(h3),
.markdown-content :deep(h4) {
  font-size: 1rem;
}

.markdown-content :deep(ul) {
  list-style: disc;
  padding-left: 1.5rem;
}

.markdown-content :deep(ol) {
  list-style: decimal;
  padding-left: 1.5rem;
}

.markdown-content :deep(li + li) {
  margin-top: 0.2rem;
}

.markdown-content :deep(blockquote) {
  border-left: 3px solid var(--border);
  color: var(--muted-foreground);
  padding-left: 0.75rem;
}

.markdown-content :deep(a) {
  color: var(--primary);
  text-decoration: underline;
  text-underline-offset: 3px;
}

.markdown-content :deep(code) {
  border-radius: 4px;
  background: var(--muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.875em;
  padding: 0.12rem 0.3rem;
}

.markdown-content :deep(pre) {
  max-width: 100%;
  overflow-x: auto;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--muted);
  padding: 0.75rem;
}

.markdown-content :deep(pre code) {
  border-radius: 0;
  background: transparent;
  font-size: 0.8125rem;
  padding: 0;
}

.markdown-content :deep(table) {
  display: block;
  max-width: 100%;
  overflow-x: auto;
  border-collapse: collapse;
}

.markdown-content :deep(th),
.markdown-content :deep(td) {
  border: 1px solid var(--border);
  padding: 0.35rem 0.6rem;
  text-align: left;
  white-space: nowrap;
}

.markdown-content :deep(th) {
  background: var(--muted);
  font-weight: 600;
}

.markdown-content :deep(hr) {
  margin: 1rem 0;
  border: 0;
  border-top: 1px solid var(--border);
}

.markdown-content :deep(img) {
  max-width: 100%;
  border-radius: 6px;
}
</style>
