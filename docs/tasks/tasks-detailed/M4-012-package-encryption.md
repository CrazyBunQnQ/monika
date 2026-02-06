# M4-012: 实现场景包加密

**任务ID**: M4-012
**任务名称**: 实现场景包加密
**预估时间**: 5 小时
**优先级**: P1
**依赖**: M4-011 (场景包压缩)
**状态**: 待开始

---

## 任务概述

实现场景包的加密功能，为敏感或付费场景包提供保护机制。支持 AES-256-GCM 对称加密，支持密码模式和密钥模式，确保只有授权用户才能访问受保护的内容。同时实现相应的解密功能。

---

## 子任务拆解

| ID | 子任务描述 | 预估时间 | 依赖 | 状态 |
|----|-----------|----------|------|------|
| M4-012-01 | 设计加密服务架构和安全方案 | 1h | M4-011 | 待开始 |
| M4-012-02 | 实现 AES-256-GCM 加密/解密核心功能 | 2h | M4-012-01 | 待开始 |
| M4-012-03 | 实现密码派生函数（PBKDF2） | 1h | M4-012-02 | 待开始 |
| M4-012-04 | 实现加密元数据和验证机制 | 0.5h | M4-012-03 | 待开始 |
| M4-012-05 | 实现加密文件格式和API接口 | 0.5h | M4-012-04 | 待开始 |

**总预估时间**: 5 小时

---

## Python 后端实现

### 1. 加密配置

```python
# backend/app/core/config.py
from pydantic import BaseSettings

class EncryptionSettings(BaseSettings):
    """加密配置"""

    # 加密算法
    algorithm: str = "aes-256-gcm"

    # 密钥派生迭代次数
    pbkdf2_iterations: int = 100000

    # Salt 长度（字节）
    salt_length: int = 16

    # Nonce 长度（字节）
    nonce_length: int = 12

    # Tag 长度（字节）
    tag_length: int = 16

    # 加密文件扩展名
    encrypted_extension: str = ".encrypted"

    # 加密文件魔数（用于识别加密文件）
    magic_number: bytes = b"SCENCRYPT"

    class Config:
        env_prefix = "ENCRYPTION_"

encryption_settings = EncryptionSettings()
```

### 2. 加密数据模型

```python
# backend/app/models/encryption.py
from pydantic import BaseModel, Field
from typing import Optional
from enum import Enum
from datetime import datetime

class EncryptionMode(str, Enum):
    """加密模式"""
    PASSWORD = "password"  # 密码模式
    KEY = "key"           # 密钥模式

class EncryptionMetadata(BaseModel):
    """加密元数据"""
    version: str = Field(default="1.0", description="加密格式版本")
    algorithm: str = Field(..., description="加密算法")
    mode: EncryptionMode = Field(..., description="加密模式")
    salt: Optional[str] = Field(None, description="盐值（密码模式）")
    nonce: str = Field(..., description="随机数")
    tag: str = Field(..., description="认证标签")
    original_filename: str = Field(..., description="原始文件名")
    original_size: int = Field(..., description="原始文件大小")
    encrypted_at: datetime = Field(default_factory=datetime.utcnow, description="加密时间")
    key_id: Optional[str] = Field(None, description="密钥ID（密钥模式）")

class EncryptionResult(BaseModel):
    """加密结果"""
    success: bool
    encrypted_path: str
    metadata: EncryptionMetadata
    duration: float

class DecryptionResult(BaseModel):
    """解密结果"""
    success: bool
    decrypted_path: str
    metadata: EncryptionMetadata
    duration: float
```

### 3. 加密服务实现

