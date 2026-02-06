# M5-014: 实现自定义表情

**任务ID**: M5-014
**标题**: 实现自定义表情/贴纸系统
**类型**: fullstack (全栈开发)
**预估工时**: 6h
**依赖**: M1 完成

---

## 任务描述

实现一个自定义表情/贴纸系统，允许玩家和 KP 在游戏交流中使用自定义表情增强表达。支持 Campaign 级别的表情包、表情搜索、快捷输入等功能。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M5-014-01 | 设计表情数据结构 | 表情包、分类、权限 | 45min |
| M5-014-02 | 实现后端表情管理 API | 上传、存储、检索 | 1.5h |
| M5-014-03 | 实现表情图片存储 | 文件上传、CDN | 1h |
| M5-014-04 | 实现前端表情选择器 | 搜索、分类、预览 | 2h |
| M5-014-05 | 实现表情快捷输入 | 快捷键、自动补全 | 1h |

---

## 完整后端代码示例 (Python + Agno)

### 数据模型

```python
# backend/app/models/emojis.py
from datetime import datetime
from sqlalchemy import Column, String, JSON, DateTime, Boolean, Integer, ForeignKey, Text
from sqlalchemy.dialects.postgresql import UUID
import uuid

from app.db.base_class import Base


class EmojiPack(Base):
    """表情包"""
    __tablename__ = "emoji_packs"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    name = Column(String(100), nullable=False)
    description = Column(Text, nullable=True)

    # 归属
    campaign_id = Column(UUID(as_uuid=True), ForeignKey("campaigns.id"), nullable=True)
    account_id = Column(UUID(as_uuid=True), ForeignKey("accounts.id"), nullable=True)

    # 是否为系统预设
    is_system = Column(Boolean, default=False)

    # 分类标签
    tags = Column(JSON, default=list)

    # 排序顺序
    order = Column(Integer, default=0)

    created_at = Column(DateTime, default=datetime.utcnow)


class Emoji(Base):
    """表情/贴纸"""
    __tablename__ = "emojis"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    pack_id = Column(UUID(as_uuid=True), ForeignKey("emoji_packs.id"), nullable=False)

    # 表情信息
    name = Column(String(100), nullable=False)
    shortcode = Column(String(50), nullable=False, unique=True)  # :shortcode:
    keywords = Column(JSON, default=list)  # 搜索关键词

    # 图片信息
    image_url = Column(String(500), nullable=False)
    thumbnail_url = Column(String(500), nullable=True)
    image_type = Column(String(20), nullable=False)  # png, gif, webp

    # 尺寸
    width = Column(Integer, nullable=True)
    height = Column(Integer, nullable=True)

    # 分类
    category = Column(String(50), nullable=True)

    # 是否启用
    is_active = Column(Boolean, default=True)

    # 使用统计
    usage_count = Column(Integer, default=0)

    created_at = Column(DateTime, default=datetime.utcnow)
```

### 服务层

