# 认证 UI 实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现完整的用户认证 UI，包括登录/注册页面、AuthContext 状态管理、角色选择界面，集成后端 API 并处理完整的用户流程。

**Architecture:** 使用 React Context 管理全局认证状态，Axios 处理 API 请求，localStorage 持久化 token，React Router 管理路由。前端使用 shadcn/ui 组件库保持一致的视觉风格。

**Tech Stack:** React 19, TypeScript, React Router v7, Axios, shadcn/ui, sonner (toast)

---

## 前置准备

### Step 1: 安装必需的依赖

```bash
cd frontend
npm install react-router-dom axios sonner
npm install -D @types/react-router-dom
```

**Step 2: 添加缺失的 shadcn/ui 组件**

```bash
npx shadcn@latest add alert
npx shadcn@latest add table
npx shadcn@latest add skeleton
```

**Step 3: 提交初始设置**

```bash
git add frontend/package.json frontend/package-lock.json
git commit -m "chore: install auth dependencies (react-router, axios, sonner)"
```

---

## Task 1: 创建 API 封装层

**Files:**
- Create: `frontend/src/lib/api.ts`

### Step 1: 创建 API 客户端基础

```typescript
// frontend/src/lib/api.ts
import axios from 'axios'

const API_BASE_URL = 'http://localhost:8000'

export const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 请求拦截器 - 添加 token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('monika_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 响应拦截器 - 处理 401
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('monika_token')
      localStorage.removeItem('monika_user')
      window.location.href = '/auth'
    }
    return Promise.reject(error)
  }
)

export default api
```

### Step 2: 创建认证 API 函数

```typescript
// 在 frontend/src/lib/api.ts 中添加

export interface LoginRequest {
  username: string
  password: string
}

export interface RegisterRequest {
  username: string
  email: string
  password: string
}

export interface AuthResponse {
  access_token: string
  token_type: string
  expires_in: number
}

export interface User {
  id: number
  username: string
  email: string
  role: string
  is_active: boolean
}

export const authApi = {
  login: async (data: LoginRequest): Promise<AuthResponse> => {
    const formData = new FormData()
    formData.append('username', data.username)
    formData.append('password', data.password)

    const response = await api.post<AuthResponse>('/auth/login', formData, {
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
      },
    })
    return response.data
  },

  register: async (data: RegisterRequest): Promise<User> => {
    const response = await api.post<User>('/auth/register', data)
    return response.data
  },

  getCurrentUser: async (): Promise<User> => {
    const response = await api.get<User>('/auth/me')
    return response.data
  },
}
```

### Step 3: 创建角色 API 函数

```typescript
// 在 frontend/src/lib/api.ts 中添加

export interface Character {
  id: number
  name: string
  archetype: string
  age: number
  occupation: string
  hp: number
  hp_max: number
  san: number
  san_max: number
  luck: number
  luck_max: number
  mp: number
  mp_max: number
  strength: number
  dexterity: number
  intelligence: number
  constitution: number
  appearance: number
  power: number
  education: number
  size: number
  skills: Record<string, number>
}

export const characterApi = {
  list: async (): Promise<Character[]> => {
    const response = await api.get<Character[]>('/characters')
    return response.data
  },

  getById: async (id: number): Promise<Character> => {
    const response = await api.get<Character>(`/characters/${id}`)
    return response.data
  },

  create: async (data: Partial<Character>): Promise<Character> => {
    const response = await api.post<Character>('/characters', data)
    return response.data
  },

  update: async (id: number, data: Partial<Character>): Promise<Character> => {
    const response = await api.put<Character>(`/characters/${id}`, data)
    return response.data
  },

  delete: async (id: number): Promise<void> => {
    await api.delete(`/characters/${id}`)
  },
}

// 导出所有 API
export { api as axiosApi }
```

### Step 4: 提交 API 层

```bash
git add frontend/src/lib/api.ts
git commit -m "feat: add API client with auth and character endpoints"
```

---

## Task 2: 创建 AuthContext

**Files:**
- Create: `frontend/src/contexts/AuthContext.tsx`

### Step 1: 创建 Context 类型定义和 Provider

