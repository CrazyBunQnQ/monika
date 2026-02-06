# M4-010: 实现场景包验证

**任务ID**: M4-010
**任务名称**: 实现场景包验证
**预估时间**: 5 小时
**优先级**: P0
**依赖**: M4-009 (JSON解析器)
**状态**: 待开始

---

## 任务概述

在JSON解析器基础上，实现场景包的完整验证逻辑，包括必填字段校验、数据类型校验、引用完整性校验、循环引用检测等，确保场景包数据完整、有效、无冲突，为后续的压缩、加密和上传提供质量保证。

---

## 子任务拆解

| ID | 子任务描述 | 预估时间 | 依赖 | 状态 |
|----|-----------|----------|------|------|
| M4-010-01 | 设计验证规则架构和验证器接口 | 1h | M4-009 | 待开始 |
| M4-010-02 | 实现必填字段和数据类型校验器 | 1h | M4-010-01 | 待开始 |
| M4-010-03 | 实现引用完整性校验器 | 1.5h | M4-010-02 | 待开始 |
| M4-010-04 | 实现循环引用检测器 | 1h | M4-010-03 | 待开始 |
| M4-010-05 | 实现验证结果生成和报告模块 | 0.5h | M4-010-04 | 待开始 |

**总预估时间**: 5 小时

---

## Python 后端实现

### 1. 验证器基础架构

```python
# backend/app/services/validators/base.py
from abc import ABC, abstractmethod
from typing import List, Any, Dict
from dataclasses import dataclass
from enum import Enum

class ValidationSeverity(str, Enum):
    """验证级别"""
    ERROR = "error"      # 阻塞性错误，必须修复
    WARNING = "warning"  # 警告，建议修复
    INFO = "info"        # 信息提示

@dataclass
class ValidationResult:
    """验证结果"""
    is_valid: bool                          # 是否通过验证
    errors: List[str]                       # 错误列表
    warnings: List[str]                     # 警告列表
    info: List[str]                         # 信息列表
    details: Dict[str, Any]                 # 详细信息

    @classmethod
    def success(cls) -> "ValidationResult":
        """创建成功结果"""
        return cls(
            is_valid=True,
            errors=[],
            warnings=[],
            info=[],
            details={}
        )

    @classmethod
    def failure(cls, errors: List[str], warnings: List[str] = None) -> "ValidationResult":
        """创建失败结果"""
        return cls(
            is_valid=False,
            errors=errors,
            warnings=warnings or [],
            info=[],
            details={}
        )

    def add_error(self, error: str) -> None:
        """添加错误"""
        self.errors.append(error)
        self.is_valid = False

    def add_warning(self, warning: str) -> None:
        """添加警告"""
        self.warnings.append(warning)

    def add_info(self, info: str) -> None:
        """添加信息"""
        self.info.append(info)

    def merge(self, other: "ValidationResult") -> None:
        """合并另一个验证结果"""
        self.errors.extend(other.errors)
        self.warnings.extend(other.warnings)
        self.info.extend(other.info)
        self.details.update(other.details)
        if not other.is_valid:
            self.is_valid = False

class BaseValidator(ABC):
    """验证器基类"""

    def __init__(self):
        self.name = self.__class__.__name__

    @abstractmethod
    def validate(self, data: Dict[str, Any]) -> ValidationResult:
        """
        执行验证

        Args:
            data: 待验证的数据

        Returns:
            ValidationResult: 验证结果
        """
        pass

    def _create_result(self, is_valid: bool = True) -> ValidationResult:
        """创建验证结果"""
        return ValidationResult(is_valid=is_valid, errors=[], warnings=[], info=[], details={})
```

### 2. 字段验证器

