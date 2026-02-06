# M6-012: 实现 GET /game/leads

**任务ID**: M6-012
**标题**: 实现 GET /game/leads
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M6-005

---

## 任务描述

实现获取游戏 Leads 列表的 API 端点。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-012-01 | 设计 API 规范 | 请求/响应格式 | 20min |
| M6-012-02 | 实现查询参数处理 | 过滤、排序、分页 | 30min |
| M6-012-03 | 实现 API 端点 | 核心逻辑 | 30min |
| M6-012-04 | 实现权限控制 | 访问验证 | 15min |
| M6-012-05 | 编写 API 文档 | OpenAPI 规范 | 15min |
| M6-012-06 | 编写单元测试 | 测试覆盖 | 10min |

---

## API 规范

```python
# app/api/leads.py
from fastapi import APIRouter, Depends, Query, HTTPException, status
from typing import List, Optional
from pydantic import BaseModel

router = APIRouter(prefix="/game/{session_id}/leads", tags=["leads"])

class LeadResponse(BaseModel):
    """Lead 响应模型"""
    lead_id: str
    title: str
    description: str
    category: str
    type: str
    priority: int
    urgency: str
    status: str
    action: dict
    related: dict
    source: dict
    created_at: str
    updated_at: Optional[str] = None

class LeadsListResponse(BaseModel):
    """Leads 列表响应"""
    leads: List[LeadResponse]
    total: int
    page: int
    page_size: int
    has_more: bool

@router.get("", response_model=LeadsListResponse)
async def get_leads(
    session_id: str,
    category: Optional[str] = Query(None, description="按类别过滤"),
    type: Optional[str] = Query(None, description="按类型过滤"),
    status: Optional[str] = Query("available", description="按状态过滤"),
    min_priority: Optional[int] = Query(None, description="最小优先级"),
    max_priority: Optional[int] = Query(None, description="最大优先级"),
    sort_by: str = Query("priority", description="排序字段"),
    sort_order: str = Query("desc", description="排序方向"),
    page: int = Query(1, ge=1, description="页码"),
    page_size: int = Query(20, ge=1, le=100, description="每页数量"),
    current_user: dict = Depends(get_current_user),
    leads_manager: 'LeadsStateManager' = Depends(get_leads_manager)
):
    """
    获取游戏 Leads 列表

    参数:
    - session_id: 游戏会话 ID
    - category: 过滤类别 (investigate, action, social, prep, etc.)
    - type: 过滤类型 (clue_follow, npc_talk, location_search, etc.)
    - status: 过滤状态 (available, in_progress, completed, etc.)
    - min_priority: 最小优先级 (0-100)
    - max_priority: 最大优先级 (0-100)
    - sort_by: 排序字段 (priority, urgency, created_at)
    - sort_order: 排序方向 (asc, desc)
    - page: 页码 (从 1 开始)
    - page_size: 每页数量 (1-100)

    返回:
    - leads: Lead 列表
    - total: 总数
    - page: 当前页
    - page_size: 每页数量
    - has_more: 是否有更多
    """
    # 验证访问权限
    if not await _can_access_session(current_user, session_id):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="无权访问此会话"
        )

    # 获取状态
    state = await leads_manager.get_state(session_id)
    if not state:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="会话不存在"
        )

    # 过滤
    filtered_leads = _filter_leads(
        state.available,
        category=category,
        type=type,
        status=status,
        min_priority=min_priority,
        max_priority=max_priority
    )

    # 排序
    sorted_leads = _sort_leads(
        filtered_leads,
        sort_by=sort_by,
        sort_order=sort_order
    )

    # 分页
    total = len(sorted_leads)
    start = (page - 1) * page_size
    end = start + page_size
    paginated_leads = sorted_leads[start:end]

    return LeadsListResponse(
        leads=[LeadResponse(**lead.dict()) for lead in paginated_leads],
        total=total,
        page=page,
        page_size=page_size,
        has_more=end < total
    )
```

---

## 查询参数处理

```python
# app/api/leads/filters.py
from typing import List, Optional
from app.core.types.leads import LeadItem, LeadStatus

def _filter_leads(
    leads: List[LeadItem],
    category: Optional[str] = None,
    type: Optional[str] = None,
    status: Optional[str] = None,
    min_priority: Optional[int] = None,
    max_priority: Optional[int] = None
) -> List[LeadItem]:
    """过滤 Leads"""
    filtered = leads

    # 按类别过滤
    if category:
        filtered = [l for l in filtered if l.category == category]

    # 按类型过滤
    if type:
        filtered = [l for l in filtered if l.type == type]

    # 按状态过滤
    if status:
        status_enum = LeadStatus(status)
        filtered = [l for l in filtered if l.status == status_enum]

    # 按优先级范围过滤
    if min_priority is not None:
        filtered = [l for l in filtered if l.priority >= min_priority]

    if max_priority is not None:
        filtered = [l for l in filtered if l.priority <= max_priority]

    return filtered

def _sort_leads(
    leads: List[LeadItem],
    sort_by: str = "priority",
    sort_order: str = "desc"
) -> List[LeadItem]:
    """排序 Leads"""
    reverse = sort_order == "desc"

    if sort_by == "priority":
        return sorted(leads, key=lambda x: x.priority, reverse=reverse)
    elif sort_by == "urgency":
        urgency_order = {"urgent": 4, "high": 3, "medium": 2, "low": 1}
        return sorted(
            leads,
            key=lambda x: urgency_order.get(x.urgency, 0),
            reverse=reverse
        )
    elif sort_by == "created_at":
        return sorted(leads, key=lambda x: x.timestamp, reverse=reverse)
    else:
        return leads
```

