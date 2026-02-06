# M4-009: 实现 JSON 解析器

**任务ID**: M4-009
**任务名称**: 实现 JSON 解析器
**预估时间**: 4 小时
**优先级**: P0
**依赖**: M0 (场景格式规范)
**状态**: 待开始

---

## 任务概述

实现场景包 JSON 解析器，负责解析模组上传的场景包文件，提取元数据、场景信息、NPC、线索等核心数据，为后续的验证、压缩、加密等功能提供基础数据支持。

---

## 子任务拆解

| ID | 子任务描述 | 预估时间 | 依赖 | 状态 |
|----|-----------|----------|------|------|
| M4-009-01 | 设计 JSON 解析器架构和数据模型 | 1h | - | 待开始 |
| M4-009-02 | 实现基础 JSON 文件读取和解析功能 | 1h | M4-009-01 | 待开始 |
| M4-009-03 | 实现元数据提取逻辑 | 1h | M4-009-02 | 待开始 |
| M4-009-04 | 实现场景、NPC、线索数据提取 | 1h | M4-009-03 | 待开始 |

**总预估时间**: 4 小时

---

## Python 后端实现

### 1. 数据模型定义

```python
# backend/app/models/schema.py
from pydantic import BaseModel, Field, validator
from typing import List, Optional, Dict, Any
from datetime import datetime
from enum import Enum

class SceneFormatVersion(str, Enum):
    """场景格式版本枚举"""
    V1_0 = "1.0"
    V1_1 = "1.1"

class Metadata(BaseModel):
    """场景包元数据"""
    name: str = Field(..., min_length=1, max_length=200, description="场景包名称")
    version: SceneFormatVersion = Field(default=SceneFormatVersion.V1_0, description="格式版本")
    author: str = Field(..., min_length=1, max_length=100, description="作者")
    description: Optional[str] = Field(None, max_length=2000, description="描述")
    tags: List[str] = Field(default_factory=list, description="标签列表")
    created_at: Optional[datetime] = Field(default_factory=datetime.utcnow, description="创建时间")
    min_players: Optional[int] = Field(default=3, ge=1, le=10, description="最少玩家数")
    max_players: Optional[int] = Field(default=6, ge=1, le=10, description="最多玩家数")
    difficulty: Optional[str] = Field(default="普通", description="难度等级")

    @validator('tags')
    def validate_tags(cls, v):
        """验证标签"""
        if len(v) > 10:
            raise ValueError("标签数量不能超过10个")
        return [tag.strip() for tag in v if tag.strip()]

    @validator('max_players')
    def validate_player_range(cls, v, values):
        """验证玩家数量范围"""
        if 'min_players' in values and v < values['min_players']:
            raise ValueError("最大玩家数不能小于最小玩家数")
        return v

class NPC(BaseModel):
    """NPC数据模型"""
    id: str = Field(..., description="NPC唯一标识")
    name: str = Field(..., min_length=1, description="NPC名称")
    description: Optional[str] = Field(None, description="NPC描述")
    personality: Optional[str] = Field(None, description="性格特征")
    background: Optional[str] = Field(None, description="背景故事")
    is_key: bool = Field(default=False, description="是否为关键NPC")

    @validator('id')
    def validate_id(cls, v):
        """验证ID格式"""
        if not v or not v.strip():
            raise ValueError("NPC ID不能为空")
        return v.strip()

class Clue(BaseModel):
    """线索数据模型"""
    id: str = Field(..., description="线索唯一标识")
    name: str = Field(..., min_length=1, description="线索名称")
    description: str = Field(..., description="线索描述")
    location: Optional[str] = Field(None, description="线索位置")
    is_key: bool = Field(default=False, description="是否为关键线索")
    related_npcs: List[str] = Field(default_factory=list, description="关联NPC ID列表")

class Scene(BaseModel):
    """场景数据模型"""
    id: str = Field(..., description="场景唯一标识")
    name: str = Field(..., min_length=1, description="场景名称")
    description: str = Field(..., description="场景描述")
    location: Optional[str] = Field(None, description="场景位置")
    npcs: List[NPC] = Field(default_factory=list, description="NPC列表")
    clues: List[Clue] = Field(default_factory=list, description="线索列表")
    atmosphere: Optional[str] = Field(None, description="氛围描述")

class ScenarioPackage(BaseModel):
    """场景包完整数据模型"""
    metadata: Metadata = Field(..., description="元数据")
    scenes: List[Scene] = Field(default_factory=list, description="场景列表")
    global_npcs: List[NPC] = Field(default_factory=list, description="全局NPC列表")
    global_clues: List[Clue] = Field(default_factory=list, description="全局线索列表")
    custom_data: Dict[str, Any] = Field(default_factory=dict, description="自定义数据")

    class Config:
        """Pydantic配置"""
        json_encoders = {
            datetime: lambda v: v.isoformat()
        }

    def get_all_npcs(self) -> Dict[str, NPC]:
        """获取所有NPC（包括场景内和全局）"""
        npcs = {npc.id: npc for npc in self.global_npcs}
        for scene in self.scenes:
            for npc in scene.npcs:
                npcs[npc.id] = npc
        return npcs

    def get_all_clues(self) -> Dict[str, Clue]:
        """获取所有线索（包括场景内和全局）"""
        clues = {clue.id: clue for clue in self.global_clues}
        for scene in self.scenes:
            for clue in scene.clues:
                clues[clue.id] = clue
        return clues
```

