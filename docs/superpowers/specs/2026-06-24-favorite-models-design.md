# 偏好模型分组 — 设计文档

> 日期：2026-06-24 | 状态：待实现

## 概述

在 ModelPicker 下拉列表新增「偏好模型」分组，允许用户收藏常用模型。收藏列表存储在浏览器 localStorage 中，纯前端功能，无需后端改动。

## 功能

1. **收藏操作**：ModelPicker 下拉列表悬停模型行时，右侧显示星标按钮。点击星标切换收藏状态（☆ / ★）。
2. **偏好分组**：列表顶部以「★ 偏好模型」标题展示所有已收藏模型，跨 provider 聚合。
3. **持久化**：收藏列表存入 localStorage，重启应用后自动恢复。

## 数据层

### 存储格式

- **localStorage key**: `monika:favorite_models`
- **值**: JSON 字符串数组，每项格式 `"providerId:modelId"`，例 `["openai:gpt-4o-mini", "deepseek:deepseek-v3"]`
- providerId 和 modelId 均不含冒号，不存在解析歧义。

### Zustand store 变更 (`frontend/src/store/index.ts`)

新增状态：

```typescript
favoriteModels: string[]  // 初始值: []
```

新增方法：

- `toggleFavoriteModel(providerId: string, modelId: string): void` — 切换收藏状态，同步写入 localStorage
- `isFavoriteModel(providerId: string, modelId: string): boolean` — 查询是否已收藏（派生方法）

初始化流程：

1. store 初始化时调用 `loadFromLocalStorage()`
2. 读取 `localStorage.getItem('monika:favorite_models')`
3. 解析失败或不存在 → 初始化为 `[]`，不报错

## UI 层

### 修改文件: `frontend/src/components/Chat/ModelPicker.tsx`

**FlatItem 类型扩展：**

```typescript
type FlatItem =
    | { type: 'favorite-header' }
    | { type: 'provider'; provider: ProviderInfo }
    | { type: 'model'; provider: ProviderInfo; model: ModelInfo; isFavorite?: boolean }
```

**flatItems 构建逻辑（`useMemo`）：**

1. 从 store 读取 `favoriteModels`，结合 `modelsByProvider` 解析出当前可用的收藏模型列表
2. 搜索过滤时按 `DisplayName` / `ID` 匹配
3. 有效收藏模型非空 → 先插入 `favorite-header`，再插入所有收藏模型项（每项携带 provider 信息和 `isFavorite: true`）
4. 然后按原逻辑遍历各 provider，构建 model 项时附带 `isFavorite` 标记
5. 收藏模型同时出现在偏好分组和原 provider 分组中（两处均显示星标）

**渲染分支：**

- `favorite-header`：金色字体 ± 偏好模型 标题（不参与键盘导航计数）
- `model` 行：右侧追加星标按钮

**星标按钮交互：**

- 默认不显示，CSS hover 时出现
- 已收藏（`isFavorite`）始终显示金色 ★
- 点击调用 `toggleFavoriteModel`，`e.stopPropagation()` 阻止冒泡到选中模型逻辑
- `isGenerating` 期间仍可操作收藏（不影响会话状态）

**偏好分组中模型项：**

- 每个模型旁显示所属 provider 名称（灰色小字）
- 仍可正常选中作为当前使用模型

### 修改文件: `frontend/src/store/index.ts`

初始化方法中添加 `favoriteModels` 加载。`toggleFavoriteModel` 读写 localStorage。

## 边界情况

| 场景 | 行为 |
|------|------|
| 收藏的模型被禁用/删除 | 从偏好分组中不显示，localStorage 保留记录（不清理孤儿数据） |
| 无收藏 | 不渲染偏好分组，不占空白行 |
| 搜索过滤后偏好分组为空 | 不显示偏好分组标题 |
| localStorage 为空或解析失败 | favoriteModels 初始化为 `[]` |
| 数据格式异常（缺少冒号） | 跳过该项，不报错 |
| 生成消息中（isGenerating） | 仍可操作收藏，不可选中模型 |

## 影响范围

- `frontend/src/components/Chat/ModelPicker.tsx` — 主要改动
- `frontend/src/store/index.ts` — 新增 favoriteModels 状态和方法
- 无后端变更（`internal/` 不变）
- 无新增文件

## 不做什么

- 不在 Settings 页面添加收藏管理 UI
- 不对 `config.json` 或后端数据结构做任何修改
- 不引入新依赖
