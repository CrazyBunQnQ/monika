# M1-015: 实现登录页面 LoginPage

**任务ID**: M1-015
**标题**: 实现登录页面 LoginPage
**类型**: frontend (前端开发)
**预估工时**: 2h
**依赖**: M1-014

---

## 任务描述

实现用户登录页面 UI，包括表单验证、记住我功能、登录失败处理等。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-015-01 | 设计登录页面布局 | 页面结构 | 20min |
| M1-015-02 | 实现登录表单组件 | 表单字段 | 30min |
| M1-015-03 | 实现记住我功能 | 本地存储 | 20min |
| M1-015-04 | 集成登录 API | 调用后端 | 25min |
| M1-015-05 | 实现 Token 存储 | 本地存储 | 20min |
| M1-015-06 | 实现登录跳转 | 跳转逻辑 | 10min |
| M1-015-07 | 实现错误提示 | 错误处理 | 10min |

---

## 登录页面组件

```tsx
// frontend/src/pages/LoginPage.tsx
import { useState, useEffect } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'

export default function LoginPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const { login, isAuthenticated } = useAuth()

  const [formData, setFormData] = useState({
    username: '',
    password: '',
  })
  const [rememberMe, setRememberMe] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  // 如果已登录，重定向
  useEffect(() => {
    if (isAuthenticated) {
      const params = new URLSearchParams(location.search)
      const redirectTo = params.get('redirect') || '/game'
      navigate(redirectTo)
    }
  }, [isAuthenticated, location, navigate])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      // 调用登录 API
      const response = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          username: formData.username,
          password: formData.password,
          remember_me: rememberMe,
        }),
      })

      const data = await response.json()

      if (!response.ok) {
        throw new Error(data.detail || '登录失败')
      }

      // 保存 Token
      localStorage.setItem('access_token', data.access_token)
      if (data.refresh_token) {
        localStorage.setItem('refresh_token', data.refresh_token)
      }

      // 更新认证状态
      await login(data.access_token)

      // 跳转到原目标页面
      const params = new URLSearchParams(location.search)
      const redirectTo = params.get('redirect') || '/game'
      navigate(redirectTo)
    } catch (err: any) {
      setError(err.message || '登录失败，请检查用户名和密码')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>登录</CardTitle>
          <CardDescription>
            登录 CoC 跑团平台
          </CardDescription>
        </CardHeader>

        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}

            {location.state?.message && (
              <Alert>
                <AlertDescription>{location.state.message}</AlertDescription>
              </Alert>
            )}

            <div className="space-y-2">
              <div>
                <Label htmlFor="username">用户名或邮箱</Label>
                <Input
                  id="username"
                  type="text"
                  placeholder="输入用户名或邮箱"
                  value={formData.username}
                  onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                  required
                  autoFocus
                />
              </div>

              <div>
                <Label htmlFor="password">密码</Label>
                <Input
                  id="password"
                  type="password"
                  placeholder="输入密码"
                  value={formData.password}
                  onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                  required
                />
              </div>

              <div className="flex items-center space-x-2">
                <Checkbox
                  id="remember"
                  checked={rememberMe}
                  onCheckedChange={setRememberMe}
                />
                <Label htmlFor="remember" className="text-sm">
                  记住我
                </Label>
              </div>

              <Button type="submit" className="w-full" disabled={loading}>
                {loading ? '登录中...' : '登录'}
              </Button>
            </div>

            <div className="text-center">
              <a
                href="/forgot-password"
                className="text-sm text-muted-foreground hover:underline"
              >
                忘记密码?
              </a>
            </div>
          </form>
        </CardContent>

        <CardFooter className="flex justify-between">
          <p className="text-sm text-muted-foreground">
            还没有账号?{' '}
            <a href="/register" className="underline">
              注册
            </a>
          </p>
        </CardFooter>
      </Card>
    </div>
  )
}
```

---

## 验收标准

- [ ] 登录页面布局正确
- [ ] 表单验证有效
- [ ] 记住我功能正常
- [ ] Token 正确保存
- [ ] 登录成功后跳转
- [ ] 错误提示友好

---

## 参考文档

- M1-007: JWT Token 中间件
- M1-014: 注册页面

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
