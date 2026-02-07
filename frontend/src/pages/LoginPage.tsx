import { useState } from 'react'
import { Button } from '@/components/ui/button'

interface LoginPageProps {
  onLogin: () => void
}

export function LoginPage({ onLogin }: LoginPageProps) {
  const [isRegister, setIsRegister] = useState(false)
  const [formData, setFormData] = useState({
    username: '',
    email: '',
    password: '',
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    // TODO: 实现登录/注册逻辑
    console.log('Form submitted:', formData)
    onLogin()
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-muted/50">
      <div className="w-full max-w-md p-8 bg-card rounded-xl shadow-lg">
        <h1 className="text-2xl font-bold text-center mb-6">
          {isRegister ? '注册' : '登录'} Monika
        </h1>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">用户名</label>
            <input
              type="text"
              value={formData.username}
              onChange={(e) =>
                setFormData({ ...formData, username: e.target.value })
              }
              className="w-full px-4 py-2 bg-muted rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary"
              required
            />
          </div>

          {isRegister && (
            <div>
              <label className="block text-sm font-medium mb-1">邮箱</label>
              <input
                type="email"
                value={formData.email}
                onChange={(e) =>
                  setFormData({ ...formData, email: e.target.value })
                }
                className="w-full px-4 py-2 bg-muted rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                required
              />
            </div>
          )}

          <div>
            <label className="block text-sm font-medium mb-1">密码</label>
            <input
              type="password"
              value={formData.password}
              onChange={(e) =>
                setFormData({ ...formData, password: e.target.value })
              }
              className="w-full px-4 py-2 bg-muted rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary"
              required
            />
          </div>

          <Button type="submit" className="w-full">
            {isRegister ? '注册' : '登录'}
          </Button>
        </form>

        <p className="text-center text-sm text-muted-foreground mt-4">
          {isRegister ? '已有账号？' : '没有账号？'}
          <button
            type="button"
            className="ml-1 text-primary hover:underline"
            onClick={() => setIsRegister(!isRegister)}
          >
            {isRegister ? '去登录' : '去注册'}
          </button>
        </p>
      </div>
    </div>
  )
}
