# M4-011: 实现场景包压缩

**任务ID**: M4-011
**任务名称**: 实现场景包压缩
**预估时间**: 4 小时
**优先级**: P1
**依赖**: M4-010 (场景包验证)
**状态**: 待开始

---

## 任务概述

实现场景包的压缩功能，将验证通过的 JSON 场景包文件压缩为 `.scenario` 格式的压缩包，支持多种压缩算法（gzip、bz2、lzma），并提供相应的解压缩功能，以减少存储空间占用和加快传输速度。

---

## 子任务拆解

| ID | 子任务描述 | 预估时间 | 依赖 | 状态 |
|----|-----------|----------|------|------|
| M4-011-01 | 设计压缩服务架构和接口定义 | 0.5h | M4-010 | 待开始 |
| M4-011-02 | 实现 JSON 到压缩包的转换功能 | 1.5h | M4-011-01 | 待开始 |
| M4-011-03 | 实现压缩包到 JSON 的解压功能 | 1h | M4-011-02 | 待开始 |
| M4-011-04 | 实现多算法支持（gzip/bz2/lzma） | 0.5h | M4-011-02 | 待开始 |
| M4-011-05 | 实现压缩率统计和元数据管理 | 0.5h | M4-011-04 | 待开始 |

**总预估时间**: 4 小时

---

## Python 后端实现

### 1. 压缩服务配置

```python
# backend/app/core/config.py
from pydantic import BaseSettings

class CompressionSettings(BaseSettings):
    """压缩配置"""

    # 默认压缩算法
    default_algorithm: str = "gzip"

    # 支持的压缩算法
    supported_algorithms: list = ["gzip", "bz2", "lzma"]

    # 压缩级别 (0-9)
    compression_level: int = 6

    # 最大文件大小 (100MB)
    max_file_size: int = 100 * 1024 * 1024

    # 压缩文件扩展名
    compressed_extension: str = ".scenario"

    class Config:
        env_prefix = "COMPRESSION_"

compression_settings = CompressionSettings()
```

### 2. 压缩算法枚举

```python
# backend/app/models/compression.py
from enum import Enum
from typing import Dict, Any
from dataclasses import dataclass
import time

class CompressionAlgorithm(str, Enum):
    """压缩算法枚举"""
    GZIP = "gzip"
    BZ2 = "bz2"
    LZMA = "lzma"

    @property
    def extension(self) -> str:
        """获取文件扩展名"""
        return {
            CompressionAlgorithm.GZIP: ".gz",
            CompressionAlgorithm.BZ2: ".bz2",
            CompressionAlgorithm.LZMA: ".xz",
        }[self]

    @property
    def mime_type(self) -> str:
        """获取MIME类型"""
        return {
            CompressionAlgorithm.GZIP: "application/gzip",
            CompressionAlgorithm.BZ2: "application/x-bzip2",
            CompressionAlgorithm.LZMA: "application/x-xz",
        }[self]

@dataclass
class CompressionResult:
    """压缩结果"""
    success: bool
    original_size: int
    compressed_size: int
    compression_ratio: float
    algorithm: CompressionAlgorithm
    duration: float  # 压缩耗时（秒）
    file_path: str
    metadata: Dict[str, Any]

    @classmethod
    def create(
        cls,
        original_size: int,
        compressed_size: int,
        algorithm: CompressionAlgorithm,
        file_path: str,
        duration: float
    ) -> "CompressionResult":
        """创建压缩结果"""
        compression_ratio = (1 - compressed_size / original_size) * 100 if original_size > 0 else 0

        return cls(
            success=True,
            original_size=original_size,
            compressed_size=compressed_size,
            compression_ratio=round(compression_ratio, 2),
            algorithm=algorithm,
            duration=round(duration, 3),
            file_path=file_path,
            metadata={}
        )

@dataclass
class DecompressionResult:
    """解压结果"""
    success: bool
    file_size: int
    algorithm: CompressionAlgorithm
    duration: float
    file_path: str
```

### 3. 压缩服务实现

