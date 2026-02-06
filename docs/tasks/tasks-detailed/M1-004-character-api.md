# M1-004: 角色卡 CRUD API

**任务ID**: M1-004
**标题**: 角色卡 CRUD API
**类型**: backend (后端开发)
**预估工时**: 5h
**依赖**: M1-003

---

## 任务描述

实现角色卡的完整 CRUD API，包括创建、读取、更新、删除、列表操作。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-004-01 | 设计 Character Schema | Pydantic 模型 | 30min |
| M1-004-02 | 实现 POST /characters | 创建角色卡 | 45min |
| M1-004-03 | 实现 GET /characters | 列表角色卡 (分页) | 30min |
| M1-004-04 | 实现 GET /characters/:id | 获取单个角色卡 | 20min |
| M1-004-05 | 实现 PUT /characters/:id | 更新角色卡 | 45min |
| M1-004-06 | 实现 DELETE /characters/:id | 删除角色卡 | 20min |
| M1-004-07 | 实现权限控制 | 只能操作自己的角色卡 | 30min |
| M1-004-08 | 编写 API 测试 | 单元测试 | 30min |
| M1-004-09 | 添加 OpenAPI 文档 | 接口文档 | 10min |

---

## API 端点设计

### POST /characters
创建新角色卡

```yaml
requestBody:
  required: true
  content:
    application/json:
      schema:
        type: object
        required: [name, attributes, skills]
        properties:
          name:
            type: string
            description: 角色名称
          age:
            type: integer
            description: 年龄
          occupation:
            type: string
            description: 职业
          attributes:
            type: object
            properties:
              STR: { type: integer }
              CON: { type: integer }
              DEX: { type: integer }
              APP: { type: integer }
              POW: { type: integer }
              INT: { type: integer }
              SIZ: { type: integer }
              EDU: { type: integer }
          skills:
            type: object
            additionalProperties:
              type: integer
          backstory:
            type: string

responses:
  201:
    description: 创建成功
  400:
    description: 请求参数错误
  401:
    description: 未认证
```

### GET /characters
获取角色卡列表

```yaml
parameters:
  - name: page
    in: query
    schema:
      type: integer
      default: 1
  - name: limit
    in: query
    schema:
      type: integer
      default: 20
  - name: sort
    in: query
    schema:
      type: string
      enum: [created_at, updated_at, name]
      default: -created_at

responses:
  200:
    description: 成功
    content:
      application/json:
        schema:
          type: object
          properties:
            total:
              type: integer
            page:
              type: integer
            limit:
              type: integer
            data:
              type: array
              items:
                $ref: '#/components/schemas/Character'
```

---

## Pydantic Schema

```python
# app/schemas/character.py
from pydantic import BaseModel, Field
from typing import Dict, Optional

class AttributeSchema(BaseModel):
    STR: int = Field(..., ge=0, le=100)
    CON: int = Field(..., ge=0, le=100)
    DEX: int = Field(..., ge=0, le=100)
    APP: int = Field(..., ge=0, le=100)
    POW: int = Field(..., ge=0, le=100)
    INT: int = Field(..., ge=0, le=100)
    SIZ: int = Field(..., ge=0, le=100)
    EDU: int = Field(..., ge=0, le=100)

class CharacterCreate(BaseModel):
    name: str = Field(..., min_length=1, max_length=100)
    age: Optional[int] = Field(None, ge=0, le=150)
    occupation: Optional[str] = Field(None, max_length=100)
    attributes: AttributeSchema
    skills: Dict[str, int] = Field(default_factory=dict)
    backstory: Optional[str] = None

class CharacterUpdate(BaseModel):
    name: Optional[str] = Field(None, min_length=1, max_length=100)
    age: Optional[int] = Field(None, ge=0, le=150)
    occupation: Optional[str] = Field(None, max_length=100)
    attributes: Optional[AttributeSchema] = None
    skills: Optional[Dict[str, int]] = None
    backstory: Optional[str] = None

class CharacterResponse(CharacterCreate):
    id: int
    user_id: int
    derived: Dict[str, int]  # HP, MP, SAN, Luck, Move
    created_at: datetime
    updated_at: datetime

    class Config:
        from_attributes = True
```

---

## 路由实现

```python
# app/api/characters.py
from fastapi import APIRouter, Depends, status
from sqlalchemy.orm import Session
from app.api.deps import get_current_user, get_db
from app.schemas.character import CharacterCreate, CharacterUpdate, CharacterResponse
from app.services.character import CharacterService

router = APIRouter(prefix="/characters", tags=["characters"])

@router.post("", response_model=CharacterResponse, status_code=status.HTTP_201_CREATED)
async def create_character(
    data: CharacterCreate,
    current_user = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """创建新角色卡"""
    service = CharacterService(db)
    return service.create(user_id=current_user.id, data=data)

@router.get("", response_model=List[CharacterResponse])
async def list_characters(
    page: int = 1,
    limit: int = 20,
    current_user = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """获取角色卡列表"""
    service = CharacterService(db)
    return service.list(user_id=current_user.id, page=page, limit=limit)

@router.get("/{character_id}", response_model=CharacterResponse)
async def get_character(
    character_id: int,
    current_user = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """获取单个角色卡"""
    service = CharacterService(db)
    return service.get(character_id=character_id, user_id=current_user.id)

@router.put("/{character_id}", response_model=CharacterResponse)
async def update_character(
    character_id: int,
    data: CharacterUpdate,
    current_user = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """更新角色卡"""
    service = CharacterService(db)
    return service.update(character_id=character_id, user_id=current_user.id, data=data)

@router.delete("/{character_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_character(
    character_id: int,
    current_user = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """删除角色卡"""
    service = CharacterService(db)
    service.delete(character_id=character_id, user_id=current_user.id)
    return None
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/schemas/character.py` | 创建 | 角色卡 Schema |
| `app/api/characters.py` | 创建 | 角色卡 API 路由 |
| `app/services/character.py` | 更新 | 添加 CRUD 方法 |
| `tests/api/test_characters.py` | 创建 | API 测试 |

---

## 验收标准

- [ ] 创建角色卡成功
- [ ] 读取角色卡正确
- [ ] 更新角色卡生效
- [ ] 删除角色卡成功
- [ ] 列表分页正常
- [ ] 权限控制有效
- [ ] 派生数值自动计算

---

## 参考文档

- M1-003: 角色卡数据模型
- RESTful API 设计规范

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
