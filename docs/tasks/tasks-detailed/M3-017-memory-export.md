# M3-017: 实现记忆导出功能

**任务ID**: M3-017
**标题**: 实现记忆导出功能
**类型**: backend + frontend (全栈开发)
**预估工时**: 4h
**依赖**: M3-001, M3-014

---

## 任务描述

实现游戏记忆的导出功能，允许用户将游戏会话、事件日志、摘要等内容导出为多种格式，便于保存和分享。支持：
- 导出为 Markdown 文档
- 导出为 PDF
- 导出为 JSON 数据
- 导出为 HTML 网页
- 批量导出和打包下载

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M3-017-01 | 设计导出配置 | 导出选项和格式定义 | 20min |
| M3-017-02 | 实现 Markdown 导出 | 生成 Markdown 格式 | 45min |
| M3-017-03 | 实现 PDF 导出 | 生成 PDF 文件 | 1h |
| M3-017-04 | 实现 JSON 导出 | 导出结构化数据 | 30min |
| M3-017-05 | 实现 HTML 导出 | 生成可浏览网页 | 45min |
| M3-017-06 | 实现批量导出 | 打包多个会话 | 30min |
| M3-017-07 | 实现导出 UI 组件 | 导出选项界面 | 30min |
| M3-017-08 | 编写导出测试 | 测试覆盖 | 20min |

---

## 后端代码示例

### 导出服务

