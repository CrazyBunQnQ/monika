# M5-016: 实现插件系统

**任务ID**: M5-016
**标题**: 实现插件系统
**类型**: fullstack (全栈开发)
**预估工时**: 16h
**依赖**: M0, M5-015 完成

---

## 任务描述

实现一个可扩展的插件系统，允许第三方开发者创建和分发插件来扩展平台功能。插件可以添加新的命令、集成外部服务、自定义游戏机制等。系统需要提供插件 SDK、生命周期管理、权限沙箱、API 暴露等能力。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M5-016-01 | 设计插件架构与沙箱 | 插件接口、权限隔离 | 2h |
| M5-016-02 | 实现插件加载器 | 动态加载、依赖解析 | 3h |
| M5-016-03 | 实现插件生命周期管理 | 安装、启用、禁用、卸载 | 2h |
| M5-016-04 | 实现插件 API 网关 | 安全的 API 暴露 | 2h |
| M5-016-05 | 实现插件 SDK | Python SDK 文档和示例 | 2h |
| M5-016-06 | 实现前端插件管理界面 | 插件市场、管理、配置 | 3h |
| M5-016-07 | 编写插件开发文档 | SDK 使用指南 | 2h |

---

## 完整后端代码示例 (Python + Agno)

### 插件数据模型

```python
# backend/app/models/plugins.py
from datetime import datetime
from typing import List, Optional, Dict, Any
from enum import Enum
from sqlalchemy import Column, String, JSON, DateTime, Boolean, Integer, ForeignKey, Text
from sqlalchemy.dialects.postgresql import UUID
import uuid
import json

from app.db.base_class import Base


class PluginPermission(str, Enum):
    """插件权限"""
    READ_CHARACTER = "read_character"  # 读取角色数据
    MODIFY_CHARACTER = "modify_character"  # 修改角色数据
    READ_SESSION = "read_session"  # 读取会话数据
    MODIFY_SESSION = "modify_session"  # 修改会话数据
    SEND_MESSAGE = "send_message"  # 发送消息
    EXECUTE_COMMAND = "execute_command"  # 执行命令
    WEBHOOK = "webhook"  # Webhook 调用
    STORAGE = "storage"  # 文件存储访问
    NETWORK = "network"  # 网络请求


class PluginState(str, Enum):
    """插件状态"""
    INSTALLED = "installed"
    ENABLED = "enabled"
    DISABLED = "disabled"
    ERROR = "error"


class Plugin(Base):
    """插件表"""
    __tablename__ = "plugins"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)

    # 插件标识
    plugin_id = Column(String(100), nullable=False, unique=True)  # 例如: author.plugin-name
    version = Column(String(20), nullable=False)

    # 插件信息
    name = Column(String(100), nullable=False)
    description = Column(Text, nullable=True)
    author = Column(String(100), nullable=False)
    homepage = Column(String(500), nullable=True)
    repository = Column(String(500), nullable=True)

    # 安装信息
    install_path = Column(String(500), nullable=False)

    # 依赖
    dependencies = Column(JSON, default=list)  # [{"plugin_id": "...", "version": "..."}]

    # 所需权限
    permissions = Column(JSON, default=list)  # List[PluginPermission]

    # 配置架构（JSON Schema）
    config_schema = Column(JSON, nullable=True)

    # 当前配置
    config = Column(JSON, default=dict)

    # 状态
    state = Column(String(20), default=PluginState.INSTALLED)

    # 是否为系统插件
    is_system = Column(Boolean, default=False)

    # 安装信息
    installed_at = Column(DateTime, default=datetime.utcnow)
    installed_by = Column(UUID(as_uuid=True), ForeignKey("accounts.id"))

    # 启用/禁用信息
    enabled_at = Column(DateTime, nullable=True)
    disabled_at = Column(DateTime, nullable=True)

    # 错误信息
    error_message = Column(Text, nullable=True)


class PluginInstallationLog(Base):
    """插件安装日志"""
    __tablename__ = "plugin_installation_logs"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    plugin_id = Column(UUID(as_uuid=True), ForeignKey("plugins.id"), nullable=False)

    action = Column(String(20), nullable=False)  # install, enable, disable, uninstall
    status = Column(String(20), nullable=False)  # success, error

    message = Column(Text, nullable=True)
    created_at = Column(DateTime, default=datetime.utcnow)
```

### 插件基础类

