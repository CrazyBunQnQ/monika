# M5-017: 实现数据备份

**任务ID**: M5-017
**标题**: 实现数据备份系统
**类型**: backend (后端开发)
**预估工时**: 8h
**依赖**: M0 完成

---

## 任务描述

实现一个完整的数据备份系统，支持自动备份、手动备份、增量备份、备份恢复等功能。备份数据应包括用户数据、游戏数据、配置等所有关键信息。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M5-017-01 | 设计备份策略 | 备份类型、保留策略 | 1h |
| M5-017-02 | 实现数据库备份 | PostgreSQL dump/restore | 2h |
| M5-017-03 | 实现文件备份 | 上传文件、静态资源 | 1.5h |
| M5-017-04 | 实现备份调度器 | 自动备份、定时任务 | 1.5h |
| M5-017-05 | 实现备份管理 API | 创建、列表、删除备份 | 1h |
| M5-017-06 | 实现备份监控与告警 | 备份失败通知 | 1h |

---

## 完整后端代码示例 (Python + Agno)

### 备份数据模型

```python
# backend/app/models/backups.py
from datetime import datetime
from typing import Optional
from enum import Enum
from sqlalchemy import Column, String, JSON, DateTime, Boolean, BigInteger, Integer, ForeignKey, Text
from sqlalchemy.dialects.postgresql import UUID
import uuid

from app.db.base_class import Base


class BackupType(str, Enum):
    """备份类型"""
    FULL = "full"  # 完整备份
    INCREMENTAL = "incremental"  # 增量备份
    DATABASE = "database"  # 仅数据库
    FILES = "files"  # 仅文件


class BackupStatus(str, Enum):
    """备份状态"""
    PENDING = "pending"
    RUNNING = "running"
    COMPLETED = "completed"
    FAILED = "failed"
    CANCELLED = "cancelled"


class Backup(Base):
    """备份记录表"""
    __tablename__ = "backups"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)

    # 备份信息
    name = Column(String(255), nullable=False)
    description = Column(Text, nullable=True)
    backup_type = Column(String(20), nullable=False)

    # 状态
    status = Column(String(20), default=BackupStatus.PENDING)

    # 文件信息
    file_path = Column(String(500), nullable=True)  # 备份文件路径
    file_size = Column(BigInteger, nullable=True)  # 文件大小（字节）

    # 备份内容
    includes_database = Column(Boolean, default=True)
    includes_files = Column(Boolean, default=False)
    included_tables = Column(JSON, default=list)  # 备份的表列表
    included_directories = Column(JSON, default=list)  # 备份的目录列表

    # 基础备份（用于增量备份）
    base_backup_id = Column(UUID(as_uuid=True), ForeignKey("backups.id"), nullable=True)

    # 统计
    total_records = Column(Integer, nullable=True)  # 总记录数
    total_files = Column(Integer, nullable=True)  # 总文件数

    # 时间
    created_at = Column(DateTime, default=datetime.utcnow)
    started_at = Column(DateTime, nullable=True)
    completed_at = Column(DateTime, nullable=True)

    # 创建者
    created_by = Column(UUID(as_uuid=True), ForeignKey("accounts.id"))

    # 错误信息
    error_message = Column(Text, nullable=True)


class BackupSchedule(Base):
    """备份计划表"""
    __tablename__ = "backup_schedules"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)

    # 计划信息
    name = Column(String(255), nullable=False)
    description = Column(Text, nullable=True)

    # 备份配置
    backup_type = Column(String(20), nullable=False)
    includes_database = Column(Boolean, default=True)
    includes_files = Column(Boolean, default=False)

    # 调度配置
    schedule_type = Column(String(20), nullable=False)  # "daily", "weekly", "monthly"
    schedule_config = Column(JSON, nullable=False)  # {"hour": 2, "day_of_week": 0}

    # 保留策略
    retention_count = Column(Integer, default=7)  # 保留最近 N 个备份
    retention_days = Column(Integer, default=30)  # 保留 N 天内的备份

    # 是否启用
    is_active = Column(Boolean, default=True)

    # 最后执行
    last_run_at = Column(DateTime, nullable=True)
    last_backup_id = Column(UUID(as_uuid=True), ForeignKey("backups.id"), nullable=True)

    # 下次执行
    next_run_at = Column(DateTime, nullable=True)

    # 创建时间
    created_at = Column(DateTime, default=datetime.utcnow)
```

