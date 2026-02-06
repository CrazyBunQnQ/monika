# M5-006: 实现富文本编辑器

**任务ID**: M5-006
**标题**: 实现富文本编辑器
**类型**: frontend (前端开发)
**预估工时**: 2h
**依赖**: 无

---

## 任务描述

实现富文本编辑器组件，用于场景描述、角色背景、手递物内容等的编辑，支持格式化、插入图片、表格等功能。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M5-006-01 | 选择编辑器库 | Library Selection | 10min |
| M5-006-02 | 实现基础编辑器 | Basic Editor | 30min |
| M5-006-03 | 实现工具栏 | Toolbar | 25min |
| M5-006-04 | 实现图片上传 | Image Upload | 20min |
| M5-006-05 | 实现表格支持 | Table Support | 20min |
| M5-006-06 | 实现预览模式 | Preview Mode | 15min |
| M5-006-07 | 编写编辑器测试 | 测试覆盖 | 10min |

---

## 技术选型

推荐使用 **Tiptap** 作为富文本编辑器：
- 基于 ProseMirror，性能优秀
- 支持 React
- 模块化设计
- 支持扩展

```bash
npm install @tiptap/react @tiptap/starter-kit @tiptap/extension-image @tiptap/extension-table @tiptap/extension-table-row
```

---

## 富文本编辑器组件

```tsx
// frontend/src/components/ui/rich-text-editor.tsx
import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Image from '@tiptap/extension-image'
import Table from '@tiptap/extension-table'
import TableRow from '@tiptap/extension-table-row'
import TableCell from '@tiptap/extension-table-cell'
import TableHeader from '@tiptap/extension-table-header'
import Underline from '@tiptap/extension-underline'
import { useCallback, useEffect } from 'react'
import {
  Bold,
  Italic,
  Underline as UnderlineIcon,
  List,
  ListOrdered,
  Heading1,
  Heading2,
  Undo,
  Redo,
  Image as ImageIcon,
  Table as TableIcon,
} from 'lucide-react'
import { Button } from './button'
import { Separator } from './separator'
import { Toggle } from './toggle'

interface RichTextEditorProps {
  content: string
  onChange: (content: string) => void
  placeholder?: string
  editable?: boolean
  minHeight?: string
}

export function RichTextEditor({
  content,
  onChange,
  placeholder = '输入内容...',
  editable = true,
  minHeight = '200px',
}: RichTextEditorProps) {
  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        heading: {
          levels: [1, 2],
        },
      }),
      Underline,
      Image.configure({
        inline: true,
        allowBase64: true,
      }),
      Table.configure({
        resizable: true,
      }),
      TableRow,
      TableHeader,
      TableCell,
    ],
    content,
    editable,
    onUpdate: ({ editor }) => {
      onChange(editor.getHTML())
    },
    editorProps: {
      attributes: {
        class: 'prose prose-sm max-w-none focus:outline-none min-h-[200px] px-4 py-3',
      },
    },
  })

  // 同步外部内容变化
  useEffect(() => {
    if (editor && content !== editor.getHTML()) {
      editor.commands.setContent(content, false)
    }
  }, [content, editor])

  const handleImageUpload = useCallback(() => {
    const input = document.createElement('input')
    input.type = 'file'
    input.accept = 'image/*'
    input.onchange = async (e) => {
      const file = (e.target as HTMLInputElement).files?.[0]
      if (!file) return

      // 转换为 base64
      const reader = new FileReader()
      reader.onload = (e) => {
        const src = e.target?.result as string
        editor?.chain().focus().setImage({ src }).run()
      }
      reader.readAsDataURL(file)
    }
    input.click()
  }, [editor])

  const insertTable = useCallback(() => {
    editor?.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()
  }, [editor])

  if (!editor) {
    return null
  }

  return (
    <div className="border rounded-md overflow-hidden">
      {/* 工具栏 */}
      {editable && (
        <div className="border-b bg-muted/50 p-2 flex flex-wrap gap-1">
          {/* 撤销/重做 */}
          <Button
            size="sm"
            variant="ghost"
            onClick={() => editor.chain().focus().undo().run()}
            disabled={!editor.can().undo()}
          >
            <Undo className="h-4 w-4" />
          </Button>
          <Button
            size="sm"
            variant="ghost"
            onClick={() => editor.chain().focus().redo().run()}
            disabled={!editor.can().redo()}
          >
            <Redo className="h-4 w-4" />
          </Button>

          <Separator orientation="vertical" className="mx-1 h-6" />

          {/* 标题 */}
          <Toggle
            size="sm"
            pressed={editor.isActive('heading', { level: 1 })}
            onPressedChange={() => editor.chain().focus().toggleHeading({ level: 1 }).run()}
          >
            <Heading1 className="h-4 w-4" />
          </Toggle>
          <Toggle
            size="sm"
            pressed={editor.isActive('heading', { level: 2 })}
            onPressedChange={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
          >
            <Heading2 className="h-4 w-4" />
          </Toggle>

          <Separator orientation="vertical" className="mx-1 h-6" />

          {/* 格式 */}
          <Toggle
            size="sm"
            pressed={editor.isActive('bold')}
            onPressedChange={() => editor.chain().focus().toggleBold().run()}
          >
            <Bold className="h-4 w-4" />
          </Toggle>
          <Toggle
            size="sm"
            pressed={editor.isActive('italic')}
            onPressedChange={() => editor.chain().focus().toggleItalic().run()}
          >
            <Italic className="h-4 w-4" />
          </Toggle>
          <Toggle
            size="sm"
            pressed={editor.isActive('underline')}
            onPressedChange={() => editor.chain().focus().toggleUnderline().run()}
          >
            <UnderlineIcon className="h-4 w-4" />
          </Toggle>

          <Separator orientation="vertical" className="mx-1 h-6" />

          {/* 列表 */}
          <Toggle
            size="sm"
            pressed={editor.isActive('bulletList')}
            onPressedChange={() => editor.chain().focus().toggleBulletList().run()}
          >
            <List className="h-4 w-4" />
          </Toggle>
          <Toggle
            size="sm"
            pressed={editor.isActive('orderedList')}
            onPressedChange={() => editor.chain().focus().toggleOrderedList().run()}
          >
            <ListOrdered className="h-4 w-4" />
          </Toggle>

          <Separator orientation="vertical" className="mx-1 h-6" />

          {/* 插入 */}
          <Button
            size="sm"
            variant="ghost"
            onClick={handleImageUpload}
          >
            <ImageIcon className="h-4 w-4" />
          </Button>
          <Button
            size="sm"
            variant="ghost"
            onClick={insertTable}
          >
            <TableIcon className="h-4 w-4" />
          </Button>
        </div>
      )}

      {/* 编辑区 */}
      <EditorContent editor={editor} style={{ minHeight }} />

      {/* 字数统计 */}
      <div className="border-t bg-muted/30 px-3 py-1 text-xs text-muted-foreground flex justify-between">
        <span>{editor.storage.characterCount.words()} 词</span>
        <span>{editor.storage.characterCount.characters()} 字符</span>
      </div>
    </div>
  )
}
```