```python
# backend/app/plugins/base.py
from abc import ABC, abstractmethod
from typing import Dict, Any, List, Optional
from pydantic import BaseModel


class PluginContext:
    """插件上下文 - 插件与系统交互的接口"""

    def __init__(
        self,
        plugin_id: str,
        config: Dict[str, Any],
        permissions: List[str],
        api_client: "PluginAPIClient"
    ):
        self.plugin_id = plugin_id
        self.config = config
        self.permissions = permissions
        self.api = api_client


class PluginAPIClient:
    """插件 API 客户端 - 提供安全的 API 访问"""

    def __init__(
        self,
        plugin_id: str,
        session_id: Optional[str] = None,
        campaign_id: Optional[str] = None
    ):
        self.plugin_id = plugin_id
        self.session_id = session_id
        self.campaign_id = campaign_id

    async def get_character(self, character_id: str) -> Dict[str, Any]:
        """获取角色信息"""
        # 权限检查: READ_CHARACTER
        from app.services.plugin_service import PluginService
        return await PluginService.api_get_character(
            self.plugin_id,
            character_id,
            self.session_id
        )

    async def update_character(self, character_id: str, data: Dict[str, Any]) -> Dict[str, Any]:
        """更新角色信息"""
        # 权限检查: MODIFY_CHARACTER
        from app.services.plugin_service import PluginService
        return await PluginService.api_update_character(
            self.plugin_id,
            character_id,
            data
        )

    async def send_message(self, content: str, visibility: str = "public") -> Dict[str, Any]:
        """发送消息到游戏台"""
        # 权限检查: SEND_MESSAGE
        from app.services.plugin_service import PluginService
        return await PluginService.api_send_message(
            self.plugin_id,
            self.session_id,
            content,
            visibility
        )

    async def execute_command(self, command: str) -> Dict[str, Any]:
        """执行游戏命令"""
        # 权限检查: EXECUTE_COMMAND
        from app.services.plugin_service import PluginService
        return await PluginService.api_execute_command(
            self.plugin_id,
            command,
            self.session_id
        )

    async def storage_get(self, key: str) -> Optional[Any]:
        """获取插件存储数据"""
        # 权限检查: STORAGE
        from app.services.plugin_service import PluginService
        return await PluginService.api_storage_get(self.plugin_id, key)

    async def storage_set(self, key: str, value: Any):
        """设置插件存储数据"""
        from app.services.plugin_service import PluginService
        return await PluginService.api_storage_set(self.plugin_id, key, value)

    async def http_request(self, url: str, method: str = "GET", **kwargs) -> Any:
        """发送 HTTP 请求"""
        # 权限检查: NETWORK
        import httpx
        async with httpx.AsyncClient() as client:
            response = await client.request(method, url, **kwargs)
            return response.json()


class BasePlugin(ABC):
    """插件基类 - 所有插件必须继承此类"""

    def __init__(self, context: PluginContext):
        self.context = context

    @property
    def plugin_id(self) -> str:
        return self.context.plugin_id

    @property
    def config(self) -> Dict[str, Any]:
        return self.context.config

    @abstractmethod
    async def on_enable(self):
        """插件启用时调用"""
        pass

    @abstractmethod
    async def on_disable(self):
        """插件禁用时调用"""
        pass

    async def on_config_change(self, old_config: Dict[str, Any], new_config: Dict[str, Any]):
        """配置变更时调用"""
        pass

    async def on_command(self, command: str, args: List[str]) -> Optional[str]:
        """
        处理插件命令

        Args:
            command: 命令名称
            args: 命令参数

        Returns:
            响应消息，如果返回 None 则不响应
        """
        return None

    async def on_webhook(self, event: str, data: Dict[str, Any]):
        """
        处理 Webhook 事件

        Args:
            event: 事件名称
            data: 事件数据
        """
        pass


class PluginManifest(BaseModel):
    """插件清单"""

    plugin_id: str
    version: str
    name: str
    description: Optional[str] = None
    author: str
    homepage: Optional[str] = None
    repository: Optional[str] = None

    dependencies: List[Dict[str, str]] = []
    permissions: List[str] = []

    config_schema: Optional[Dict] = None

    entry_point: str  # 插件入口文件


def load_plugin_manifest(plugin_path: str) -> PluginManifest:
    """加载插件清单"""
    manifest_path = f"{plugin_path}/plugin.json"

    with open(manifest_path, "r") as f:
        data = json.load(f)

    return PluginManifest(**data)
```