```python
# backend/app/services/emoji_service.py
from typing import List, Optional, Dict, Any
from sqlalchemy.orm import Session
from fastapi import UploadFile, HTTPException

from app.models.emojis import EmojiPack, Emoji
import uuid
import os
from pathlib import Path


class EmojiService:
    """表情服务"""

    UPLOAD_DIR = "uploads/emojis"

    @staticmethod
    async def upload_emoji_image(file: UploadFile) -> Dict[str, str]:
        """上传表情图片"""
        # 生成唯一文件名
        ext = file.filename.split(".")[-1]
        filename = f"{uuid.uuid4()}.{ext}"
        file_path = Path(EmojiService.UPLOAD_DIR) / filename

        # 确保目录存在
        file_path.parent.mkdir(parents=True, exist_ok=True)

        # 保存文件
        with file_path.open("wb") as f:
            content = await file.read()
            f.write(content)

        # 返回 URL
        return {
            "image_url": f"/static/emojis/{filename}",
            "thumbnail_url": f"/static/emojis/thumbs/{filename}"
        }

    @staticmethod
    def create_emoji_pack(
        db: Session,
        name: str,
        campaign_id: Optional[str] = None,
        account_id: Optional[str] = None,
        description: Optional[str] = None,
        tags: Optional[List[str]] = None,
        is_system: bool = False
    ) -> EmojiPack:
        """创建表情包"""
        pack = EmojiPack(
            name=name,
            description=description,
            campaign_id=campaign_id,
            account_id=account_id,
            tags=tags or [],
            is_system=is_system
        )

        db.add(pack)
        db.commit()
        db.refresh(pack)

        return pack

    @staticmethod
    def create_emoji(
        db: Session,
        pack_id: str,
        name: str,
        shortcode: str,
        image_url: str,
        keywords: Optional[List[str]] = None,
        category: Optional[str] = None,
        image_type: str = "png",
        thumbnail_url: Optional[str] = None,
        width: Optional[int] = None,
        height: Optional[int] = None
    ) -> Emoji:
        """创建表情"""
        # 检查 shortcode 唯一性
        existing = db.query(Emoji).filter(Emoji.shortcode == shortcode).first()
        if existing:
            raise HTTPException(status_code=400, detail="Shortcode already exists")

        emoji = Emoji(
            pack_id=pack_id,
            name=name,
            shortcode=shortcode.lstrip(":").rstrip(":"),  # 移除冒号
            keywords=keywords or [],
            image_url=image_url,
            thumbnail_url=thumbnail_url,
            image_type=image_type,
            category=category,
            width=width,
            height=height
        )

        db.add(emoji)
        db.commit()
        db.refresh(emoji)

        return emoji

    @staticmethod
    def search_emojis(
        db: Session,
        query: str,
        campaign_id: Optional[str] = None,
        limit: int = 20
    ) -> List[Emoji]:
        """搜索表情"""
        emojis = db.query(Emoji).filter(Emoji.is_active == True)

        # 搜索名称、shortcode、关键词
        search_pattern = f"%{query}%"
        emojis = emojis.filter(
            (Emoji.name.ilike(search_pattern)) |
            (Emoji.shortcode.ilike(search_pattern)) |
            (Emoji.keywords.any(search_pattern))  # 假设 JSON 支持
        )

        return emojis.limit(limit).all()

    @staticmethod
    def get_campaign_emojis(
        db: Session,
        campaign_id: str
    ) -> List[Emoji]:
        """获取 Campaign 的所有表情"""
        # 获取系统表情 + Campaign 表情
        system_packs = db.query(EmojiPack).filter(EmojiPack.is_system == True).all()
        campaign_packs = db.query(EmojiPack).filter(
            EmojiPack.campaign_id == campaign_id
        ).all()

        pack_ids = [p.id for p in system_packs + campaign_packs]

        return db.query(Emoji).filter(
            Emoji.pack_id.in_(pack_ids),
            Emoji.is_active == True
        ).all()

    @staticmethod
    def increment_usage(db: Session, emoji_id: str):
        """增加使用次数"""
        emoji = db.query(Emoji).filter(Emoji.id == emoji_id).first()
        if emoji:
            emoji.usage_count += 1
            db.commit()
```

### API 路由

