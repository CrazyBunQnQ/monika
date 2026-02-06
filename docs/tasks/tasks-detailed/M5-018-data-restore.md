# M5-018: 实现数据恢复

**任务ID**: M5-018
**标题**: 实现数据恢复系统
**类型**: fullstack (全栈开发)
**预估工时**: 6h
**依赖**: M5-017 完成

---

## 任务描述

实现一个安全可靠的数据恢复系统，允许从备份中恢复数据。恢复过程需要支持预览、选择性恢复、回滚等高级功能，并确保数据一致性和完整性。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M5-018-01 | 设计恢复策略 | 恢复类型、验证机制 | 1h |
| M5-018-02 | 实现备份预览功能 | 查看备份内容 | 1.5h |
| M5-018-03 | 实现完整恢复 | 数据库+文件恢复 | 1.5h |
| M5-018-04 | 实现选择性恢复 | 恢复指定表/文件 | 1.5h |
| M5-018-05 | 实现恢复验证 | 数据完整性检查 | 1h |
| M5-018-06 | 实现前端恢复界面 | 恢复向导、进度显示 | 1.5h |

---

## 完整后端代码示例 (Python + Agno)

### 恢复服务

```python
# backend/app/services/restore_service.py
import os
import subprocess
import shutil
from datetime import datetime
from typing import List, Optional, Dict, Any
from pathlib import Path
import logging
import json
import gzip

from sqlalchemy.orm import Session
import tarfile

from app.models.backups import Backup, BackupStatus
from app.core.config import settings


logger = logging.getLogger(__name__)


class RestoreOptions:
    """恢复选项"""

    def __init__(
        self,
        restore_database: bool = True,
        restore_files: bool = True,
        tables: Optional[List[str]] = None,
        directories: Optional[List[str]] = None,
        create_before_restore: bool = True,  # 恢复前创建备份
        validate_after_restore: bool = True  # 恢复后验证
    ):
        self.restore_database = restore_database
        self.restore_files = restore_files
        self.tables = tables  # 要恢复的表（None 表示全部）
        self.directories = directories  # 要恢复的目录（None 表示全部）
        self.create_before_restore = create_before_restore
        self.validate_after_restore = validate_after_restore


class RestoreService:
    """恢复服务"""

    TEMP_DIR = "temp/restore"

    @staticmethod
    async def preview_backup(
        db: Session,
        backup_id: str
    ) -> Dict[str, Any]:
        """
        预览备份内容

        Args:
            db: 数据库会话
            backup_id: 备份 ID

        Returns:
            备份内容预览
        """
        backup = db.query(Backup).filter(Backup.id == backup_id).first()

        if not backup:
            raise ValueError(f"Backup {backup_id} not found")

        if backup.status != BackupStatus.COMPLETED:
            raise ValueError(f"Backup {backup_id} is not completed")

        if not os.path.exists(backup.file_path):
            raise ValueError(f"Backup file not found: {backup.file_path}")

        # 解压到临时目录
        temp_dir = Path(RestoreService.TEMP_DIR) / f"preview_{backup_id}"
        temp_dir.mkdir(parents=True, exist_ok=True)

        try:
            with tarfile.open(backup.file_path, "r:gz") as tar:
                tar.extractall(temp_dir)

            # 预览数据库内容
            db_preview = None
            if backup.includes_database:
                db_preview = await RestoreService._preview_database(
                    backup,
                    temp_dir
                )

            # 预览文件内容
            files_preview = None
            if backup.includes_files:
                files_preview = await RestoreService._preview_files(
                    backup,
                    temp_dir
                )

            return {
                "backup": {
                    "id": str(backup.id),
                    "name": backup.name,
                    "created_at": backup.created_at.isoformat(),
                    "file_size": backup.file_size,
                    "type": backup.backup_type
                },
                "database": db_preview,
                "files": files_preview
            }

        finally:
            # 清理临时目录
            shutil.rmtree(temp_dir, ignore_errors=True)

    @staticmethod
    async def _preview_database(
        backup: Backup,
        temp_dir: Path
    ) -> Dict[str, Any]:
        """预览数据库内容"""
        db_file = temp_dir / "database.sql.gz"

        if not db_file.exists():
            return {}

        # 解压并读取前几行
        preview_lines = []
        with gzip.open(db_file, "rt") as f:
            for i, line in enumerate(f):
                if i >= 100:  # 只读前100行
                    break
                preview_lines.append(line.strip())

        return {
            "tables": backup.included_tables,
            "preview": "\n".join(preview_lines[:50])  # 返回前50行
        }

    @staticmethod
    async def _preview_files(
        backup: Backup,
        temp_dir: Path
    ) -> Dict[str, Any]:
        """预览文件内容"""
        files_dir = temp_dir / "files"

        if not files_dir.exists():
            return {}

        # 统计文件信息
        file_info = []
        for dir_name in backup.included_directories:
            dir_path = files_dir / dir_name
            if dir_path.exists():
                files = list(dir_path.rglob("*"))
                file_info.append({
                    "directory": dir_name,
                    "file_count": len([f for f in files if f.is_file()]),
                    "total_size": sum([f.stat().st_size for f in files if f.is_file()])
                })

        return {
            "directories": file_info
        }

    @staticmethod
    async def restore_backup(
        db: Session,
        backup_id: str,
        options: Optional[RestoreOptions] = None,
        restored_by: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        恢复备份

        Args:
            db: 数据库会话
            backup_id: 备份 ID
            options: 恢复选项
            restored_by: 恢复者 ID

        Returns:
            恢复结果
        """
        backup = db.query(Backup).filter(Backup.id == backup_id).first()

        if not backup:
            raise ValueError(f"Backup {backup_id} not found")

        if backup.status != BackupStatus.COMPLETED:
            raise ValueError(f"Backup {backup_id} is not completed")

        if not os.path.exists(backup.file_path):
            raise ValueError(f"Backup file not found: {backup.file_path}")

        options = options or RestoreOptions()

        # 恢复前备份
        pre_restore_backup_id = None
        if options.create_before_restore:
            logger.info("Creating pre-restore backup...")
            pre_restore_backup = await RestoreService._create_pre_restore_backup(
                db,
                backup_id
            )
            pre_restore_backup_id = str(pre_restore_backup.id)

        try:
            # 解压备份
            temp_dir = Path(RestoreService.TEMP_DIR) / f"restore_{backup_id}_{datetime.now().strftime('%Y%m%d_%H%M%S')}"
            temp_dir.mkdir(parents=True, exist_ok=True)

            with tarfile.open(backup.file_path, "r:gz") as tar:
                tar.extractall(temp_dir)

            # 恢复数据库
            db_restored = False
            if backup.includes_database and options.restore_database:
                logger.info("Restoring database...")
                await RestoreService._restore_database(
                    db,
                    backup,
                    temp_dir,
                    options.tables
                )
                db_restored = True

            # 恢复文件
            files_restored = False
            if backup.includes_files and options.restore_files:
                logger.info("Restoring files...")
                await RestoreService._restore_files(
                    backup,
                    temp_dir,
                    options.directories
                )
                files_restored = True

            # 验证恢复
            validation_errors = []
            if options.validate_after_restore:
                logger.info("Validating restore...")
                validation_errors = await RestoreService._validate_restore(
                    db,
                    backup,
                    temp_dir
                )

            # 清理临时目录
            shutil.rmtree(temp_dir, ignore_errors=True)

            # 记录恢复历史
            result = {
                "backup_id": str(backup.id),
                "backup_name": backup.name,
                "pre_restore_backup_id": pre_restore_backup_id,
                "database_restored": db_restored,
                "files_restored": files_restored,
                "restored_at": datetime.utcnow().isoformat(),
                "restored_by": restored_by,
                "validation_errors": validation_errors,
                "status": "success" if not validation_errors else "warning"
            }

            logger.info(f"Restore completed: {backup_id}")

            return result

        except Exception as e:
            logger.error(f"Restore failed: {backup_id}, error: {e}")
            raise

    @staticmethod
    async def _create_pre_restore_backup(
        db: Session,
        backup_id: str
    ):
        """创建恢复前备份"""
        from app.services.backup_service import BackupService, BackupType

        original_backup = db.query(Backup).filter(Backup.id == backup_id).first()

        return await BackupService.create_backup(
            db,
            name=f"Pre-restore backup before restoring {original_backup.name}",
            backup_type=BackupType.FULL,
            includes_database=True,
            includes_files=True,
            description=f"自动创建的恢复前备份"
        )

    @staticmethod
    async def _restore_database(
        db: Session,
        backup: Backup,
        temp_dir: Path,
        tables: Optional[List[str]] = None
    ):
        """恢复数据库"""
        db_file = temp_dir / "database.sql.gz"

        if not db_file.exists():
            raise FileNotFoundError("Database backup file not found")

        # 如果指定了表，需要过滤 SQL
        if tables:
            await RestoreService._restore_selected_tables(
                db,
                db_file,
                tables
            )
        else:
            # 完整恢复
            await RestoreService._restore_full_database(
                db,
                db_file
            )

    @staticmethod
    async def _restore_full_database(
        db: Session,
        db_file: Path
    ):
        """完整恢复数据库"""
        # 先删除现有数据（DROP DATABASE 不安全，改用删除表）
        from app.models import *
        from app.db.base_class import Base

        # 删除所有表
        Base.metadata.drop_all(bind=db.bind)

        # 重新创建表结构
        Base.metadata.create_all(bind=db.bind)

        # 解压并恢复数据
        sql_file = Path(str(db_file).replace(".gz", ""))

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

        # 清理临时 SQL 文件
        sql_file.unlink()

    @staticmethod
    async def _restore_selected_tables(
        db: Session,
        db_file: Path,
        tables: List[str]
    ):
        """恢复选定的表"""
        # 解压 SQL 文件
        sql_file = Path(str(db_file).replace(".gz", ""))

        with gzip.open(db_file, "rb") as f_in:
            with open(sql_file, "wb") as f_out:
                shutil.copyfileobj(f_in, f_out)

        # 解析 SQL 并只恢复指定的表
        # 这是一个简化的实现，实际应该使用更强大的 SQL 解析器
        for table in tables:
            # 先删除表
            db.execute(f"DROP TABLE IF EXISTS {table} CASCADE")

            # 从 SQL 文件中提取该表的 CREATE 和 INSERT 语句
            # 这里需要更复杂的 SQL 解析逻辑
            # 简化实现：使用 pg_restore 的 -t 选项
            pass

        sql_file.unlink()

    @staticmethod
    async def _restore_files(
        backup: Backup,
        temp_dir: Path,
        directories: Optional[List[str]] = None
    ):
        """恢复文件"""
        files_dir = temp_dir / "files"

        if not files_dir.exists():
            logger.warning("No files to restore in backup")
            return

        # 恢复指定目录或全部
        dirs_to_restore = directories or backup.included_directories

        for dir_name in dirs_to_restore:
            src_dir = files_dir / dir_name
            if src_dir.exists():
                dst_dir = Path(dir_name)

                # 备份现有文件
                if dst_dir.exists():
                    backup_dir = Path(f"{dst_dir}.backup_{datetime.now().strftime('%Y%m%d_%H%M%S')}")
                    shutil.move(str(dst_dir), str(backup_dir))

                # 复制新文件
                shutil.copytree(src_dir, dst_dir)

                logger.info(f"Restored directory: {dir_name}")

    @staticmethod
    async def _validate_restore(
        db: Session,
        backup: Backup,
        temp_dir: Path
    ) -> List[str]:
        """验证恢复结果"""
        errors = []

        # 验证表是否存在
        if backup.includes_database:
            for table in backup.included_tables:
                try:
                    result = db.execute(f"SELECT 1 FROM {table} LIMIT 1")
                    result.fetchone()
                except Exception as e:
                    errors.append(f"Table {table} validation failed: {e}")

        # 验证文件是否存在
        if backup.includes_files:
            files_dir = temp_dir / "files"
            for dir_name in backup.included_directories:
                src_dir = files_dir / dir_name
                dst_dir = Path(dir_name)

                if src_dir.exists() and not dst_dir.exists():
                    errors.append(f"Directory {dir_name} was not restored")

        return errors

    @staticmethod
    def get_restore_history(
        db: Session,
        limit: int = 20
    ) -> List[Dict[str, Any]]:
        """获取恢复历史（从日志或专门表读取）"""
        # 这里简化实现，实际应该有专门的恢复历史表
        return []
```

