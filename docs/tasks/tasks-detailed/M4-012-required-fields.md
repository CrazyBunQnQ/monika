# M4-012: 实现场景包必填字段校验

**任务ID**: M4-012
**标题**: 实现场景包必填字段校验
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M0-023

---

## 任务描述

实现场景包的必填字段校验，确保上传的场景包符合最低要求。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M4-012-01 | 定义必填字段列表 | 确定哪些字段必填 | 20min |
| M4-012-02 | 实现字段检查函数 | 校验逻辑 | 30min |
| M4-012-03 | 实现嵌套检查 | 深度检查对象 | 30min |
| M4-012-04 | 实现错误报告 | 清晰的错误信息 | 30min |
| M4-012-05 | 编写校验测试 | 各种场景测试 | 25min |
| M4-012-06 | 集成到上传流程 | 上传时自动校验 | 10min |

---

## 必填字段定义

```python
# app/services/validation.py
from typing import Dict, List, Any
from dataclasses import dataclass

@dataclass
class ValidationError:
    """校验错误"""
    path: str              # 错误路径，如 "metadata.title"
    message: str           # 错误消息
    code: str             # 错误代码
    value: Any = None      # 错误值

class RequiredFieldsValidator:
    """必填字段校验器"""

    # 必填字段定义
    REQUIRED_FIELDS = {
        'metadata': ['id', 'title', 'version', 'author'],
        'scenes': {
            '_self': ['id', 'title', 'order', 'narrative'],
            'narrative': ['opening']
        },
        'shared': {
            'npcs': ['id', 'name'],
            'locations': ['id', 'name'],
            'clues': ['id', 'description'],
        }
    }

    def __init__(self):
        self.errors: List[ValidationError] = []

    def validate(self, data: Dict) -> List[ValidationError]:
        """校验场景包数据"""
        self.errors = []

        # 检查顶层结构
        self._check_top_level(data)

        # 检查 metadata
        if 'metadata' in data:
            self._check_metadata(data['metadata'])

        # 检查 scenes
        if 'scenes' in data:
            self._check_scenes(data['scenes'])

        # 检查 shared
        if 'shared' in data:
            self._check_shared(data['shared'])

        return self.errors

    def _check_top_level(self, data: Dict):
        """检查顶层必填字段"""
        required = ['metadata', 'scenes']

        for field in required:
            if field not in data:
                self.errors.append(ValidationError(
                    path=field,
                    message=f"Missing required field: {field}",
                    code="MISSING_REQUIRED"
                ))

    def _check_metadata(self, metadata: Dict):
        """检查元数据"""
        required = self.REQUIRED_FIELDS['metadata']

        for field in required:
            if field not in metadata:
                self.errors.append(ValidationError(
                    path=f"metadata.{field}",
                    message=f"Missing required metadata field: {field}",
                    code="MISSING_REQUIRED"
                ))
            elif not metadata[field]:
                self.errors.append(ValidationError(
                    path=f"metadata.{field}",
                    message=f"Field cannot be empty: {field}",
                    code="EMPTY_VALUE",
                    value=metadata[field]
                ))

    def _check_scenes(self, scenes: Dict):
        """检查场景集合"""
        if not scenes:
            self.errors.append(ValidationError(
                path="scenes",
                message="At least one scene is required",
                code="NO_SCENES"
            ))
            return

        required = self.REQUIRED_FIELDS['scenes']

        for scene_id, scene in scenes.items():
            # 检查场景必填字段
            for field in required['_self']:
                if field not in scene:
                    self.errors.append(ValidationError(
                        path=f"scenes.{scene_id}.{field}",
                        message=f"Missing required scene field: {field}",
                        code="MISSING_REQUIRED"
                    ))

            # 检查嵌套的 narrative
            if 'narrative' in scene:
                self._check_narrative(scene['narrative'], f"scenes.{scene_id}")

            # 检查数组字段不为空
            for array_field in ['npcs', 'locations', 'clues']:
                if array_field in scene and isinstance(scene[array_field], list):
                    pass  # 可以为空数组，不检查

    def _check_narrative(self, narrative: Dict, path: str):
        """检查叙事结构"""
        if 'opening' not in narrative:
            self.errors.append(ValidationError(
                path=f"{path}.narrative.opening",
                message="Narrative opening is required",
                code="MISSING_REQUIRED"
            ))
        elif not narrative['opening'].strip():
            self.errors.append(ValidationError(
                path=f"{path}.narrative.opening",
                message="Narrative opening cannot be empty",
                code="EMPTY_VALUE"
            ))

    def _check_shared(self, shared: Dict):
        """检查共享资源"""
        # 检查 NPC
        if 'npcs' in shared:
            for npc_id, npc in shared['npcs'].items():
                for field in self.REQUIRED_FIELDS['shared']['npcs']:
                    if field not in npc:
                        self.errors.append(ValidationError(
                            path=f"shared.npcs.{npc_id}.{field}",
                            message=f"Missing required NPC field: {field}",
                            code="MISSING_REQUIRED"
                        ))

        # 检查 Location
        if 'locations' in shared:
            for loc_id, loc in shared['locations'].items():
                for field in self.REQUIRED_FIELDS['shared']['locations']:
                    if field not in loc:
                        self.errors.append(ValidationError(
                            path=f"shared.locations.{loc_id}.{field}",
                            message=f"Missing required location field: {field}",
                            code="MISSING_REQUIRED"
                        ))

        # 检查 Clue
        if 'clues' in shared:
            for clue_id, clue in shared['clues'].items():
                for field in self.REQUIRED_FIELDS['shared']['clues']:
                    if field not in clue:
                        self.errors.append(ValidationError(
                            path=f"shared.clues.{clue_id}.{field}",
                            message=f"Missing required clue field: {field}",
                            code="MISSING_REQUIRED"
                        ))
```

---

## 使用示例

```python
# 在上传接口中使用
from app.services.validation import RequiredFieldsValidator

@router.post("/scripts/upload")
async def upload_script(
    file: UploadFile,
    current_user = Depends(get_current_user)
):
    # 读取文件
    content = await file.read()
    data = json.loads(content)

    # 校验必填字段
    validator = RequiredFieldsValidator()
    errors = validator.validate(data)

    if errors:
        return JSONResponse(
            status_code=400,
            content={
                "error": "Validation failed",
                "errors": [
                    {
                        "field": e.path,
                        "message": e.message,
                        "code": e.code
                    }
                    for e in errors
                ]
            }
        )

    # 继续处理...
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/validation.py` | 创建 | 校验服务 |
| `app/api/scripts.py` | 更新 | 集成校验 |
| `tests/test_validation.py` | 创建 | 校验测试 |

---

## 验收标准

- [ ] 所有必填字段被检查
- [ ] 嵌套字段正确检查
- [ ] 错误信息清晰有用
- [ ] 空值被正确处理
- [ ] 测试覆盖各种场景

---

## 参考文档

- M0-023: 必填字段校验规则
- M4-008: JSON 解析器

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
