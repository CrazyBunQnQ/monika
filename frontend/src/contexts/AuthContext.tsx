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