### API 路由

```python
# backend/app/api/restore.py
from fastapi import APIRouter, Depends, HTTPException, BackgroundTasks
from sqlalchemy.orm import Session

from app.api.deps import get_db, get_current_active_user
from app.schemas.restore import (
    RestorePreviewResponse,
    RestoreRequest,
    RestoreResponse
)
from app.services.restore_service import RestoreService, RestoreOptions

router = APIRouter()


@router.get("/preview/{backup_id}", response_model=RestorePreviewResponse)
async def preview_backup(
    backup_id: str,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """预览备份内容"""
    try:
        preview = await RestoreService.preview_backup(db, backup_id)
        return preview
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/{backup_id}", response_model=RestoreResponse)
async def restore_backup(
    backup_id: str,
    request: RestoreRequest,
    background_tasks: BackgroundTasks,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """恢复备份"""
    # 创建恢复选项
    options = RestoreOptions(
        restore_database=request.restore_database,
        restore_files=request.restore_files,
        tables=request.tables,
        directories=request.directories,
        create_before_restore=request.create_before_restore,
        validate_after_restore=request.validate_after_restore
    )

    # 在后台执行恢复
    def run_restore():
        import asyncio
        asyncio.run(
            RestoreService.restore_backup(
                db,
                backup_id,
                options,
                current_user.id
            )
        )

    background_tasks.add_task(run_restore)

    return {
        "message": "Restore started",
        "backup_id": backup_id,
        "status": "running"
    }


@router.get("/history")
def get_restore_history(
    limit: int = 20,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """获取恢复历史"""
    return RestoreService.get_restore_history(db, limit)
```