### 2. JSON 解析器实现

```python
# backend/app/services/json_parser.py
import json
import logging
from pathlib import Path
from typing import Union, Dict, Any
from fastapi import UploadFile, HTTPException

from app.models.schema import ScenarioPackage
from app.core.exceptions import ParseError, ValidationError

logger = logging.getLogger(__name__)

class JSONParser:
    """JSON解析器服务"""

    def __init__(self):
        self.supported_versions = ["1.0", "1.1"]

    async def parse_file(self, file_path: Union[str, Path]) -> ScenarioPackage:
        """
        解析JSON文件

        Args:
            file_path: JSON文件路径

        Returns:
            ScenarioPackage: 解析后的场景包对象

        Raises:
            ParseError: 解析失败时抛出
        """
        try:
            file_path = Path(file_path)
            if not file_path.exists():
                raise ParseError(f"文件不存在: {file_path}")

            if file_path.suffix.lower() != '.json':
                raise ParseError(f"不支持的文件格式: {file_path.suffix}")

            with open(file_path, 'r', encoding='utf-8') as f:
                data = json.load(f)

            return self.parse_dict(data)

        except json.JSONDecodeError as e:
            logger.error(f"JSON解析失败: {e}")
            raise ParseError(f"JSON格式错误: {str(e)}")
        except Exception as e:
            logger.error(f"文件读取失败: {e}")
            raise ParseError(f"文件读取失败: {str(e)}")

    async def parse_upload(self, upload_file: UploadFile) -> ScenarioPackage:
        """
        解析上传的文件

        Args:
            upload_file: FastAPI UploadFile对象

        Returns:
            ScenarioPackage: 解析后的场景包对象

        Raises:
            ParseError: 解析失败时抛出
        """
        try:
            # 验证文件类型
            if not upload_file.filename.endswith('.json'):
                raise ParseError(f"不支持的文件格式，仅支持 .json 文件")

            # 读取文件内容
            content = await upload_file.read()
            if not content:
                raise ParseError("文件内容为空")

            # 解析JSON
            try:
                data = json.loads(content.decode('utf-8'))
            except json.JSONDecodeError as e:
                raise ParseError(f"JSON格式错误: {str(e)}")

            return self.parse_dict(data)

        except ParseError:
            raise
        except Exception as e:
            logger.error(f"文件解析失败: {e}")
            raise ParseError(f"文件解析失败: {str(e)}")
        finally:
            await upload_file.close()

    def parse_dict(self, data: Dict[str, Any]) -> ScenarioPackage:
        """
        解析字典数据

        Args:
            data: JSON解析后的字典数据

        Returns:
            ScenarioPackage: 解析后的场景包对象

        Raises:
            ParseError: 解析失败时抛出
        """
        try:
            # 验证基本结构
            if not isinstance(data, dict):
                raise ParseError("根节点必须是对象")

            if 'metadata' not in data:
                raise ParseError("缺少必需字段: metadata")

            # 使用Pydantic进行数据验证和解析
            scenario = ScenarioPackage(**data)

            logger.info(f"成功解析场景包: {scenario.metadata.name}")
            return scenario

        except Exception as e:
            logger.error(f"数据解析失败: {e}")
            raise ParseError(f"数据解析失败: {str(e)}")

    def validate_structure(self, data: Dict[str, Any]) -> tuple[bool, list[str]]:
        """
        验证JSON结构

        Args:
            data: JSON数据字典

        Returns:
            tuple: (是否有效, 错误信息列表)
        """
        errors = []

        # 检查必需字段
        required_fields = ['metadata']
        for field in required_fields:
            if field not in data:
                errors.append(f"缺少必需字段: {field}")

        # 检查元数据
        if 'metadata' in data:
            metadata = data['metadata']
            if not isinstance(metadata, dict):
                errors.append("metadata 必须是对象")
            else:
                required_metadata = ['name', 'author']
                for field in required_metadata:
                    if field not in metadata:
                        errors.append(f"metadata 缺少必需字段: {field}")

        # 检查场景列表
        if 'scenes' in data and data['scenes'] is not None:
            if not isinstance(data['scenes'], list):
                errors.append("scenes 必须是数组")
            else:
                for idx, scene in enumerate(data['scenes']):
                    if not isinstance(scene, dict):
                        errors.append(f"场景 {idx} 必须是对象")
                    elif 'id' not in scene:
                        errors.append(f"场景 {idx} 缺少 id 字段")

        return len(errors) == 0, errors

    async def parse_with_validation(self, file_path: Union[str, Path]) -> tuple[ScenarioPackage, list[str]]:
        """
        解析文件并进行结构验证

        Args:
            file_path: JSON文件路径

        Returns:
            tuple: (场景包对象, 警告信息列表)
        """
        # 先解析
        scenario = await self.parse_file(file_path)

        # 收集警告信息
        warnings = []

        # 检查是否有场景
        if not scenario.scenes:
            warnings.append("场景包不包含任何场景")

        # 检查NPC和线索引用
        all_npcs = scenario.get_all_npcs()
        all_clues = scenario.get_all_clues()

        for scene in scenario.scenes:
            for clue in scene.clues:
                for npc_id in clue.related_npcs:
                    if npc_id not in all_npcs:
                        warnings.append(f"线索 {clue.id} 引用了不存在的NPC: {npc_id}")

        return scenario, warnings
```

