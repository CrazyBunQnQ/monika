# M1-027 实现角色卡导入 POST /characters/import

## 概述
实现从 JSON 导入角色卡的 API 端点,支持完整验证和错误处理。

## 验收标准
- [ ] 实现 POST /characters/import 端点
- [ ] 支持 JSON 文件上传
- [ ] 验证导入数据格式
- [ ] 处理 ID 冲突
- [ ] 返回导入结果和错误详情
- [ ] 支持批量导入

## 技术方案

### API 端点

```python
from fastapi import APIRouter, Depends, UploadFile, File, HTTPException
from sqlalchemy.orm import Session
import json
from typing import List, Dict
from datetime import datetime

router = APIRouter(prefix="/characters", tags=["characters"])

@router.post("/import")
async def import_character_json(
    file: UploadFile = File(..., description="JSON 文件"),
    resolve_id_conflict: str = "error",
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """
    导入角色卡 JSON

    参数:
    - file: JSON 文件
    - resolve_id_conflict: ID 冲突处理方式
      - "error": 报错(默认)
      - "skip": 跳过
      - "overwrite": 覆盖
      - "rename": 重命名

    返回:
    - success: 成功导入的角色列表
    - failed: 失败的角色列表(含错误原因)
    - skipped: 跳过的角色列表
    """
    # 验证文件类型
    if not file.filename.endswith('.json'):
        raise HTTPException(
            status_code=400,
            detail="只支持 JSON 文件"
        )

    try:
        # 读取文件
        content = await file.read()
        data = json.loads(content.decode('utf-8'))
    except json.JSONDecodeError as e:
        raise HTTPException(
            status_code=400,
            detail=f"JSON 解析失败: {str(e)}"
        )
    except Exception as e:
        raise HTTPException(
            status_code=500,
            detail=f"文件读取失败: {str(e)}"
        )

    # 处理单个或批量导入
    if isinstance(data, list):
        return await import_characters_batch(
            data,
            resolve_id_conflict,
            current_user,
            db
        )
    else:
        return await import_character_single(
            data,
            resolve_id_conflict,
            current_user,
            db
        )
```

### 单个导入

```python
async def import_character_single(
    data: dict,
    resolve_id_conflict: str,
    current_user: User,
    db: Session
):
    """导入单个角色卡"""
    # 验证格式
    try:
        validated_data = CharacterImport(**data)
    except ValidationError as e:
        return {
            "success": [],
            "failed": [{
                "id": data.get("id", "unknown"),
                "reason": format_validation_error(e)
            }],
            "skipped": []
        }

    # 处理 ID 冲突
    character_id = validated_data.id
    existing = db.query(Character).filter(
        Character.id == character_id
    ).first()

    if existing:
        if resolve_id_conflict == "error":
            return {
                "success": [],
                "failed": [{
                    "id": character_id,
                    "reason": "ID 冲突: 角色已存在"
                }],
                "skipped": []
            }
        elif resolve_id_conflict == "skip":
            return {
                "success": [],
                "failed": [],
                "skipped": [{
                    "id": character_id,
                    "reason": "已存在相同 ID 的角色"
                }]
            }
        elif resolve_id_conflict == "overwrite":
            # 删除旧角色
            db.delete(existing)
            db.commit()
        elif resolve_id_conflict == "rename":
            # 生成新 ID
            character_id = generate_new_id(character_id, db)

    # 创建角色
    character = Character(
        id=character_id,
        user_id=current_user.id,
        name=validated_data.name,
        age=validated_data.age,
        occupation=validated_data.occupation,
        player=validated_data.player,

        str=validated_data.attributes.STR,
        con=validated_data.attributes.CON,
        dex=validated_data.attributes.DEX,
        app=validated_data.attributes.APP,
        pow=validated_data.attributes.POW,
        int=validated_data.attributes.INT,
        siz=validated_data.attributes.SIZ,
        edu=validated_data.attributes.EDU,

        hp=validated_data.derived.HP,
        hp_max=validated_data.derived.HP_max,
        mp=validated_data.derived.MP,
        mp_max=validated_data.derived.MP_max,
        san=validated_data.derived.SAN,
        san_max=validated_data.derived.SAN_max,
        luck=validated_data.derived.Luck,
        luck_max=validated_data.derived.Luck_max,
        move=validated_data.derived.Move,
        db=validated_data.derived.DB,
        build=validated_data.derived.Build,

        skills=validated_data.skills,
        status=validated_data.status,
        inventory=validated_data.inventory,
        notes=validated_data.notes
    )

    db.add(character)
    db.commit()
    db.refresh(character)

    return {
        "success": [{
            "id": character.id,
            "name": character.name,
            "action": "created"
        }],
        "failed": [],
        "skipped": []
    }
```

### 批量导入

```python
async def import_characters_batch(
    data_list: List[dict],
    resolve_id_conflict: str,
    current_user: User,
    db: Session
):
    """批量导入角色卡"""
    success = []
    failed = []
    skipped = []

    for data in data_list:
        result = await import_character_single(
            data,
            resolve_id_conflict,
            current_user,
            db
        )

        success.extend(result["success"])
        failed.extend(result["failed"])
        skipped.extend(result["skipped"])

    return {
        "total": len(data_list),
        "success_count": len(success),
        "failed_count": len(failed),
        "skipped_count": len(skipped),
        "success": success,
        "failed": failed,
        "skipped": skipped
    }
```

### 导入模型

