# M4-013: 实现资源 CDN 上传

**任务ID**: M4-013
**任务名称**: 实现资源 CDN 上传
**预估时间**: 6 小时
**优先级**: P0
**依赖**: M4-012 (场景包加密)
**状态**: 待开始

---

## 任务概述

实现场景包资源的 CDN 上传功能，支持将场景包文件上传到云存储服务（AWS S3、阿里云 OSS、腾讯云 COS 等），实现资源的分布式存储和全球加速访问。包括文件分片上传、断点续传、上传进度追踪等功能。

---

## 子任务拆解

| ID | 子任务描述 | 预估时间 | 依赖 | 状态 |
|----|-----------|----------|------|------|
| M4-013-01 | 设计存储服务抽象层和多云支持架构 | 1h | M4-012 | 待开始 |
| M4-013-02 | 实现 AWS S3 存储适配器 | 2h | M4-013-01 | 待开始 |
| M4-013-03 | 实现阿里云 OSS 存储适配器 | 1.5h | M4-013-01 | 待开始 |
| M4-013-04 | 实现文件分片上传和断点续传 | 1h | M4-013-02 | 待开始 |
| M4-013-05 | 实现上传进度追踪和回调机制 | 0.5h | M4-013-04 | 待开始 |

**总预估时间**: 6 小时

---

## Python 后端实现

### 1. 存储配置

```python
# backend/app/core/config.py
from pydantic import BaseSettings
from typing import Optional

class StorageSettings(BaseSettings):
    """存储配置"""

    # 存储提供商
    provider: str = "s3"  # s3, oss, cos, local

    # AWS S3 配置
    aws_access_key_id: Optional[str] = None
    aws_secret_access_key: Optional[str] = None
    aws_region: str = "us-east-1"
    aws_bucket_name: str = "monika-scenarios"

    # 阿里云 OSS 配置
    oss_access_key_id: Optional[str] = None
    oss_access_key_secret: Optional[str] = None
    oss_endpoint: str = "oss-cn-hangzhou.aliyuncs.com"
    oss_bucket_name: str = "monika-scenarios"

    # 腾讯云 COS 配置
    cos_secret_id: Optional[str] = None
    cos_secret_key: Optional[str] = None
    cos_region: str = "ap-guangzhou"
    cos_bucket_name: str = "monika-scenarios"

    # 本地存储配置
    local_storage_path: str = "./storage"

    # 上传配置
    max_file_size: int = 500 * 1024 * 1024  # 500MB
    chunk_size: int = 5 * 1024 * 1024  # 5MB
    allowed_extensions: list = [".scenario", ".encrypted", ".json"]

    # CDN 配置
    cdn_enabled: bool = False
    cdn_domain: Optional[str] = None

    class Config:
        env_prefix = "STORAGE_"

storage_settings = StorageSettings()
```

### 2. 存储适配器基类