### 3. 异常类定义

```python
# backend/app/core/exceptions.py
class ParseError(Exception):
    """解析错误"""
    def __init__(self, message: str, details: dict = None):
        self.message = message
        self.details = details or {}
        super().__init__(self.message)

class ValidationError(Exception):
    """验证错误"""
    def __init__(self, message: str, field: str = None):
        self.message = message
        self.field = field
        super().__init__(self.message)
```

### 4. API 路由

```python
# backend/app/api/v1/endpoints/parser.py
from fastapi import APIRouter, UploadFile, File, HTTPException, Depends
from fastapi.responses import JSONResponse
from typing import Dict, Any

from app.services.json_parser import JSONParser
from app.core.exceptions import ParseError
from app.api.deps import get_current_user

router = APIRouter()
parser = JSONParser()

@router.post("/parse", response_model=Dict[str, Any])
async def parse_scenario_file(
    file: UploadFile = File(...),
    current_user = Depends(get_current_user)
):
    """
    解析场景包JSON文件

    - **file**: 上传的JSON文件
    - 返回解析后的场景包数据
    """
    try:
        scenario = await parser.parse_upload(file)
        return {
            "success": True,
            "data": scenario.dict(),
            "message": "解析成功"
        }
    except ParseError as e:
        raise HTTPException(status_code=400, detail=e.message)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.post("/validate", response_model=Dict[str, Any])
async def validate_scenario_file(
    file: UploadFile = File(...),
    current_user = Depends(get_current_user)
):
    """
    解析并验证场景包文件

    - **file**: 上传的JSON文件
    - 返回解析结果和验证信息
    """
    try:
        # 先读取文件内容
        content = await file.read()
        data = __import__('json').loads(content.decode('utf-8'))

        # 验证结构
        is_valid, errors = parser.validate_structure(data)

        # 尝试解析
        scenario = None
        warnings = []
        if is_valid:
            try:
                scenario, warnings = await parser.parse_with_validation(data)
            except Exception as e:
                is_valid = False
                errors.append(str(e))

        return {
            "success": is_valid,
            "errors": errors,
            "warnings": warnings,
            "data": scenario.dict() if scenario else None
        }

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
    finally:
        await file.close()
```