```python
# backend/app/services/compression_service.py
import gzip
import bz2
import lzma
import json
import shutil
import logging
from pathlib import Path
from typing import Union, Optional
import time

from app.models.compression import (
    CompressionAlgorithm,
    CompressionResult,
    DecompressionResult
)
from app.core.config import compression_settings
from app.core.exceptions import ParseError

logger = logging.getLogger(__name__)

class CompressionService:
    """压缩服务"""

    def __init__(self):
        self.default_algorithm = CompressionAlgorithm(compression_settings.default_algorithm)

    def compress_json(
        self,
        json_data: Union[dict, str],
        output_path: Union[str, Path],
        algorithm: Optional[CompressionAlgorithm] = None
    ) -> CompressionResult:
        """
        压缩JSON数据到文件

        Args:
            json_data: JSON数据（字典或JSON字符串）
            output_path: 输出文件路径
            algorithm: 压缩算法，默认使用配置的默认算法

        Returns:
            CompressionResult: 压缩结果

        Raises:
            ParseError: 压缩失败时抛出
        """
        algorithm = algorithm or self.default_algorithm
        output_path = Path(output_path)

        try:
            start_time = time.time()

            # 转换为JSON字符串
            if isinstance(json_data, dict):
                json_str = json.dumps(json_data, ensure_ascii=False, indent=2)
            else:
                json_str = json_data

            original_size = len(json_str.encode('utf-8'))

            # 根据算法选择压缩方法
            if algorithm == CompressionAlgorithm.GZIP:
                compressed_data = self._compress_gzip(json_str)
            elif algorithm == CompressionAlgorithm.BZ2:
                compressed_data = self._compress_bz2(json_str)
            elif algorithm == CompressionAlgorithm.LZMA:
                compressed_data = self._compress_lzma(json_str)
            else:
                raise ParseError(f"不支持的压缩算法: {algorithm}")

            # 确保输出目录存在
            output_path.parent.mkdir(parents=True, exist_ok=True)

            # 写入压缩文件
            with open(output_path, 'wb') as f:
                f.write(compressed_data)

            duration = time.time() - start_time
            compressed_size = len(compressed_data)

            logger.info(
                f"压缩成功: {original_size} -> {compressed_size} bytes "
                f"({algorithm.value}, {duration:.3f}s)"
            )

            return CompressionResult.create(
                original_size=original_size,
                compressed_size=compressed_size,
                algorithm=algorithm,
                file_path=str(output_path),
                duration=duration
            )

        except Exception as e:
            logger.error(f"压缩失败: {e}")
            raise ParseError(f"压缩失败: {str(e)}")

    def decompress_to_json(
        self,
        file_path: Union[str, Path],
        output_path: Optional[Union[str, Path]] = None
    ) -> DecompressionResult:
        """
        解压文件到JSON

        Args:
            file_path: 压缩文件路径
            output_path: 可选的输出JSON文件路径

        Returns:
            DecompressionResult: 解压结果

        Raises:
            ParseError: 解压失败时抛出
        """
        file_path = Path(file_path)

        if not file_path.exists():
            raise ParseError(f"文件不存在: {file_path}")

        try:
            start_time = time.time()

            # 检测压缩算法
            algorithm = self._detect_algorithm(file_path)

            # 读取压缩文件
            with open(file_path, 'rb') as f:
                compressed_data = f.read()

            # 根据算法解压
            if algorithm == CompressionAlgorithm.GZIP:
                json_str = self._decompress_gzip(compressed_data)
            elif algorithm == CompressionAlgorithm.BZ2:
                json_str = self._decompress_bz2(compressed_data)
            elif algorithm == CompressionAlgorithm.LZMA:
                json_str = self._decompress_lzma(compressed_data)
            else:
                raise ParseError(f"无法检测压缩算法: {file_path.suffix}")

            # 验证JSON格式
            json_data = json.loads(json_str)

            duration = time.time() - start_time
            file_size = len(compressed_data)

            # 如果指定了输出路径，保存JSON文件
            if output_path:
                output_path = Path(output_path)
                output_path.parent.mkdir(parents=True, exist_ok=True)
                with open(output_path, 'w', encoding='utf-8') as f:
                    f.write(json_str)

            logger.info(
                f"解压成功: {file_size} bytes "
                f"({algorithm.value}, {duration:.3f}s)"
            )

            return DecompressionResult(
                success=True,
                file_size=file_size,
                algorithm=algorithm,
                duration=round(duration, 3),
                file_path=str(output_path) if output_path else "",
            )

        except json.JSONDecodeError as e:
            raise ParseError(f"JSON格式错误: {str(e)}")
        except Exception as e:
            logger.error(f"解压失败: {e}")
            raise ParseError(f"解压失败: {str(e)}")

    def _compress_gzip(self, data: str) -> bytes:
        """使用gzip压缩"""
        return gzip.compress(
            data.encode('utf-8'),
            compresslevel=compression_settings.compression_level
        )

    def _decompress_gzip(self, data: bytes) -> str:
        """使用gzip解压"""
        return gzip.decompress(data).decode('utf-8')

    def _compress_bz2(self, data: str) -> bytes:
        """使用bz2压缩"""
        return bz2.compress(
            data.encode('utf-8'),
            compresslevel=compression_settings.compression_level
        )

    def _decompress_bz2(self, data: bytes) -> str:
        """使用bz2解压"""
        return bz2.decompress(data).decode('utf-8')

    def _compress_lzma(self, data: str) -> bytes:
        """使用lzma压缩"""
        return lzma.compress(
            data.encode('utf-8'),
            preset=compression_settings.compression_level
        )

    def _decompress_lzma(self, data: bytes) -> str:
        """使用lzma解压"""
        return lzma.decompress(data).decode('utf-8')

    def _detect_algorithm(self, file_path: Path) -> CompressionAlgorithm:
        """根据文件扩展名检测压缩算法"""
        suffix_map = {
            '.gz': CompressionAlgorithm.GZIP,
            '.bz2': CompressionAlgorithm.BZ2,
            '.xz': CompressionAlgorithm.LZMA,
        }

        suffix = file_path.suffix.lower()
        if suffix in suffix_map:
            return suffix_map[suffix]

        # 尝试自动检测
        try:
            with open(file_path, 'rb') as f:
                magic = f.read(10)

            if magic.startswith(b'\x1f\x8b'):
                return CompressionAlgorithm.GZIP
            elif magic.startswith(b'BZh'):
                return CompressionAlgorithm.BZ2
            elif magic.startswith(b'\xfd7zXZ\x00'):
                return CompressionAlgorithm.LZMA

        except Exception:
            pass

        raise ParseError(f"无法检测压缩算法: {file_path}")

    def create_scenario_package(
        self,
        json_file_path: Union[str, Path],
        output_dir: Optional[Union[str, Path]] = None,
        algorithm: Optional[CompressionAlgorithm] = None
    ) -> CompressionResult:
        """
        创建场景包（.scenario文件）

        Args:
            json_file_path: JSON源文件路径
            output_dir: 输出目录，默认为源文件同目录
            algorithm: 压缩算法

        Returns:
            CompressionResult: 压缩结果
        """
        json_file_path = Path(json_file_path)

        if not json_file_path.exists():
            raise ParseError(f"JSON文件不存在: {json_file_path}")

        # 读取JSON数据
        with open(json_file_path, 'r', encoding='utf-8') as f:
            json_data = json.load(f)

        # 确定输出路径
        if output_dir is None:
            output_dir = json_file_path.parent
        else:
            output_dir = Path(output_dir)

        # 生成输出文件名
        base_name = json_file_path.stem
        algorithm = algorithm or self.default_algorithm
        output_filename = f"{base_name}{compression_settings.compressed_extension}"
        output_path = output_dir / output_filename

        # 压缩
        return self.compress_json(json_data, output_path, algorithm)

    def get_algorithm_info(self) -> dict:
        """获取所有压缩算法的信息"""
        return {
            "default": compression_settings.default_algorithm,
            "supported": [
                {
                    "name": algo.value,
                    "extension": algo.extension,
                    "mime_type": algo.mime_type,
                }
                for algo in CompressionAlgorithm
            ]
        }
```