```python
# backend/app/services/storage/base.py
from abc import ABC, abstractmethod
from typing import Optional, Dict, Any, Callable
from dataclasses import dataclass
from enum import Enum
import logging

logger = logging.getLogger(__name__)

class StorageProvider(str, Enum):
    """存储提供商"""
    S3 = "s3"
    OSS = "oss"
    COS = "cos"
    LOCAL = "local"

@dataclass
class UploadResult:
    """上传结果"""
    success: bool
    file_url: str
    file_key: str
    file_size: int
    provider: StorageProvider
    metadata: Dict[str, Any]

@dataclass
class UploadProgress:
    """上传进度"""
    uploaded_bytes: int
    total_bytes: int
    percentage: float
    current_chunk: int
    total_chunks: int

class StorageAdapter(ABC):
    """存储适配器基类"""

    def __init__(self):
        self.provider: StorageProvider = StorageProvider.LOCAL

    @abstractmethod
    async def upload_file(
        self,
        file_path: str,
        key: str,
        metadata: Optional[Dict[str, Any]] = None,
        progress_callback: Optional[Callable[[UploadProgress], None]] = None
    ) -> UploadResult:
        """
        上传文件

        Args:
            file_path: 本地文件路径
            key: 存储键名
            metadata: 文件元数据
            progress_callback: 进度回调函数

        Returns:
            UploadResult: 上传结果
        """
        pass

    @abstractmethod
    async def upload_bytes(
        self,
        data: bytes,
        key: str,
        metadata: Optional[Dict[str, Any]] = None
    ) -> UploadResult:
        """
        上传字节数据

        Args:
            data: 字节数据
            key: 存储键名
            metadata: 文件元数据

        Returns:
            UploadResult: 上传结果
        """
        pass

    @abstractmethod
    async def download_file(
        self,
        key: str,
        local_path: str
    ) -> str:
        """
        下载文件

        Args:
            key: 存储键名
            local_path: 本地保存路径

        Returns:
            str: 本地文件路径
        """
        pass

    @abstractmethod
    async def delete_file(self, key: str) -> bool:
        """
        删除文件

        Args:
            key: 存储键名

        Returns:
            bool: 是否删除成功
        """
        pass

    @abstractmethod
    async def file_exists(self, key: str) -> bool:
        """
        检查文件是否存在

        Args:
            key: 存储键名

        Returns:
            bool: 文件是否存在
        """
        pass

    @abstractmethod
    async def get_file_url(self, key: str, expires_in: int = 3600) -> str:
        """
        获取文件访问URL

        Args:
            key: 存储键名
            expires_in: 过期时间（秒）

        Returns:
            str: 文件URL
        """
        pass

    @abstractmethod
    async def get_file_metadata(self, key: str) -> Dict[str, Any]:
        """
        获取文件元数据

        Args:
            key: 存储键名

        Returns:
            Dict: 文件元数据
        """
        pass

    def generate_key(self, filename: str, user_id: Optional[str] = None) -> str:
        """
        生成存储键名

        Args:
            filename: 文件名
            user_id: 用户ID（可选）

        Returns:
            str: 存储键名
        """
        import uuid
        from datetime import datetime

        ext = filename.rsplit('.', 1)[-1] if '.' in filename else ''
        unique_id = str(uuid.uuid4())
        date_path = datetime.utcnow().strftime('%Y/%m/%d')

        if user_id:
            return f"scenarios/{user_id}/{date_path}/{unique_id}.{ext}"
        else:
            return f"scenarios/public/{date_path}/{unique_id}.{ext}"
```

### 3. AWS S3 适配器