### 插件加载器

```python
# backend/app/services/plugin_loader.py
import importlib.util
import sys
from pathlib import Path
from typing import Dict, Optional, Any
import logging

from app.models.plugins import Plugin, PluginState
from app.plugins.base import BasePlugin, PluginContext, PluginAPIClient, load_plugin_manifest


logger = logging.getLogger(__name__)


class PluginLoader:
    """插件加载器"""

    _loaded_plugins: Dict[str, BasePlugin] = {}

    @staticmethod
    async def load_plugin(plugin: Plugin, session_id: Optional[str] = None) -> BasePlugin:
        """
        加载插件

        Args:
            plugin: 插件数据库记录
            session_id: 会话 ID（可选，用于上下文）

        Returns:
            插件实例
        """
        # 检查是否已加载
        if plugin.plugin_id in PluginLoader._loaded_plugins:
            return PluginLoader._loaded_plugins[plugin.plugin_id]

        try:
            # 加载插件清单
            manifest = load_plugin_manifest(plugin.install_path)

            # 创建插件上下文
            api_client = PluginAPIClient(
                plugin_id=plugin.plugin_id,
                session_id=session_id,
                campaign_id=plugin.campaign_id
            )

            context = PluginContext(
                plugin_id=plugin.plugin_id,
                config=plugin.config,
                permissions=plugin.permissions,
                api_client=api_client
            )

            # 动态加载插件模块
            module_path = Path(plugin.install_path) / manifest.entry_point
            spec = importlib.util.spec_from_file_location(
                plugin.plugin_id,
                module_path
            )
            module = importlib.util.module_from_spec(spec)
            sys.modules[plugin.plugin_id] = module
            spec.loader.exec_module(module)

            # 获取插件类
            plugin_class = getattr(module, "Plugin")

            # 实例化插件
            plugin_instance: BasePlugin = plugin_class(context)

            # 缓存插件实例
            PluginLoader._loaded_plugins[plugin.plugin_id] = plugin_instance

            logger.info(f"Loaded plugin: {plugin.plugin_id}")

            return plugin_instance

        except Exception as e:
            logger.error(f"Failed to load plugin {plugin.plugin_id}: {e}")
            raise

    @staticmethod
    async def unload_plugin(plugin_id: str):
        """卸载插件"""
        if plugin_id in PluginLoader._loaded_plugins:
            plugin_instance = PluginLoader._loaded_plugins[plugin_id]
            await plugin_instance.on_disable()
            del PluginLoader._loaded_plugins[plugin_id]

            if plugin_id in sys.modules:
                del sys.modules[plugin_id]

            logger.info(f"Unloaded plugin: {plugin_id}")

    @staticmethod
    def get_loaded_plugin(plugin_id: str) -> Optional[BasePlugin]:
        """获取已加载的插件实例"""
        return PluginLoader._loaded_plugins.get(plugin_id)
```

### 插件服务