---

## 只读模式组件

```tsx
// frontend/src/components/ui/rich-text-viewer.tsx
import { generateHTML } from '@tiptap/html'
import StarterKit from '@tiptap/starter-kit'
import Image from '@tiptap/extension-image'
import Table from '@tiptap/extension-table'
import Underline from '@tiptap/extension-underline'

interface RichTextViewerProps {
  content: string
  className?: string
}

export function RichTextViewer({ content, className = '' }: RichTextViewerProps) {
  const html = generateHTML(
    JSON.parse(content || '{"type":"doc","content":[]}'),
    [
      StarterKit,
      Image,
      Table,
      Underline,
    ]
  )

  return (
    <div
      className={`prose prose-sm max-w-none ${className}`}
      dangerouslySetInnerHTML={{ __html: html }}
    />
  )
}
```

---

## 扩展：字数统计

```typescript
// frontend/src/lib/tiptap/extensions/character-count.ts
import { Extension } from '@tiptap/core'

export const CharacterCount = Extension.create({
  name: 'characterCount',

  addStorage() {
    return {
      characters() {
        return this.editor.text.length
      },
      words() {
        return this.editor.text.split(/\s+/).filter(word => word !== '').length
      },
    }
  },
})
```

---

## 扩展：代码高亮

```bash
npm install @tiptap/extension-code-block-lowlight lowlight
```

```typescript
// frontend/src/lib/tiptap/extensions/code-block.ts
import CodeBlockLowlight from '@tiptap/extension-code-block-lowlight'
import { common, createLowlight } from 'lowlight'

const lowlight = createLowlight(common)

export const CodeBlock = CodeBlockLowlight.configure({
  lowlight,
  defaultLanguage: null,
})
```

---

## 使用示例

```tsx
// 使用示例
import { useState } from 'react'
import { RichTextEditor } from '@/components/ui/rich-text-editor'
import { RichTextViewer } from '@/components/ui/rich-text-viewer'

export function SceneDescriptionForm() {
  const [description, setDescription] = useState('<p>场景描述...</p>')
  const [isPreview, setIsPreview] = useState(false)

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <label className="font-medium">场景描述</label>
        <Button
          size="sm"
          variant="outline"
          onClick={() => setIsPreview(!isPreview)}
        >
          {isPreview ? '编辑' : '预览'}
        </Button>
      </div>

      {isPreview ? (
        <RichTextViewer content={description} className="border rounded p-4" />
      ) : (
        <RichTextEditor
          content={description}
          onChange={setDescription}
          placeholder="输入场景描述..."
          minHeight="300px"
        />
      )}
    </div>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `frontend/src/components/ui/rich-text-editor.tsx` | 创建 | 富文本编辑器组件 |
| `frontend/src/components/ui/rich-text-viewer.tsx` | 创建 | 富文本查看器组件 |
| `frontend/src/lib/tiptap/extensions/character-count.ts` | 创建 | 字数统计扩展 |
| `frontend/src/lib/tiptap/extensions/code-block.ts` | 创建 | 代码块扩展 |

---

## 验收标准

- [ ] 基础编辑功能完整
- [ ] 工具栏按钮有效
- [ ] 图片上传正常
- [ ] 表格插入可用
- [ ] 预览模式正确
- [ ] 性能流畅

---

## 参考文档

- Tiptap 官方文档
- ProseMirror 文档

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