### 备份服务

```python
# backend/app/services/backup_service.py
import os
import subprocess
import shutil
from datetime import datetime, timedelta
from typing import List, Optional, Dict, Any
from pathlib import Path
import logging
import gzip
import json

from sqlalchemy.orm import Session

from app.models.backups import Backup, BackupType, BackupStatus, BackupSchedule
from app.core.config import settings

logger = logging.getLogger(__name__)


class BackupService:
    """备份服务"""

    BACKUP_DIR = "backups"
    TEMP_DIR = "temp/backups"

    @staticmethod
    def get_backup_path(backup_id: str) -> str:
        """获取备份文件路径"""
        return f"{BackupService.BACKUP_DIR}/{backup_id}.tar.gz"

    @staticmethod
    async def create_backup(
        db: Session,
        name: str,
        backup_type: BackupType,
        includes_database: bool = True,
        includes_files: bool = False,
        description: Optional[str] = None,
        base_backup_id: Optional[str] = None,
        created_by: Optional[str] = None
    ) -> Backup:
        """
        创建备份

        Args:
            db: 数据库会话
            name: 备份名称
            backup_type: 备份类型
            includes_database: 是否包含数据库
            includes_files: 是否包含文件
            description: 描述
            base_backup_id: 基础备份 ID（用于增量备份）
            created_by: 创建者 ID

        Returns:
            Backup 对象
        """
        # 创建备份记录
        backup = Backup(
            name=name,
            description=description,
            backup_type=backup_type.value,
            includes_database=includes_database,
            includes_files=includes_files,
            base_backup_id=base_backup_id,
            created_by=created_by,
            status=BackupStatus.PENDING
        )

        db.add(backup)
        db.commit()
        db.refresh(backup)

        # 异步执行备份
        import asyncio
        asyncio.create_task(
            BackupService._execute_backup(
                db,
                str(backup.id)
            )
        )

        return backup

    @staticmethod
    async def _execute_backup(db: Session, backup_id: str):
        """执行备份（后台任务）"""
        backup = db.query(Backup).filter(Backup.id == backup_id).first()

        if not backup:
            return

        try:
            # 更新状态
            backup.status = BackupStatus.RUNNING
            backup.started_at = datetime.utcnow()
            db.commit()

            # 创建临时目录
            temp_dir = Path(BackupService.TEMP_DIR) / backup_id
            temp_dir.mkdir(parents=True, exist_ok=True)

            # 备份数据库
            if backup.includes_database:
                await BackupService._backup_database(
                    db,
                    backup,
                    temp_dir
                )

            # 备份文件
            if backup.includes_files:
                await BackupService._backup_files(
                    backup,
                    temp_dir
                )

            # 打包压缩
            backup_file = BackupService.get_backup_path(backup_id)
            BackupService._create_tarball(temp_dir, backup_file)

            # 获取文件大小
            backup.file_size = os.path.getsize(backup_file)
            backup.file_path = backup_file

            # 清理临时目录
            shutil.rmtree(temp_dir)

            # 更新状态
            backup.status = BackupStatus.COMPLETED
            backup.completed_at = datetime.utcnow()

            db.commit()

            logger.info(f"Backup completed: {backup_id}")

        except Exception as e:
            backup.status = BackupStatus.FAILED
            backup.error_message = str(e)
            backup.completed_at = datetime.utcnow()
            db.commit()

            logger.error(f"Backup failed: {backup_id}, error: {e}")

    @staticmethod
    async def _backup_database(
        db: Session,
        backup: Backup,
        temp_dir: Path
    ):
        """备份数据库"""
        logger.info(f"Backing up database for {backup.id}")

        # 使用 pg_dump
        db_file = temp_dir / "database.sql"

        # 构建命令
        cmd = [
            "pg_dump",
            f"--dbname={settings.DATABASE_URL}",
            "--format=plain",
            "--no-owner",
            "--no-acl"
        ]

        # 执行备份
        with open(db_file, "w") as f:
            result = subprocess.run(
                cmd,
                stdout=f,
                stderr=subprocess.PIPE,
                text=True
            )

            if result.returncode != 0:
                raise Exception(f"Database backup failed: {result.stderr}")

        # 压缩
        with open(db_file, "rb") as f_in:
            with gzip.open(f"{db_file}.gz", "wb") as f_out:
                shutil.copyfileobj(f_in, f_out)

        os.remove(db_file)

        # 记录备份的表
        backup.included_tables = [
            "accounts", "campaigns", "sessions", "characters",
            "events", "scripts", "scenes", "triggers", "macros"
        ]

    @staticmethod
    async def _backup_files(
        backup: Backup,
        temp_dir: Path
    ):
        """备份文件"""
        logger.info(f"Backing up files for {backup.id}")

        files_dir = temp_dir / "files"
        files_dir.mkdir(exist_ok=True)

        # 备份上传的文件
        upload_dirs = ["uploads", "static"]

        for dir_name in upload_dirs:
            src_dir = Path(dir_name)
            if src_dir.exists():
                dst_dir = files_dir / dir_name
                shutil.copytree(src_dir, dst_dir)

                # 统计文件数
                backup.total_files = sum(
                    1 for _ in dst_dir.rglob("*") if _.is_file()
                )

        backup.included_directories = upload_dirs

    @staticmethod
    def _create_tarball(source_dir: Path, output_file: str):
        """创建 tar.gz 压缩包"""
        import tarfile

        with tarfile.open(output_file, "w:gz") as tar:
            tar.add(source_dir, arcname="")

    @staticmethod
    async def restore_backup(
        db: Session,
        backup_id: str
    ):
        """
        恢复备份

        Args:
            db: 数据库会话
            backup_id: 备份 ID
        """
        backup = db.query(Backup).filter(Backup.id == backup_id).first()

        if not backup:
            raise ValueError(f"Backup {backup_id} not found")

        if backup.status != BackupStatus.COMPLETED:
            raise ValueError(f"Backup {backup_id} is not completed")

        if not os.path.exists(backup.file_path):
            raise ValueError(f"Backup file not found: {backup.file_path}")

        try:
            # 解压
            temp_dir = Path(BackupService.TEMP_DIR) / f"restore_{backup_id}"
            temp_dir.mkdir(parents=True, exist_ok=True)

            BackupService._extract_tarball(backup.file_path, temp_dir)

            # 恢复数据库
            if backup.includes_database:
                await BackupService._restore_database(
                    db,
                    backup,
                    temp_dir
                )

            # 恢复文件
            if backup.includes_files:
                await BackupService._restore_files(
                    backup,
                    temp_dir
                )

            # 清理临时目录
            shutil.rmtree(temp_dir)

            logger.info(f"Backup restored: {backup_id}")

        except Exception as e:
            logger.error(f"Restore failed: {backup_id}, error: {e}")
            raise

    @staticmethod
    async def _restore_database(
        db: Session,
        backup: Backup,
        temp_dir: Path
    ):
        """恢复数据库"""
        logger.info(f"Restoring database from {backup.id}")

        db_file = temp_dir / "database.sql.gz"

        # 解压
        import gzip
        sql_file = temp_dir / "database.sql"
        with gzip.open(db_file, "rb") as f_in:
            with open(sql_file, "wb") as f_out:
                shutil.copyfileobj(f_in, f_out)

        # 使用 psql 恢复
        cmd = [
            "psql",
            f"--dbname={settings.DATABASE_URL}",
            "-f", str(sql_file)
        ]

        result = subprocess.run(cmd, capture_output=True, text=True)

        if result.returncode != 0:
            raise Exception(f"Database restore failed: {result.stderr}")

    @staticmethod
    async def _restore_files(
        backup: Backup,
        temp_dir: Path
    ):
        """恢复文件"""
        logger.info(f"Restoring files from {backup.id}")

        files_dir = temp_dir / "files"

        for dir_name in backup.included_directories:
            src_dir = files_dir / dir_name
            if src_dir.exists():
                dst_dir = Path(dir_name)
                # 删除旧文件
                if dst_dir.exists():
                    shutil.rmtree(dst_dir)
                # 复制新文件
                shutil.copytree(src_dir, dst_dir)

    @staticmethod
    def _extract_tarball(tarball_path: str, output_dir: Path):
        """解压 tar.gz"""
        import tarfile

        with tarfile.open(tarball_path, "r:gz") as tar:
            tar.extractall(output_dir)

    @staticmethod
    def delete_backup(db: Session, backup_id: str) -> bool:
        """删除备份"""
        backup = db.query(Backup).filter(Backup.id == backup_id).first()

        if not backup:
            return False

        # 删除文件
        if backup.file_path and os.path.exists(backup.file_path):
            os.remove(backup.file_path)

        # 删除记录
        db.delete(backup)
        db.commit()

        return True

    @staticmethod
    def get_backups(
        db: Session,
        limit: int = 50,
        offset: int = 0
    ) -> List[Backup]:
        """获取备份列表"""
        return db.query(Backup).order_by(
            Backup.created_at.desc()
        ).offset(offset).limit(limit).all()

    @staticmethod
    def get_backup_stats(db: Session) -> Dict[str, Any]:
        """获取备份统计"""
        total_backups = db.query(Backup).count()
        total_size = db.query(Backup).with_entities(
            Backup.file_size
        ).all()

        size_bytes = sum([s[0] or 0 for s in total_size])

        # 最近备份
        latest_backup = db.query(Backup).filter(
            Backup.status == BackupStatus.COMPLETED
        ).order_by(
            Backup.created_at.desc()
        ).first()

        return {
            "total_backups": total_backups,
            "total_size_bytes": size_bytes,
            "total_size_gb": round(size_bytes / (1024**3), 2),
            "latest_backup": latest_backup.created_at if latest_backup else None
        }
```