```python
# backend/app/services/plugin_service.py
from typing import Dict, Any, List, Optional
from sqlalchemy.orm import Session
from pathlib import Path
import shutil
import json
import logging

from app.models.plugins import Plugin, PluginState, PluginPermission
from app.services.plugin_loader import PluginLoader


logger = logging.getLogger(__name__)


class PluginService:
    """插件服务"""

    PLUGIN_DIR = "plugins"

    @staticmethod
    async def install_plugin(
        db: Session,
        plugin_file_path: str,
        installed_by: str
    ) -> Plugin:
        """
        安装插件

        Args:
            db: 数据库会话
            plugin_file_path: 插件压缩包路径
            installed_by: 安装者 ID

        Returns:
            插件记录
        """
        import zipfile

        # 解压插件
        plugin_dir = Path(PluginService.PLUGIN_DIR)
        plugin_dir.mkdir(parents=True, exist_ok=True)

        with zipfile.ZipFile(plugin_file_path, "r") as zip_ref:
            # 读取清单
            manifest_data = json.loads(zip_ref.read("plugin.json"))
            plugin_id = manifest_data["plugin_id"]
            version = manifest_data["version"]

            # 检查是否已安装
            existing = db.query(Plugin).filter(
                Plugin.plugin_id == plugin_id
            ).first()

            if existing:
                raise ValueError(f"Plugin {plugin_id} already installed")

            # 解压到插件目录
            install_path = plugin_dir / plugin_id
            zip_ref.extractall(install_path)

        # 创建数据库记录
        plugin = Plugin(
            plugin_id=plugin_id,
            version=version,
            name=manifest_data["name"],
            description=manifest_data.get("description"),
            author=manifest_data["author"],
            homepage=manifest_data.get("homepage"),
            repository=manifest_data.get("repository"),
            install_path=str(install_path),
            dependencies=manifest_data.get("dependencies", []),
            permissions=manifest_data.get("permissions", []),
            config_schema=manifest_data.get("config_schema"),
            installed_by=installed_by
        )

        db.add(plugin)
        db.commit()
        db.refresh(plugin)

        logger.info(f"Installed plugin: {plugin_id}")

        return plugin

    @staticmethod
    async def enable_plugin(db: Session, plugin_id: str) -> Plugin:
        """启用插件"""
        plugin = db.query(Plugin).filter(Plugin.plugin_id == plugin_id).first()

        if not plugin:
            raise ValueError(f"Plugin {plugin_id} not found")

        # 加载插件
        plugin_instance = await PluginLoader.load_plugin(plugin)

        # 调用启用回调
        await plugin_instance.on_enable()

        # 更新状态
        plugin.state = PluginState.ENABLED
        plugin.enabled_at = datetime.utcnow()
        plugin.error_message = None

        db.commit()
        db.refresh(plugin)

        return plugin

    @staticmethod
    async def disable_plugin(db: Session, plugin_id: str) -> Plugin:
        """禁用插件"""
        plugin = db.query(Plugin).filter(Plugin.plugin_id == plugin_id).first()

        if not plugin:
            raise ValueError(f"Plugin {plugin_id} not found")

        # 卸载插件
        await PluginLoader.unload_plugin(plugin_id)

        # 更新状态
        plugin.state = PluginState.DISABLED
        plugin.disabled_at = datetime.utcnow()

        db.commit()
        db.refresh(plugin)

        return plugin

    @staticmethod
    async def uninstall_plugin(db: Session, plugin_id: str):
        """卸载插件"""
        plugin = db.query(Plugin).filter(Plugin.plugin_id == plugin_id).first()

        if not plugin:
            raise ValueError(f"Plugin {plugin_id} not found")

        # 先禁用
        if plugin.state == PluginState.ENABLED:
            await PluginService.disable_plugin(db, plugin_id)

        # 删除文件
        install_path = Path(plugin.install_path)
        if install_path.exists():
            shutil.rmtree(install_path)

        # 删除数据库记录
        db.delete(plugin)
        db.commit()

        logger.info(f"Uninstalled plugin: {plugin_id}")

    @staticmethod
    async def update_plugin_config(
        db: Session,
        plugin_id: str,
        new_config: Dict[str, Any]
    ) -> Plugin:
        """更新插件配置"""
        plugin = db.query(Plugin).filter(Plugin.plugin_id == plugin_id).first()

        if not plugin:
            raise ValueError(f"Plugin {plugin_id} not found")

        old_config = plugin.config

        # 更新配置
        plugin.config = new_config
        db.commit()
        db.refresh(plugin)

        # 如果插件已加载，调用配置变更回调
        if plugin.state == PluginState.ENABLED:
            plugin_instance = PluginLoader.get_loaded_plugin(plugin_id)
            if plugin_instance:
                await plugin_instance.on_config_change(old_config, new_config)

        return plugin

    # API 网关方法
    @staticmethod
    async def api_get_character(
        plugin_id: str,
        character_id: str,
        session_id: Optional[str] = None
    ) -> Dict[str, Any]:
        """API: 获取角色信息"""
        # 权限检查
        plugin = _get_plugin_and_check_permission(plugin_id, PluginPermission.READ_CHARACTER)

        from app.services.character_service import CharacterService
        # TODO: 获取角色信息
        return {}

    @staticmethod
    async def api_send_message(
        plugin_id: str,
        session_id: str,
        content: str,
        visibility: str = "public"
    ) -> Dict[str, Any]:
        """API: 发送消息"""
        # 权限检查
        _get_plugin_and_check_permission(plugin_id, PluginPermission.SEND_MESSAGE)

        # TODO: 发送消息到游戏台
        return {"status": "sent"}


def _get_plugin_and_check_permission(plugin_id: str, permission: PluginPermission) -> Plugin:
    """获取插件并检查权限"""
    from app.database import SessionLocal

    db = SessionLocal()
    plugin = db.query(Plugin).filter(Plugin.plugin_id == plugin_id).first()

    if not plugin:
        raise PermissionError(f"Plugin {plugin_id} not found")

    if permission.value not in plugin.permissions:
        raise PermissionError(f"Plugin {plugin_id} missing permission: {permission}")

    return plugin
```