```python
# backend/app/services/storage/s3_adapter.py
import os
import asyncio
from typing import Optional, Dict, Any, Callable
import boto3
from botocore.exceptions import ClientError
import logging

from app.services.storage.base import (
    StorageAdapter,
    StorageProvider,
    UploadResult,
    UploadProgress
)
from app.core.config import storage_settings
from app.core.exceptions import ParseError

logger = logging.getLogger(__name__)

class S3Adapter(StorageAdapter):
    """AWS S3 存储适配器"""

    def __init__(self):
        super().__init__()
        self.provider = StorageProvider.S3
        self.s3_client = boto3.client(
            's3',
            aws_access_key_id=storage_settings.aws_access_key_id,
            aws_secret_access_key=storage_settings.aws_secret_access_key,
            region_name=storage_settings.aws_region
        )
        self.bucket_name = storage_settings.aws_bucket_name

    async def upload_file(
        self,
        file_path: str,
        key: str,
        metadata: Optional[Dict[str, Any]] = None,
        progress_callback: Optional[Callable[[UploadProgress], None]] = None
    ) -> UploadResult:
        """上传文件到 S3"""
        try:
            file_size = os.path.getsize(file_path)

            # 检查是否需要分片上传
            if file_size > storage_settings.chunk_size:
                return await self._upload_multipart(
                    file_path, key, metadata, progress_callback
                )
            else:
                return await self._upload_simple(
                    file_path, key, metadata
                )

        except ClientError as e:
            logger.error(f"S3 上传失败: {e}")
            raise ParseError(f"S3 上传失败: {str(e)}")

    async def _upload_simple(
        self,
        file_path: str,
        key: str,
        metadata: Optional[Dict[str, Any]] = None
    ) -> UploadResult:
        """简单上传"""
        extra_args = {}
        if metadata:
            extra_args['Metadata'] = metadata

        # 添加缓存控制
        extra_args['CacheControl'] = 'max-age=31536000'  # 1年

        loop = asyncio.get_event_loop()
        await loop.run_in_executor(
            None,
            lambda: self.s3_client.upload_file(
                file_path,
                self.bucket_name,
                key,
                ExtraArgs=extra_args
            )
        )

        file_url = self._get_public_url(key)
        file_size = os.path.getsize(file_path)

        return UploadResult(
            success=True,
            file_url=file_url,
            file_key=key,
            file_size=file_size,
            provider=self.provider,
            metadata=metadata or {}
        )

    async def _upload_multipart(
        self,
        file_path: str,
        key: str,
        metadata: Optional[Dict[str, Any]] = None,
        progress_callback: Optional[Callable[[UploadProgress], None]] = None
    ) -> UploadResult:
        """分片上传"""
        file_size = os.path.getsize(file_path)
        chunk_size = storage_settings.chunk_size
        total_chunks = (file_size + chunk_size - 1) // chunk_size

        loop = asyncio.get_event_loop()

        # 初始化分片上传
        mpu = await loop.run_in_executor(
            None,
            lambda: self.s3_client.create_multipart_upload(
                Bucket=self.bucket_name,
                Key=key
            )
        )
        upload_id = mpu['UploadId']

        parts = []

        try:
            with open(file_path, 'rb') as f:
                for chunk_num in range(total_chunks):
                    offset = chunk_num * chunk_size
                    f.seek(offset)
                    chunk_data = f.read(chunk_size)

                    # 上传分片
                    part = await loop.run_in_executor(
                        None,
                        lambda: self.s3_client.upload_part(
                            Bucket=self.bucket_name,
                            Key=key,
                            PartNumber=chunk_num + 1,
                            UploadId=upload_id,
                            Body=chunk_data
                        )
                    )

                    parts.append({
                        'PartNumber': chunk_num + 1,
                        'ETag': part['ETag']
                    })

                    # 进度回调
                    if progress_callback:
                        uploaded = offset + len(chunk_data)
                        progress_callback(UploadProgress(
                            uploaded_bytes=min(uploaded, file_size),
                            total_bytes=file_size,
                            percentage=(uploaded / file_size) * 100,
                            current_chunk=chunk_num + 1,
                            total_chunks=total_chunks
                        ))

            # 完成分片上传
            await loop.run_in_executor(
                None,
                lambda: self.s3_client.complete_multipart_upload(
                    Bucket=self.bucket_name,
                    Key=key,
                    UploadId=upload_id,
                    MultipartUpload={'Parts': parts}
                )
            )

            file_url = self._get_public_url(key)

            return UploadResult(
                success=True,
                file_url=file_url,
                file_key=key,
                file_size=file_size,
                provider=self.provider,
                metadata=metadata or {}
            )

        except Exception as e:
            # 取消上传
            await loop.run_in_executor(
                None,
                lambda: self.s3_client.abort_multipart_upload(
                    Bucket=self.bucket_name,
                    Key=key,
                    UploadId=upload_id
                )
            )
            raise ParseError(f"分片上传失败: {str(e)}")

    async def upload_bytes(
        self,
        data: bytes,
        key: str,
        metadata: Optional[Dict[str, Any]] = None
    ) -> UploadResult:
        """上传字节数据"""
        try:
            extra_args = {}
            if metadata:
                extra_args['Metadata'] = metadata
            extra_args['CacheControl'] = 'max-age=31536000'

            loop = asyncio.get_event_loop()
            await loop.run_in_executor(
                None,
                lambda: self.s3_client.put_object(
                    Bucket=self.bucket_name,
                    Key=key,
                    Body=data,
                    **extra_args
                )
            )

            file_url = self._get_public_url(key)

            return UploadResult(
                success=True,
                file_url=file_url,
                file_key=key,
                file_size=len(data),
                provider=self.provider,
                metadata=metadata or {}
            )

        except ClientError as e:
            logger.error(f"S3 上传失败: {e}")
            raise ParseError(f"S3 上传失败: {str(e)}")

    async def download_file(self, key: str, local_path: str) -> str:
        """下载文件"""
        try:
            loop = asyncio.get_event_loop()
            await loop.run_in_executor(
                None,
                lambda: self.s3_client.download_file(
                    self.bucket_name,
                    key,
                    local_path
                )
            )
            return local_path
        except ClientError as e:
            raise ParseError(f"下载失败: {str(e)}")

    async def delete_file(self, key: str) -> bool:
        """删除文件"""
        try:
            loop = asyncio.get_event_loop()
            await loop.run_in_executor(
                None,
                lambda: self.s3_client.delete_object(
                    Bucket=self.bucket_name,
                    Key=key
                )
            )
            return True
        except ClientError:
            return False

    async def file_exists(self, key: str) -> bool:
        """检查文件是否存在"""
        try:
            loop = asyncio.get_event_loop()
            await loop.run_in_executor(
                None,
                lambda: self.s3_client.head_object(
                    Bucket=self.bucket_name,
                    Key=key
                )
            )
            return True
        except ClientError:
            return False

    async def get_file_url(self, key: str, expires_in: int = 3600) -> str:
        """获取文件URL"""
        loop = asyncio.get_event_loop()
        url = await loop.run_in_executor(
            None,
            lambda: self.s3_client.generate_presigned_url(
                'get_object',
                Params={'Bucket': self.bucket_name, 'Key': key},
                ExpiresIn=expires_in
            )
        )
        return url

    async def get_file_metadata(self, key: str) -> Dict[str, Any]:
        """获取文件元数据"""
        try:
            loop = asyncio.get_event_loop()
            response = await loop.run_in_executor(
                None,
                lambda: self.s3_client.head_object(
                    Bucket=self.bucket_name,
                    Key=key
                )
            )
            return {
                'content_length': response.get('ContentLength'),
                'content_type': response.get('ContentType'),
                'last_modified': response.get('LastModified'),
                'metadata': response.get('Metadata', {}),
                'etag': response.get('ETag')
            }
        except ClientError:
            return {}

    def _get_public_url(self, key: str) -> str:
        """获取公开URL"""
        if storage_settings.cdn_enabled and storage_settings.cdn_domain:
            return f"https://{storage_settings.cdn_domain}/{key}"
        else:
            region = storage_settings.aws_region
            return f"https://{self.bucket_name}.s3.{region}.amazonaws.com/{key}"
```