```python
# backend/app/api/emojis.py
from fastapi import APIRouter, Depends, UploadFile, File, HTTPException, Form
from sqlalchemy.orm import Session

from app.api.deps import get_db, get_current_active_user
from app.services.emoji_service import EmojiService

router = APIRouter()


@router.post("/upload")
async def upload_emoji(
    file: UploadFile = File(...),
    current_user = Depends(get_current_active_user)
):
    """上传表情图片"""
    return await EmojiService.upload_emoji_image(file)


@router.post("/packs")
def create_pack(
    name: str = Form(...),
    description: str = Form(None),
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """创建表情包"""
    return EmojiService.create_emoji_pack(
        db,
        name,
        account_id=current_user.id,
        description=description
    )


@router.post("/packs/{pack_id}/emojis")
def create_emoji(
    pack_id: str,
    name: str = Form(...),
    shortcode: str = Form(...),
    image_url: str = Form(...),
    keywords: str = Form(None),
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """添加表情到表情包"""
    import json
    kw_list = json.loads(keywords) if keywords else []

    return EmojiService.create_emoji(
        db,
        pack_id,
        name,
        shortcode,
        image_url,
        keywords=kw_list
    )


@router.get("/search")
def search_emojis(
    q: str,
    campaign_id: str = None,
    limit: int = 20,
    db: Session = Depends(get_db)
):
    """搜索表情"""
    return EmojiService.search_emojis(db, q, campaign_id, limit)


@router.get("/campaign/{campaign_id}")
def get_campaign_emojis(
    campaign_id: str,
    db: Session = Depends(get_db)
):
    """获取 Campaign 表情"""
    return EmojiService.get_campaign_emojis(db, campaign_id)


@router.post("/{emoji_id}/use")
def use_emoji(
    emoji_id: str,
    db: Session = Depends(get_db)
):
    """记录表情使用"""
    EmojiService.increment_usage(db, emoji_id)
    return {"message": "Usage recorded"}
```

---

## 完整前端代码示例 (TypeScript + React + shadcn/ui)

### 类型定义

```typescript
// frontend/src/types/emojis.ts
export interface EmojiPack {
  id: string;
  name: string;
  description?: string;
  campaign_id?: string;
  account_id?: string;
  is_system: boolean;
  tags: string[];
  order: number;
  created_at: string;
}

export interface Emoji {
  id: string;
  pack_id: string;
  name: string;
  shortcode: string;
  keywords: string[];
  image_url: string;
  thumbnail_url?: string;
  image_type: string;
  width?: number;
  height?: number;
  category?: string;
  is_active: boolean;
  usage_count: number;
  created_at: string;
}
```

### 表情选择器组件