```typescript
// frontend/src/contexts/AuthContext.tsx
import { createContext, useContext, useState, useEffect, ReactNode } from 'react'
import { authApi, type User } from '@/lib/api'
import { toast } from 'sonner'

interface AuthContextType {
  user: User | null
  token: string | null
  isAuthenticated: boolean
  isLoading: boolean
  login: (username: string, password: string, rememberMe?: boolean) => Promise<void>
  register: (username: string, email: string, password: string) => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [token, setToken] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  // 从 localStorage 恢复登录状态
  useEffect(() => {
    const storedToken = localStorage.getItem('monika_token')
    const storedUser = localStorage.getItem('monika_user')

    if (storedToken && storedUser) {
      setToken(storedToken)
      setUser(JSON.parse(storedUser))
    }
    setIsLoading(false)
  }, [])

  const login = async (username: string, password: string, rememberMe = true) => {
    setIsLoading(true)
    try {
      const response = await authApi.login({ username, password })

      // 获取用户信息
      const userData = await authApi.getCurrentUser()

      setToken(response.access_token)
      setUser(userData)

      if (rememberMe) {
        localStorage.setItem('monika_token', response.access_token)
        localStorage.setItem('monika_user', JSON.stringify(userData))
      }

      toast.success('登录成功')
    } catch (error: any) {
      const message = error.response?.data?.detail || '登录失败'
      toast.error(message)
      throw error
    } finally {
      setIsLoading(false)
    }
  }

  const register = async (username: string, email: string, password: string) => {
    setIsLoading(true)
    try {
      await authApi.register({ username, email, password })

      // 注册成功后自动登录
      await login(username, password)
      toast.success('注册成功')
    } catch (error: any) {
      const message = error.response?.data?.detail || '注册失败'
      toast.error(message)
      throw error
    } finally {
      setIsLoading(false)
    }
  }

  const logout = async () => {
    setIsLoading(true)
    try {
      localStorage.removeItem('monika_token')
      localStorage.removeItem('monika_user')
      setToken(null)
      setUser(null)
      toast.success('已登出')
    } finally {
      setIsLoading(false)
    }
  }

  const value: AuthContextType = {
    user,
    token,
    isAuthenticated: !!user,
    isLoading,
    login,
    register,
    logout,
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}
```

### Step 2: 提交 AuthContext

```bash
git add frontend/src/contexts/AuthContext.tsx
git commit -m "feat: add AuthContext with login/register/logout"
```

---

## Task 3: 创建 ProtectedRoute 组件

**Files:**
- Create: `frontend/src/components/ProtectedRoute.tsx`

### Step 1: 实现路由保护组件

```typescript
// frontend/src/components/ProtectedRoute.tsx
import { Navigate } from 'react-router-dom'
import { useAuth } from '@/contexts/AuthContext'
import { Loader2 } from 'lucide-react'

export function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth()

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    )
  }

  if (!isAuthenticated) {
    return <Navigate to="/auth" replace />
  }

  return <>{children}</>
}
```

### Step 2: 提交 ProtectedRoute

```bash
git add frontend/src/components/ProtectedRoute.tsx
git commit -m "feat: add ProtectedRoute component"
```

---

## Task 4: 重构 AuthPage（替换 LoginPage）

**Files:**
- Modify: `frontend/src/pages/LoginPage.tsx` → 重命名为 `AuthPage.tsx`

### Step 1: 创建新的 AuthPage