### 4. 阿里云 OSS 适配器

```python
# backend/app/services/storage/oss_adapter.py
import oss2
import asyncio
from typing import Optional, Dict, Any, Callable
import logging

from app.services.storage.base import (
    StorageAdapter,
    StorageProvider,
    UploadResult,
    UploadProgress
)
from app.core.config import storage_settings
from app.core.exceptions import ParseError

logger = logging.getLogger(__name__)

class OSSAdapter(StorageAdapter):
    """阿里云 OSS 存储适配器"""

    def __init__(self):
        super().__init__()
        self.provider = StorageProvider.OSS
        auth = oss2.Auth(
            storage_settings.oss_access_key_id,
            storage_settings.oss_access_key_secret
        )
        self.bucket = oss2.Bucket(
            auth,
            storage_settings.oss_endpoint,
            storage_settings.oss_bucket_name
        )

    async def upload_file(
        self,
        file_path: str,
        key: str,
        metadata: Optional[Dict[str, Any]] = None,
        progress_callback: Optional[Callable[[UploadProgress], None]] = None
    ) -> UploadResult:
        """上传文件到 OSS"""
        try:
            import os
            file_size = os.path.getsize(file_path)

            # 分片上传
            if file_size > 100 * 1024:  # 大于100KB使用分片上传
                return await self._upload_multipart(
                    file_path, key, metadata, progress_callback
                )
            else:
                return await self._upload_simple(
                    file_path, key, metadata
                )

        except oss2.exceptions.OssError as e:
            logger.error(f"OSS 上传失败: {e}")
            raise ParseError(f"OSS 上传失败: {str(e)}")

    async def _upload_simple(
        self,
        file_path: str,
        key: str,
        metadata: Optional[Dict[str, Any]] = None
    ) -> UploadResult:
        """简单上传"""
        loop = asyncio.get_event_loop()

        headers = {}
        if metadata:
            for k, v in metadata.items():
                headers[f'x-oss-meta-{k}'] = str(v)

        await loop.run_in_executor(
            None,
            lambda: self.bucket.put_object_from_file(
                key,
                file_path,
                headers=headers
            )
        )

        file_url = self._get_public_url(key)
        file_size = os.path.getsize(file_path)

        return UploadResult(
            success=True,
            file_url=file_url,
            file_key=key,
            file_size=file_size,
            provider=self.provider,
            metadata=metadata or {}
        )

    async def _upload_multipart(
        self,
        file_path: str,
        key: str,
        metadata: Optional[Dict[str, Any]] = None,
        progress_callback: Optional[Callable[[UploadProgress], None]] = None
    ) -> UploadResult:
        """分片上传"""
        import os

        file_size = os.path.getsize(file_path)
        chunk_size = storage_settings.chunk_size
        total_chunks = (file_size + chunk_size - 1) // chunk_size

        loop = asyncio.get_event_loop()

        # 初始化分片上传
        upload_id = await loop.run_in_executor(
            None,
            lambda: self.bucket.init_multipart_upload(key).upload_id
        )

        parts = []

        try:
            with open(file_path, 'rb') as f:
                for chunk_num in range(total_chunks):
                    offset = chunk_num * chunk_size
                    f.seek(offset)
                    chunk_data = f.read(chunk_size)

                    # 上传分片
                    part = await loop.run_in_executor(
                        None,
                        lambda: self.bucket.upload_part(
                            key,
                            upload_id,
                            chunk_num + 1,
                            chunk_data
                        )
                    )

                    parts.append(oss2.models.PartInfo(chunk_num + 1, part.etag))

                    # 进度回调
                    if progress_callback:
                        uploaded = offset + len(chunk_data)
                        progress_callback(UploadProgress(
                            uploaded_bytes=min(uploaded, file_size),
                            total_bytes=file_size,
                            percentage=(uploaded / file_size) * 100,
                            current_chunk=chunk_num + 1,
                            total_chunks=total_chunks
                        ))

            # 完成分片上传
            await loop.run_in_executor(
                None,
                lambda: self.bucket.complete_multipart_upload(
                    key,
                    upload_id,
                    parts
                )
            )

            file_url = self._get_public_url(key)

            return UploadResult(
                success=True,
                file_url=file_url,
                file_key=key,
                file_size=file_size,
                provider=self.provider,
                metadata=metadata or {}
            )

        except Exception as e:
            # 取消上传
            await loop.run_in_executor(
                None,
                lambda: self.bucket.abort_multipart_upload(key, upload_id)
            )
            raise ParseError(f"分片上传失败: {str(e)}")

    async def upload_bytes(
        self,
        data: bytes,
        key: str,
        metadata: Optional[Dict[str, Any]] = None
    ) -> UploadResult:
        """上传字节数据"""
        loop = asyncio.get_event_loop()

        headers = {}
        if metadata:
            for k, v in metadata.items():
                headers[f'x-oss-meta-{k}'] = str(v)

        await loop.run_in_executor(
            None,
            lambda: self.bucket.put_object(key, data, headers=headers)
        )

        file_url = self._get_public_url(key)

        return UploadResult(
            success=True,
            file_url=file_url,
            file_key=key,
            file_size=len(data),
            provider=self.provider,
            metadata=metadata or {}
        )

    async def download_file(self, key: str, local_path: str) -> str:
        """下载文件"""
        loop = asyncio.get_event_loop()
        await loop.run_in_executor(
            None,
            lambda: self.bucket.get_object_to_file(key, local_path)
        )
        return local_path

    async def delete_file(self, key: str) -> bool:
        """删除文件"""
        try:
            loop = asyncio.get_event_loop()
            await loop.run_in_executor(
                None,
                lambda: self.bucket.delete_object(key)
            )
            return True
        except oss2.exceptions.OssError:
            return False

    async def file_exists(self, key: str) -> bool:
        """检查文件是否存在"""
        try:
            loop = asyncio.get_event_loop()
            await loop.run_in_executor(
                None,
                lambda: self.bucket.head_object(key)
            )
            return True
        except oss2.exceptions.OssError:
            return False

    async def get_file_url(self, key: str, expires_in: int = 3600) -> str:
        """获取文件URL"""
        url = self.bucket.sign_url('GET', key, expires_in)
        return url

    async def get_file_metadata(self, key: str) -> Dict[str, Any]:
        """获取文件元数据"""
        try:
            loop = asyncio.get_event_loop()
            meta = await loop.run_in_executor(
                None,
                lambda: self.bucket.head_object(key)
            )
            return {
                'content_length': meta.content_length,
                'content_type': meta.content_type,
                'last_modified': meta.last_modified,
                'metadata': dict(meta.metadata) if meta.metadata else {},
                'etag': meta.etag
            }
        except oss2.exceptions.OssError:
            return {}

    def _get_public_url(self, key: str) -> str:
        """获取公开URL"""
        if storage_settings.cdn_enabled and storage_settings.cdn_domain:
            return f"https://{storage_settings.cdn_domain}/{key}"
        else:
            return f"https://{self.bucket.bucket_name}.{storage_settings.oss_endpoint}/{key}"
```