### 示例插件

```python
# plugins/example-hello/plugin.py
from app.plugins.base import BasePlugin


class Plugin(BasePlugin):
    """示例插件: Hello World"""

    async def on_enable(self):
        print(f"Hello plugin enabled! Config: {self.config}")

    async def on_disable(self):
        print("Hello plugin disabled!")

    async def on_command(self, command: str, args: list) -> str:
        if command == "hello":
            name = args[0] if args else "World"
            return f"Hello, {name}!"
        return None
```

```json
// plugins/example-hello/plugin.json
{
  "plugin_id": "example.hello",
  "version": "1.0.0",
  "name": "Hello World",
  "description": "A simple hello world plugin",
  "author": "Your Name",
  "homepage": "https://example.com",
  "repository": "https://github.com/example/hello-plugin",
  "permissions": ["send_message"],
  "entry_point": "plugin.py"
}
```

---

## 完整前端代码示例 (TypeScript + React + shadcn/ui)

### 插件管理界面

```tsx
// frontend/src/components/plugins/PluginManagement.tsx
import React, { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Upload, Power, PowerOff, Trash2, Settings, Download } from "lucide-react";

import { Plugin } from "@/types/plugins";

export function PluginManagement() {
  const [plugins, setPlugins] = useState<Plugin[]>([]);
  const [showUploadDialog, setShowUploadDialog] = useState(false);
  const [selectedPlugin, setSelectedPlugin] = useState<Plugin | null>(null);

  useEffect(() => {
    loadPlugins();
  }, []);

  const loadPlugins = async () => {
    const res = await fetch("/api/plugins");
    const data = await res.json();
    setPlugins(data);
  };

  const handleTogglePlugin = async (pluginId: string, currentState: string) => {
    const action = currentState === "enabled" ? "disable" : "enable";
    await fetch(`/api/plugins/${pluginId}/${action}`, { method: "POST" });
    loadPlugins();
  };

  const handleUninstall = async (pluginId: string) => {
    if (!confirm("确定要卸载此插件吗？这将删除所有相关数据。")) return;

    await fetch(`/api/plugins/${pluginId}`, { method: "DELETE" });
    loadPlugins();
  };

  const handleUploadPlugin = async (file: File) => {
    const formData = new FormData();
    formData.append("plugin", file);

    await fetch("/api/plugins/install", {
      method: "POST",
      body: formData
    });

    setShowUploadDialog(false);
    loadPlugins();
  };

  return (
    <div className="space-y-6">
      {/* 头部 */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">插件管理</h2>
          <p className="text-muted-foreground">管理已安装的插件</p>
        </div>
        <Dialog open={showUploadDialog} onOpenChange={setShowUploadDialog}>
          <DialogTrigger asChild>
            <Button>
              <Upload className="w-4 h-4 mr-2" />
              安装插件
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>上传插件</DialogTitle>
            </DialogHeader>
            <PluginUpload onUpload={handleUploadPlugin} onCancel={() => setShowUploadDialog(false)} />
          </DialogContent>
        </Dialog>
      </div>

      {/* 插件列表 */}
      <div className="grid gap-4">
        {plugins.map((plugin) => (
          <Card key={plugin.id}>
            <CardContent className="p-4">
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <h3 className="font-semibold">{plugin.name}</h3>
                    <Badge variant="outline">{plugin.version}</Badge>
                    <Badge
                      variant={plugin.state === "enabled" ? "default" : "secondary"}
                    >
                      {plugin.state}
                    </Badge>
                  </div>

                  <p className="text-sm text-muted-foreground mt-1">
                    {plugin.description}
                  </p>

                  <div className="flex items-center gap-4 mt-2 text-sm text-muted-foreground">
                    <span>作者: {plugin.author}</span>
                    <span>ID: {plugin.plugin_id}</span>
                  </div>

                  {/* 权限标签 */}
                  <div className="flex flex-wrap gap-2 mt-3">
                    {plugin.permissions.map((permission) => (
                      <Badge key={permission} variant="outline" className="text-xs">
                        {permission}
                      </Badge>
                    ))}
                  </div>
                </div>

                {/* 操作按钮 */}
                <div className="flex gap-2">
                  {plugin.state === "enabled" ? (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleTogglePlugin(plugin.plugin_id, plugin.state)}
                    >
                      <PowerOff className="w-4 h-4" />
                    </Button>
                  ) : (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleTogglePlugin(plugin.plugin_id, plugin.state)}
                    >
                      <Power className="w-4 h-4" />
                    </Button>
                  )}
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setSelectedPlugin(plugin)}
                  >
                    <Settings className="w-4 h-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleUninstall(plugin.plugin_id)}
                  >
                    <Trash2 className="w-4 h-4" />
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* 插件配置对话框 */}
      {selectedPlugin && (
        <Dialog open={!!selectedPlugin} onOpenChange={() => setSelectedPlugin(null)}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>配置插件: {selectedPlugin.name}</DialogTitle>
            </DialogHeader>
            <PluginConfigEditor plugin={selectedPlugin} onSave={loadPlugins} />
          </DialogContent>
        </Dialog>
      )}
    </div>
  );
}

// 插件上传组件
function PluginUpload({
  onUpload,
  onCancel
}: {
  onUpload: (file: File) => void;
  onCancel: () => void;
}) {
  const [file, setFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!file) return;

    setUploading(true);
    try {
      await onUpload(file);
    } finally {
      setUploading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="border-2 border-dashed rounded-lg p-8 text-center">
        <input
          type="file"
          accept=".zip"
          onChange={(e) => setFile(e.target.files?.[0] || null)}
          className="hidden"
          id="plugin-upload"
        />
        <label htmlFor="plugin-upload" className="cursor-pointer">
          <div className="space-y-2">
            <Upload className="w-12 h-12 mx-auto text-muted-foreground" />
            <p className="text-sm text-muted-foreground">
              {file ? file.name : "点击选择插件文件 (.zip)"}
            </p>
          </div>
        </label>
      </div>

      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onCancel}>
          取消
        </Button>
        <Button type="submit" disabled={!file || uploading}>
          {uploading ? "上传中..." : "安装"}
        </Button>
      </div>
    </form>
  );
}

// 插件配置编辑器
function PluginConfigEditor({
  plugin,
  onSave
}: {
  plugin: Plugin;
  onSave: () => void;
}) {
  const [config, setConfig] = useState(plugin.config || {});
  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    setSaving(true);
    try {
      await fetch(`/api/plugins/${plugin.plugin_id}/config`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(config)
      });
      onSave();
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <pre className="bg-muted p-4 rounded text-sm">
          {JSON.stringify(config, null, 2)}
        </pre>
        <textarea
          value={JSON.stringify(config, null, 2)}
          onChange={(e) => {
            try {
              setConfig(JSON.parse(e.target.value));
            } catch {
              // 忽略无效 JSON
            }
          }}
          rows={10}
          className="w-full font-mono text-sm"
        />
      </div>

      <div className="flex justify-end">
        <Button onClick={handleSave} disabled={saving}>
          {saving ? "保存中..." : "保存配置"}
        </Button>
      </div>
    </div>
  );
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `backend/app/models/plugins.py` | 创建 | 插件数据模型 |
| `backend/app/plugins/base.py` | 创建 | 插件基类和接口 |
| `backend/app/services/plugin_loader.py` | 创建 | 插件加载器 |
| `backend/app/services/plugin_service.py` | 创建 | 插件服务 |
| `backend/app/api/plugins.py` | 创建 | 插件 API 路由 |
| `frontend/src/types/plugins.ts` | 创建 | 类型定义 |
| `frontend/src/components/plugins/PluginManagement.tsx` | 创建 | 插件管理界面 |
| `docs/plugin-sdk.md` | 创建 | 插件 SDK 文档 |

---

## 验收标准

- [ ] 插件可以成功安装
- [ ] 插件可以启用和禁用
- [ ] 插件权限正确隔离
- [ ] 插件可以访问授权的 API
- [ ] 插件可以存储配置
- [ ] 插件错误不影响系统稳定性
- [ ] 插件可以完全卸载

---

## 参考文档

- VS Code 插件系统设计
- Discord Bot 开发指南
- Python 插件系统最佳实践

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