```typescript
// frontend/src/pages/AuthPage.tsx
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/contexts/AuthContext'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Checkbox } from '@/components/ui/checkbox'
import { Loader2, AlertCircle } from 'lucide-react'
import { cn } from '@/lib/utils'

type AuthMode = 'login' | 'register'

export function AuthPage() {
  const [mode, setMode] = useState<AuthMode>('login')
  const [formData, setFormData] = useState({
    username: '',
    email: '',
    password: '',
    confirmPassword: '',
  })
  const [rememberMe, setRememberMe] = useState(true)
  const [error, setError] = useState<string('')
  const { login, register, isLoading } = useAuth()
  const navigate = useNavigate()

  const validateForm = (): string | null => {
    if (formData.username.length < 3) {
      return '用户名至少需要3个字符'
    }

    if (mode === 'register') {
      if (!formData.email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
        return '请输入有效的邮箱地址'
      }

      if (formData.password.length < 8) {
        return '密码至少需要8个字符'
      }

      const hasLetter = /[a-zA-Z]/.test(formData.password)
      const hasNumber = /[0-9]/.test(formData.password)
      const hasSpecial = /[!@#$%^&*(),.?":{}|<>]/.test(formData.password)

      if (!hasLetter || !hasNumber || !hasSpecial) {
        return '密码必须包含字母、数字和特殊字符'
      }

      if (formData.password !== formData.confirmPassword) {
        return '两次输入的密码不一致'
      }
    }

    return null
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    const validationError = validateForm()
    if (validationError) {
      setError(validationError)
      return
    }

    try {
      if (mode === 'login') {
        await login(formData.username, formData.password, rememberMe)
      } else {
        await register(formData.username, formData.email, formData.password)
      }
      navigate('/select-character')
    } catch (err) {
      // Error already handled by AuthContext
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-muted/50 p-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle className="text-2xl text-center">
            {mode === 'login' ? '登录' : '注册'} Monika
          </CardTitle>
        </CardHeader>

        <CardContent>
          {error && (
            <Alert variant="destructive" className="mb-4">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="username">用户名</Label>
              <Input
                id="username"
                type="text"
                value={formData.username}
                onChange={(e) =>
                  setFormData({ ...formData, username: e.target.value })
                }
                disabled={isLoading}
                required
              />
            </div>

            {mode === 'register' && (
              <div className="space-y-2">
                <Label htmlFor="email">邮箱</Label>
                <Input
                  id="email"
                  type="email"
                  value={formData.email}
                  onChange={(e) =>
                    setFormData({ ...formData, email: e.target.value })
                  }
                  disabled={isLoading}
                  required
                />
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="password">密码</Label>
              <Input
                id="password"
                type="password"
                value={formData.password}
                onChange={(e) =>
                  setFormData({ ...formData, password: e.target.value })
                }
                disabled={isLoading}
                required
              />
            </div>

            {mode === 'register' && (
              <div className="space-y-2">
                <Label htmlFor="confirmPassword">确认密码</Label>
                <Input
                  id="confirmPassword"
                  type="password"
                  value={formData.confirmPassword}
                  onChange={(e) =>
                    setFormData({ ...formData, confirmPassword: e.target.value })
                  }
                  disabled={isLoading}
                  required
                />
              </div>
            )}

            {mode === 'login' && (
              <div className="flex items-center space-x-2">
                <Checkbox
                  id="remember"
                  checked={rememberMe}
                  onCheckedChange={(checked) => setRememberMe(checked as boolean)}
                  disabled={isLoading}
                />
                <Label
                  htmlFor="remember"
                  className="text-sm font-normal cursor-pointer"
                >
                  记住我
                </Label>
              </div>
            )}

            <Button type="submit" className="w-full" disabled={isLoading}>
              {isLoading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  {mode === 'login' ? '登录中...' : '注册中...'}
                </>
              ) : (
                mode === 'login' ? '登录' : '注册'
              )}
            </Button>
          </form>

          <div className="mt-4 text-center text-sm text-muted-foreground">
            {mode === 'login' ? '没有账号？' : '已有账号？'}
            <button
              type="button"
              className="ml-1 text-primary hover:underline"
              onClick={() => {
                setMode(mode === 'login' ? 'register' : 'login')
                setError('')
              }}
            >
              {mode === 'login' ? '去注册' : '去登录'}
            </button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
```

### Step 2: 添加 Checkbox 组件（如果不存在）

```bash
npx shadcn@latest add checkbox
```

### Step 3: 提交 AuthPage

```bash
git add frontend/src/pages/AuthPage.tsx
git rm frontend/src/pages/LoginPage.tsx
git commit -m "feat: add AuthPage with login/register modes"
```

---

## Task 5: 创建 CharacterSelectScreen 组件

**Files:**
- Create: `frontend/src/components/CharacterSelectScreen.tsx`

### Step 1: 实现角色选择组件