```python
# backend/app/services/encryption_service.py
import os
import time
import hashlib
import secrets
import logging
from pathlib import Path
from typing import Union, Optional, Tuple
from cryptography.hazmat.primitives.ciphers.aead import AESGCM
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2HMAC
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.backends import default_backend

from app.models.encryption import (
    EncryptionMode,
    EncryptionMetadata,
    EncryptionResult,
    DecryptionResult
)
from app.core.config import encryption_settings
from app.core.exceptions import ParseError

logger = logging.getLogger(__name__)

class EncryptionService:
    """加密服务"""

    def __init__(self):
        self.backend = default_backend()

    def encrypt_file(
        self,
        input_path: Union[str, Path],
        output_path: Optional[Union[str, Path]] = None,
        password: Optional[str] = None,
        key: Optional[bytes] = None
    ) -> EncryptionResult:
        """
        加密文件

        Args:
            input_path: 输入文件路径
            output_path: 输出文件路径，默认在输入文件同目录
            password: 密码（密码模式）
            key: 密钥（密钥模式，32字节）

        Returns:
            EncryptionResult: 加密结果

        Raises:
            ParseError: 加密失败时抛出
        """
        # 验证参数
        if not password and not key:
            raise ParseError("必须提供密码或密钥")
        if password and key:
            raise ParseError("密码和密钥只能二选一")

        # 确定加密模式
        mode = EncryptionMode.PASSWORD if password else EncryptionMode.KEY

        input_path = Path(input_path)
        if not input_path.exists():
            raise ParseError(f"文件不存在: {input_path}")

        try:
            start_time = time.time()

            # 读取文件内容
            with open(input_path, 'rb') as f:
                plaintext = f.read()

            # 生成 nonce
            nonce = secrets.token_bytes(encryption_settings.nonce_length)

            # 生成密钥
            if mode == EncryptionMode.PASSWORD:
                salt = secrets.token_bytes(encryption_settings.salt_length)
                derived_key = self._derive_key_from_password(password, salt)
            else:
                salt = None
                derived_key = key

            # 加密
            aesgcm = AESGCM(derived_key)
            ciphertext_with_tag = aesgcm.encrypt(nonce, plaintext, None)

            # 分离密文和 tag
            tag_length = encryption_settings.tag_length
            ciphertext = ciphertext_with_tag[:-tag_length]
            tag = ciphertext_with_tag[-tag_length:]

            # 确定输出路径
            if output_path is None:
                output_path = input_path.parent / f"{input_path.name}{encryption_settings.encrypted_extension}"
            else:
                output_path = Path(output_path)

            # 创建加密元数据
            metadata = EncryptionMetadata(
                algorithm=encryption_settings.algorithm,
                mode=mode,
                salt=salt.hex() if salt else None,
                nonce=nonce.hex(),
                tag=tag.hex(),
                original_filename=input_path.name,
                original_size=len(plaintext)
            )

            # 写入加密文件：魔数 + 元数据JSON长度 + 元数据JSON + 密文
            output_path.parent.mkdir(parents=True, exist_ok=True)
            with open(output_path, 'wb') as f:
                # 写入魔数
                f.write(encryption_settings.magic_number)

                # 写入元数据
                metadata_json = metadata.json().encode('utf-8')
                metadata_length = len(metadata_json).to_bytes(4, 'big')
                f.write(metadata_length)
                f.write(metadata_json)

                # 写入密文
                f.write(ciphertext)

            duration = time.time() - start_time

            logger.info(
                f"加密成功: {input_path.name} -> {output_path.name} "
                f"({mode.value}, {duration:.3f}s)"
            )

            return EncryptionResult(
                success=True,
                encrypted_path=str(output_path),
                metadata=metadata,
                duration=round(duration, 3)
            )

        except Exception as e:
            logger.error(f"加密失败: {e}")
            raise ParseError(f"加密失败: {str(e)}")

    def decrypt_file(
        self,
        input_path: Union[str, Path],
        output_path: Optional[Union[str, Path]] = None,
        password: Optional[str] = None,
        key: Optional[bytes] = None
    ) -> DecryptionResult:
        """
        解密文件

        Args:
            input_path: 输入加密文件路径
            output_path: 输出文件路径
            password: 密码（密码模式）
            key: 密钥（密钥模式）

        Returns:
            DecryptionResult: 解密结果

        Raises:
            ParseError: 解密失败时抛出
        """
        if not password and not key:
            raise ParseError("必须提供密码或密钥")

        input_path = Path(input_path)
        if not input_path.exists():
            raise ParseError(f"文件不存在: {input_path}")

        try:
            start_time = time.time()

            # 读取加密文件
            with open(input_path, 'rb') as f:
                # 验证魔数
                magic = f.read(len(encryption_settings.magic_number))
                if magic != encryption_settings.magic_number:
                    raise ParseError("无效的加密文件格式")

                # 读取元数据
                metadata_length_bytes = f.read(4)
                metadata_length = int.from_bytes(metadata_length_bytes, 'big')
                metadata_json = f.read(metadata_length).decode('utf-8')
                metadata = EncryptionMetadata.parse_raw(metadata_json)

                # 读取密文
                ciphertext = f.read()

            # 生成密钥
            if metadata.mode == EncryptionMode.PASSWORD:
                if not password:
                    raise ParseError("该文件使用密码加密，请提供密码")
                salt = bytes.fromhex(metadata.salt)
                derived_key = self._derive_key_from_password(password, salt)
            else:
                if not key:
                    raise ParseError("该文件使用密钥加密，请提供密钥")
                derived_key = key

            # 重组密文和 tag
            nonce = bytes.fromhex(metadata.nonce)
            tag = bytes.fromhex(metadata.tag)
            ciphertext_with_tag = ciphertext + tag

            # 解密
            aesgcm = AESGCM(derived_key)
            plaintext = aesgcm.decrypt(nonce, ciphertext_with_tag, None)

            # 确定输出路径
            if output_path is None:
                output_path = input_path.parent / metadata.original_filename
            else:
                output_path = Path(output_path)

            # 写入解密文件
            output_path.parent.mkdir(parents=True, exist_ok=True)
            with open(output_path, 'wb') as f:
                f.write(plaintext)

            duration = time.time() - start_time

            logger.info(
                f"解密成功: {input_path.name} -> {output_path.name} "
                f"({duration:.3f}s)"
            )

            return DecryptionResult(
                success=True,
                decrypted_path=str(output_path),
                metadata=metadata,
                duration=round(duration, 3)
            )

        except Exception as e:
            logger.error(f"解密失败: {e}")
            raise ParseError(f"解密失败: {str(e)}")

    def _derive_key_from_password(self, password: str, salt: bytes) -> bytes:
        """
        从密码派生密钥

        Args:
            password: 密码
            salt: 盐值

        Returns:
            bytes: 32字节密钥
        """
        kdf = PBKDF2HMAC(
            algorithm=hashes.SHA256(),
            length=32,
            salt=salt,
            iterations=encryption_settings.pbkdf2_iterations,
            backend=self.backend
        )
        return kdf.derive(password.encode('utf-8'))

    def generate_key(self) -> bytes:
        """
        生成随机密钥

        Returns:
            bytes: 32字节随机密钥
        """
        return secrets.token_bytes(32)

    def verify_password(
        self,
        encrypted_file: Union[str, Path],
        password: str
    ) -> bool:
        """
        验证密码是否正确

        Args:
            encrypted_file: 加密文件路径
            password: 密码

        Returns:
            bool: 密码是否正确
        """
        try:
            # 尝试解密前几个字节
            import tempfile
            with tempfile.NamedTemporaryFile(delete=False) as tmp:
                tmp_path = tmp.name

            self.decrypt_file(encrypted_file, tmp_path, password)

            # 清理临时文件
            os.unlink(tmp_path)

            return True
        except Exception:
            return False

    def get_encryption_info(self, encrypted_file: Union[str, Path]) -> EncryptionMetadata:
        """
        获取加密文件信息（不解密）

        Args:
            encrypted_file: 加密文件路径

        Returns:
            EncryptionMetadata: 加密元数据
        """
        encrypted_file = Path(encrypted_file)

        with open(encrypted_file, 'rb') as f:
            # 验证魔数
            magic = f.read(len(encryption_settings.magic_number))
            if magic != encryption_settings.magic_number:
                raise ParseError("无效的加密文件格式")

            # 读取元数据
            metadata_length_bytes = f.read(4)
            metadata_length = int.from_bytes(metadata_length_bytes, 'big')
            metadata_json = f.read(metadata_length).decode('utf-8')
            metadata = EncryptionMetadata.parse_raw(metadata_json)

        return metadata
```