```python
# backend/app/services/validators/field_validator.py
from typing import Dict, Any, List, Set
from .base import BaseValidator, ValidationResult

class FieldValidator(BaseValidator):
    """字段验证器"""

    # 必需字段定义
    REQUIRED_ROOT_FIELDS = {"metadata"}
    REQUIRED_METADATA_FIELDS = {"name", "author"}
    REQUIRED_SCENE_FIELDS = {"id", "name", "description"}

    def validate(self, data: Dict[str, Any]) -> ValidationResult:
        """
        验证必需字段和数据类型

        Args:
            data: 场景包数据

        Returns:
            ValidationResult: 验证结果
        """
        result = self._create_result()

        # 验证根级字段
        self._validate_root_fields(data, result)

        # 验证元数据
        if "metadata" in data:
            self._validate_metadata(data["metadata"], result)

        # 验证场景列表
        if "scenes" in data and data["scenes"]:
            self._validate_scenes(data["scenes"], result)

        # 验证NPC
        if "global_npcs" in data and data["global_npcs"]:
            self._validate_npcs(data["global_npcs"], result)

        # 验证线索
        if "global_clues" in data and data["global_clues"]:
            self._validate_clues(data["global_clues"], result)

        return result

    def _validate_root_fields(self, data: Dict[str, Any], result: ValidationResult) -> None:
        """验证根级必需字段"""
        missing = self.REQUIRED_ROOT_FIELDS - set(data.keys())
        if missing:
            for field in missing:
                result.add_error(f"缺少根级必需字段: {field}")

    def _validate_metadata(self, metadata: Dict[str, Any], result: ValidationResult) -> None:
        """验证元数据"""
        if not isinstance(metadata, dict):
            result.add_error("metadata 必须是对象类型")
            return

        # 验证必需字段
        missing = self.REQUIRED_METADATA_FIELDS - set(metadata.keys())
        if missing:
            for field in missing:
                result.add_error(f"metadata 缺少必需字段: {field}")

        # 验证字段类型
        if "name" in metadata and not isinstance(metadata.get("name"), str):
            result.add_error("metadata.name 必须是字符串")

        if "author" in metadata and not isinstance(metadata.get("author"), str):
            result.add_error("metadata.author 必须是字符串")

        if "version" in metadata:
            version = metadata.get("version")
            if version not in ["1.0", "1.1"]:
                result.add_warning(f"不支持的版本号: {version}，建议使用 1.0 或 1.1")

        if "tags" in metadata:
            tags = metadata.get("tags")
            if not isinstance(tags, list):
                result.add_error("metadata.tags 必须是数组")
            elif len(tags) > 10:
                result.add_warning("标签数量超过10个，可能影响显示效果")

        # 验证玩家数量
        min_players = metadata.get("min_players", 3)
        max_players = metadata.get("max_players", 6)

        if isinstance(min_players, int) and isinstance(max_players, int):
            if min_players < 1 or min_players > 10:
                result.add_error("min_players 必须在 1-10 之间")
            if max_players < 1 or max_players > 10:
                result.add_error("max_players 必须在 1-10 之间")
            if min_players > max_players:
                result.add_error("min_players 不能大于 max_players")

    def _validate_scenes(self, scenes: List[Dict[str, Any]], result: ValidationResult) -> None:
        """验证场景列表"""
        if not isinstance(scenes, list):
            result.add_error("scenes 必须是数组类型")
            return

        scene_ids: Set[str] = set()

        for idx, scene in enumerate(scenes):
            if not isinstance(scene, dict):
                result.add_error(f"场景 [{idx}] 必须是对象类型")
                continue

            # 验证必需字段
            missing = self.REQUIRED_SCENE_FIELDS - set(scene.keys())
            if missing:
                for field in missing:
                    result.add_error(f"场景 [{idx}] 缺少必需字段: {field}")

            # 验证ID唯一性
            scene_id = scene.get("id")
            if scene_id:
                if not isinstance(scene_id, str):
                    result.add_error(f"场景 [{idx}] 的 id 必须是字符串")
                elif scene_id in scene_ids:
                    result.add_error(f"场景 ID 重复: {scene_id}")
                else:
                    scene_ids.add(scene_id)

            # 验证NPC列表
            if "npcs" in scene:
                npcs = scene.get("npcs")
                if not isinstance(npcs, list):
                    result.add_error(f"场景 [{idx}] 的 npcs 必须是数组")

            # 验证线索列表
            if "clues" in scene:
                clues = scene.get("clues")
                if not isinstance(clues, list):
                    result.add_error(f"场景 [{idx}] 的 clues 必须是数组")

    def _validate_npcs(self, npcs: List[Dict[str, Any]], result: ValidationResult) -> None:
        """验证NPC列表"""
        if not isinstance(npcs, list):
            result.add_error("global_npcs 必须是数组类型")
            return

        npc_ids: Set[str] = set()

        for idx, npc in enumerate(npcs):
            if not isinstance(npc, dict):
                result.add_error(f"全局NPC [{idx}] 必须是对象类型")
                continue

            if "id" not in npc:
                result.add_error(f"全局NPC [{idx}] 缺少必需字段: id")
            else:
                npc_id = npc.get("id")
                if npc_id in npc_ids:
                    result.add_error(f"全局NPC ID 重复: {npc_id}")
                else:
                    npc_ids.add(npc_id)

            if "name" not in npc:
                result.add_warning(f"全局NPC [{idx}] 缺少名称")

    def _validate_clues(self, clues: List[Dict[str, Any]], result: ValidationResult) -> None:
        """验证线索列表"""
        if not isinstance(clues, list):
            result.add_error("global_clues 必须是数组类型")
            return

        clue_ids: Set[str] = set()

        for idx, clue in enumerate(clues):
            if not isinstance(clue, dict):
                result.add_error(f"全局线索 [{idx}] 必须是对象类型")
                continue

            if "id" not in clue:
                result.add_error(f"全局线索 [{idx}] 缺少必需字段: id")
            else:
                clue_id = clue.get("id")
                if clue_id in clue_ids:
                    result.add_error(f"全局线索 ID 重复: {clue_id}")
                else:
                    clue_ids.add(clue_id)

            if "description" not in clue:
                result.add_warning(f"全局线索 [{idx}] 缺少描述信息")
```