```typescript
// frontend/src/components/CharacterSelectScreen.tsx
import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { characterApi, type Character } from '@/lib/api'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogTitle,
} from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { Play, Edit, Trash2, UserPlus, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import { CharacterForm } from './CharacterForm'

export function CharacterSelectScreen() {
  const [characters, setCharacters] = useState<Character[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [deleteDialog, setDeleteDialog] = useState(false)
  const [characterToDelete, setCharacterToDelete] = useState<Character | null>(null)
  const [isDeleting, setIsDeleting] = useState(false)
  const { user } = useAuth()
  const navigate = useNavigate()

  useEffect(() => {
    loadCharacters()
  }, [])

  const loadCharacters = async () => {
    try {
      const data = await characterApi.list()
      setCharacters(data)
    } catch (error) {
      toast.error('加载角色列表失败')
    } finally {
      setIsLoading(false)
    }
  }

  const handlePlay = (character: Character) => {
    navigate('/game', { state: { character } })
  }

  const handleEdit = (character: Character) => {
    navigate(`/character/${character.id}/edit`)
  }

  const handleDeleteClick = (character: Character) => {
    setCharacterToDelete(character)
    setDeleteDialog(true)
  }

  const handleDeleteConfirm = async () => {
    if (!characterToDelete) return

    setIsDeleting(true)
    try {
      await characterApi.delete(characterToDelete.id)
      setCharacters(characters.filter((c) => c.id !== characterToDelete.id))
      toast.success('角色已删除')
    } catch (error) {
      toast.error('删除角色失败')
    } finally {
      setIsDeleting(false)
      setDeleteDialog(false)
      setCharacterToDelete(null)
    }
  }

  const handleCharacterCreated = (character: Character) => {
    setCharacters([...characters, character])
  }

  const handleCharacterUpdated = (character: Character) => {
    setCharacters(
      characters.map((c) => (c.id === character.id ? character : c))
    )
  }

  if (isLoading) {
    return (
      <div className="container mx-auto py-8 px-4 max-w-4xl">
        <div className="flex justify-between items-center mb-6">
          <Skeleton className="h-8 w-32" />
          <Skeleton className="h-10 w-32" />
        </div>
        <div className="space-y-2">
          {[...Array(5)].map((_, i) => (
            <Skeleton key={i} className="h-16 w-full" />
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="container mx-auto py-8 px-4 max-w-4xl">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">我的角色</h1>
        <Button onClick={() => navigate('/character/new')}>
          <UserPlus className="mr-2 h-4 w-4" />
          创建新角色
        </Button>
      </div>

      {characters.length === 0 ? (
        <Card>
          <CardContent className="py-12">
            <div className="text-center">
              <UserPlus className="mx-auto h-12 w-12 text-muted-foreground mb-4" />
              <h3 className="text-lg font-semibold mb-2">还没有角色</h3>
              <p className="text-sm text-muted-foreground mb-6">
                创建你的第一个角色开始冒险吧！
              </p>
              <CharacterForm
                onSave={handleCharacterCreated}
                onCancel={() => {}}
              />
            </div>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>角色名</TableHead>
                  <TableHead>原型</TableHead>
                  <TableHead>HP</TableHead>
                  <TableHead>SAN</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {characters.map((character) => (
                  <TableRow key={character.id}>
                    <TableCell className="font-medium">
                      {character.name}
                    </TableCell>
                    <TableCell>{character.archetype}</TableCell>
                    <TableCell>
                      {character.hp}/{character.hp_max}
                    </TableCell>
                    <TableCell>
                      {character.san}/{character.san_max}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-2">
                        <Button
                          size="sm"
                          variant="default"
                          onClick={() => handlePlay(character)}
                        >
                          <Play className="h-4 w-4" />
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => handleEdit(character)}
                        >
                          <Edit className="h-4 w-4" />
                        </Button>
                        <Button
                          size="sm"
                          variant="destructive"
                          onClick={() => handleDeleteClick(character)}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      <Dialog open={deleteDialog} onOpenChange={setDeleteDialog}>
        <DialogContent>
          <DialogTitle>确认删除</DialogTitle>
          <DialogDescription>
            确定要删除角色"{characterToDelete?.name}"吗？此操作无法撤销。
          </DialogDescription>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setDeleteDialog(false)}
              disabled={isDeleting}
            >
              取消
            </Button>
            <Button
              variant="destructive"
              onClick={handleDeleteConfirm}
              disabled={isDeleting}
            >
              {isDeleting ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  删除中...
                </>
              ) : (
                '删除'
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
```

### Step 2: 提交 CharacterSelectScreen

```bash
git add frontend/src/components/CharacterSelectScreen.tsx
git commit -m "feat: add CharacterSelectScreen with CRUD operations"
```

---

## Task 6: 创建 CharacterSelectPage 包装器

**Files:**
- Create: `frontend/src/pages/CharacterSelectPage.tsx`

### Step 1: 创建页面组件