```python
# app/services/export.py
from typing import List, Dict, Any, Optional
from datetime import datetime
import json
import markdown
import io
from reportlab.lib.pagesizes import letter, A4
from reportlab.lib.styles import getSampleStyleSheet, ParagraphStyle
from reportlab.lib.units import inch
from reportlab.platypus import SimpleDocTemplate, Paragraph, Spacer, PageBreak, Table, TableStyle
from reportlab.lib.enums import TA_CENTER, TA_LEFT
from reportlab.lib import colors

from sqlalchemy.orm import Session
from app.services.summary import SummaryService
from app.core.logger import EventLogger

class ExportService:
    """记忆导出服务"""

    def __init__(self, db: Session):
        self.db = db
        self.summary_service = SummaryService(db)
        self.logger = EventLogger()

    async def export_session_markdown(
        self,
        session_id: str,
        include_events: bool = True,
        include_summary: bool = True,
        include_statistics: bool = True,
    ) -> str:
        """导出会话为 Markdown

        Args:
            session_id: 会话 ID
            include_events: 是否包含事件
            include_summary: 是否包含摘要
            include_statistics: 是否包含统计

        Returns:
            Markdown 文本
        """
        # 获取会话信息
        session_info = await self._get_session_info(session_id)

        # 构建 Markdown
        md_lines = []

        # 标题
        md_lines.append(f"# {session_info.get('title', '游戏会话')}\n")
        md_lines.append(f"**会话 ID**: {session_id}\n")
        md_lines.append(f"**开始时间**: {session_info.get('started_at', '')}\n")
        if session_info.get('ended_at'):
            md_lines.append(f"**结束时间**: {session_info.get('ended_at')}\n")
        md_lines.append("\n---\n")

        # 摘要
        if include_summary:
            summary = await self.summary_service.generate_summary(
                campaign_id=session_info['campaign_id'],
                start_time=datetime.fromisoformat(session_info['started_at']),
                end_time=datetime.fromisoformat(session_info['ended_at']) if session_info.get('ended_at') else None,
            )

            md_lines.append("## 叙事摘要\n")
            md_lines.append(f"{summary.get('summary', '')}\n\n")

            if summary.get('key_events'):
                md_lines.append("### 关键事件\n")
                for event in summary['key_events']:
                    md_lines.append(f"- {event}\n")
                md_lines.append("\n")

        # 事件日志
        if include_events:
            events = await self.logger.get_events(
                campaign_id=session_info['campaign_id'],
                start_time=datetime.fromisoformat(session_info['started_at']),
                end_time=datetime.fromisoformat(session_info['ended_at']) if session_info.get('ended_at') else None,
            )

            md_lines.append("## 事件日志\n\n")
            for event in events:
                timestamp = event.timestamp.strftime("%H:%M:%S")
                md_lines.append(f"### [{timestamp}] {event.description}\n")
                if event.data:
                    md_lines.append(f"```json\n{json.dumps(event.data, ensure_ascii=False, indent=2)}\n```\n")
                md_lines.append("\n")

        # 统计信息
        if include_statistics:
            stats = await self._calculate_session_stats(session_id)
            md_lines.append("## 统计信息\n\n")
            md_lines.append(f"- **消息数**: {stats.get('message_count', 0)}\n")
            md_lines.append(f"- **检定数**: {stats.get('roll_count', 0)}\n")
            md_lines.append(f"- **战斗数**: {stats.get('combat_count', 0)}\n")
            md_lines.append(f"- **SAN检定**: {stats.get('san_check_count', 0)}\n")
            md_lines.append("\n")

        return "".join(md_lines)

    async def export_session_pdf(
        self,
        session_id: str,
        include_events: bool = True,
        include_summary: bool = True,
    ) -> bytes:
        """导出会话为 PDF

        Args:
            session_id: 会话 ID
            include_events: 是否包含事件
            include_summary: 是否包含摘要

        Returns:
            PDF 字节流
        """
        # 获取会话信息
        session_info = await self._get_session_info(session_id)

        # 创建 PDF
        buffer = io.BytesIO()
        doc = SimpleDocTemplate(buffer, pagesize=A4)

        # 样式
        styles = getSampleStyleSheet()
        title_style = ParagraphStyle(
            'CustomTitle',
            parent=styles['Heading1'],
            fontSize=18,
            textColor=colors.HexColor('#1a1a1a'),
            spaceAfter=30,
        )
        heading_style = ParagraphStyle(
            'CustomHeading',
            parent=styles['Heading2'],
            fontSize=14,
            textColor=colors.HexColor('#333333'),
            spaceAfter=12,
        )

        story = []

        # 标题
        story.append(Paragraph(session_info.get('title', '游戏会话'), title_style))
        story.append(Spacer(1, 0.2 * inch))

        # 会话信息
        info_data = [
            ['会话 ID', session_id],
            ['开始时间', session_info.get('started_at', '')],
            ['结束时间', session_info.get('ended_at', '进行中')],
        ]
        info_table = Table(info_data, colWidths=[1.5 * inch, 4 * inch])
        info_table.setStyle(TableStyle([
            ('BACKGROUND', (0, 0), (0, -1), colors.grey),
            ('TEXTCOLOR', (0, 0), (0, -1), colors.whitesmoke),
            ('ALIGN', (0, 0), (-1, -1), 'LEFT'),
            ('FONTNAME', (0, 0), (-1, -1), 'Helvetica'),
            ('FONTSIZE', (0, 0), (-1, -1), 10),
            ('BOTTOMPADDING', (0, 0), (-1, -1), 12),
            ('TOPPADDING', (0, 0), (-1, -1), 12),
        ]))
        story.append(info_table)
        story.append(Spacer(1, 0.3 * inch))

        # 摘要
        if include_summary:
            story.append(Paragraph('叙事摘要', heading_style))
            summary = await self.summary_service.generate_summary(
                campaign_id=session_info['campaign_id'],
                start_time=datetime.fromisoformat(session_info['started_at']),
                end_time=datetime.fromisoformat(session_info['ended_at']) if session_info.get('ended_at') else None,
            )

            for para in summary.get('summary', '').split('\n\n'):
                story.append(Paragraph(para, styles['Normal']))
                story.append(Spacer(1, 0.1 * inch))

            story.append(Spacer(1, 0.2 * inch))

        # 事件
        if include_events:
            story.append(Paragraph('事件日志', heading_style))
            events = await self.logger.get_events(
                campaign_id=session_info['campaign_id'],
                start_time=datetime.fromisoformat(session_info['started_at']),
                end_time=datetime.fromisoformat(session_info['ended_at']) if session_info.get('ended_at') else None,
            )

            for event in events[:50]:  # 限制事件数量
                timestamp = event.timestamp.strftime("%H:%M:%S")
                story.append(Paragraph(f"[{timestamp}] {event.description}", styles['Normal']))
                story.append(Spacer(1, 0.05 * inch))

        # 构建 PDF
        doc.build(story)
        pdf_bytes = buffer.getvalue()
        buffer.close()

        return pdf_bytes

    async def export_session_json(
        self,
        session_id: str,
        include_events: bool = True,
        include_summary: bool = True,
    ) -> Dict[str, Any]:
        """导出会话为 JSON

        Args:
            session_id: 会话 ID
            include_events: 是否包含事件
            include_summary: 是否包含摘要

        Returns:
            JSON 数据
        """
        session_info = await self._get_session_info(session_id)

        export_data = {
            "version": "1.0",
            "exported_at": datetime.utcnow().isoformat(),
            "session": {
                "id": session_id,
                "title": session_info.get('title'),
                "started_at": session_info.get('started_at'),
                "ended_at": session_info.get('ended_at'),
                "campaign_id": session_info.get('campaign_id'),
            },
        }

        # 摘要
        if include_summary:
            summary = await self.summary_service.generate_summary(
                campaign_id=session_info['campaign_id'],
                start_time=datetime.fromisoformat(session_info['started_at']),
                end_time=datetime.fromisoformat(session_info['ended_at']) if session_info.get('ended_at') else None,
            )
            export_data["summary"] = summary

        # 事件
        if include_events:
            events = await self.logger.get_events(
                campaign_id=session_info['campaign_id'],
                start_time=datetime.fromisoformat(session_info['started_at']),
                end_time=datetime.fromisoformat(session_info['ended_at']) if session_info.get('ended_at') else None,
            )

            export_data["events"] = [
                {
                    "id": event.id,
                    "timestamp": event.timestamp.isoformat(),
                    "type": event.type,
                    "description": event.description,
                    "data": event.data,
                    "user_id": event.user_id,
                    "character_id": event.character_id,
                }
                for event in events
            ]

        # 统计
        stats = await self._calculate_session_stats(session_id)
        export_data["statistics"] = stats

        return export_data

    async def export_session_html(
        self,
        session_id: str,
        include_events: bool = True,
        include_summary: bool = True,
    ) -> str:
        """导出会话为 HTML

        Args:
            session_id: 会话 ID
            include_events: 是否包含事件
            include_summary: 是否包含摘要

        Returns:
            HTML 文本
        """
        # 先获取 Markdown
        md_content = await self.export_session_markdown(
            session_id=session_id,
            include_events=include_events,
            include_summary=include_summary,
        )

        # 转换为 HTML
        html_body = markdown.markdown(md_content)

        # 添加 HTML 模板
        html_template = f"""<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>游戏会话 - {session_id}</title>
    <style>
        body {{
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            background: #f5f5f5;
        }}
        .container {{
            background: white;
            padding: 40px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }}
        h1 {{
            color: #1a1a1a;
            border-bottom: 2px solid #e0e0e0;
            padding-bottom: 10px;
        }}
        h2 {{
            color: #333;
            margin-top: 30px;
        }}
        h3 {{
            color: #666;
        }}
        code {{
            background: #f5f5f5;
            padding: 2px 6px;
            border-radius: 3px;
            font-family: 'Courier New', monospace;
        }}
        pre {{
            background: #f5f5f5;
            padding: 15px;
            border-radius: 5px;
            overflow-x: auto;
        }}
        blockquote {{
            border-left: 4px solid #ddd;
            padding-left: 20px;
            color: #666;
            margin: 20px 0;
        }}
        table {{
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
        }}
        th, td {{
            border: 1px solid #ddd;
            padding: 12px;
            text-align: left;
        }}
        th {{
            background: #f5f5f5;
        }}
        .event {{
            margin: 15px 0;
            padding: 15px;
            background: #f9f9f9;
            border-left: 3px solid #ddd;
            border-radius: 3px;
        }}
        .event-time {{
            color: #999;
            font-size: 0.9em;
        }}
    </style>
</head>
<body>
    <div class="container">
        {html_body}
    </div>
</body>
</html>"""

        return html_template

    async def batch_export(
        self,
        session_ids: List[str],
        format: str = "markdown",
    ) -> bytes:
        """批量导出

        Args:
            session_ids: 会话 ID 列表
            format: 导出格式

        Returns:
            打包的 ZIP 文件字节流
        """
        import zipfile

        buffer = io.BytesIO()
        with zipfile.ZipFile(buffer, 'w', zipfile.ZIP_DEFLATED) as zip_file:
            for session_id in session_ids:
                if format == "markdown":
                    content = await self.export_session_markdown(session_id)
                    filename = f"{session_id}.md"
                elif format == "pdf":
                    content = await self.export_session_pdf(session_id)
                    filename = f"{session_id}.pdf"
                elif format == "json":
                    content_data = await self.export_session_json(session_id)
                    content = json.dumps(content_data, ensure_ascii=False, indent=2)
                    filename = f"{session_id}.json"
                elif format == "html":
                    content = await self.export_session_html(session_id)
                    filename = f"{session_id}.html"
                else:
                    continue

                zip_file.writestr(filename, content)

        buffer.seek(0)
        return buffer.read()

    async def _get_session_info(self, session_id: str) -> Dict[str, Any]:
        """获取会话信息"""
        # 简化实现，实际应该从数据库加载
        return {
            "id": session_id,
            "title": "游戏会话",
            "started_at": datetime.utcnow().isoformat(),
            "ended_at": None,
            "campaign_id": "default",
        }

    async def _calculate_session_stats(self, session_id: str) -> Dict[str, int]:
        """计算会话统计"""
        # 简化实现
        return {
            "message_count": 0,
            "roll_count": 0,
            "combat_count": 0,
            "san_check_count": 0,
        }
```