### 5. 存储服务类

```python
# backend/app/services/storage_service.py
from typing import Optional, Dict, Any, Callable
import logging

from app.services.storage.base import (
    StorageAdapter,
    StorageProvider,
    UploadResult,
    UploadProgress
)
from app.services.storage.s3_adapter import S3Adapter
from app.services.storage.oss_adapter import OSSAdapter
from app.core.config import storage_settings
from app.core.exceptions import ParseError

logger = logging.getLogger(__name__)

class StorageService:
    """存储服务"""

    def __init__(self):
        self.adapter = self._create_adapter()

    def _create_adapter(self) -> StorageAdapter:
        """创建存储适配器"""
        provider = storage_settings.provider.lower()

        if provider == StorageProvider.S3:
            return S3Adapter()
        elif provider == StorageProvider.OSS:
            return OSSAdapter()
        else:
            raise ParseError(f"不支持的存储提供商: {provider}")

    async def upload_scenario_file(
        self,
        file_path: str,
        filename: str,
        user_id: Optional[str] = None,
        metadata: Optional[Dict[str, Any]] = None,
        progress_callback: Optional[Callable[[UploadProgress], None]] = None
    ) -> UploadResult:
        """
        上传场景包文件

        Args:
            file_path: 本地文件路径
            filename: 文件名
            user_id: 用户ID
            metadata: 额外元数据
            progress_callback: 进度回调

        Returns:
            UploadResult: 上传结果
        """
        key = self.adapter.generate_key(filename, user_id)

        if metadata:
            metadata['original_filename'] = filename
            metadata['user_id'] = user_id or 'anonymous'

        return await self.adapter.upload_file(
            file_path,
            key,
            metadata,
            progress_callback
        )

    async def delete_file(self, key: str) -> bool:
        """删除文件"""
        return await self.adapter.delete_file(key)

    async def get_file_url(self, key: str, expires_in: int = 3600) -> str:
        """获取文件URL"""
        return await self.adapter.get_file_url(key, expires_in)
```