```typescript
// frontend/src/pages/CharacterSelectPage.tsx
import { CharacterSelectScreen } from '@/components/CharacterSelectScreen'

export function CharacterSelectPage() {
  return <CharacterSelectScreen />
}
```

### Step 2: 提交页面

```bash
git add frontend/src/pages/CharacterSelectPage.tsx
git commit -m "feat: add CharacterSelectPage wrapper"
```

---

## Task 7: 创建 LandingPage

**Files:**
- Create: `frontend/src/pages/LandingPage.tsx`

### Step 1: 创建落地页

```typescript
// frontend/src/pages/LandingPage.tsx
import { useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'

export function LandingPage() {
  const navigate = useNavigate()

  return (
    <div className="min-h-screen bg-gradient-to-b from-background to-muted/20">
      <div className="container mx-auto px-4 py-16">
        <div className="max-w-3xl mx-auto text-center space-y-8">
          <h1 className="text-5xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-primary to-primary/60">
            Monika
          </h1>
          <p className="text-xl text-muted-foreground">
            AI 驱动的《克苏鲁的呼唤》在线跑团平台
          </p>
          <p className="text-muted-foreground max-w-xl mx-auto">
            与 AI 守密人一起探索未知的世界，体验前所未有的 TRPG 冒险。
          </p>

          <div className="flex gap-4 justify-center">
            <Button size="lg" onClick={() => navigate('/auth')}>
              开始冒险
            </Button>
            <Button size="lg" variant="outline" onClick={() => navigate('/auth')}>
              了解更多
            </Button>
          </div>

          <div className="grid md:grid-cols-3 gap-6 mt-16">
            <Card>
              <CardContent className="pt-6">
                <h3 className="font-semibold mb-2">AI 守密人</h3>
                <p className="text-sm text-muted-foreground">
                  24/7 可用的 AI KP，随时开始你的冒险
                </p>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="pt-6">
                <h3 className="font-semibold mb-2">自动化规则</h3>
                <p className="text-sm text-muted-foreground">
                  检定、战斗、SAN 值自动处理，专注于角色扮演
                </p>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="pt-6">
                <h3 className="font-semibold mb-2">长团记忆</h3>
                <p className="text-sm text-muted-foreground">
                  结构化记忆系统，随时回顾历史，断点续跑
                </p>
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    </div>
  )
}
```

### Step 2: 提交 LandingPage

```bash
git add frontend/src/pages/LandingPage.tsx
git commit -m "feat: add LandingPage with hero and features"
```

---

## Task 8: 更新 CharacterForm 组件以支持 API 集成

**Files:**
- Modify: `frontend/src/components/CharacterForm.tsx`

### Step 1: 添加 onSave 和 onCancel props

查看现有的 CharacterForm，确保它接受这些 props：

```typescript
// 在 CharacterForm.tsx 中确保有这些 props
interface CharacterFormProps {
  characterId?: number
  onSave?: (character: Character) => void
  onCancel?: () => void
}

export function CharacterForm({ characterId, onSave, onCancel }: CharacterFormProps) {
  // ... 现有代码

  // 在保存函数中调用 API
  const handleSave = async (data: CharacterData) => {
    try {
      if (characterId) {
        const updated = await characterApi.update(characterId, data)
        onSave?.(updated)
      } else {
        const created = await characterApi.create(data)
        onSave?.(created)
      }
      toast.success('角色保存成功')
    } catch (error) {
      toast.error('保存角色失败')
    }
  }

  // ... 其余代码
}
```

### Step 2: 提交更新

```bash
git add frontend/src/components/CharacterForm.tsx
git commit -m "feat: integrate CharacterForm with API"
```

---

## Task 9: 更新 App.tsx 配置路由

**Files:**
- Modify: `frontend/src/App.tsx`

### Step 1: 完全重写 App.tsx