### 备份调度器

```python
# backend/app/services/backup_scheduler.py
from datetime import datetime, timedelta
from typing import List
import logging

from sqlalchemy.orm import Session
from apscheduler.schedulers.asyncio import AsyncIOScheduler

from app.models.backups import BackupSchedule, BackupType
from app.services.backup_service import BackupService


logger = logging.getLogger(__name__)


class BackupScheduler:
    """备份调度器"""

    def __init__(self):
        self.scheduler = AsyncIOScheduler()

    async def _run_backup(self, schedule_id: str, db: Session):
        """执行计划的备份"""
        schedule = db.query(BackupSchedule).filter(
            BackupSchedule.id == schedule_id
        ).first()

        if not schedule or not schedule.is_active:
            return

        try:
            # 创建备份
            backup = await BackupService.create_backup(
                db,
                name=f"{schedule.name} - {datetime.now().strftime('%Y-%m-%d %H:%M')}",
                backup_type=BackupType(schedule.backup_type),
                includes_database=schedule.includes_database,
                includes_files=schedule.includes_files,
                description=f"自动备份（计划: {schedule.name}）"
            )

            # 更新计划信息
            schedule.last_run_at = datetime.utcnow()
            schedule.last_backup_id = backup.id

            # 计算下次执行时间
            schedule.next_run_at = self._calculate_next_run(schedule)

            db.commit()

            logger.info(f"Scheduled backup executed: {schedule_id}")

        except Exception as e:
            logger.error(f"Scheduled backup failed: {schedule_id}, error: {e}")

    def _calculate_next_run(self, schedule: BackupSchedule) -> datetime:
        """计算下次执行时间"""
        config = schedule.schedule_config

        if schedule.schedule_type == "daily":
            # 每天执行
            hour = config.get("hour", 0)
            next_run = datetime.now().replace(hour=hour, minute=0, second=0, microsecond=0)
            if next_run <= datetime.now():
                next_run += timedelta(days=1)
            return next_run

        elif schedule.schedule_type == "weekly":
            # 每周执行
            day_of_week = config.get("day_of_week", 0)  # 0 = Monday
            hour = config.get("hour", 0)
            next_run = datetime.now()
            days_ahead = (day_of_week - next_run.weekday() + 7) % 7
            if days_ahead == 0 and next_run.hour >= hour:
                days_ahead = 7
            next_run = next_run + timedelta(days=days_ahead)
            next_run = next_run.replace(hour=hour, minute=0, second=0, microsecond=0)
            return next_run

        elif schedule.schedule_type == "monthly":
            # 每月执行
            day_of_month = config.get("day_of_month", 1)
            hour = config.get("hour", 0)
            next_run = datetime.now()
            if next_run.day > day_of_month:
                # 下个月
                if next_run.month == 12:
                    next_run = next_run.replace(year=next_run.year + 1, month=1, day=day_of_month)
                else:
                    next_run = next_run.replace(month=next_run.month + 1, day=day_of_month)
            else:
                next_run = next_run.replace(day=day_of_month)

            next_run = next_run.replace(hour=hour, minute=0, second=0, microsecond=0)
            return next_run

        return datetime.now() + timedelta(days=1)

    def add_schedule(self, schedule: BackupSchedule, db: Session):
        """添加备份计划"""
        # 计算下次执行时间
        schedule.next_run_at = self._calculate_next_run(schedule)

        # 添加到调度器
        self.scheduler.add_job(
            self._run_backup,
            'date',
            run_date=schedule.next_run_at,
            args=[str(schedule.id), db],
            id=str(schedule.id)
        )

        logger.info(f"Added backup schedule: {schedule.id}, next run: {schedule.next_run_at}")

    def start(self):
        """启动调度器"""
        self.scheduler.start()
        logger.info("Backup scheduler started")

    def shutdown(self):
        """关闭调度器"""
        self.scheduler.shutdown()
        logger.info("Backup scheduler shutdown")
```