### 4. API 路由

```python
# backend/app/api/v1/endpoints/compression.py
from fastapi import APIRouter, UploadFile, File, Form, Depends, HTTPException
from fastapi.responses import FileResponse
from typing import Optional

from app.services.compression_service import CompressionService
from app.models.compression import CompressionAlgorithm
from app.api.deps import get_current_user

router = APIRouter()
compression_service = CompressionService()

@router.post("/compress")
async def compress_scenario(
    file: UploadFile = File(...),
    algorithm: Optional[str] = Form(None),
    current_user = Depends(get_current_user)
):
    """
    压缩场景包

    - **file**: 上传的JSON文件
    - **algorithm**: 压缩算法 (gzip/bz2/lzma)，可选
    - 返回压缩后的文件
    """
    try:
        # 读取JSON内容
        content = await file.read()
        json_data = __import__('json').loads(content.decode('utf-8'))

        # 确定算法
        algo = None
        if algorithm:
            try:
                algo = CompressionAlgorithm(algorithm)
            except ValueError:
                raise HTTPException(status_code=400, detail=f"不支持的压缩算法: {algorithm}")

        # 生成临时文件路径
        import tempfile
        with tempfile.NamedTemporaryFile(suffix='.scenario', delete=False) as tmp:
            output_path = tmp.name

        # 压缩
        result = compression_service.compress_json(json_data, output_path, algo)

        # 返回文件
        response = FileResponse(
            path=result.file_path,
            media_type=result.algorithm.mime_type,
            filename=file.filename.replace('.json', '.scenario')
        )

        # 添加压缩信息到响应头
        response.headers["X-Original-Size"] = str(result.original_size)
        response.headers["X-Compressed-Size"] = str(result.compressed_size)
        response.headers["X-Compression-Ratio"] = f"{result.compression_ratio}%"
        response.headers["X-Compression-Algorithm"] = result.algorithm.value
        response.headers["X-Compression-Duration"] = f"{result.duration}s"

        return response

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.post("/decompress")
async def decompress_scenario(
    file: UploadFile = File(...),
    current_user = Depends(get_current_user)
):
    """
    解压场景包

    - **file**: 上传的压缩文件
    - 返回解压后的JSON数据
    """
    try:
        # 保存临时文件
        import tempfile
        with tempfile.NamedTemporaryFile(delete=False) as tmp:
            content = await file.read()
            tmp.write(content)
            tmp_path = tmp.name

        # 解压
        result = compression_service.decompress_to_json(tmp_path)

        # 读取JSON数据
        with open(result.file_path, 'r', encoding='utf-8') as f:
            json_data = __import__('json').load(f)

        # 清理临时文件
        import os
        os.unlink(tmp_path)
        if result.file_path:
            os.unlink(result.file_path)

        return {
            "success": True,
            "data": json_data,
            "algorithm": result.algorithm.value,
            "duration": result.duration,
        }

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/algorithms")
async def get_algorithms():
    """获取支持的压缩算法列表"""
    return compression_service.get_algorithm_info()
```