---

## TypeScript/React 前端实现

### 1. 类型定义

```typescript
// frontend/src/types/scenario.ts
export enum SceneFormatVersion {
  V1_0 = "1.0",
  V1_1 = "1.1"
}

export interface Metadata {
  name: string;
  version: SceneFormatVersion;
  author: string;
  description?: string;
  tags: string[];
  created_at?: string;
  min_players?: number;
  max_players?: number;
  difficulty?: string;
}

export interface NPC {
  id: string;
  name: string;
  description?: string;
  personality?: string;
  background?: string;
  is_key: boolean;
}

export interface Clue {
  id: string;
  name: string;
  description: string;
  location?: string;
  is_key: boolean;
  related_npcs: string[];
}

export interface Scene {
  id: string;
  name: string;
  description: string;
  location?: string;
  npcs: NPC[];
  clues: Clue[];
  atmosphere?: string;
}

export interface ScenarioPackage {
  metadata: Metadata;
  scenes: Scene[];
  global_npcs: NPC[];
  global_clues: Clue[];
  custom_data?: Record<string, any>;
}

export interface ParseResult {
  success: boolean;
  data?: ScenarioPackage;
  errors: string[];
  warnings: string[];
}
```

### 2. API 服务

```typescript
// frontend/src/services/api/scenario.ts
import api from './client';
import { ScenarioPackage, ParseResult } from '@/types/scenario';

class ScenarioService {
  /**
   * 上传并解析场景包文件
   */
  async parseScenario(file: File): Promise<ParseResult> {
    const formData = new FormData();
    formData.append('file', file);

    try {
      const response = await api.post<ParseResult>('/api/v1/parser/parse', formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      });
      return response.data;
    } catch (error: any) {
      return {
        success: false,
        errors: [error.response?.data?.detail || '解析失败'],
        warnings: [],
      };
    }
  }

  /**
   * 验证场景包文件
   */
  async validateScenario(file: File): Promise<ParseResult> {
    const formData = new FormData();
    formData.append('file', file);

    try {
      const response = await api.post<ParseResult>('/api/v1/parser/validate', formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      });
      return response.data;
    } catch (error: any) {
      return {
        success: false,
        errors: [error.response?.data?.detail || '验证失败'],
        warnings: [],
      };
    }
  }

  /**
   * 本地JSON文件解析（用于快速预览）
   */
  async parseLocalFile(file: File): Promise<ParseResult> {
    return new Promise((resolve) => {
      const reader = new FileReader();

      reader.onload = (e) => {
        try {
          const content = e.target?.result as string;
          const data = JSON.parse(content);

          // 基础结构验证
          const errors: string[] = [];
          const warnings: string[] = [];

          if (!data.metadata) {
            errors.push('缺少必需字段: metadata');
          } else {
            if (!data.metadata.name) errors.push('缺少场景包名称');
            if (!data.metadata.author) errors.push('缺少作者信息');
          }

          if (!data.scenes || !Array.isArray(data.scenes)) {
            warnings.push('场景列表为空');
          }

          if (errors.length > 0) {
            resolve({ success: false, errors, warnings });
          } else {
            resolve({ success: true, data, errors: [], warnings });
          }
        } catch (error) {
          resolve({
            success: false,
            errors: ['JSON格式错误'],
            warnings: [],
          });
        }
      };

      reader.onerror = () => {
        resolve({
          success: false,
          errors: ['文件读取失败'],
          warnings: [],
        });
      };

      reader.readAsText(file);
    });
  }
}

export default new ScenarioService();
```

### 3. React 组件