```python
from pydantic import BaseModel, Field, validator

class CharacterImport(BaseModel):
    """角色卡导入模型"""
    version: str = "1.0"
    format: str = "coc7e_character"

    # 基本信息
    id: str = Field(..., min_length=1, max_length=50)
    name: str = Field(..., min_length=1, max_length=100)
    age: int = Field(..., ge=15, le=90)
    occupation: str = Field(..., min_length=1, max_length=100)
    player: str = Field(..., min_length=1, max_length=100)

    # 属性
    attributes: Dict[str, int]
    derived: Dict[str, any]
    skills: Dict[str, int] = {}

    # 状态
    status: str = "alive"
    inventory: List[str] = []
    notes: Optional[str] = None

    # 时间戳(可选)
    created_at: Optional[str] = None
    updated_at: Optional[str] = None

    @validator('attributes')
    def validate_attributes(cls, v):
        """验证属性"""
        required = ['STR', 'CON', 'DEX', 'APP', 'POW', 'INT', 'SIZ', 'EDU']
        for attr in required:
            if attr not in v:
                raise ValueError(f"缺少属性: {attr}")
            if not (0 <= v[attr] <= 100):
                raise ValueError(f"属性 {attr} 必须在 0-100 之间")
        return v

    @validator('derived')
    def validate_derived(cls, v):
        """验证派生属性"""
        required = ['HP', 'HP_max', 'MP', 'MP_max', 'SAN', 'SAN_max', 'Luck', 'Move', 'DB', 'Build']
        for attr in required:
            if attr not in v:
                raise ValueError(f"缺少派生属性: {attr}")
        return v
```

### ID 处理

```python
import uuid
import re

def generate_new_id(base_id: str, db: Session) -> str:
    """
    生成新 ID

    规则:
    1. 尝试在原 ID 后添加 _2, _3, ...
    2. 如果失败,生成 UUID
    """
    # 检查是否有数字后缀
    match = re.match(r'^(.*)_?(\d+)$', base_id)
    if match:
        prefix = match.group(1)
        num = int(match.group(2))
        new_id = f"{prefix}_{num + 1}"
    else:
        new_id = f"{base_id}_2"

    # 检查是否存在
    existing = db.query(Character).filter(Character.id == new_id).first()
    if not existing:
        return new_id

    # 递归尝试
    return generate_new_id(new_id, db)

def resolve_id_conflict_handler(
    character_id: str,
    resolve_id_conflict: str,
    current_user: User,
    db: Session
) -> str:
    """
    处理 ID 冲突
    """
    existing = db.query(Character).filter(
        Character.id == character_id
    ).first()

    if not existing:
        return character_id

    if resolve_id_conflict == "error":
        raise ValueError(f"ID 冲突: {character_id}")

    elif resolve_id_conflict == "skip":
        raise ValueError(f"跳过: {character_id}")

    elif resolve_id_conflict == "overwrite":
        # 检查所有权
        if existing.user_id != current_user.id:
            raise ValueError("无权限覆盖其他用户的角色")
        return character_id

    elif resolve_id_conflict == "rename":
        return generate_new_id(character_id, db)

    return character_id
```

### 错误格式化

```python
def format_validation_error(error: ValidationError) -> str:
    """格式化验证错误"""
    messages = []
    for err in error.errors():
        field = ".".join(str(loc) for loc in err["loc"])
        messages.append(f"{field}: {err['msg']}")
    return "; ".join(messages)
```

### 导入历史记录

```python
def record_import(
    user_id: str,
    character_id: str,
    action: str,
    db: Session
):
    """记录导入历史"""
    history = ImportHistory(
        id=str(uuid.uuid4()),
        user_id=user_id,
        character_id=character_id,
        action=action,  # "created", "updated", "skipped", "failed"
        imported_at=datetime.utcnow()
    )

    db.add(history)
    db.commit()
```

### 响应格式

```python
# 成功响应
{
    "success": [
        {
            "id": "char_001",
            "name": "侦探约翰",
            "action": "created"
        }
    ],
    "failed": [],
    "skipped": []
}

# 批量导入响应
{
    "total": 5,
    "success_count": 3,
    "failed_count": 1,
    "skipped_count": 1,
    "success": [
        {"id": "char_001", "name": "角色1", "action": "created"},
        {"id": "char_002", "name": "角色2", "action": "created"},
        {"id": "char_003", "name": "角色3", "action": "created"}
    ],
    "failed": [
        {"id": "char_004", "reason": "缺少属性: STR"}
    ],
    "skipped": [
        {"id": "char_005", "reason": "已存在相同 ID 的角色"}
    ]
}
```

### 客户端示例

```javascript
// JavaScript 客户端导入示例
async function importCharacter(file, options = {}) {
  const formData = new FormData();
  formData.append('file', file);

  if (options.resolveIdConflict) {
    formData.append('resolve_id_conflict', options.resolveIdConflict);
  }

  const response = await fetch('/api/characters/import', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`
    },
    body: formData
  });

  if (!response.ok) {
    throw new Error('导入失败');
  }

  const result = await response.json();

  // 显示结果
  console.log(`成功导入 ${result.success_count} 个角色`);
  if (result.failed_count > 0) {
    console.error('失败:', result.failed);
  }
  if (result.skipped_count > 0) {
    console.warn('跳过:', result.skipped);
  }

  return result;
}
```

## 依赖关系
- 前置任务: M1-026 实现角色卡导出 JSON
- 被依赖: M1-031 实现 JSON 导入/导出组件

## 预估工时
2h
