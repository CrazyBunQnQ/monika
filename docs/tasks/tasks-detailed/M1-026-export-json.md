# M1-026 实现角色卡导出 JSON GET /characters/:id/export

## 概述
实现角色卡导出为 JSON 格式的 API 端点,支持完整数据和自定义字段选择。

## 验收标准
- [ ] 实现 GET /characters/:id/export 端点
- [ ] 支持完整 JSON 导出
- [ ] 支持字段选择(query 参数)
- [ ] 设置正确的 Content-Type
- [ ] 支持文件名自定义
- [ ] 包含导出时间戳

## 技术方案

### API 端点

```python
from fastapi import APIRouter, Depends, Query, HTTPException
from fastapi.responses import JSONResponse
from typing import Optional, List
from datetime import datetime
import json
import urllib.parse

router = APIRouter(prefix="/characters", tags=["characters"])

@router.get("/{character_id}/export")
async def export_character_json(
    character_id: str,
    fields: Optional[str] = Query(None, description="导出字段(逗号分隔)"),
    include_metadata: bool = Query(True, description="包含元数据"),
    filename: Optional[str] = Query(None, description="自定义文件名"),
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """
    导出角色卡为 JSON

    参数:
    - fields: 导出字段列表(逗号分隔),如 "name,age,attributes"
    - include_metadata: 是否包含导出元数据
    - filename: 自定义文件名(不含扩展名)

    返回: JSON 文件下载
    """
    # 查询角色
    character = db.query(Character).filter(
        Character.id == character_id
    ).first()

    if not character:
        raise HTTPException(status_code=404, detail="角色不存在")

    # 权限检查
    if character.user_id != current_user.id:
        raise HTTPException(status_code=403, detail="无权限导出此角色")

    # 构建导出数据
    export_data = build_export_data(character, fields, include_metadata)

    # 生成文件名
    if filename:
        safe_filename = sanitize_filename(filename)
    else:
        safe_filename = f"{character.name}_{character.id[:8]}"

    filename_with_ext = f"{safe_filename}.json"

    # 返回 JSON 响应
    return JSONResponse(
        content=export_data,
        media_type="application/json",
        headers={
            "Content-Disposition": f'attachment; filename="{urllib.parse.quote(filename_with_ext)}"'
        }
    )
```

### 导出数据构建

```python
def build_export_data(
    character: Character,
    fields: Optional[str],
    include_metadata: bool
) -> dict:
    """
    构建导出数据
    """
    # 完整字段列表
    all_fields = {
        # 基本信息
        "id": character.id,
        "name": character.name,
        "age": character.age,
        "occupation": character.occupation,
        "player": character.player,

        # 属性
        "attributes": {
            "STR": character.str,
            "CON": character.con,
            "DEX": character.dex,
            "APP": character.app,
            "POW": character.pow,
            "INT": character.int,
            "SIZ": character.siz,
            "EDU": character.edu
        },

        # 派生属性
        "derived": {
            "HP": character.hp,
            "HP_max": character.hp_max,
            "MP": character.mp,
            "MP_max": character.mp_max,
            "SAN": character.san,
            "SAN_max": character.san_max,
            "Luck": character.luck,
            "Luck_max": character.luck_max,
            "Move": character.move,
            "DB": character.db,
            "Build": character.build
        },

        # 技能
        "skills": character.skills or {},

        # 状态
        "status": character.status,
        "inventory": character.inventory or [],
        "notes": character.notes,

        # 时间戳
        "created_at": character.created_at.isoformat(),
        "updated_at": character.updated_at.isoformat()
    }

    # 字段选择
    if fields:
        field_list = [f.strip() for f in fields.split(',')]
        export_data = {}
        for field in field_list:
            if field in all_fields:
                export_data[field] = all_fields[field]
    else:
        export_data = all_fields

    # 添加元数据
    if include_metadata:
        export_data["_metadata"] = {
            "version": "1.0",
            "exported_at": datetime.utcnow().isoformat(),
            "format": "coc7e_character"
        }

    return export_data
```

### 文件名处理

```python
import re
from typing import Optional

def sanitize_filename(filename: str) -> str:
    """
    清理文件名,移除不安全字符
    """
    # 移除或替换不安全字符
    safe = re.sub(r'[<>:"/\\|?*]', '_', filename)

    # 移除控制字符
    safe = re.sub(r'[\x00-\x1f\x7f]', '', safe)

    # 限制长度
    max_length = 200
    if len(safe) > max_length:
        safe = safe[:max_length]

    # 移除首尾空格和点
    safe = safe.strip('. ')

    # 如果为空,使用默认名称
    if not safe:
        safe = "character"

    return safe
```