```tsx
// frontend/src/components/emojis/EmojiPicker.tsx
import React, { useState, useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import { Smile, Search } from "lucide-react";

import { Emoji } from "@/types/emojis";

interface EmojiPickerProps {
  campaignId: string;
  onInsert: (emoji: Emoji) => void;
  children: React.ReactNode;
}

export function EmojiPicker({ campaignId, onInsert, children }: EmojiPickerProps) {
  const [open, setOpen] = useState(false);
  const [emojis, setEmojis] = useState<Emoji[]>([]);
  const [filtered, setFiltered] = useState<Emoji[]>([]);
  const [search, setSearch] = useState("");
  const [selectedCategory, setSelectedCategory] = useState("all");

  useEffect(() => {
    loadEmojis();
  }, [campaignId]);

  useEffect(() => {
    if (search) {
      const query = search.toLowerCase();
      setFiltered(
        emojis.filter(
          e =>
            e.name.toLowerCase().includes(query) ||
            e.shortcode.toLowerCase().includes(query) ||
            e.keywords.some(k => k.toLowerCase().includes(query))
        )
      );
    } else {
      setFiltered(emojis);
    }
  }, [search, emojis]);

  const loadEmojis = async () => {
    const res = await fetch(`/api/emojis/campaign/${campaignId}`);
    const data = await res.json();
    setEmojis(data);
    setFiltered(data);
  };

  const categories = ["all", ...new Set(emojis.map(e => e.category || "uncategorized"))];

  const displayedEmojis = filtered.filter(e =>
    selectedCategory === "all" || (e.category || "uncategorized") === selectedCategory
  );

  const handleInsert = (emoji: Emoji) => {
    onInsert(emoji);
    setOpen(false);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent className="w-80 p-0" align="start">
        <div className="space-y-2 p-2">
          {/* 搜索框 */}
          <div className="relative">
            <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="搜索表情..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-8"
            />
          </div>

          {/* 分类标签 */}
          <Tabs value={selectedCategory} onValueChange={setSelectedCategory}>
            <TabsList className="w-full justify-start overflow-x-auto">
              <TabsTrigger value="all">全部</TabsTrigger>
              {categories.slice(1).map(cat => (
                <TabsTrigger key={cat} value={cat} className="capitalize">
                  {cat}
                </TabsTrigger>
              ))}
            </TabsList>

            <TabsContent value={selectedCategory} className="mt-2">
              <ScrollArea className="h-64">
                <div className="grid grid-cols-6 gap-1 p-2">
                  {displayedEmojis.map(emoji => (
                    <button
                      key={emoji.id}
                      onClick={() => handleInsert(emoji)}
                      className="aspect-square hover:bg-accent rounded flex items-center justify-center transition-colors"
                      title={`${emoji.name} :${emoji.shortcode}:`}
                    >
                      <img
                        src={emoji.thumbnail_url || emoji.image_url}
                        alt={emoji.name}
                        className="w-10 h-10 object-cover"
                      />
                    </button>
                  ))}
                </div>
                {displayedEmojis.length === 0 && (
                  <div className="text-center text-muted-foreground py-8">
                    {search ? "没有找到匹配的表情" : "暂无表情"}
                  </div>
                )}
              </ScrollArea>
            </TabsContent>
          </Tabs>

          {/* 最近使用 */}
          {search === "" && (
            <div className="border-t pt-2">
              <p className="text-xs text-muted-foreground px-2 mb-2">最近使用</p>
              <div className="flex gap-1 overflow-x-auto px-2">
                {emojis
                  .sort((a, b) => b.usage_count - a.usage_count)
                  .slice(0, 12)
                  .map(emoji => (
                    <button
                      key={emoji.id}
                      onClick={() => handleInsert(emoji)}
                      className="flex-shrink-0 w-8 h-8 hover:bg-accent rounded flex items-center justify-center"
                    >
                      <img
                        src={emoji.thumbnail_url || emoji.image_url}
                        alt={emoji.name}
                        className="w-6 h-6 object-cover"
                      />
                    </button>
                  ))}
              </div>
            </div>
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
}
```

### 消息框中的表情支持

```tsx
// frontend/src/components/game/MessageInput.tsx
import React, { useState, useRef, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { EmojiPicker } from "@/components/emojis/EmojiPicker";
import { Send, Smile, AtSlash } from "lucide-react";

import { Emoji } from "@/types/emojis";

interface MessageInputProps {
  campaignId: string;
  onSend: (content: string) => void;
  disabled?: boolean;
}

export function MessageInput({ campaignId, onSend, disabled }: MessageInputProps) {
  const [content, setContent] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // 处理表情插入
  const handleInsertEmoji = (emoji: Emoji) => {
    const insert = `:${emoji.shortcode}:`;
    const textarea = textareaRef.current;
    if (!textarea) return;

    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const before = content.substring(0, start);
    const after = content.substring(end);

    setContent(before + insert + after);

    // 重新定位光标
    setTimeout(() => {
      textarea.selectionStart = textarea.selectionEnd = start + insert.length;
      textarea.focus();
    }, 0);
  };

  // 解析表情
  const parseEmojis = (text: string): string => {
    // 将 :shortcode: 替换为图片标签
    return text.replace(/:([a-zA-Z0-9_+-]+):/g, (match, shortcode) => {
      return `<emoji data-shortcode="${shortcode}">${match}</emoji>`;
    });
  };

  const handleSend = () => {
    if (content.trim()) {
      onSend(parseEmojis(content));
      setContent("");
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="flex items-end gap-2">
      <div className="flex-1 relative">
        <Textarea
          ref={textareaRef}
          value={content}
          onChange={(e) => setContent(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="输入消息... (使用 :shortcode: 输入表情)"
          disabled={disabled}
          rows={2}
          className="resize-none pr-20"
        />

        {/* 表情按钮 */}
        <div className="absolute right-2 bottom-2 flex gap-1">
          <EmojiPicker campaignId={campaignId} onInsert={handleInsertEmoji}>
            <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
              <Smile className="h-4 w-4" />
            </Button>
          </EmojiPicker>
        </div>
      </div>

      <Button onClick={handleSend} disabled={disabled || !content.trim()}>
        <Send className="h-4 w-4" />
      </Button>
    </div>
  );
}
```