### 6. API 路由

```python
# backend/app/api/v1/endpoints/storage.py
from fastapi import APIRouter, UploadFile, File, Depends, HTTPException, BackgroundTasks
from fastapi.responses import JSONResponse
import tempfile
import os

from app.services.storage_service import StorageService
from app.api.deps import get_current_user

router = APIRouter()
storage_service = StorageService()

@router.post("/upload")
async def upload_scenario(
    background_tasks: BackgroundTasks,
    file: UploadFile = File(...),
    current_user = Depends(get_current_user)
):
    """
    上传场景包到 CDN

    - **file**: 上传的文件
    - 返回上传结果和文件URL
    """
    try:
        # 保存临时文件
        with tempfile.NamedTemporaryFile(delete=False) as tmp:
            content = await file.read()
            tmp.write(content)
            tmp_path = tmp.name

        # 上传到 CDN
        result = await storage_service.upload_scenario_file(
            tmp_path,
            file.filename,
            user_id=str(current_user.id) if current_user else None
        )

        # 后台删除临时文件
        background_tasks.add_task(os.unlink, tmp_path)

        return JSONResponse(content={
            "success": True,
            "file_url": result.file_url,
            "file_key": result.file_key,
            "file_size": result.file_size,
            "provider": result.provider.value,
            "metadata": result.metadata
        })

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.delete("/files/{key}")
async def delete_file(
    key: str,
    current_user = Depends(get_current_user)
):
    """
    删除文件

    - **key**: 文件键名
    """
    try:
        success = await storage_service.delete_file(key)
        return {
            "success": success
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
```