### 4. API 路由

```python
# backend/app/api/v1/endpoints/encryption.py
from fastapi import APIRouter, UploadFile, File, Form, Depends, HTTPException
from fastapi.responses import FileResponse
from typing import Optional
import tempfile
import os

from app.services.encryption_service import EncryptionService
from app.models.encryption import EncryptionMode
from app.api.deps import get_current_user

router = APIRouter()
encryption_service = EncryptionService()

@router.post("/encrypt")
async def encrypt_scenario(
    file: UploadFile = File(...),
    password: Optional[str] = Form(None),
    current_user = Depends(get_current_user)
):
    """
    加密场景包

    - **file**: 上传的文件（.scenario 或 .json）
    - **password**: 加密密码（可选，如果不提供则使用系统密钥）
    - 返回加密后的文件
    """
    try:
        # 保存临时文件
        with tempfile.NamedTemporaryFile(delete=False, suffix='.scenario') as tmp:
            content = await file.read()
            tmp.write(content)
            tmp_path = tmp.name

        # 加密
        result = encryption_service.encrypt_file(tmp_path, password=password)

        # 清理临时文件
        os.unlink(tmp_path)

        # 返回加密文件
        response = FileResponse(
            path=result.encrypted_path,
            media_type="application/octet-stream",
            filename=file.name + ".encrypted"
        )

        # 添加加密信息到响应头
        response.headers["X-Encryption-Mode"] = result.metadata.mode.value
        response.headers["X-Encryption-Algorithm"] = result.metadata.algorithm
        response.headers["X-Original-Size"] = str(result.metadata.original_size)
        response.headers["X-Encryption-Duration"] = f"{result.duration}s"

        return response

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.post("/decrypt")
async def decrypt_scenario(
    file: UploadFile = File(...),
    password: Optional[str] = Form(None),
    current_user = Depends(get_current_user)
):
    """
    解密场景包

    - **file**: 上传的加密文件
    - **password**: 解密密码（如果使用密码加密）
    - 返回解密后的JSON数据
    """
    try:
        # 保存临时文件
        with tempfile.NamedTemporaryFile(delete=False) as tmp:
            content = await file.read()
            tmp.write(content)
            tmp_path = tmp.name

        # 解密
        result = encryption_service.decrypt_file(tmp_path, password=password)

        # 读取解密后的数据
        with open(result.decrypted_path, 'rb') as f:
            decrypted_content = f.read()

        # 清理临时文件
        os.unlink(tmp_path)
        os.unlink(result.decrypted_path)

        # 判断文件类型并返回
        if result.decrypted_path.endswith('.json'):
            import json
            json_data = json.loads(decrypted_content.decode('utf-8'))
            return {
                "success": True,
                "data": json_data,
                "metadata": result.metadata.dict(),
                "duration": result.duration,
            }
        else:
            # 返回文件
            return FileResponse(
                path=result.decrypted_path,
                media_type="application/octet-stream",
                filename=result.metadata.original_filename
            )

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.post("/verify-password")
async def verify_encryption_password(
    file: UploadFile = File(...),
    password: str = Form(...)
):
    """
    验证加密密码

    - **file**: 上传的加密文件
    - **password**: 待验证的密码
    - 返回密码是否正确
    """
    try:
        # 保存临时文件
        with tempfile.NamedTemporaryFile(delete=False) as tmp:
            content = await file.read()
            tmp.write(content)
            tmp_path = tmp.name

        # 验证密码
        is_valid = encryption_service.verify_password(tmp_path, password)

        # 清理临时文件
        os.unlink(tmp_path)

        return {
            "valid": is_valid
        }

    except Exception as e:
        return {
            "valid": False,
            "error": str(e)
        }

@router.get("/info")
async def get_encryption_info(file: UploadFile = File(...)):
    """
    获取加密文件信息

    - **file**: 上传的加密文件
    - 返回加密元数据
    """
    try:
        # 保存临时文件
        with tempfile.NamedTemporaryFile(delete=False) as tmp:
            content = await file.read()
            tmp.write(content)
            tmp_path = tmp.name

        # 获取信息
        metadata = encryption_service.get_encryption_info(tmp_path)

        # 清理临时文件
        os.unlink(tmp_path)

        return metadata.dict()

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
```