### API 路由

```python
# backend/app/api/backups.py
from fastapi import APIRouter, Depends, HTTPException, BackgroundTasks
from sqlalchemy.orm import Session

from app.api.deps import get_db, get_current_active_user
from app.schemas.backups import (
    BackupCreate,
    BackupResponse,
    BackupRestoreResponse
)
from app.services.backup_service import BackupService

router = APIRouter()


@router.post("/", response_model=BackupResponse)
def create_backup(
    backup_in: BackupCreate,
    background_tasks: BackgroundTasks,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """创建备份"""
    return BackupService.create_backup(
        db,
        **backup_in.dict(),
        created_by=current_user.id
    )


@router.get("/", response_model=List[BackupResponse])
def list_backups(
    limit: int = 50,
    offset: int = 0,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """获取备份列表"""
    return BackupService.get_backups(db, limit, offset)


@router.get("/stats")
def get_backup_stats(
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """获取备份统计"""
    return BackupService.get_backup_stats(db)


@router.post("/{backup_id}/restore")
def restore_backup(
    backup_id: str,
    background_tasks: BackgroundTasks,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """恢复备份"""
    background_tasks.add_task(BackupService.restore_backup, db, backup_id)
    return {"message": "Backup restore started"}


@router.delete("/{backup_id}")
def delete_backup(
    backup_id: str,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """删除备份"""
    success = BackupService.delete_backup(db, backup_id)
    if not success:
        raise HTTPException(status_code=404, detail="Backup not found")
    return {"message": "Backup deleted"}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `backend/app/models/backups.py` | 创建 | 备份数据模型 |
| `backend/app/services/backup_service.py` | 创建 | 备份服务 |
| `backend/app/services/backup_scheduler.py` | 创建 | 备份调度器 |
| `backend/app/api/backups.py` | 创建 | 备份 API 路由 |
| `backend/app/schemas/backups.py` | 创建 | Pydantic 模型 |
| `backend/app/db/migrations/versions/xxx_create_backups.py` | 创建 | 数据库迁移 |
| `scripts/backup.sh` | 创建 | 备份脚本 |

---

## 验收标准

- [ ] 可以创建完整备份
- [ ] 可以创建增量备份
- [ ] 可以成功恢复备份
- [ ] 自动备份按计划执行
- [ ] 备份文件正确压缩和存储
- [ ] 提供备份统计信息
- [ ] 备份失败有错误日志

---

## 参考文档

- PostgreSQL 备份恢复最佳实践
- Python 异步任务调度
- 数据备份策略设计

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