### 表情渲染组件

```tsx
// frontend/src/components/emojis/EmojiRenderer.tsx
import React, { useMemo } from "react";
import Image from "next/image";

interface EmojiRendererProps {
  content: string;
  emojis: Emoji[];
}

export function EmojiRenderer({ content, emojis }: EmojiRendererProps) {
  // 构建 shortcode 到 emoji 的映射
  const emojiMap = useMemo(() => {
    const map = new Map<string, Emoji>();
    emojis.forEach(e => map.set(e.shortcode, e));
    return map;
  }, [emojis]);

  // 解析内容
  const rendered = useMemo(() => {
    const parts: React.ReactNode[] = [];
    let lastIndex = 0;
    const regex = /:([a-zA-Z0-9_+-]+):/g;

    let match;
    while ((match = regex.exec(content)) !== null) {
      // 添加表情前的文本
      if (match.index > lastIndex) {
        parts.push(
          <span key={`text-${lastIndex}`}>
            {content.substring(lastIndex, match.index)}
          </span>
        );
      }

      // 添加表情
      const shortcode = match[1];
      const emoji = emojiMap.get(shortcode);
      if (emoji) {
        parts.push(
          <img
            key={`emoji-${match.index}`}
            src={emoji.image_url}
            alt={emoji.name}
            title={`:${shortcode}:`}
            className="inline-block w-6 h-6 align-middle"
          />
        );
      } else {
        // 未找到表情，保留原文本
        parts.push(<span key={`unknown-${match.index}`}>{match[0]}</span>);
      }

      lastIndex = match.index + match[0].length;
    }

    // 添加剩余文本
    if (lastIndex < content.length) {
      parts.push(
        <span key={`text-${lastIndex}`}>{content.substring(lastIndex)}</span>
      );
    }

    return parts;
  }, [content, emojiMap]);

  return <span>{rendered}</span>;
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `backend/app/models/emojis.py` | 创建 | 表情数据模型 |
| `backend/app/services/emoji_service.py` | 创建 | 表情服务 |
| `backend/app/api/emojis.py` | 创建 | 表情 API 路由 |
| `backend/app/schemas/emojis.py` | 创建 | Pydantic 模型 |
| `backend/app/db/migrations/versions/xxx_create_emojis.py` | 创建 | 数据库迁移 |
| `frontend/src/types/emojis.ts` | 创建 | 类型定义 |
| `frontend/src/components/emojis/EmojiPicker.tsx` | 创建 | 表情选择器 |
| `frontend/src/components/emojis/EmojiRenderer.tsx` | 创建 | 表情渲染组件 |
| `frontend/src/components/game/MessageInput.tsx` | 修改 | 添加表情支持 |
| `frontend/src/components/game/MessageBubble.tsx` | 修改 | 添加表情渲染 |

---

## 验收标准

- [ ] 用户可以上传自定义表情
- [ ] 表情按分类组织
- [ ] 支持关键词搜索表情
- [ ] 表情可以在消息中正确显示
- [ ] 表情快捷码（:shortcode:）可以自动转换为图片
- [ ] 支持最近使用表情的快速访问
- [ ] 表情使用次数正确统计

---

## 参考文档

- 表情包系统设计最佳实践
- 图片文件存储与 CDN
- React 文本解析与渲染

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