---

## 权限控制

```python
# app/api/leads/auth.py
from fastapi import Depends, HTTPException, status
from app.core.auth import get_current_user

async def _can_access_session(
    current_user: dict,
    session_id: str
) -> bool:
    """检查用户是否可以访问会话"""
    # 用户是会话的创建者
    if current_user.get('owned_sessions') and session_id in current_user['owned_sessions']:
        return True

    # 用户是会话的参与者
    if current_user.get('joined_sessions') and session_id in current_user['joined_sessions']:
        return True

    return False

async def validate_session_access(
    session_id: str,
    current_user: dict = Depends(get_current_user)
):
    """验证会话访问权限（依赖注入）"""
    if not await _can_access_session(current_user, session_id):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="无权访问此会话"
        )
    return session_id
```

---

## OpenAPI 文档

```yaml
# docs/api/leads.yaml
openapi: 3.0.0
info:
  title: Leads API
  version: 1.0.0
paths:
  /game/{session_id}/leads:
    get:
      summary: 获取游戏 Leads 列表
      tags:
        - leads
      parameters:
        - name: session_id
          in: path
          required: true
          schema:
            type: string
          description: 游戏会话 ID
        - name: category
          in: query
          schema:
            type: string
            enum: [investigate, action, social, prep, explore, combat, escape]
          description: 按类别过滤
        - name: type
          in: query
          schema:
            type: string
            enum: [clue_follow, npc_talk, location_search, item_use, skill_check, etc.]
          description: 按类型过滤
        - name: status
          in: query
          schema:
            type: string
            enum: [available, in_progress, completed, failed, expired, blocked]
            default: available
          description: 按状态过滤
        - name: min_priority
          in: query
          schema:
            type: integer
            minimum: 0
            maximum: 100
          description: 最小优先级
        - name: max_priority
          in: query
          schema:
            type: integer
            minimum: 0
            maximum: 100
          description: 最大优先级
        - name: sort_by
          in: query
          schema:
            type: string
            enum: [priority, urgency, created_at]
            default: priority
          description: 排序字段
        - name: sort_order
          in: query
          schema:
            type: string
            enum: [asc, desc]
            default: desc
          description: 排序方向
        - name: page
          in: query
          schema:
            type: integer
            minimum: 1
            default: 1
          description: 页码
        - name: page_size
          in: query
          schema:
            type: integer
            minimum: 1
            maximum: 100
            default: 20
          description: 每页数量
      responses:
        '200':
          description: 成功
          content:
            application/json:
              schema:
                type: object
                properties:
                  leads:
                    type: array
                    items:
                      $ref: '#/components/schemas/Lead'
                  total:
                    type: integer
                  page:
                    type: integer
                  page_size:
                    type: integer
                  has_more:
                    type: boolean
        '403':
          description: 无权限
        '404':
          description: 会话不存在

components:
  schemas:
    Lead:
      type: object
      properties:
        lead_id:
          type: string
        title:
          type: string
        description:
          type: string
        category:
          type: string
          enum: [investigate, action, social, prep, explore, combat, escape]
        type:
          type: string
        priority:
          type: integer
          minimum: 0
          maximum: 100
        urgency:
          type: string
          enum: [low, medium, high, urgent]
        status:
          type: string
          enum: [available, in_progress, completed, failed, expired, blocked]
        action:
          type: object
        related:
          type: object
        source:
          type: object
        created_at:
          type: string
          format: date-time
        updated_at:
          type: string
          format: date-time
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/api/leads.py` | 创建 | Leads API 路由 |
| `app/api/leads/filters.py` | 创建 | 过滤和排序 |
| `app/api/leads/auth.py` | 创建 | 权限验证 |
| `docs/api/leads.yaml` | 创建 | API 文档 |
| `tests/api/leads/test_get_leads.py` | 创建 | 单元测试 |

---

## 验收标准

- [ ] API 端点正常工作
- [ ] 过滤功能完整
- [ ] 排序功能正确
- [ ] 分页功能有效
- [ ] 权限控制正确
- [ ] API 文档完整
- [ ] 单元测试通过

---

## 参考文档

- M6-005: Leads 自动刷新
- M6-004: Leads 状态管理

---

**最后更新**: 2026-02-07
**状态**: [ ] 待开始