```typescript
// frontend/src/App.tsx
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider } from '@/contexts/AuthContext'
import { ProtectedRoute } from '@/components/ProtectedRoute'
import { Toaster } from '@/components/ui/toaster'
import { LandingPage } from '@/pages/LandingPage'
import { AuthPage } from '@/pages/AuthPage'
import { CharacterSelectPage } from '@/pages/CharacterSelectPage'
import { GameConsole } from '@/components/GameConsole'

function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route path="/" element={<LandingPage />} />
          <Route path="/auth" element={<AuthPage />} />

          <Route
            path="/select-character"
            element={
              <ProtectedRoute>
                <CharacterSelectPage />
              </ProtectedRoute>
            }
          />

          <Route
            path="/game"
            element={
              <ProtectedRoute>
                <GameConsole />
              </ProtectedRoute>
            }
          />

          {/* 添加角色编辑路由 */}
          <Route
            path="/character/:id/edit"
            element={
              <ProtectedRoute>
                <div>角色编辑页（待实现）</div>
              </ProtectedRoute>
            }
          />

          <Route
            path="/character/new"
            element={
              <ProtectedRoute>
                <div>角色创建页（待实现）</div>
              </ProtectedRoute>
            }
          />

          {/* 404 重定向 */}
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
        <Toaster />
      </AuthProvider>
    </BrowserRouter>
  )
}

export default App
```

### Step 2: 提交路由配置

```bash
git add frontend/src/App.tsx
git commit -m "feat: configure React Router with protected routes"
```

---

## Task 10: 更新 main.tsx 添加 Toaster

**Files:**
- Modify: `frontend/src/main.tsx`

### Step 1: 更新 main.tsx

```typescript
// frontend/src/main.tsx
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { Toaster } from '@/components/ui/toaster'
import App from './App'
import './index.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
    <Toaster />
  </StrictMode>
)
```

### Step 2: 提交更新

```bash
git add frontend/src/main.tsx
git commit -m "chore: add Toaster to main.tsx"
```

---

## Task 11: 更新 GameConsole 添加登出按钮

**Files:**
- Modify: `frontend/src/components/GameConsole.tsx`

### Step 1: 在 Header 组件中添加登出按钮

查看现有的 Header 组件并添加登出功能：

```typescript
// 在 Header.tsx 中添加登出按钮
import { Button } from '@/components/ui/button'
import { LogOut } from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'

export function Header({ onLogout }: { onLogout?: () => void }) {
  const { logout, user } = useAuth()

  const handleLogout = async () => {
    await logout()
    onLogout?.()
  }

  return (
    <header className="border-b px-6 py-3 flex items-center justify-between">
      <div>
        <h1 className="font-bold">Monika</h1>
        {user && <p className="text-sm text-muted-foreground">{user.username}</p>}
      </div>
      <Button variant="ghost" size="sm" onClick={handleLogout}>
        <LogOut className="h-4 w-4" />
      </Button>
    </header>
  )
}
```

### Step 2: 提交更新

```bash
git add frontend/src/components/Header.tsx frontend/src/components/GameConsole.tsx
git commit -m "feat: add logout button to GameConsole"
```

---

## 验收测试

### Step 1: 启动开发服务器

```bash
cd frontend
npm run dev
```

### Step 2: 启动后端服务器

```bash
cd backend
uv run python -m uvicorn src.main:app --reload
```

### Step 3: 手动测试流程

1. 访问 `http://localhost:5173`
2. 应该看到 LandingPage
3. 点击"开始冒险"跳转到 `/auth`
4. 测试注册新用户
5. 测试登录
6. 登录后应该跳转到 `/select-character`
7. 如果没有角色，应该看到快速创建表单
8. 创建角色后应该显示在列表中
9. 测试游玩、编辑、删除功能
10. 测试登出功能

### Step 4: 提交完成

```bash
git add .
git commit -m "feat: complete authentication UI implementation (M1-014, M1-015, M1-016, M1-028, M1-030)"
```

---

## 附录：密码验证正则表达式

如果需要在后端也验证密码，可以使用以下正则：

```typescript
const passwordRegex = /^(?=.*[a-zA-Z])(?=.*\d)(?=.*[!@#$%^&*(),.?":{}|<>]).{8,}$/
```

或者分开验证：
```typescript
const hasLength = password.length >= 8
const hasLetter = /[a-zA-Z]/.test(password)
const hasNumber = /\d/.test(password)
const hasSpecial = /[!@#$%^&*(),.?":{}|<>]/.test(password)

return hasLength && hasLetter && hasNumber && hasSpecial
```

---

## 任务映射

- **M1-014** RegisterPage → AuthPage（注册模式）✓
- **M1-015** LoginPage → AuthPage（登录模式）✓
- **M1-016** AuthContext → `frontend/src/contexts/AuthContext.tsx` ✓
- **M1-028** CharacterList → `frontend/src/components/CharacterSelectScreen.tsx` ✓
- **M1-030** CharacterPreview → 表格行展示 ✓