---

## TypeScript/React 前端实现

### 1. 上传服务

```typescript
// frontend/src/services/api/storage.ts
import api, { getApiUrl } from './client';

export interface UploadProgress {
  uploaded_bytes: number;
  total_bytes: number;
  percentage: number;
  current_chunk: number;
  total_chunks: number;
}

export interface UploadResult {
  success: boolean;
  file_url: string;
  file_key: string;
  file_size: number;
  provider: string;
  metadata: Record<string, any>;
}

class StorageService {
  /**
   * 上传场景包到 CDN
   */
  async uploadScenario(
    file: File,
    onProgress?: (progress: UploadProgress) => void
  ): Promise<UploadResult> {
    const formData = new FormData();
    formData.append('file', file);

    try {
      const response = await api.post<UploadResult>(
        '/api/v1/storage/upload',
        formData,
        {
          headers: {
            'Content-Type': 'multipart/form-data',
          },
          onUploadProgress: (progressEvent) => {
            if (onProgress && progressEvent.total) {
              const uploaded = progressEvent.loaded || 0;
              const total = progressEvent.total;
              onProgress({
                uploaded_bytes: uploaded,
                total_bytes: total,
                percentage: (uploaded / total) * 100,
                current_chunk: 1,
                total_chunks: 1,
              });
            }
          },
        }
      );

      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '上传失败');
    }
  }

  /**
   * 删除文件
   */
  async deleteFile(key: string): Promise<boolean> {
    try {
      const response = await api.delete<{ success: boolean }>(
        `/api/v1/storage/files/${key}`
      );
      return response.data.success;
    } catch (error) {
      return false;
    }
  }
}

export default new StorageService();
```

### 2. 上传组件

