# M1-017: 实现前端路由与导航

**任务ID**: M1-017
**标题**: 实现前端路由与导航
**类型**: frontend (前端开发)
**预估工时**: 2h
**依赖**: M1-016

---

## 任务描述

使用 React Router 实现前端路由系统，包括公共路由、受保护路由、嵌套路由等。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-017-01 | 设计路由结构 | 路由层次 | 20min |
| M1-017-02 | 实现公共路由 | 登录/注册 | 15min |
| M1-017-03 | 实现受保护路由 | 认证检查 | 25min |
| M1-017-04 | 实现嵌套路由 | 游戏内页面 | 20min |
| M1-017-05 | 实现路由守卫 | 权限检查 | 25min |
| M1-017-06 | 实现面包屑导航 | 导航辅助 | 20min |
| M1-017-07 | 编写路由测试 | 测试覆盖 | 20min |

---

## 路由配置

```tsx
// frontend/src/router/index.tsx
import { createBrowserRouter, Navigate } from 'react-router-dom'
import { useAuth } from '@/contexts/AuthContext'
import LoginPage from '@/pages/LoginPage'
import RegisterPage from '@/pages/RegisterPage'
import GameConsolePage from '@/pages/GameConsolePage'
import CampaignSelectPage from '@/pages/CampaignSelectPage'
import CharacterSheetPage from '@/pages/CharacterSheetPage'

// 受保护路由包装器
function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth()

  if (isLoading) {
    return <div>Loading...</div>
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}

// KP 路由包装器
function KPRoute({ children }: { children: React.ReactNode }) {
  const { user, isAuthenticated, isLoading } = useAuth()

  if (isLoading) {
    return <div>Loading...</div>
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  if (user?.role !== 'kp') {
    return <Navigate to="/game" replace />
  }

  return <>{children}</>
}

// 路由配置
export const router = createBrowserRouter([
  {
    path: '/',
    element: <Navigate to="/game" replace />,
  },
  {
    path: '/login',
    element: <LoginPage />,
  },
  {
    path: '/register',
    element: <RegisterPage />,
  },
  {
    path: '/game',
    element: (
      <ProtectedRoute>
        <GameConsolePage />
      </ProtectedRoute>
    ),
    children: [
      {
        index: true,
        element: <CampaignSelectPage />,
      },
      {
        path: ':campaignId',
        element: <GamePage />,
        children: [
          {
            index: true,
            element: <ScenePanel />,
          },
          {
            path: 'character',
            element: <CharacterSheetPage />,
          },
          {
            path: 'chat',
            element: <ChatPanel />,
          },
        ],
      },
    ],
  },
  {
    path: '/kp',
    element: (
      <KPRoute>
        <KPDashboard />
      </KPRoute>
    ),
    children: [
      {
        index: true,
        element: <CampaignList />,
      },
      {
        path: 'campaign/:campaignId',
        element: <CampaignDetail />,
      },
    ],
  },
  {
    path: '*',
    element: <Navigate to="/game" replace />,
  },
])
```

---

## 面包屑导航

```tsx
// frontend/src/components/navigation/Breadcrumb.tsx
import { useLocation, Link } from 'react-router-dom'

interface BreadcrumbItem {
  label: string
  path?: string
}

export function Breadcrumb() {
  const location = useLocation()

  const items: BreadcrumbItem[] = []

  // 根据路径生成面包屑
  const pathSegments = location.pathname.split('/').filter(Boolean)

  if (pathSegments[0] === 'game') {
    items.push({ label: '游戏', path: '/game' })

    if (pathSegments[1]) {
      items.push({ label: `战役 ${pathSegments[1]}` })

      if (pathSegments[2]) {
        const pageMap: Record<string, string> = {
          character: '角色卡',
          chat: '聊天',
          dice: '掷骰',
        }
        items.push({ label: pageMap[pathSegments[2]] || pathSegments[2] })
      }
    }
  } else if (pathSegments[0] === 'kp') {
    items.push({ label: 'KP 控制台', path: '/kp' })
    // ... KP 面包屑逻辑
  }

  if (items.length === 0) {
    return null
  }

  return (
    <nav className="flex items-center space-x-2 text-sm">
      {items.map((item, index) => (
        <div key={index} className="flex items-center">
          {index > 0 && <span className="mx-2 text-muted-foreground">/</span>}
          {item.path ? (
            <Link
              to={item.path}
              className="text-muted-foreground hover:text-foreground"
            >
              {item.label}
            </Link>
          ) : (
            <span className="text-foreground">{item.label}</span>
          )}
        </div>
      ))}
    </nav>
  )
}
```

---

## 路由守卫 Hook

```tsx
// frontend/src/hooks/useRequireAuth.ts
import { useEffect } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useAuth } from '@/contexts/AuthContext'

export function useRequireAuth(requireRole?: 'kp' | 'player') {
  const { isAuthenticated, isLoading, user } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()

  useEffect(() => {
    if (!isLoading) {
      if (!isAuthenticated) {
        // 保存目标位置用于登录后跳转
        navigate('/login', {
          state: { redirect: location.pathname },
        })
      } else if (requireRole && user?.role !== requireRole) {
        // 权限不足
        navigate('/game')
      }
    }
  }, [isAuthenticated, isLoading, user, requireRole, navigate, location.pathname])

  return { isAuthenticated, isLoading, user }
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `frontend/src/router/index.tsx` | 创建 | 路由配置 |
| `frontend/src/components/navigation/Breadcrumb.tsx` | 创建 | 面包屑组件 |
| `frontend/src/hooks/useRequireAuth.ts` | 创建 | 路由守卫 Hook |
| `frontend/src/App.tsx` | 更新 | 集成路由 |

---

## 验收标准

- [ ] 公共路由可访问
- [ ] 受保护路由需要登录
- [ ] KP 路由仅 KP 可访问
- [ ] 嵌套路由正确渲染
- [ ] 面包屑导航准确
- [ ] 测试覆盖主要场景

---

## 参考文档

- React Router v6 文档
- M1-016: AuthContext

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