### 3. 引用完整性验证器

```python
# backend/app/services/validators/reference_validator.py
from typing import Dict, Any, Set, List
from collections import defaultdict
from .base import BaseValidator, ValidationResult

class ReferenceValidator(BaseValidator):
    """引用完整性验证器"""

    def validate(self, data: Dict[str, Any]) -> ValidationResult:
        """
        验证引用完整性

        Args:
            data: 场景包数据

        Returns:
            ValidationResult: 验证结果
        """
        result = self._create_result()

        # 收集所有NPC和线索ID
        npc_ids = self._collect_npc_ids(data)
        clue_ids = self._collect_clue_ids(data)

        # 验证线索中的NPC引用
        self._validate_clue_npc_references(data, npc_ids, result)

        # 验证场景ID引用（如果有跨场景引用）
        self._validate_scene_references(data, result)

        return result

    def _collect_npc_ids(self, data: Dict[str, Any]) -> Set[str]:
        """收集所有NPC ID"""
        npc_ids = set()

        # 全局NPC
        if "global_npcs" in data:
            for npc in data["global_npcs"]:
                if isinstance(npc, dict) and "id" in npc:
                    npc_ids.add(npc["id"])

        # 场景内NPC
        if "scenes" in data:
            for scene in data["scenes"]:
                if isinstance(scene, dict) and "npcs" in scene:
                    for npc in scene["npcs"]:
                        if isinstance(npc, dict) and "id" in npc:
                            npc_ids.add(npc["id"])

        return npc_ids

    def _collect_clue_ids(self, data: Dict[str, Any]) -> Set[str]:
        """收集所有线索ID"""
        clue_ids = set()

        # 全局线索
        if "global_clues" in data:
            for clue in data["global_clues"]:
                if isinstance(clue, dict) and "id" in clue:
                    clue_ids.add(clue["id"])

        # 场景内线索
        if "scenes" in data:
            for scene in data["scenes"]:
                if isinstance(scene, dict) and "clues" in scene:
                    for clue in scene["clues"]:
                        if isinstance(clue, dict) and "id" in clue:
                            clue_ids.add(clue["id"])

        return clue_ids

    def _validate_clue_npc_references(
        self,
        data: Dict[str, Any],
        npc_ids: Set[str],
        result: ValidationResult
    ) -> None:
        """验证线索中对NPC的引用"""
        all_clues = []

        # 收集所有线索
        if "global_clues" in data:
            all_clues.extend(data["global_clues"])

        if "scenes" in data:
            for scene in data["scenes"]:
                if isinstance(scene, dict) and "clues" in scene:
                    all_clues.extend(scene["clues"])

        # 验证引用
        for clue in all_clues:
            if not isinstance(clue, dict):
                continue

            clue_id = clue.get("id", "unknown")
            related_npcs = clue.get("related_npcs", [])

            if not isinstance(related_npcs, list):
                result.add_error(f"线索 [{clue_id}] 的 related_npcs 必须是数组")
                continue

            for npc_id in related_npcs:
                if npc_id not in npc_ids:
                    result.add_error(
                        f"线索 [{clue_id}] 引用了不存在的NPC: {npc_id}"
                    )

    def _validate_scene_references(self, data: Dict[str, Any], result: ValidationResult) -> None:
        """验证场景间的引用（如果有）"""
        if "scenes" not in data:
            return

        scene_ids = {scene.get("id") for scene in data["scenes"] if isinstance(scene, dict)}

        # 这里可以扩展验证场景间的引用关系
        # 例如：一个场景可能引用另一个场景作为"前置场景"
```

