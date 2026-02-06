# M1-016: 实现 AuthContext 状态管理

**任务ID**: M1-016
**标题**: 实现 AuthContext 状态管理
**类型**: frontend (前端开发)
**预估工时**: 2h
**依赖**: M1-015

---

## 任务描述

实现 React Context 来管理全局认证状态，包括用户信息、Token、登录状态等。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-016-01 | 设计 AuthContext 接口 | Context API | 20min |
| M1-016-02 | 实现用户状态管理 | 用户信息 | 25min |
| M1-016-03 | 实现 Token 管理 | Token 操作 | 25min |
| M1-016-04 | 实现自动刷新 | Token 刷新 | 30min |
| M1-016-05 | 实现登录状态检查 | 认证检查 | 20min |
| M1-016-06 | 编写状态测试 | 测试状态管理 | 25min |
| M1-016-07 | 编写 Context 文档 | 使用说明 | 10min |

---

## AuthContext 实现

```tsx
// frontend/src/contexts/AuthContext.tsx
import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  ReactNode
} from 'react'
import { useNavigate, useLocation } from 'react-router-dom'

interface User {
  id: string
  username: string
  email: string
  role: 'kp' | 'player'
  created_at: string
}

interface AuthContextType {
  user: User | null
  token: string | null
  refreshToken: string | null
  isAuthenticated: boolean
  isLoading: boolean
  login: (usernameOrEmail: string, password: string, rememberMe?: boolean) => Promise<void>
  logout: () => void
  refreshToken: () => Promise<boolean>
}

const AuthContext = createContext<AuthContextType | null>(null)

const TOKEN_REFRESH_INTERVAL = 5 * 60 * 1000 // 5 分钟

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [token, setToken] = useState<string | null>(null)
  const [refreshToken, setRefreshToken] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const navigate = useNavigate()
  const location = useLocation()

  // 初始化：从本地存储加载
  useEffect(() => {
    const loadAuth = () => {
      const storedToken = localStorage.getItem('access_token')
      const storedRefreshToken = localStorage.getItem('refresh_token')
      const storedUser = localStorage.getItem('user')

      if (storedToken && storedUser) {
        try {
          setToken(storedToken)
          setRefreshToken(storedRefreshToken)
          setUser(JSON.parse(storedUser))
        } catch (error) {
          console.error('Failed to load auth:', error)
          // 清除无效数据
          localStorage.removeItem('access_token')
          localStorage.removeItem('refresh_token')
          localStorage.removeItem('user')
        }
      }

      setIsLoading(false)
    }

    loadAuth()
  }, [])

  // Token 自动刷新
  useEffect(() => {
    if (!token || !refreshToken) return

    const refreshInterval = setInterval(async () => {
      const success = await attemptRefreshToken()
      if (!success) {
        // 刷新失败，登出
        logout()
      }
    }, TOKEN_REFRESH_INTERVAL)

    return () => clearInterval(refreshInterval)
  }, [token, refreshToken])

  // 登出函数
  const logout = useCallback(() => {
    setUser(null)
    setToken(null)
    setRefreshToken(null)
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    localStorage.removeItem('user')
    localStorage.removeItem('remember_me')

    // 跳转到登录页
    navigate('/login', { state: { message: '已登出' } })
  }, [navigate])

  // Token 刷新
  const attemptRefreshToken = async (): Promise<boolean> => {
    if (!refreshToken) return false

    try {
      const response = await fetch('/api/auth/refresh', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          refresh_token: refreshToken,
        }),
      })

      if (!response.ok) {
        return false
      }

      const data = await response.json()

      // 更新 Token
      setToken(data.access_token)
      if (data.refresh_token) {
        setRefreshToken(data.refresh_token)
        localStorage.setItem('refresh_token', data.refresh_token)
      }
      localStorage.setItem('access_token', data.access_token)

      return true
    } catch (error) {
      console.error('Token refresh error:', error)
      return false
    }
  }

  const refreshTokenFn = useCallback(async (): Promise<boolean> => {
    return await attemptRefreshToken()
  }, [refreshToken, attemptRefreshToken])

  // 提供上下文值
  const value: AuthContextType = {
    user,
    token,
    refreshToken,
    isAuthenticated: !!token && !!user,
    isLoading,
    login: async () => {},  // 由登录页面实现
    logout,
    refreshToken: refreshTokenFn,
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider')
  }
  return context
}
```

---

## 验收标准

- [ ] Context API 正确实现
- [ ] 登录/登出功能正常
- [ ] Token 自动刷新工作
- [ ] 状态持久化正确
- [ ] 测试覆盖全面

---

## 参考文档

- M1-015: 登录页面
- React Context API 文档

---

**最后更新**: 2026-02-06
**状态**: [ 待开始