---

## TypeScript/React 前端实现

### 1. 加密服务

```typescript
// frontend/src/services/api/encryption.ts
import api from './client';

export enum EncryptionMode {
  PASSWORD = 'password',
  KEY = 'key',
}

export interface EncryptionMetadata {
  version: string;
  algorithm: string;
  mode: EncryptionMode;
  salt?: string;
  nonce: string;
  tag: string;
  original_filename: string;
  original_size: number;
  encrypted_at: string;
  key_id?: string;
}

export interface EncryptionResult {
  success: boolean;
  encrypted_path: string;
  metadata: EncryptionMetadata;
  duration: number;
}

export interface DecryptionResult {
  success: boolean;
  data?: any;
  metadata: EncryptionMetadata;
  duration: number;
}

class EncryptionService {
  /**
   * 加密场景包
   */
  async encryptScenario(
    file: File,
    password?: string
  ): Promise<Blob> {
    const formData = new FormData();
    formData.append('file', file);
    if (password) {
      formData.append('password', password);
    }

    try {
      const response = await api.post('/api/v1/encryption/encrypt', formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
        responseType: 'blob',
      });

      const blob = response.data as Blob;
      const metadata: EncryptionMetadata = {
        version: '1.0',
        algorithm: response.headers['x-encryption-algorithm'] || 'aes-256-gcm',
        mode: response.headers['x-encryption-mode'] as EncryptionMode || EncryptionMode.PASSWORD,
        nonce: '',
        tag: '',
        original_filename: file.name,
        original_size: parseInt(response.headers['x-original-size'] || '0'),
        encrypted_at: new Date().toISOString(),
      };

      (blob as any).encryptionMetadata = metadata;

      return blob;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '加密失败');
    }
  }

  /**
   * 解密场景包
   */
  async decryptScenario(
    file: File,
    password?: string
  ): Promise<DecryptionResult> {
    const formData = new FormData();
    formData.append('file', file);
    if (password) {
      formData.append('password', password);
    }

    try {
      const response = await api.post<DecryptionResult>(
        '/api/v1/encryption/decrypt',
        formData,
        {
          headers: {
            'Content-Type': 'multipart/form-data',
          },
        }
      );

      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '解密失败');
    }
  }

  /**
   * 验证密码
   */
  async verifyPassword(file: File, password: string): Promise<boolean> {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('password', password);

    try {
      const response = await api.post<{ valid: boolean }>(
        '/api/v1/encryption/verify-password',
        formData,
        {
          headers: {
            'Content-Type': 'multipart/form-data',
          },
        }
      );

      return response.data.valid;
    } catch (error) {
      return false;
    }
  }

  /**
   * 下载加密文件
   */
  downloadEncryptedFile(blob: Blob, filename: string): void {
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename.endsWith('.encrypted')
      ? filename
      : filename + '.encrypted';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    window.URL.revokeObjectURL(url);
  }
}

export default new EncryptionService();
```