### 4. 循环引用检测器

```python
# backend/app/services/validators/circular_reference_validator.py
from typing import Dict, Any, Set, List
from collections import defaultdict, deque
from .base import BaseValidator, ValidationResult

class CircularReferenceValidator(BaseValidator):
    """循环引用检测器"""

    def validate(self, data: Dict[str, Any]) -> ValidationResult:
        """
        检测循环引用

        Args:
            data: 场景包数据

        Returns:
            ValidationResult: 验证结果
        """
        result = self._create_result()

        # 检测线索-NPC循环引用
        self._detect_clue_npc_cycles(data, result)

        # 检测场景间循环引用（如果有前置场景等关系）
        self._detect_scene_cycles(data, result)

        return result

    def _detect_clue_npc_cycles(
        self,
        data: Dict[str, Any],
        result: ValidationResult
    ) -> None:
        """
        检测线索-NPC之间的循环引用

        构建双向图并检测环：
        - NPC -> Clues（如果NPC有相关线索）
        - Clue -> NPCs（related_npcs）
        """
        # 构建引用关系图
        clue_to_npcs: Dict[str, List[str]] = defaultdict(list)
        npc_to_clues: Dict[str, List[str]] = defaultdict(list)

        # 收集线索到NPC的引用
        all_clues = []
        if "global_clues" in data:
            all_clues.extend(data["global_clues"])
        if "scenes" in data:
            for scene in data["scenes"]:
                if isinstance(scene, dict) and "clues" in scene:
                    all_clues.extend(scene["clues"])

        for clue in all_clues:
            if not isinstance(clue, dict):
                continue
            clue_id = clue.get("id")
            if clue_id and "related_npcs" in clue:
                related_npcs = clue.get("related_npcs", [])
                if isinstance(related_npcs, list):
                    for npc_id in related_npcs:
                        clue_to_npcs[clue_id].append(npc_id)
                        npc_to_clues[npc_id].append(clue_id)

        # 检测环
        visited: Set[str] = set()
        rec_stack: Set[str] = set()

        def dfs(node: str, path: List[str]) -> bool:
            """深度优先搜索检测环"""
            visited.add(node)
            rec_stack.add(node)
            path.append(node)

            # 获取邻居节点
            neighbors = []
            if node in clue_to_npcs:
                neighbors.extend(clue_to_npcs[node])
            if node in npc_to_clues:
                neighbors.extend(npc_to_clues[node])

            for neighbor in neighbors:
                if neighbor not in visited:
                    if dfs(neighbor, path):
                        return True
                elif neighbor in rec_stack:
                    # 找到环
                    cycle_start = path.index(neighbor)
                    cycle = path[cycle_start:] + [neighbor]
                    result.add_warning(
                        f"检测到循环引用: {' -> '.join(cycle)}"
                    )
                    return True

            path.pop()
            rec_stack.remove(node)
            return False

        # 对所有节点执行DFS
        all_nodes = set(clue_to_npcs.keys()) | set(npc_to_clues.keys())
        for node in all_nodes:
            if node not in visited:
                dfs(node, [])

    def _detect_scene_cycles(
        self,
        data: Dict[str, Any],
        result: ValidationResult
    ) -> None:
        """
        检测场景间的循环引用

        假设场景可能有 "next_scenes" 或 "previous_scenes" 等字段
        """
        if "scenes" not in data:
            return

        # 构建场景引用图
        scene_graph: Dict[str, List[str]] = defaultdict(list)

        for scene in data["scenes"]:
            if not isinstance(scene, dict):
                continue
            scene_id = scene.get("id")
            if not scene_id:
                continue

            # 检查可能的引用字段
            if "next_scenes" in scene:
                next_scenes = scene.get("next_scenes", [])
                if isinstance(next_scenes, list):
                    scene_graph[scene_id].extend(next_scenes)

        # 检测环
        visited: Set[str] = set()
        rec_stack: Set[str] = set()

        def dfs(node: str, path: List[str]) -> bool:
            if node in rec_stack:
                cycle_start = path.index(node)
                cycle = path[cycle_start:] + [node]
                result.add_error(
                    f"场景间存在循环引用: {' -> '.join(cycle)}"
                )
                return True

            if node in visited:
                return False

            visited.add(node)
            rec_stack.add(node)
            path.append(node)

            for neighbor in scene_graph.get(node, []):
                if dfs(neighbor, path):
                    return True

            path.pop()
            rec_stack.remove(node)
            return False

        for scene_id in scene_graph:
            if scene_id not in visited:
                dfs(scene_id, [])
```

