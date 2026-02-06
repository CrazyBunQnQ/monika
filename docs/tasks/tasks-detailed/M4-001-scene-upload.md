# M4-001: 实现场景包上传功能

**任务ID**: M4-001
**标题**: 实现场景包上传功能
**类型**: backend (后端开发)
**预估工时**: 2.5h
**依赖**: M4-008, M0-022

---

## 任务描述

实现场景包文件上传、解析、存储功能，支持 ZIP 文件上传和 JSON 验证。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M4-001-01 | 设计上传 API 结构 | API 设计 | 20min |
| M4-001-02 | 实现文件上传端点 | Upload Endpoint | 30min |
| M4-001-03 | 实现 ZIP 解压 | Zip Extraction | 25min |
| M4-001-04 | 实现场景验证 | Schema Validation | 35min |
| M4-001-05 | 实现场景存储 | Storage | 25min |
| M4-001-06 | 实现缩略图生成 | Thumbnail | 20min |
| M4-001-07 | 编写上传测试 | 测试覆盖 | 25min |

---

## 上传 API

```python
# app/api/scene_upload.py
from fastapi import APIRouter, UploadFile, File, Depends, HTTPException
from sqlalchemy.orm import Session
import shutil
import zipfile
import json
from pathlib import Path

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.scene_parser import SceneParser
from app.services.scene_validator import SceneValidator

router = APIRouter(prefix="/scenes", tags=["scenes"])

# 配置
UPLOAD_DIR = Path("data/scenes")
UPLOAD_DIR.mkdir(parents=True, exist_ok=True)

@router.post("/upload")
async def upload_scene(
    file: UploadFile = File(...),
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """上传场景包 ZIP 文件"""
    # 验证文件类型
    if not file.filename.endswith(".zip"):
        raise HTTPException(
            status_code=400,
            detail="只支持 ZIP 格式的场景包"
        )

    # 创建临时文件
    temp_path = UPLOAD_DIR / f"temp_{file.filename}"

    try:
        # 保存上传的文件
        with open(temp_path, "wb") as buffer:
            shutil.copyfileobj(file.file, buffer)

        # 解压 ZIP
        extract_dir = UPLOAD_DIR / f"extract_{file.filename[:-4]}"
        extract_dir.mkdir(exist_ok=True)

        with zipfile.ZipFile(temp_path, 'r') as zip_ref:
            zip_ref.extractall(extract_dir)

        # 查找 scene.json
        scene_json_path = extract_dir / "scene.json"
        if not scene_json_path.exists():
            raise HTTPException(
                status_code=400,
                detail="场景包必须包含 scene.json 文件"
            )

        # 解析场景
        with open(scene_json_path, 'r', encoding='utf-8') as f:
            scene_data = json.load(f)

        # 验证场景
        validator = SceneValidator()
        validation_result = validator.validate(scene_data)

        if not validation_result.is_valid:
            raise HTTPException(
                status_code=400,
                detail=f"场景验证失败: {validation_result.errors}"
            )

        # 保存场景
        scene_id = scene_data.get("metadata", {}).get("id")
        scene_dir = UPLOAD_DIR / scene_id
        scene_dir.mkdir(exist_ok=True)

        # 移动提取的文件到最终位置
        shutil.move(str(extract_dir), str(scene_dir))

        # 生成缩略图（如果有封面图）
        cover_path = scene_dir / "assets" / "cover.png"
        if cover_path.exists():
            await _generate_thumbnail(cover_path)

        # 清理临时文件
        temp_path.unlink()
        shutil.rmtree(extract_dir, ignore_errors=True)

        return {
            "message": "场景上传成功",
            "scene_id": scene_id,
            "scene_name": scene_data.get("metadata", {}).get("name"),
        }

    except Exception as e:
        # 清理临时文件
        if temp_path.exists():
            temp_path.unlink()
        raise HTTPException(
            status_code=500,
            detail=f"上传失败: {str(e)}"
        )

async def _generate_thumbnail(image_path: Path):
    """生成缩略图"""
    from PIL import Image

    thumbnail_path = image_path.parent / f"{image_path.stem}_thumb{image_path.suffix}"

    with Image.open(image_path) as img:
        img.thumbnail((300, 400))
        img.save(thumbnail_path)
```

---

## 场景验证器

```python
# app/services/scene_validator.py
from typing import List, Dict, Any
from jsonschema import validate, ValidationError

class ValidationResult:
    """验证结果"""
    def __init__(self):
        self.is_valid = True
        self.errors: List[str] = []

    def add_error(self, error: str):
        self.is_valid = False
        self.errors.append(error)

class SceneValidator:
    """场景验证器"""

    def __init__(self):
        self.schema = self._load_schema()

    def _load_schema(self) -> Dict[str, Any]:
        """加载 JSON Schema"""
        schema_path = Path("data/schemas/scene_schema.json")
        with open(schema_path, 'r', encoding='utf-8') as f:
            return json.load(f)

    def validate(self, scene_data: Dict[str, Any]) -> ValidationResult:
        """验证场景数据"""
        result = ValidationResult()

        # JSON Schema 验证
        try:
            validate(instance=scene_data, schema=self.schema)
        except ValidationError as e:
            result.add_error(f"Schema 验证失败: {e.message}")

        # 自定义验证
        self._validate_metadata(scene_data.get("metadata", {}), result)
        self._validate_scenes(scene_data.get("scenes", []), result)
        self._validate_npcs(scene_data.get("npcs", []), result)

        return result

    def _validate_metadata(self, metadata: Dict[str, Any], result: ValidationResult):
        """验证元信息"""
        required_fields = ["id", "name", "version", "author"]

        for field in required_fields:
            if field not in metadata:
                result.add_error(f"metadata.{field} 是必需的")

        # 验证 ID 格式
        scene_id = metadata.get("id", "")
        if not self._is_valid_id(scene_id):
            result.add_error(f"无效的场景 ID: {scene_id}")

    def _validate_scenes(self, scenes: List[Dict[str, Any]], result: ValidationResult):
        """验证场景"""
        if not scenes:
            result.add_error("至少需要一个场景")

        for i, scene in enumerate(scenes):
            if "id" not in scene:
                result.add_error(f"scene[{i}].id 是必需的")

            if "name" not in scene:
                result.add_error(f"scene[{i}].name 是必需的")

    def _validate_npcs(self, npcs: List[Dict[str, Any]], result: ValidationResult):
        """验证 NPC"""
        for i, npc in enumerate(npcs):
            if "id" not in npc:
                result.add_error(f"npc[{i}].id 是必需的")

            if "name" not in npc:
                result.add_error(f"npc[{i}].name 是必需的")

    def _is_valid_id(self, value: str) -> bool:
        """验证 ID 格式"""
        import re
        pattern = r'^[a-z0-9_]+$'
        return bool(re.match(pattern, value))
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/api/scene_upload.py` | 创建 | 场景上传 API |
| `app/services/scene_validator.py` | 创建 | 场景验证器 |
| `data/scenes/` | 创建 | 场景存储目录 |
| `data/schemas/scene_schema.json` | 创建 | 场景 Schema |

---

## 验收标准

- [ ] 文件上传功能正常
- [ ] ZIP 解压成功
- [ ] Schema 验证准确
- [ ] 错误提示友好
- [ ] 缩略图生成正确
- [ ] 测试覆盖完整

---

## 参考文档

- M0-022: 场景包 JSON Schema
- M4-008: JSON 解析器

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