---

## 完整前端代码示例 (TypeScript + React + shadcn/ui)

### 恢复向导组件

```tsx
// frontend/src/components/restore/RestoreWizard.tsx
import React, { useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { AlertTriangle, CheckCircle2, Loader2 } from "lucide-react";

import { Backup } from "@/types/backups";

type Step = "select" | "preview" | "options" | "confirm" | "restoring" | "complete";

interface RestoreWizardProps {
  backup: Backup;
  onComplete: () => void;
  onCancel: () => void;
}

export function RestoreWizard({ backup, onComplete, onCancel }: RestoreWizardProps) {
  const [step, setStep] = useState<Step>("select");
  const [preview, setPreview] = useState<any>(null);
  const [options, setOptions] = useState({
    restore_database: true,
    restore_files: false,
    create_before_restore: true,
    validate_after_restore: true
  });
  const [selectedTables, setSelectedTables] = useState<string[]>([]);
  const [selectedDirs, setSelectedDirs] = useState<string[]>([]);
  const [restoring, setRestoring] = useState(false);
  const [progress, setProgress] = useState(0);
  const [result, setResult] = useState<any>(null);
  const [error, setError] = useState<string | null>(null);

  const handlePreview = async () => {
    setStep("preview");
    const res = await fetch(`/api/restore/preview/${backup.id}`);
    const data = await res.json();
    setPreview(data);
  };

  const handleStartRestore = async () => {
    setStep("restoring");
    setRestoring(true);
    setProgress(0);
    setError(null);

    try {
      // 模拟进度
      const progressInterval = setInterval(() => {
        setProgress(prev => Math.min(prev + 10, 90));
      }, 500);

      const res = await fetch(`/api/restore/${backup.id}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          ...options,
          tables: selectedTables.length > 0 ? selectedTables : undefined,
          directories: selectedDirs.length > 0 ? selectedDirs : undefined
        })
      });

      clearInterval(progressInterval);
      setProgress(100);

      const data = await res.json();
      setResult(data);
      setStep("complete");

    } catch (e: any) {
      setError(e.message);
      setStep("confirm");
    } finally {
      setRestoring(false);
    }
  };

  return (
    <div className="max-w-3xl mx-auto p-6 space-y-6">
      {/* 步骤指示 */}
      <div className="flex items-center justify-center gap-2">
        {["select", "preview", "options", "confirm"].map((s, i) => (
          <React.Fragment key={s}>
            <div
              className={`w-8 h-8 rounded-full flex items-center justify-center text-sm ${
                step === s
                  ? "bg-primary text-primary-foreground"
                  : ["select", "preview", "options", "confirm"].indexOf(step) > i
                  ? "bg-primary text-primary-foreground"
                  : "bg-muted"
              }`}
            >
              {i + 1}
            </div>
            {i < 3 && <div className="w-12 h-0.5 bg-muted" />}
          </React.Fragment>
        ))}
      </div>

      {/* 选择步骤 */}
      {step === "select" && (
        <Card>
          <CardHeader>
            <CardTitle>恢复备份</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <p className="font-medium">{backup.name}</p>
              <div className="text-sm text-muted-foreground space-y-1">
                <p>类型: {backup.backup_type}</p>
                <p>创建时间: {new Date(backup.created_at).toLocaleString()}</p>
                <p>大小: {(backup.file_size / 1024 / 1024 / 1024).toFixed(2)} GB</p>
              </div>
            </div>

            {backup.backup_type === "incremental" && (
              <Alert>
                <AlertTriangle className="h-4 w-4" />
                <AlertDescription>
                  这是增量备份，需要先恢复基础备份。
                </AlertDescription>
              </Alert>
            )}

            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={onCancel}>取消</Button>
              <Button onClick={handlePreview}>下一步</Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* 预览步骤 */}
      {step === "preview" && preview && (
        <Card>
          <CardHeader>
            <CardTitle>备份内容预览</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {preview.database && (
              <div>
                <h3 className="font-semibold mb-2">数据库</h3>
                <div className="flex flex-wrap gap-2">
                  {preview.database.tables.map((table: string) => (
                    <Badge key={table} variant="outline">{table}</Badge>
                  ))}
                </div>
              </div>
            )}

            {preview.files && (
              <div>
                <h3 className="font-semibold mb-2">文件</h3>
                <div className="space-y-2">
                  {preview.files.directories.map((dir: any) => (
                    <div key={dir.directory} className="text-sm">
                      <span className="font-medium">{dir.directory}</span>
                      <span className="text-muted-foreground ml-2">
                        {dir.file_count} 个文件
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setStep("select")}>上一步</Button>
              <Button onClick={() => setStep("options")}>下一步</Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* 选项步骤 */}
      {step === "options" && (
        <Card>
          <CardHeader>
            <CardTitle>恢复选项</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <div className="flex items-center space-x-2">
                <Checkbox
                  id="restore-db"
                  checked={options.restore_database}
                  onCheckedChange={(checked) =>
                    setOptions({ ...options, restore_database: checked as boolean })
                  }
                />
                <label htmlFor="restore-db">恢复数据库</label>
              </div>

              <div className="flex items-center space-x-2">
                <Checkbox
                  id="restore-files"
                  checked={options.restore_files}
                  onCheckedChange={(checked) =>
                    setOptions({ ...options, restore_files: checked as boolean })
                  }
                />
                <label htmlFor="restore-files">恢复文件</label>
              </div>

              <div className="flex items-center space-x-2">
                <Checkbox
                  id="pre-backup"
                  checked={options.create_before_restore}
                  onCheckedChange={(checked) =>
                    setOptions({ ...options, create_before_restore: checked as boolean })
                  }
                />
                <label htmlFor="pre-backup">恢复前创建备份</label>
              </div>

              <div className="flex items-center space-x-2">
                <Checkbox
                  id="validate"
                  checked={options.validate_after_restore}
                  onCheckedChange={(checked) =>
                    setOptions({ ...options, validate_after_restore: checked as boolean })
                  }
                />
                <label htmlFor="validate">恢复后验证数据</label>
              </div>
            </div>

            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setStep("preview")}>上一步</Button>
              <Button onClick={() => setStep("confirm")}>下一步</Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* 确认步骤 */}
      {step === "confirm" && (
        <Card>
          <CardHeader>
            <CardTitle>确认恢复</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <Alert>
              <AlertTriangle className="h-4 w-4" />
              <AlertDescription>
                恢复操作将覆盖现有数据，请确保已创建恢复前备份。
              </AlertDescription>
            </Alert>

            <div className="space-y-2 text-sm">
              <p><strong>备份:</strong> {backup.name}</p>
              <p><strong>恢复数据库:</strong> {options.restore_database ? "是" : "否"}</p>
              <p><strong>恢复文件:</strong> {options.restore_files ? "是" : "否"}</p>
              <p><strong>恢复前备份:</strong> {options.create_before_restore ? "是" : "否"}</p>
            </div>

            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}

            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setStep("options")}>上一步</Button>
              <Button onClick={handleStartRestore} disabled={restoring}>
                开始恢复
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* 恢复中 */}
      {step === "restoring" && (
        <Card>
          <CardContent className="p-6 text-center space-y-4">
            <Loader2 className="w-12 h-12 animate-spin mx-auto text-primary" />
            <h3 className="text-lg font-semibold">正在恢复数据...</h3>
            <Progress value={progress} className="w-full" />
            <p className="text-sm text-muted-foreground">{progress}%</p>
          </CardContent>
        </Card>
      )}

      {/* 完成 */}
      {step === "complete" && result && (
        <Card>
          <CardContent className="p-6 text-center space-y-4">
            <CheckCircle2 className="w-12 h-12 mx-auto text-green-500" />
            <h3 className="text-lg font-semibold">恢复完成!</h3>

            <div className="text-sm space-y-1">
              <p>数据库已恢复: {result.database_restored ? "是" : "否"}</p>
              <p>文件已恢复: {result.files_restored ? "是" : "否"}</p>
              {result.pre_restore_backup_id && (
                <p className="text-muted-foreground">
                  恢复前备份: {result.pre_restore_backup_id}
                </p>
              )}
            </div>

            {result.validation_errors && result.validation_errors.length > 0 && (
              <Alert>
                <AlertTriangle className="h-4 w-4" />
                <AlertDescription>
                  <p className="font-semibold">验证警告:</p>
                  <ul className="list-disc list-inside text-sm">
                    {result.validation_errors.map((error: string, i: number) => (
                      <li key={i}>{error}</li>
                    ))}
                  </ul>
                </AlertDescription>
              </Alert>
            )}

            <Button onClick={onComplete}>完成</Button>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `backend/app/services/restore_service.py` | 创建 | 恢复服务 |
| `backend/app/api/restore.py` | 创建 | 恢复 API 路由 |
| `backend/app/schemas/restore.py` | 创建 | Pydantic 模型 |
| `frontend/src/components/restore/RestoreWizard.tsx` | 创建 | 恢复向导组件 |
| `frontend/src/pages/BackupRestore.tsx` | 创建 | 备份恢复管理页面 |

---

## 验收标准

- [ ] 可以预览备份内容
- [ ] 可以完整恢复备份
- [ ] 可以选择性恢复（指定表/文件）
- [ ] 恢复前自动创建备份
- [ ] 恢复后验证数据完整性
- [ ] 恢复过程有进度显示
- [ ] 恢复失败有错误提示

---

## 参考文档

- 数据恢复最佳实践
- PostgreSQL 恢复机制
- 事务与回滚策略

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