### 5. 验证服务类

```python
# backend/app/services/validation_service.py
from typing import Dict, Any, List
import logging

from app.services.validators.base import ValidationResult
from app.services.validators.field_validator import FieldValidator
from app.services.validators.reference_validator import ReferenceValidator
from app.services.validators.circular_reference_validator import CircularReferenceValidator

logger = logging.getLogger(__name__)

class ValidationService:
    """验证服务"""

    def __init__(self):
        self.validators = [
            FieldValidator(),
            ReferenceValidator(),
            CircularReferenceValidator(),
        ]

    def validate(self, data: Dict[str, Any]) -> ValidationResult:
        """
        执行完整验证

        Args:
            data: 待验证的场景包数据

        Returns:
            ValidationResult: 综合验证结果
        """
        final_result = ValidationResult.success()

        for validator in self.validators:
            try:
                result = validator.validate(data)
                final_result.merge(result)
            except Exception as e:
                logger.error(f"验证器 {validator.name} 执行失败: {e}")
                final_result.add_error(f"验证过程出错: {str(e)}")

        return final_result

    def validate_with_details(
        self,
        data: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        执行验证并返回详细信息

        Returns:
            Dict: 包含详细验证信息的字典
        """
        result = self.validate(data)

        return {
            "is_valid": result.is_valid,
            "errors": result.errors,
            "warnings": result.warnings,
            "info": result.info,
            "summary": self._generate_summary(result),
            "details": result.details
        }

    def _generate_summary(self, result: ValidationResult) -> str:
        """生成验证摘要"""
        error_count = len(result.errors)
        warning_count = len(result.warnings)
        info_count = len(result.info)

        if result.is_valid:
            parts = ["验证通过"]
            if warning_count > 0:
                parts.append(f"{warning_count} 个警告")
            if info_count > 0:
                parts.append(f"{info_count} 条提示")
        else:
            parts = [f"验证失败，{error_count} 个错误"]
            if warning_count > 0:
                parts.append(f"{warning_count} 个警告")

        return "，".join(parts) + "。"
```

