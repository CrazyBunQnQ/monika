# Markdown 文件预览 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 PreviewPanel 中为 .md/.mdx/.markdown 文件增加只读渲染视图，所有文件默认只读模式，工具栏切换编辑。

**Architecture:** 新建 `MarkdownPreview.tsx` 子组件封装 Markdown 渲染逻辑与预览专用样式。PreviewPanel 在读模式 + Markdown 文件时显示该组件，隐藏 CodeMirror；写模式切换回 CodeMirror。两者通过 `display: none` 互斥，不销毁 DOM。

**Tech Stack:** React, react-markdown, remark-gfm, rehype-highlight, TypeScript

---

### Task 1: 创建 MarkdownPreview 子组件

**Files:**
- Create: `frontend/src/components/Preview/MarkdownPreview.tsx`

- [ ] **Step 1: 创建 MarkdownPreview.tsx**

```tsx
import React, { useCallback, useRef, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeHighlight from 'rehype-highlight'
import { Browser } from '@wailsio/runtime'

function CodeBlock({ children, ...rest }: React.ComponentPropsWithoutRef<'pre'>) {
    const ref = useRef<HTMLPreElement>(null)
    const [copied, setCopied] = useState(false)

    const handleCopy = useCallback(() => {
        const el = ref.current
        if (!el) return
        const text = el.textContent || ''
        navigator.clipboard.writeText(text)
        setCopied(true)
        setTimeout(() => setCopied(false), 1500)
    }, [])

    return (
        <div className="relative group/codeblock">
            <pre ref={ref} {...rest}>{children}</pre>
            <button
                className="absolute top-[6px] right-[6px] opacity-0 group-hover/codeblock:opacity-100 transition-opacity
                   text-[10px] font-semibold uppercase tracking-[0.04em] rounded px-1.5 py-0.5
                   hover:bg-[rgba(255,255,255,0.08)] cursor-pointer"
                style={{ color: 'var(--text-dim)' }}
                onClick={handleCopy}
                aria-label="Copy code"
            >
                {copied ? 'Copied' : 'Copy'}
            </button>
        </div>
    )
}

function ExternalLink({ href, children, ...rest }: React.ComponentPropsWithoutRef<'a'>) {
    const handleClick = useCallback((e: React.MouseEvent<HTMLAnchorElement>) => {
        e.preventDefault()
        if (href) Browser.OpenURL(href)
    }, [href])

    return <a href={href} onClick={handleClick} target="_blank" rel="noopener noreferrer" {...rest}>{children}</a>
}

function resolveImageUrl(src: string | undefined, filePath: string): string {
    if (!src) return ''
    if (/^(https?:|data:|\/)/.test(src)) return src
    const dir = filePath.substring(0, filePath.lastIndexOf('/'))
    // 在 Wails 环境中使用 file:// 协议
    const sep = dir.endsWith('/') ? '' : '/'
    return `file://${dir}${sep}${src}`
}

interface MarkdownPreviewProps {
    content: string
    filePath: string
}