### 批量导出

```python
@router.post("/export-batch")
async def export_characters_batch(
    character_ids: List[str],
    fields: Optional[str] = Query(None),
    include_metadata: bool = Query(True),
    archive_format: str = Query("zip", regex="^(zip|tar)$"),
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """
    批量导出角色卡

    支持打包为 ZIP 或 TAR
    """
    # 查询角色
    characters = db.query(Character).filter(
        Character.id.in_(character_ids),
        Character.user_id == current_user.id
    ).all()

    if not characters:
        raise HTTPException(status_code=404, detail="未找到角色")

    # 创建临时目录
    import tempfile
    import zipfile
    import tarfile
    from io import BytesIO

    with tempfile.TemporaryDirectory() as tmpdir:
        archive_name = f"characters_{datetime.utcnow().strftime('%Y%m%d_%H%M%S')}"

        if archive_format == "zip":
            # 创建 ZIP
            zip_buffer = BytesIO()
            with zipfile.ZipFile(zip_buffer, 'w', zipfile.ZIP_DEFLATED) as zip_file:
                for character in characters:
                    # 构建导出数据
                    export_data = build_export_data(character, fields, include_metadata)

                    # 写入 JSON
                    json_str = json.dumps(export_data, ensure_ascii=False, indent=2)
                    filename = f"{sanitize_filename(character.name)}_{character.id[:8]}.json"

                    zip_file.writestr(filename, json_str)

            zip_buffer.seek(0)

            from fastapi.responses import Response
            return Response(
                content=zip_buffer.getvalue(),
                media_type="application/zip",
                headers={
                    "Content-Disposition": f'attachment; filename="{archive_name}.zip"'
                }
            )
```

### 导出格式验证

```python
from pydantic import BaseModel

class CharacterExportFormat(BaseModel):
    """角色卡导出格式"""
    version: str = "1.0"
    format: str = "coc7e_character"

    # 基本信息
    id: str
    name: str
    age: int
    occupation: str
    player: str

    # 属性
    attributes: dict
    derived: dict
    skills: dict

    # 状态
    status: str
    inventory: list
    notes: Optional[str]

    # 时间戳
    created_at: str
    updated_at: str

    # 元数据(可选)
    _metadata: Optional[dict] = None

    class Config:
        extra = "allow"

def validate_export_format(data: dict) -> bool:
    """验证导出格式"""
    try:
        CharacterExportFormat(**data)
        return True
    except Exception:
        return False
```

### 导出历史

```python
class ExportHistory(Base):
    """导出历史"""
    __tablename__ = "export_history"

    id = Column(String, primary_key=True)
    user_id = Column(String, ForeignKey("users.id"))
    character_id = Column(String, ForeignKey("characters.id"))

    exported_at = Column(DateTime, default=datetime.utcnow)
    fields = Column(Text, nullable=True)  # JSON 字符串
    filename = Column(String)

    # 关联
    user = relationship("User")
    character = relationship("Character")

def record_export(
    user_id: str,
    character_id: str,
    fields: Optional[str],
    filename: str,
    db: Session
):
    """记录导出历史"""
    history = ExportHistory(
        id=str(uuid.uuid4()),
        user_id=user_id,
        character_id=character_id,
        fields=json.dumps(fields.split(',')) if fields else None,
        filename=filename
    )

    db.add(history)
    db.commit()
```

### 客户端示例

```javascript
// JavaScript 客户端导出示例
async function exportCharacter(characterId, options = {}) {
  const params = new URLSearchParams();

  if (options.fields) {
    params.append('fields', options.fields.join(','));
  }

  if (options.includeMetadata !== undefined) {
    params.append('include_metadata', options.includeMetadata);
  }

  if (options.filename) {
    params.append('filename', options.filename);
  }

  const response = await fetch(
    `/api/characters/${characterId}/export?${params.toString()}`,
    {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Accept': 'application/json'
      }
    }
  );

  if (!response.ok) {
    throw new Error('导出失败');
  }

  // 获取文件名
  const contentDisposition = response.headers.get('Content-Disposition');
  const filenameMatch = /filename="(.+)"/.exec(contentDisposition);
  const filename = filenameMatch ? filenameMatch[1] : 'character.json';

  // 下载文件
  const blob = await response.blob();
  const url = URL.createObjectURL(blob);

  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}
```

## 依赖关系
- 前置任务: M1-020 实现获取角色卡 GET /characters/:id
- 被依赖: M1-031 实现 JSON 导入/导出组件

## 预估工时
2h