---

## TypeScript/React 前端实现

### 1. 压缩服务

```typescript
// frontend/src/services/api/compression.ts
import api from './client';

export enum CompressionAlgorithm {
  GZIP = 'gzip',
  BZ2 = 'bz2',
  LZMA = 'lzma',
}

export interface CompressionResult {
  success: boolean;
  original_size: number;
  compressed_size: number;
  compression_ratio: number;
  algorithm: string;
  duration: number;
  file_path: string;
}

export interface DecompressionResult {
  success: boolean;
  data: any;
  algorithm: string;
  duration: number;
}

export interface AlgorithmInfo {
  name: string;
  extension: string;
  mime_type: string;
}

class CompressionService {
  /**
   * 压缩场景包
   */
  async compressScenario(
    file: File,
    algorithm?: CompressionAlgorithm
  ): Promise<Blob> {
    const formData = new FormData();
    formData.append('file', file);
    if (algorithm) {
      formData.append('algorithm', algorithm);
    }

    try {
      const response = await api.post('/api/v1/compression/compress', formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
        responseType: 'blob',
      });

      // 从响应头获取压缩信息
      const compressionResult: CompressionResult = {
        success: true,
        original_size: parseInt(response.headers['x-original-size'] || '0'),
        compressed_size: parseInt(response.headers['x-compressed-size'] || '0'),
        compression_ratio: parseFloat(
          response.headers['x-compression-ratio'] || '0'
        ),
        algorithm: response.headers['x-compression-algorithm'] || 'gzip',
        duration: parseFloat(response.headers['x-compression-duration'] || '0'),
        file_path: '',
      };

      // 将信息附加到 blob
      const blob = response.data as Blob;
      (blob as any).compressionResult = compressionResult;

      return blob;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '压缩失败');
    }
  }

  /**
   * 解压场景包
   */
  async decompressScenario(file: File): Promise<DecompressionResult> {
    const formData = new FormData();
    formData.append('file', file);

    try {
      const response = await api.post<DecompressionResult>(
        '/api/v1/compression/decompress',
        formData,
        {
          headers: {
            'Content-Type': 'multipart/form-data',
          },
        }
      );

      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '解压失败');
    }
  }

  /**
   * 获取支持的算法列表
   */
  async getAlgorithms(): Promise<{ default: string; supported: AlgorithmInfo[] }> {
    try {
      const response = await api.get('/api/v1/compression/algorithms');
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '获取算法列表失败');
    }
  }

  /**
   * 下载压缩后的文件
   */
  downloadCompressedFile(blob: Blob, filename: string): void {
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename.endsWith('.scenario')
      ? filename
      : filename.replace('.json', '.scenario');
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    window.URL.revokeObjectURL(url);
  }
}

export default new CompressionService();
```

