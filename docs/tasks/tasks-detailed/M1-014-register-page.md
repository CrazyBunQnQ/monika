# M1-014: 实现注册页面 RegisterPage

**任务ID**: M1-014
**标题**: 实现注册页面 RegisterPage
**类型**: frontend (前端开发)
**预估工时**: 2h
**依赖**: M0

---

## 任务描述

实现用户注册页面 UI，包括表单验证、错误处理、注册成功后跳转等功能。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-014-01 | 设计注册页面布局 | 页面结构 | 20min |
| M1-014-02 | 实现注册表单组件 | 表单字段 | 30min |
| M1-014-03 | 实现表单验证 | 前端验证 | 25min |
| M1-014-04 | 实现密码强度指示 | 密码提示 | 20min |
| M1-014-05 | 集成注册 API | 调用后端 | 20min |
| M1-014-06 | 实现错误处理 | 错误提示 | 15min |
| M1-014-07 | 实现成功跳转 | 跳转逻辑 | 10min |

---

## 注册页面组件

```tsx
// frontend/src/pages/RegisterPage.tsx
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { PasswordStrength } from '@/components/auth/PasswordStrength'

export default function RegisterPage() {
  const navigate = useNavigate()
  const [formData, setFormData] = useState({
    username: '',
    email: '',
    password: '',
    confirmPassword: '',
  })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [passwordStrength, setPasswordStrength] = useState(0)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)

    // 验证密码
    if (formData.password !== formData.confirmPassword) {
      setError('密码不匹配')
      setLoading(false)
      return
    }

    try {
      const response = await fetch('/api/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          username: formData.username,
          email: formData.email,
          password: formData.password,
        }),
      })

      const data = await response.json()

      if (!response.ok) {
        throw new Error(data.detail || '注册失败')
      }

      // 注册成功，跳转到登录页
      navigate('/login', {
        state: { message: '注册成功，请登录' }
      })
    } catch (err: any) {
      setError(err.message || '注册失败，请稍后重试')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>创建账号</CardTitle>
          <CardDescription>
            注册 CoC 跑团平台账号
          </CardDescription>
        </CardHeader>

        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}

            <div className="space-y-2">
              <div>
                <Label htmlFor="username">用户名</Label>
                <Input
                  id="username"
                  type="text"
                  placeholder="输入用户名"
                  value={formData.username}
                  onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                  required
                  minLength={3}
                  maxLength={50}
                />
              </div>

              <div>
                <Label htmlFor="email">邮箱</Label>
                <Input
                  id="email"
                  type="email"
                  placeholder="your@email.com"
                  value={formData.email}
                  onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                  required
                />
              </div>

              <div>
                <Label htmlFor="password">密码</Label>
                <Input
                  id="password"
                  type="password"
                  placeholder="输入密码"
                  value={formData.password}
                  onChange={(e) => {
                    setFormData({ ...formData, password: e.target.value })
                    setPasswordStrength(calculateStrength(e.target.value))
                  }}
                  required
                />
                <PasswordStrength strength={passwordStrength} />
              </div>

              <div>
                <Label htmlFor="confirmPassword">确认密码</Label>
                <Input
                  id="confirmPassword"
                  type="password"
                  placeholder="再次输入密码"
                  value={formData.confirmPassword}
                  onChange={(e) => setFormData({ ...formData, confirmPassword: e.target.value })}
                  required
                />
              </div>

              <Button type="submit" className="w-full" disabled={loading}>
                {loading ? '注册中...' : '注册'}
              </Button>
            </div>
          </form>
        </CardContent>

        <CardFooter className="flex justify-between">
          <p className="text-sm text-muted-foreground">
            已有账号?{' '}
            <a href="/login" className="underline">登录</a>
          </p>
        </CardFooter>
      </Card>
    </div>
  )
}

// 密码强度计算
function calculateStrength(password: string): number {
  let strength = 0

  if (password.length >= 8) strength++
  if (password.length >= 12) strength++
  if (/[a-z]/.test(password)) strength++
  if (/[A-Z]/.test(password)) strength++
  if (/[0-9]/.test(password)) strength++
  if (/[^a-zA-Z0-9]/.test(password)) strength++

  return Math.min(5, strength)
}
```

---

## 密码强度组件

```tsx
// frontend/src/components/auth/PasswordStrength.tsx
interface PasswordStrengthProps {
  strength: number  // 0-5
}

export function PasswordStrength({ strength }: PasswordStrengthProps) {
  const levels = ['非常弱', '弱', '一般', '强', '非常强']
  const colors = [
    'bg-red-500',
    'bg-orange-500',
    'bg-yellow-500',
    'bg-green-500',
    'bg-emerald-500',
  ]

  return (
    <div className="mt-2 space-y-1">
      <div className="flex justify-between text-xs">
        <span>密码强度</span>
        <span className={colors[strength]}>{levels[strength]}</span>
      </div>
      <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
        <div
          className={`h-full transition-all duration-300 ${colors[strength]}`}
          style={{ width: `${(strength + 1) * 20}%` }}
        />
      </div>
    </div>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `frontend/src/pages/RegisterPage.tsx` | 创建 | 注册页面 |
| `frontend/src/components/auth/PasswordStrength.tsx` | 创建 | 密码强度组件 |
| `frontend/src/components/ui/form.tsx` | 创建 | 表单组件 |

---

## 路由配置

```tsx
// frontend/src/App.tsx
import RegisterPage from '@/pages/RegisterPage'

const router = [
  {
    path: '/register',
    element: <RegisterPage />,
  },
  // ... 其他路由
]
```

---

## 验收标准

- [ ] 注册页面布局正确
- [ ] 表单验证有效
- [ ] 密码强度指示准确
- [ ] API 调用成功
- [ ] 错误处理完善
- [ ] 注册成功后正确跳转

---

## 参考文档

- M0-039: 配色方案
- M1-006: 用户注册 API

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