export default function MarkdownPreview({ content, filePath }: MarkdownPreviewProps) {
    const imgDir = filePath.substring(0, filePath.lastIndexOf('/'))

    return (
        <div
            className="markdown-preview markdown-body"
            style={{
                flex: 1,
                overflow: 'auto',
                padding: '24px 32px',
                background: '#08090d',
            }}
        >
            <ReactMarkdown
                remarkPlugins={[remarkGfm]}
                rehypePlugins={[rehypeHighlight]}
                components={{
                    pre: CodeBlock,
                    a: ExternalLink,
                    img: ({ src, alt, ...rest }) => (
                        <img src={resolveImageUrl(src, filePath)} alt={alt} {...rest} />
                    ),
                }}
            >
                {content}
            </ReactMarkdown>
        </div>
    )
}
```

- [ ] **Step 2: 提交**

```bash
git add frontend/src/components/Preview/MarkdownPreview.tsx
git commit -m "feat: add MarkdownPreview component for file preview"
```

---

### Task 2: 添加 Markdown 预览专用 CSS 样式

**Files:**
- Modify: `frontend/src/index.css`

- [ ] **Step 1: 在 `index.css` 的 `.markdown-body` 样式块之后追加 `.markdown-preview` 覆盖样式**

在现有 `.markdown-body ol li::before { display: none; }` 之后（约第 500 行），追加：

```css
  /* ---- Preview Markdown (wider spacing for file reading) ---- */
  .markdown-preview {
    font-size: 15px;
    line-height: 1.8;
  }
  .markdown-preview h1 {
    font-size: 1.6em;
    margin-top: 28px;
    padding-bottom: 8px;
  }
  .markdown-preview h2 {
    font-size: 1.35em;
    margin-top: 24px;
    padding-bottom: 6px;
  }
  .markdown-preview h3 {
    font-size: 1.18em;
    margin-top: 20px;
  }
  .markdown-preview h4 { font-size: 1.06em; }
  .markdown-preview h5 { font-size: 1.0em; }
  .markdown-preview h6 { font-size: 0.95em; }
  .markdown-preview p {
    margin-top: 12px;
    margin-bottom: 12px;
  }
  .markdown-preview ul,
  .markdown-preview ol {
    margin-top: 10px;
    margin-bottom: 10px;
    padding-left: 2em;
  }
  .markdown-preview blockquote {
    margin: 16px 0;
    padding: 8px 16px;
  }
  .markdown-preview table {
    margin: 16px 0;
  }
  .markdown-preview pre {
    margin: 16px 0;
    padding: 14px 16px;
    border-radius: 6px;
    font-size: 13.5px;
    line-height: 1.6;
  }
  .markdown-preview code {
    font-size: 13.5px;
  }
  .markdown-preview img {
    margin: 16px 0;
    border-radius: 6px;
    box-shadow: 0 2px 12px rgba(0,0,0,0.3);
  }
  .markdown-preview hr {
    margin: 24px 0;
  }
```

- [ ] **Step 2: 提交**

```bash
git add frontend/src/index.css
git commit -m "feat: add markdown-preview CSS for file reading layout"
```

---

### Task 3: 集成 MarkdownPreview 到 PreviewPanel

**Files:**
- Modify: `frontend/src/components/Preview/PreviewPanel.tsx`

- [ ] **Step 1: 添加 import**

在 PreviewPanel.tsx 顶部的 import 区域，在 `import { CodeMinimap } from './CodeMinimap'` 之后添加：

```tsx
import MarkdownPreview from './MarkdownPreview'
```

- [ ] **Step 2: 添加 isMarkdown 判断**

在 PreviewPanel 函数内，`const showFile = ...` 之后添加：

```tsx
const isMarkdown = /\.(md|mdx|markdown)$/i.test(preview.filePath || '')
const showMarkdownPreview = showFile && !editMode && isMarkdown
```

- [ ] **Step 3: 修改 CodeMirror 容器的 display 逻辑**

找到 `<div ref={containerRef} style={{ flex: 1, overflow: 'hidden' }}`（约第 1075 行），将 `style` 修改为：

```tsx
<div ref={containerRef} style={{ flex: 1, overflow: 'hidden', display: showMarkdownPreview ? 'none' : 'block' }} onContextMenu={e => {
```

- [ ] **Step 4: 在 CodeMirror 容器同级添加 MarkdownPreview**

找到 `{showSymbols && (` 之前（约第 1085 行），在其前面插入 MarkdownPreview 渲染：

```tsx
                    {showMarkdownPreview && (
                        <MarkdownPreview
                            content={displayContent}
                            filePath={preview.filePath || ''}
                        />
                    )}
```

- [ ] **Step 5: 提交**

```bash
git add frontend/src/components/Preview/PreviewPanel.tsx
git commit -m "feat: integrate MarkdownPreview into PreviewPanel — rendered view for md files"
```

---

### Task 4: 验证并修复

- [ ] **Step 1: 运行前端构建确认无类型错误**

```bash
cd frontend && npx tsc --noEmit
```

Expected: 无错误

- [ ] **Step 2: 启动 dev 模式手动验证**

```bash
cd d:/git/monika && wails3 dev
```

验证清单：
1. 点击 .md 文件 → 显示渲染后的 Markdown 内容
2. 点击工具栏编辑按钮 → 切换到 CodeMirror 原始文本
3. 再次点击 → 切换回渲染视图
4. 点击非 Markdown 文件（如 .go / .ts）→ 显示只读 CodeMirror
5. 非 Markdown 文件点击编辑按钮 → 可编辑

- [ ] **Step 3: 修复发现的问题并提交**

```bash
git add -A
git commit -m "fix: address review findings from markdown preview testing"
```