### 2. 加密组件

```typescript
// frontend/src/components/scenario/Encryptor.tsx
import React, { useState } from 'react';
import {
  Card,
  Button,
  Input,
  Space,
  message,
  Alert,
  Progress,
} from 'antd';
import {
  LockOutlined,
  UnlockOutlined,
  DownloadOutlined,
} from '@ant-design/icons';
import encryptionService from '@/services/api/encryption';

interface EncryptorProps {
  file: File;
  mode?: 'encrypt' | 'decrypt';
}

const Encryptor: React.FC<EncryptorProps> = ({ file, mode = 'encrypt' }) => {
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [passwordValid, setPasswordValid] = useState<boolean | null>(null);

  const handleEncrypt = async () => {
    if (!password) {
      message.warning('请输入加密密码');
      return;
    }

    setLoading(true);
    try {
      const blob = await encryptionService.encryptScenario(file, password);
      message.success('加密成功');
      encryptionService.downloadEncryptedFile(blob, file.name);
    } catch (error: any) {
      message.error(error.message || '加密失败');
    } finally {
      setLoading(false);
    }
  };

  const handleDecrypt = async () => {
    if (!password) {
      message.warning('请输入解密密码');
      return;
    }

    setLoading(true);
    try {
      const result = await encryptionService.decryptScenario(file, password);
      if (result.success) {
        message.success('解密成功');
        // 这里可以处理解密后的数据
      }
    } catch (error: any) {
      message.error(error.message || '解密失败');
    } finally {
      setLoading(false);
    }
  };

  const handleVerifyPassword = async () => {
    if (!password) {
      message.warning('请输入密码');
      return;
    }

    setLoading(true);
    try {
      const valid = await encryptionService.verifyPassword(file, password);
      setPasswordValid(valid);
      message.success(valid ? '密码正确' : '密码错误');
    } catch (error) {
      setPasswordValid(false);
      message.error('验证失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card title={mode === 'encrypt' ? '加密场景包' : '解密场景包'} bordered={false}>
      <Space direction="vertical" style={{ width: '100%' }} size="large">
        {/* 密码输入 */}
        <div>
          <div style={{ marginBottom: 8 }}>
            <LockOutlined /> 密码:
          </div>
          <Input.Password
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="请输入密码"
            size="large"
          />
        </div>

        {/* 密码验证结果 */}
        {passwordValid !== null && (
          <Alert
            type={passwordValid ? 'success' : 'error'}
            message={passwordValid ? '密码正确' : '密码错误'}
            showIcon
          />
        )}

        {/* 操作按钮 */}
        {mode === 'encrypt' ? (
          <Button
            type="primary"
            icon={<LockOutlined />}
            onClick={handleEncrypt}
            loading={loading}
            size="large"
            block
          >
            加密并下载
          </Button>
        ) : (
          <>
            <Button
              icon={<UnlockOutlined />}
              onClick={handleVerifyPassword}
              loading={loading}
              size="large"
              block
            >
              验证密码
            </Button>
            <Button
              type="primary"
              icon={<UnlockOutlined />}
              onClick={handleDecrypt}
              loading={loading}
              size="large"
              block
            >
              解密
            </Button>
          </>
        )}
      </Space>
    </Card>
  );
};

export default Encryptor;
```