```typescript
// frontend/src/components/scenario/ScenarioUploader.tsx
import React, { useState } from 'react';
import {
  Upload,
  Button,
  Progress,
  Card,
  message,
  Space,
  Tag,
} from 'antd';
import {
  CloudUploadOutlined,
  CheckCircleOutlined,
  LinkOutlined,
} from '@ant-design/icons';
import type { UploadChangeParam } from 'antd/es/upload';
import storageService, {
  UploadProgress,
  UploadResult,
} from '@/services/api/storage';

const { Dragger } = Upload;

const ScenarioUploader: React.FC = () => {
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);
  const [uploadResult, setUploadResult] = useState<UploadResult | null>(null);

  const handleUpload = async (file: File) => {
    setUploading(true);
    setProgress(0);
    setUploadResult(null);

    try {
      const result = await storageService.uploadScenario(
        file,
        (progressData: UploadProgress) => {
          setProgress(progressData.percentage);
        }
      );

      setUploadResult(result);
      message.success('上传成功');
    } catch (error: any) {
      message.error(error.message || '上传失败');
    } finally {
      setUploading(false);
    }
  };

  const copyUrl = () => {
    if (uploadResult?.file_url) {
      navigator.clipboard.writeText(uploadResult.file_url);
      message.success('链接已复制');
    }
  };

  const uploadProps = {
    name: 'file',
    multiple: false,
    accept: '.scenario,.encrypted,.json',
    showUploadList: false,
    beforeUpload: (file: File) => {
      const maxSize = 500 * 1024 * 1024; // 500MB
      if (file.size > maxSize) {
        message.error('文件大小不能超过 500MB');
        return Upload.LIST_IGNORE;
      }
      handleUpload(file);
      return false;
    },
  };

  return (
    <Card title="上传到 CDN" bordered={false}>
      <Space direction="vertical" style={{ width: '100%' }} size="large">
        <Dragger {...uploadProps} disabled={uploading}>
          <p className="ant-upload-drag-icon">
            <CloudUploadOutlined style={{ fontSize: 48, color: '#1890ff' }} />
          </p>
          <p className="ant-upload-text">点击或拖拽文件到此区域上传</p>
          <p className="ant-upload-hint">支持 .scenario, .encrypted, .json 格式，最大 500MB</p>
        </Dragger>

        {uploading && (
          <div>
            <div style={{ marginBottom: 8 }}>上传中...</div>
            <Progress percent={Math.round(progress)} status="active" />
          </div>
        )}

        {uploadResult && (
          <Card type="inner" title={<><CheckCircleOutlined /> 上传成功</>}>
            <Space direction="vertical" style={{ width: '100%' }}>
              <div>
                <Tag color="blue">文件 URL</Tag>
                <div style={{ marginTop: 8, wordBreak: 'break-all' }}>
                  {uploadResult.file_url}
                </div>
              </div>

              <div>
                <Tag color="green">存储服务</Tag>
                <span style={{ marginLeft: 8 }}>
                  {uploadResult.provider.toUpperCase()}
                </span>
              </div>

              <div>
                <Tag color="purple">文件大小</Tag>
                <span style={{ marginLeft: 8 }}>
                  {(uploadResult.file_size / 1024 / 1024).toFixed(2)} MB
                </span>
              </div>

              <Button
                type="primary"
                icon={<LinkOutlined />}
                onClick={copyUrl}
              >
                复制链接
              </Button>
            </Space>
          </Card>
        )}
      </Space>
    </Card>
  );
};

export default ScenarioUploader;
```

---

## 涉及文件清单

### 新建文件

| 文件路径 | 说明 |
|---------|------|
| `/backend/app/services/storage/base.py` | 存储适配器基类 |
| `/backend/app/services/storage/s3_adapter.py` | AWS S3 适配器 |
| `/backend/app/services/storage/oss_adapter.py` | 阿里云 OSS 适配器 |
| `/backend/app/services/storage_service.py` | 存储服务类 |
| `/backend/app/api/v1/endpoints/storage.py` | 存储API路由 |
| `/backend/app/core/config.py` | 添加存储配置 |

| 文件路径 | 说明 |
|---------|------|
| `/frontend/src/services/api/storage.ts` | 存储服务API |
| `/frontend/src/components/scenario/ScenarioUploader.tsx` | 上传组件 |

---

## 验收标准

### 功能验收

- [ ] 成功上传文件到 AWS S3
- [ ] 成功上传文件到阿里云 OSS
- [ ] 支持大文件分片上传（>100MB）
- [ ] 实时显示上传进度
- [ ] 上传失败自动重试
- [ ] 返回可访问的 CDN URL

### 性能验收

- [ ] 10MB 文件上传时间 < 30秒
- [ ] 分片上传支持断点续传
- [ ] 并发上传支持

### 异常处理验收

- [ ] 文件过大返回明确错误
- [ ] 网络错误自动重试
- [ ] 无效文件类型拒绝上传

---

## 参考文档

### 内部文档

- [M4-012: 场景包加密](/Users/guochangxi/git/monika/docs/tasks/tasks-detailed/M4-012-package-encryption.md)

### 技术文档

- [AWS S3 Documentation](https://docs.aws.amazon.com/s3/)
- [Boto3 Documentation](https://boto3.amazonaws.com/v1/documentation/api/latest/index.html)
- [阿里云 OSS 文档](https://help.aliyun.com/product/31815.html)
- [Multipart Upload](https://docs.aws.amazon.com/AmazonS3/latest/userguide/mpuoverview.html)

---

**创建日期**: 2026-02-06
**最后更新**: 2026-02-06