```typescript
// frontend/src/components/scenario/ScenarioUploader.tsx
import React, { useState, useCallback } from 'react';
import { Upload, message, Card, Alert, Spin, Button, Space } from 'antd';
import { InboxOutlined, FileTextOutlined, CheckCircleOutlined } from '@ant-design/icons';
import type { UploadFile, UploadChangeParam } from 'antd/es/upload/interface';
import scenarioService from '@/services/api/scenario';
import { ScenarioPackage } from '@/types/scenario';
import './ScenarioUploader.css';

const { Dragger } = Upload;

interface ScenarioUploaderProps {
  onParsed?: (data: ScenarioPackage) => void;
  showPreview?: boolean;
}

const ScenarioUploader: React.FC<ScenarioUploaderProps> = ({ onParsed, showPreview = true }) => {
  const [loading, setLoading] = useState(false);
  const [parsedData, setParsedData] = useState<ScenarioPackage | null>(null);
  const [errors, setErrors] = useState<string[]>([]);
  const [warnings, setWarnings] = useState<string[]>([]);

  const handleParse = useCallback(async (file: File) => {
    setLoading(true);
    setErrors([]);
    setWarnings([]);
    setParsedData(null);

    try {
      // 本地快速预览解析
      const localResult = await scenarioService.parseLocalFile(file);

      if (!localResult.success) {
        setErrors(localResult.errors);
        setWarnings(localResult.warnings);
        message.error('文件解析失败');
        return;
      }

      // 服务端完整解析
      const serverResult = await scenarioService.parseScenario(file);

      if (serverResult.success && serverResult.data) {
        setParsedData(serverResult.data);
        setWarnings(serverResult.warnings);

        if (serverResult.warnings.length > 0) {
          message.warning(`解析成功，但有 ${serverResult.warnings.length} 个警告`);
        } else {
          message.success('解析成功');
        }

        onParsed?.(serverResult.data);
      } else {
        setErrors(serverResult.errors);
        setWarnings(serverResult.warnings);
        message.error('文件验证失败');
      }
    } catch (error) {
      message.error('解析过程出错');
      setErrors(['未知错误']);
    } finally {
      setLoading(false);
    }
  }, [onParsed]);

  const uploadProps = {
    name: 'file',
    multiple: false,
    accept: '.json',
    showUploadList: false,
    beforeUpload: (file: File) => {
      if (!file.name.endsWith('.json')) {
        message.error('仅支持 .json 格式的文件');
        return Upload.LIST_IGNORE;
      }
      handleParse(file);
      return false; // 阻止自动上传
    },
  };

  return (
    <div className="scenario-uploader">
      <Card title="上传场景包" bordered={false}>
        <Dragger {...uploadProps}>
          <p className="ant-upload-drag-icon">
            <InboxOutlined />
          </p>
          <p className="ant-upload-text">点击或拖拽文件到此区域上传</p>
          <p className="ant-upload-hint">仅支持 .json 格式的场景包文件</p>
        </Dragger>

        {loading && (
          <div style={{ textAlign: 'center', padding: '20px' }}>
            <Spin size="large" tip="正在解析..." />
          </div>
        )}

        {errors.length > 0 && (
          <Alert
            type="error"
            message="解析错误"
            description={
              <ul style={{ margin: 0, paddingLeft: 20 }}>
                {errors.map((error, idx) => (
                  <li key={idx}>{error}</li>
                ))}
              </ul>
            }
            showIcon
            closable
            style={{ marginTop: 16 }}
          />
        )}

        {warnings.length > 0 && (
          <Alert
            type="warning"
            message="解析警告"
            description={
              <ul style={{ margin: 0, paddingLeft: 20 }}>
                {warnings.map((warning, idx) => (
                  <li key={idx}>{warning}</li>
                ))}
              </ul>
            }
            showIcon
            closable
            style={{ marginTop: 16 }}
          />
        )}

        {parsedData && showPreview && (
          <Card
            type="inner"
            title={
              <Space>
                <CheckCircleOutlined style={{ color: '#52c41a' }} />
                解析成功
              </Space>
            }
            style={{ marginTop: 16 }}
          >
            <ScenarioPreview data={parsedData} />
          </Card>
        )}
      </Card>
    </div>
  );
};

const ScenarioPreview: React.FC<{ data: ScenarioPackage }> = ({ data }) => {
  return (
    <div className="scenario-preview">
      <h3>{data.metadata.name}</h3>
      <p><strong>作者:</strong> {data.metadata.author}</p>
      <p><strong>版本:</strong> {data.metadata.version}</p>
      <p><strong>难度:</strong> {data.metadata.difficulty || '普通'}</p>
      <p><strong>玩家数:</strong> {data.metadata.min_players}-{data.metadata.max_players}</p>
      {data.metadata.description && (
        <p><strong>描述:</strong> {data.metadata.description}</p>
      )}
      <p><strong>场景数量:</strong> {data.scenes.length}</p>
      <p><strong>NPC数量:</strong> {data.global_npcs.length}</p>
      <p><strong>线索数量:</strong> {data.global_clues.length}</p>
      {data.metadata.tags.length > 0 && (
        <p><strong>标签:</strong> {data.metadata.tags.join(', ')}</p>
      )}
    </div>
  );
};

export default ScenarioUploader;
```