### 导出 API

```python
# app/api/export.py
from fastapi import APIRouter, Depends, HTTPException
from fastapi.responses import StreamingResponse
from sqlalchemy.orm import Session
from pydantic import BaseModel
from typing import List, Optional

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.export import ExportService

router = APIRouter(prefix="/export", tags=["export"])

class ExportRequest(BaseModel):
    session_id: str
    format: str = "markdown"  # markdown/pdf/json/html
    include_events: bool = True
    include_summary: bool = True
    include_statistics: bool = True

class BatchExportRequest(BaseModel):
    session_ids: List[str]
    format: str = "markdown"

@router.post("/session")
async def export_session(
    request: ExportRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """导出单个会话"""
    service = ExportService(db)

    if request.format == "markdown":
        content = await service.export_session_markdown(
            session_id=request.session_id,
            include_events=request.include_events,
            include_summary=request.include_summary,
            include_statistics=request.include_statistics,
        )
        filename = f"{request.session_id}.md"
        media_type = "text/markdown"

    elif request.format == "pdf":
        content = await service.export_session_pdf(
            session_id=request.session_id,
            include_events=request.include_events,
            include_summary=request.include_summary,
        )
        filename = f"{request.session_id}.pdf"
        media_type = "application/pdf"

    elif request.format == "json":
        content_data = await service.export_session_json(
            session_id=request.session_id,
            include_events=request.include_events,
            include_summary=request.include_summary,
        )
        content = json.dumps(content_data, ensure_ascii=False, indent=2)
        filename = f"{request.session_id}.json"
        media_type = "application/json"

    elif request.format == "html":
        content = await service.export_session_html(
            session_id=request.session_id,
            include_events=request.include_events,
            include_summary=request.include_summary,
        )
        filename = f"{request.session_id}.html"
        media_type = "text/html"

    else:
        raise HTTPException(status_code=400, detail="不支持的导出格式")

    return StreamingResponse(
        io.BytesIO(content.encode() if isinstance(content, str) else content),
        media_type=media_type,
        headers={
            "Content-Disposition": f'attachment; filename="{filename}"'
        }
    )

@router.post("/batch")
async def batch_export_sessions(
    request: BatchExportRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """批量导出会话"""
    service = ExportService(db)

    content = await service.batch_export(
        session_ids=request.session_ids,
        format=request.format,
    )

    filename = f"sessions_{datetime.utcnow().strftime('%Y%m%d_%H%M%S')}.zip"

    return StreamingResponse(
        io.BytesIO(content),
        media_type="application/zip",
        headers={
            "Content-Disposition": f'attachment; filename="{filename}"'
        }
    )
```