### 6. API 路由

```python
# backend/app/api/v1/endpoints/validation.py
from fastapi import APIRouter, UploadFile, File, Depends, HTTPException
from fastapi.responses import JSONResponse

from app.services.validation_service import ValidationService
from app.services.json_parser import JSONParser
from app.api.deps import get_current_user

router = APIRouter()
validation_service = ValidationService()
parser = JSONParser()

@router.post("/validate")
async def validate_scenario(
    file: UploadFile = File(...),
    current_user = Depends(get_current_user)
):
    """
    验证场景包文件

    - **file**: 上传的JSON文件
    - 返回验证结果
    """
    try:
        # 先解析文件
        scenario = await parser.parse_upload(file)

        # 执行验证
        validation_result = validation_service.validate_with_details(
            scenario.dict()
        )

        return JSONResponse(content=validation_result)

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
```

---

## TypeScript/React 前端实现

### 1. 验证服务

```typescript
// frontend/src/services/api/validation.ts
import api from './client';

export interface ValidationResult {
  is_valid: boolean;
  errors: string[];
  warnings: string[];
  info: string[];
  summary: string;
  details: Record<string, any>;
}

class ValidationService {
  /**
   * 验证场景包文件
   */
  async validateScenario(file: File): Promise<ValidationResult> {
    const formData = new FormData();
    formData.append('file', file);

    try {
      const response = await api.post<ValidationResult>(
        '/api/v1/validation/validate',
        formData,
        {
          headers: {
            'Content-Type': 'multipart/form-data',
          },
        }
      );
      return response.data;
    } catch (error: any) {
      return {
        is_valid: false,
        errors: [error.response?.data?.detail || '验证失败'],
        warnings: [],
        info: [],
        summary: '验证失败',
        details: {},
      };
    }
  }
}

export default new ValidationService();
```

### 2. 验证结果组件

```typescript
// frontend/src/components/scenario/ValidationResult.tsx
import React from 'react';
import { Alert, Card, Collapse, Tag, Space } from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  WarningOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons';
import { ValidationResult } from '@/services/api/validation';
import './ValidationResult.css';

const { Panel } = Collapse;

interface ValidationResultProps {
  result: ValidationResult;
  showDetails?: boolean;
}

const ValidationResultDisplay: React.FC<ValidationResultProps> = ({
  result,
  showDetails = true,
}) => {
  const { is_valid, errors, warnings, info, summary } = result;

  return (
    <div className="validation-result">
      {/* 总体状态 */}
      <Alert
        message={summary}
        type={is_valid ? 'success' : 'error'}
        icon={is_valid ? <CheckCircleOutlined /> : <CloseCircleOutlined />}
        showIcon
        closable
        style={{ marginBottom: 16 }}
      />

      {/* 错误列表 */}
      {errors.length > 0 && (
        <Alert
          message={
            <Space>
              <CloseCircleOutlined />
              <span>错误 ({errors.length})</span>
            </Space>
          }
          description={
            <ul className="validation-list">
              {errors.map((error, idx) => (
                <li key={`error-${idx}`} className="validation-item error">
                  {error}
                </li>
              ))}
            </ul>
          }
          type="error"
          style={{ marginBottom: 16 }}
        />
      )}

      {/* 警告列表 */}
      {warnings.length > 0 && (
        <Alert
          message={
            <Space>
              <WarningOutlined />
              <span>警告 ({warnings.length})</span>
            </Space>
          }
          description={
            <ul className="validation-list">
              {warnings.map((warning, idx) => (
                <li key={`warning-${idx}`} className="validation-item warning">
                  {warning}
                </li>
              ))}
            </ul>
          }
          type="warning"
          style={{ marginBottom: 16 }}
        />
      )}

      {/* 信息列表 */}
      {info.length > 0 && showDetails && (
        <Collapse ghost>
          <Panel
            header={
              <Space>
                <InfoCircleOutlined />
                <span>信息 ({info.length})</span>
              </Space>
            }
            key="info"
          >
            <ul className="validation-list">
              {info.map((item, idx) => (
                <li key={`info-${idx}`} className="validation-item info">
                  {item}
                </li>
              ))}
            </ul>
          </Panel>
        </Collapse>
      )}

      {/* 详细信息 */}
      {showDetails && Object.keys(result.details).length > 0 && (
        <Card title="详细信息" size="small" style={{ marginTop: 16 }}>
          <Space direction="vertical" style={{ width: '100%' }}>
            {Object.entries(result.details).map(([key, value]) => (
              <div key={key}>
                <Tag color="blue">{key}</Tag>
                <span>{String(value)}</span>
              </div>
            ))}
          </Space>
        </Card>
      )}
    </div>
  );
};

export default ValidationResultDisplay;
```

