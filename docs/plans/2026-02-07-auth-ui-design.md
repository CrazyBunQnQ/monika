# 认证 UI 设计文档

**日期**: 2026-02-07
**任务**: M1-014, M1-015, M1-016, M1-028, M1-030
**状态**: 已确认

---

## 概述

本文档描述 Monika 项目 M1 里程碑中待实现的认证 UI 设计，包括登录/注册页面、认证上下文、角色选择界面等组件。

---

## 1. 整体架构与组件结构

### 核心组件

- **AuthPage** - 统一的登录/注册页面
- **CharacterSelectScreen** - 角色选择界面
- **AuthContext** - 全局认证状态管理
- **API 封装** - 统一的 API 请求处理

### 技术栈

- React 19 + TypeScript
- React Router v7
- shadcn/ui 组件库
- Axios（API 请求）
- localStorage（持久化）

---

## 2. AuthPage 设计

### 功能

- 登录和注册模式切换
- 表单验证（提交时触发）
- "记住我"功能（localStorage）
- 密码重置链接
- 加载状态和错误提示

### 表单字段

**登录模式**：
- username/email（组合输入）
- password
- rememberMe（复选框）

**注册模式**：
- username
- email
- password
- confirmPassword

### 密码验证规则

- 最少 8 位字符
- 必须包含字母
- 必须包含数字
- 必须包含特殊字符

### UI 布局

使用 Card 容器，包含：
- 头部：标题和 Logo
- 错误 Banner（Alert 组件）
- 表单区域
- 底部链接（模式切换、忘记密码）

---

## 3. AuthContext 设计

### Context 接口

```typescript
interface AuthContextType {
  user: User | null
  token: string | null
  isAuthenticated: boolean
  isLoading: boolean
  login: (username: string, password: string, rememberMe?: boolean) => Promise<void>
  register: (username: string, email: string, password: string) => Promise<void>
  logout: () => Promise<void>
  refreshToken: () => Promise<void>
}
```

### Token 存储

- localStorage 键名：`monika_token`、`monika_user`
- 应用启动时自动恢复登录状态
- 401 错误时自动清除并跳转登录页

### API 拦截器

- 请求拦截器：自动添加 Authorization 头
- 响应拦截器：捕获 401 错误并处理

---

## 4. CharacterSelectScreen 设计

### 功能

- 显示用户角色列表
- 游玩、编辑、删除操作
- 空状态时显示快速创建表单

### 表格列

- 角色名
- 原型
- HP（当前/最大）
- SAN
- 操作按钮（游玩、编辑、删除）

### 操作行为

- **游玩**：设置当前角色，导航到 `/game`
- **编辑**：导航到角色编辑页
- **删除**：显示确认对话框后删除

### 空状态

列表为空时显示：
- 友好的空状态提示
- 完整的 CharacterForm（内联）

---

## 5. 数据流与状态管理

### 认证流程

```
用户输入 → 表单提交 → AuthContext → API 调用
→ 存储 token → 更新状态 → 导航到角色选择
```

### 错误处理策略

| 错误类型 | 处理方式 |
|---------|---------|
| 网络错误 | Toast "网络连接失败" |
| 400 验证 | 表单顶部 Banner 显示错误 |
| 401 未授权 | 自动登出，跳转登录 |
| 409 冲突 | Banner 显示具体冲突 |
| 500 服务器 | Toast "服务器错误" |

---

## 6. 加载状态与用户反馈

### 骨架屏

- 表单加载时显示骨架屏
- 角色列表加载时显示表格骨架

### Toast 通知

- 成功：3 秒自动消失
- 错误：5 秒自动消失

### 按钮加载状态

- 提交时显示加载图标
- 按钮禁用防止重复提交

---

## 7. 文件结构

```
frontend/src/
├── contexts/
│   └── AuthContext.tsx
├── lib/
│   └── api.ts
├── pages/
│   ├── AuthPage.tsx
│   ├── CharacterSelectPage.tsx
│   └── LandingPage.tsx
├── components/
│   ├── CharacterSelectScreen.tsx
│   └── ui/
│       ├── alert.tsx
│       ├── dialog.tsx
│       ├── table.tsx
│       └── skeleton.tsx
└── App.tsx
```

### 路由配置

- `/` - LandingPage
- `/auth` - AuthPage
- `/select-character` - CharacterSelectPage（需认证）
- `/game` - GameConsole（需认证）

---

## 8. 需要添加的 shadcn/ui 组件

```bash
npx shadcn@latest add alert
npx shadcn@latest add dialog
npx shadcn@latest add table
npx shadcn@latest add skeleton
```

---

## 9. 任务映射

| 任务 ID | 组件 | 状态 |
|---------|------|------|
| M1-014 | AuthPage（注册模式） | 待实现 |
| M1-015 | AuthPage（登录模式） | 待实现 |
| M1-016 | AuthContext | 待实现 |
| M1-028 | CharacterSelectScreen | 待实现 |
| M1-030 | CharacterPreview（表格行） | 待实现 |