```css
/* frontend/src/components/scenario/ScenarioUploader.css */
.scenario-uploader {
  max-width: 800px;
  margin: 0 auto;
}

.scenario-preview {
  padding: 16px;
}

.scenario-preview h3 {
  margin-top: 0;
  color: #1890ff;
}

.scenario-preview p {
  margin: 8px 0;
  line-height: 1.6;
}
```

---

## 涉及文件清单

### 新建文件

| 文件路径 | 说明 |
|---------|------|
| `/backend/app/models/schema.py` | 数据模型定义（Metadata, NPC, Clue, Scene, ScenarioPackage） |
| `/backend/app/services/json_parser.py` | JSON解析器服务实现 |
| `/backend/app/core/exceptions.py` | 自定义异常类（ParseError, ValidationError） |
| `/backend/app/api/v1/endpoints/parser.py` | 解析器API路由 |

| 文件路径 | 说明 |
|---------|------|
| `/frontend/src/types/scenario.ts` | TypeScript类型定义 |
| `/frontend/src/services/api/scenario.ts` | 场景包API服务 |
| `/frontend/src/components/scenario/ScenarioUploader.tsx` | 上传组件 |
| `/frontend/src/components/scenario/ScenarioUploader.css` | 上传组件样式 |

### 修改文件

| 文件路径 | 修改内容 |
|---------|---------|
| `/backend/app/api/v1/router.py` | 添加解析器路由注册 |
| `/frontend/src/router/index.tsx` | 添加上传页面路由 |

---

## 验收标准

### 功能验收

- [ ] 能够成功解析符合规范的场景包JSON文件
- [ ] 正确提取元数据（名称、作者、版本、描述等）
- [ ] 正确提取场景列表及场景内数据
- [ ] 正确提取NPC和线索数据
- [ ] 支持全局和场景级别的NPC/线索
- [ ] 提供详细的错误提示信息
- [ ] 提供数据验证警告信息

### 性能验收

- [ ] 单个10MB的JSON文件解析时间 < 2秒
- [ ] 支持100+场景的大型场景包解析
- [ ] 内存占用合理（100MB文件 < 500MB内存）

### 异常处理验收

- [ ] 无效JSON格式返回明确错误
- [ ] 缺少必需字段返回明确错误
- [ ] 数据类型不匹配返回明确错误
- [ ] 文件读取失败返回明确错误

---

## 参考文档

### 内部文档

- [M0-014: 场景格式规范](/Users/guochangxi/git/monika/docs/tasks/tasks-detailed/M0-014-scene-format.md)
- [M0-015: 元数据规范](/Users/guochangxi/git/monika/docs/tasks/tasks-detailed/M0-015-metadata.md)
- [05-m4-resource-web.md](/Users/guochangxi/git/monika/docs/tasks/05-m4-resource-web.md)

### 技术文档

- [Pydantic 官方文档](https://docs.pydantic.dev/)
- [FastAPI 文件上传文档](https://fastapi.tiangolo.com/tutorial/request-files/)
- [JSON Schema 规范](https://json-schema.org/)
- [Ant Design Upload 组件](https://ant.design/components/upload-cn/)

### 示例文件

场景包JSON示例结构：
```json
{
  "metadata": {
    "name": "示例场景包",
    "version": "1.0",
    "author": "作者名",
    "description": "场景描述",
    "tags": ["恐怖", "现代"],
    "min_players": 3,
    "max_players": 6,
    "difficulty": "普通"
  },
  "scenes": [
    {
      "id": "scene_001",
      "name": "开场场景",
      "description": "场景描述",
      "npcs": [
        {
          "id": "npc_001",
          "name": "NPC名称",
          "description": "NPC描述",
          "is_key": true
        }
      ],
      "clues": [
        {
          "id": "clue_001",
          "name": "线索名称",
          "description": "线索描述",
          "is_key": false
        }
      ]
    }
  ],
  "global_npcs": [],
  "global_clues": []
}
```

---

**创建日期**: 2026-02-06
**最后更新**: 2026-02-06