```css
/* frontend/src/components/scenario/ValidationResult.css */
.validation-result {
  margin: 16px 0;
}

.validation-list {
  margin: 8px 0;
  padding-left: 20px;
}

.validation-item {
  padding: 4px 0;
  line-height: 1.6;
}

.validation-item.error {
  color: #ff4d4f;
}

.validation-item.warning {
  color: #faad14;
}

.validation-item.info {
  color: #1890ff;
}
```

---

## 涉及文件清单

### 新建文件

| 文件路径 | 说明 |
|---------|------|
| `/backend/app/services/validators/base.py` | 验证器基类和结果数据类 |
| `/backend/app/services/validators/field_validator.py` | 字段验证器 |
| `/backend/app/services/validators/reference_validator.py` | 引用完整性验证器 |
| `/backend/app/services/validators/circular_reference_validator.py` | 循环引用检测器 |
| `/backend/app/services/validation_service.py` | 验证服务主类 |
| `/backend/app/api/v1/endpoints/validation.py` | 验证API路由 |

| 文件路径 | 说明 |
|---------|------|
| `/frontend/src/services/api/validation.ts` | 验证服务API |
| `/frontend/src/components/scenario/ValidationResult.tsx` | 验证结果展示组件 |
| `/frontend/src/components/scenario/ValidationResult.css` | 验证结果组件样式 |

---

## 验收标准

### 功能验收

- [ ] 正确验证所有必需字段
- [ ] 正确验证字段数据类型
- [ ] 正确检测ID重复
- [ ] 正确检测NPC/线索引用完整性
- [ ] 正确检测循环引用
- [ ] 提供清晰的错误信息
- [ ] 提供有价值的警告信息

### 性能验收

- [ ] 100个场景的场景包验证时间 < 1秒
- [ ] 内存占用合理

### 代码质量验收

- [ ] 单元测试覆盖率 > 80%
- [ ] 所有验证器都有清晰的错误消息
- [ ] 代码符合项目规范

---

## 参考文档

### 内部文档

- [M4-009: JSON解析器](/Users/guochangxi/git/monika/docs/tasks/tasks-detailed/M4-009-json-parser.md)
- [M0-014: 场景格式规范](/Users/guochangxi/git/monika/docs/tasks/tasks-detailed/M0-014-scene-format.md)

### 技术文档

- [JSON Schema Validation](https://json-schema.org/understanding-json-schema/reference/validation.html)
- [Graph Cycle Detection Algorithms](https://en.wikipedia.org/wiki/Cycle_detection)

---

**创建日期**: 2026-02-06
**最后更新**: 2026-02-06