---

## 涉及文件清单

### 新建文件

| 文件路径 | 说明 |
|---------|------|
| `/backend/app/models/encryption.py` | 加密相关数据模型 |
| `/backend/app/services/encryption_service.py` | 加密服务实现 |
| `/backend/app/api/v1/endpoints/encryption.py` | 加密API路由 |
| `/backend/app/core/config.py` | 添加加密配置 |

| 文件路径 | 说明 |
|---------|------|
| `/frontend/src/services/api/encryption.ts` | 加密服务API |
| `/frontend/src/components/scenario/Encryptor.tsx` | 加密/解密组件 |

---

## 验收标准

### 功能验收

- [ ] 成功使用密码加密场景包
- [ ] 成功使用密钥加密场景包
- [ ] 正确解密加密的场景包
- [ ] 错误密码无法解密
- [ ] 正确验证密码
- [ ] 准确读取加密元数据
- [ ] 加密文件可以正常下载

### 安全性验收

- [ ] 使用 AES-256-GCM 算法
- [ ] 密码派生使用 PBKDF2 + SHA256
- [ ] 每次加密使用随机 nonce
- [ ] 包含认证标签（AEAD）
- [ ] 密码不在日志中输出

### 性能验收

- [ ] 1MB 文件加密时间 < 2秒
- [ ] 解密时间 < 加密时间

---

## 参考文档

### 内部文档

- [M4-011: 场景包压缩](/Users/guochangxi/git/monika/docs/tasks/tasks-detailed/M4-011-package-compression.md)
- [M4-010: 场景包验证](/Users/guochangxi/git/monika/docs/tasks/tasks-detailed/M4-010-package-validation.md)

### 技术文档

- [Cryptography Library Documentation](https://cryptography.io/en/latest/)
- [AES-GCM Specification](https://csrc.nist.gov/publications/detail/sp/800-38d/final)
- [PBKDF2 RFC 2898](https://tools.ietf.org/html/rfc2898)
- [Python cryptography.hazmat.primitives](https://cryptography.io/en/latest/hazmat/primitives/)

---

**创建日期**: 2026-02-06
**最后更新**: 2026-02-06