### 2. 压缩组件

```typescript
// frontend/src/components/scenario/Compressor.tsx
import React, { useState } from 'react';
import {
  Card,
  Button,
  Select,
  Space,
  Progress,
  Statistic,
  Row,
  Col,
  message,
} from 'antd';
import {
  CompressOutlined,
  DownloadOutlined,
  FileZipOutlined,
} from '@ant-design/icons';
import compressionService, {
  CompressionAlgorithm,
  CompressionResult,
} from '@/services/api/compression';

interface CompressorProps {
  file: File;
  onCompressed?: (result: CompressionResult) => void;
}

const Compressor: React.FC<CompressorProps> = ({ file, onCompressed }) => {
  const [algorithm, setAlgorithm] = useState<CompressionAlgorithm>(CompressionAlgorithm.GZIP);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<CompressionResult | null>(null);

  const handleCompress = async () => {
    setLoading(true);
    setResult(null);

    try {
      const blob = await compressionService.compressScenario(file, algorithm);
      const compressionResult = (blob as any).compressionResult as CompressionResult;

      setResult(compressionResult);
      message.success('压缩成功');

      // 下载文件
      const filename = file.name.replace('.json', '.scenario');
      compressionService.downloadCompressedFile(blob, filename);

      onCompressed?.(compressionResult);
    } catch (error: any) {
      message.error(error.message || '压缩失败');
    } finally {
      setLoading(false);
    }
  };

  const formatSize = (bytes: number): string => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
  };

  return (
    <Card title="压缩场景包" bordered={false}>
      <Space direction="vertical" style={{ width: '100%' }} size="large">
        {/* 原始文件信息 */}
        <Row gutter={16}>
          <Col span={12}>
            <Statistic
              title="原始文件"
              value={formatSize(file.size)}
              prefix={<FileZipOutlined />}
            />
          </Col>
          <Col span={12}>
            <Statistic title="文件名" value={file.name} />
          </Col>
        </Row>

        {/* 算法选择 */}
        <div>
          <span style={{ marginRight: 8 }}>压缩算法:</span>
          <Select
            value={algorithm}
            onChange={setAlgorithm}
            style={{ width: 200 }}
            options={[
              { label: 'GZIP (推荐)', value: CompressionAlgorithm.GZIP },
              { label: 'BZ2 (高压缩)', value: CompressionAlgorithm.BZ2 },
              { label: 'LZMA (最高压缩)', value: CompressionAlgorithm.LZMA },
            ]}
          />
        </div>

        {/* 压缩按钮 */}
        <Button
          type="primary"
          icon={<CompressOutlined />}
          onClick={handleCompress}
          loading={loading}
          size="large"
          block
        >
          开始压缩
        </Button>

        {/* 压缩结果 */}
        {result && (
          <Card type="inner" title="压缩结果">
            <Row gutter={16}>
              <Col span={8}>
                <Statistic
                  title="压缩后大小"
                  value={formatSize(result.compressed_size)}
                  valueStyle={{ color: '#3f8600' }}
                />
              </Col>
              <Col span={8}>
                <Statistic
                  title="压缩率"
                  value={result.compression_ratio}
                  suffix="%"
                  valueStyle={{ color: '#cf1322' }}
                />
              </Col>
              <Col span={8}>
                <Statistic
                  title="耗时"
                  value={result.duration}
                  suffix="秒"
                />
              </Col>
            </Row>

            {/* 压缩进度可视化 */}
            <div style={{ marginTop: 16 }}>
              <Progress
                percent={Math.round(result.compression_ratio)}
                status="active"
                strokeColor={{
                  '0%': '#108ee9',
                  '100%': '#87d068',
                }}
              />
            </div>

            <div style={{ marginTop: 16, textAlign: 'center' }}>
              <Button
                icon={<DownloadOutlined />}
                onClick={() => message.info('文件已开始下载')}
              >
                重新下载
              </Button>
            </div>
          </Card>
        )}
      </Space>
    </Card>
  );
};

export default Compressor;
```

