# Markdown 文件预览设计

## 概述

在 PreviewPanel 中增加 Markdown 渲染能力。用户点击文件树中的 .md/.mdx/.markdown 文件时，只读模式下显示渲染后的 Markdown 内容；切换到写模式时显示 CodeMirror 原始文本编辑器。所有文件默认只读模式。

## 状态与切换逻辑

- `editMode` 默认 `false`（只读），所有文件统一行为
- 新增 `isMarkdown` 判断：检测 `.md` / `.mdx` / `.markdown` 扩展名
- **读模式 + Markdown 文件**：隐藏 CodeMirror 容器，显示 MarkdownPreview 渲染区域
- **读模式 + 非 Markdown 文件**：显示只读 CodeMirror（现有行为不变）
- **写模式（任何文件）**：显示 CodeMirror，隐藏 MarkdownPreview
- 切换时 CodeMirror 用 `display: none` 保留在 DOM 中，避免重建
- 工具栏 eye/edit 按钮行为不变

## MarkdownPreview 子组件

**文件**：`frontend/src/components/Preview/MarkdownPreview.tsx`

**Props**：
- `content: string` — Markdown 原始文本
- `filePath: string` — 文件路径，用于解析图片相对路径

**渲染**：
- 使用 `react-markdown` + `remark-gfm` + `rehype-highlight`（已在项目依赖中）
- 自定义 components：
  - `img`：相对路径基于 `filePath` 所在目录解析
  - `a`：外部链接用 `Browser.OpenURL` 打开
  - `pre`：带复制按钮的代码块

**样式**（暗色主题，与 PreviewPanel 风格一致）：
- 行高 1.7，段落间距更宽
- 标题醒目，表格有边框和斑马纹
- 代码块圆角 + 背景色
- 容器可滚动，padding 合理

**图片路径解析**：
- 提取 `filePath` 目录部分
- 相对路径（不以 `http` / `https` / `/` / `data:` 开头）拼接为绝对路径
- 使用 `file://` 协议加载，若 Wails 环境有限制则通过后端 ReadFile API 转 base64

## PreviewPanel 集成改动

**文件**：`frontend/src/components/Preview/PreviewPanel.tsx`

1. 引入 MarkdownPreview 组件
2. 新增 `isMarkdown` 判断
3. CodeMirror 容器：读模式 + Markdown 时 `display: none`，其余 `flex`
4. 新增 MarkdownPreview 渲染：读模式 + Markdown 时显示，其余 `display: none`
5. 两者互斥，不互相销毁

**不改动**：工具栏、FilePreviewHeader、Diff/Task/Empty 模式、非 Markdown 文件的只读行为