---

## 前端代码示例

### 导出对话框组件

```typescript
// frontend/src/components/export/ExportDialog.tsx
import React, { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Download, FileText, FileJson, FileCode, File } from 'lucide-react';

interface ExportDialogProps {
  sessionId: string;
  sessionTitle?: string;
}

export function ExportDialog({ sessionId, sessionTitle }: ExportDialogProps) {
  const [open, setOpen] = useState(false);
  const [format, setFormat] = useState<'markdown' | 'pdf' | 'json' | 'html'>('markdown');
  const [includeEvents, setIncludeEvents] = useState(true);
  const [includeSummary, setIncludeSummary] = useState(true);
  const [includeStats, setIncludeStats] = useState(true);
  const [loading, setLoading] = useState(false);

  const formatOptions = [
    { value: 'markdown', label: 'Markdown', icon: FileText },
    { value: 'pdf', label: 'PDF', icon: File },
    { value: 'json', label: 'JSON', icon: FileJson },
    { value: 'html', label: 'HTML', icon: FileCode },
  ];

  const handleExport = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/export/session', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          session_id: sessionId,
          format,
          include_events: includeEvents,
          include_summary: includeSummary,
          include_statistics: includeStats,
        }),
      });

      if (response.ok) {
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `${sessionTitle || sessionId}.${format === 'markdown' ? 'md' : format}`;
        document.body.appendChild(a);
        a.click();
        window.URL.revokeObjectURL(url);
        document.body.removeChild(a);
        setOpen(false);
      }
    } catch (error) {
      console.error('导出失败:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm">
          <Download className="h-4 w-4 mr-2" />
          导出
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>导出会话</DialogTitle>
          <DialogDescription>
            选择导出格式和包含的内容
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6 py-4">
          {/* 格式选择 */}
          <div className="space-y-3">
            <Label>导出格式</Label>
            <RadioGroup value={format} onValueChange={(v) => setFormat(v as any)}>
              {formatOptions.map((option) => {
                const Icon = option.icon;
                return (
                  <div key={option.value} className="flex items-center space-x-2">
                    <RadioGroupItem value={option.value} id={option.value} />
                    <Label htmlFor={option.value} className="flex items-center gap-2 cursor-pointer">
                      <Icon className="h-4 w-4" />
                      {option.label}
                    </Label>
                  </div>
                );
              })}
            </RadioGroup>
          </div>

          {/* 内容选择 */}
          <div className="space-y-3">
            <Label>包含内容</Label>
            <div className="space-y-2">
              <div className="flex items-center space-x-2">
                <Checkbox
                  id="events"
                  checked={includeEvents}
                  onCheckedChange={(checked) => setIncludeEvents(checked as boolean)}
                />
                <Label htmlFor="events" className="cursor-pointer">
                  事件日志
                </Label>
              </div>
              <div className="flex items-center space-x-2">
                <Checkbox
                  id="summary"
                  checked={includeSummary}
                  onCheckedChange={(checked) => setIncludeSummary(checked as boolean)}
                />
                <Label htmlFor="summary" className="cursor-pointer">
                  叙事摘要
                </Label>
              </div>
              <div className="flex items-center space-x-2">
                <Checkbox
                  id="stats"
                  checked={includeStats}
                  onCheckedChange={(checked) => setIncludeStats(checked as boolean)}
                />
                <Label htmlFor="stats" className="cursor-pointer">
                  统计信息
                </Label>
              </div>
            </div>
          </div>
        </div>

        <div className="flex justify-end gap-2">
          <Button variant="outline" onClick={() => setOpen(false)}>
            取消
          </Button>
          <Button onClick={handleExport} disabled={loading}>
            {loading ? '导出中...' : '导出'}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/export.py` | 创建 | 导出服务 |
| `app/api/export.py` | 创建 | 导出 API |
| `frontend/src/components/export/ExportDialog.tsx` | 创建 | 导出对话框组件 |
| `tests/test_export.py` | 创建 | 导出测试 |

---

## 验收标准

- [ ] Markdown 导出格式正确
- [ ] PDF 导出排版美观
- [ ] JSON 导出结构完整
- [ ] HTML 导出样式正确
- [ ] 批量导出打包成功
- [ ] 文件下载正常
- [ ] 导出内容完整无遗漏
- [ ] 导出性能满足要求

---

## 参考文档

- M3-001: AI 总结服务
- M3-014: Summary 数据结构
- ReportLab 文档
- Markdown 规范

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