---

## 涉及文件清单

### 新建文件

| 文件路径 | 说明 |
|---------|------|
| `/backend/app/models/compression.py` | 压缩相关数据模型 |
| `/backend/app/services/compression_service.py` | 压缩服务实现 |
| `/backend/app/api/v1/endpoints/compression.py` | 压缩API路由 |
| `/backend/app/core/config.py` | 添加压缩配置 |

| 文件路径 | 说明 |
|---------|------|
| `/frontend/src/services/api/compression.ts` | 压缩服务API |
| `/frontend/src/components/scenario/Compressor.tsx` | 压缩组件 |

---

## 验收标准

### 功能验收

- [ ] 成功压缩 JSON 文件为 .scenario 格式
- [ ] 支持 gzip、bz2、lzma 三种算法
- [ ] 正确解压 .scenario 文件到 JSON
- [ ] 自动检测压缩算法
- [ ] 准确计算压缩率和耗时
- [ ] 压缩文件可以正常下载

### 性能验收

- [ ] 1MB JSON 文件压缩时间 < 1秒
- [ ] gzip 压缩率达到 60-80%
- [ ] 解压时间 < 压缩时间

### 异常处理验收

- [ ] 无效 JSON 文件返回明确错误
- [ ] 不支持的算法返回明确错误
- [ ] 文件过大时返回明确错误

---

## 参考文档

### 内部文档

- [M4-010: 场景包验证](/Users/guochangxi/git/monika/docs/tasks/tasks-detailed/M4-010-package-validation.md)
- [M4-009: JSON解析器](/Users/guochangxi/git/monika/docs/tasks/tasks-detailed/M4-009-json-parser.md)

### 技术文档

- [Python gzip 模块文档](https://docs.python.org/3/library/gzip.html)
- [Python bz2 模块文档](https://docs.python.org/3/library/bz2.html)
- [Python lzma 模块文档](https://docs.python.org/3/library/lzma.html)

---

**创建日期**: 2026-02-06
**最后更新**: 2026-02-06
