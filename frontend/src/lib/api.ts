import axios from 'axios'

// 存储键名常量
export const STORAGE_KEYS = {
  TOKEN: 'monika_token',
  USER: 'monika_user',
} as const

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8000'

export const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 请求拦截器 - 添加 token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem(STORAGE_KEYS.TOKEN)
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
      localStorage.removeItem(STORAGE_KEYS.TOKEN)
      localStorage.removeItem(STORAGE_KEYS.USER)
      window.location.href = '/auth'
    }
    return Promise.reject(error)
  }
)

// 类型定义
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

export interface Character {
  id: number
  owner_id: number
  name: string
  age: number
  gender: string
  occupation: string
  mental_illness: string
  backstory: string
  str: number
  con: number
  dex: number
  app: number
  pow: number
  int: number
  siz: number
  edu: number
  hp: number
  mp: number
  san: number
  max_san: number
  luck: number
  created_at: string
  updated_at: string
}

// 创建/更新时使用的类型（排除只读字段）
export interface CharacterCreate {
  name: string
  age: number
  gender: string
  occupation: string
  mental_illness?: string
  backstory?: string
  str: number
  con: number
  dex: number
  app: number
  pow: number
  intelligence: number  // 后端实际字段名
  siz: number
  edu: number
  luck?: number
}

// 认证 API
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

// 角色 API
export const characterApi = {
  list: async (): Promise<Character[]> => {
    const response = await api.get<Character[]>('/characters')
    return response.data
  },

  getById: async (id: number): Promise<Character> => {
    const response = await api.get<Character>(`/characters/${id}`)
    return response.data
  },

  create: async (data: CharacterCreate): Promise<Character> => {
    const response = await api.post<Character>('/characters', data)
    return response.data
  },

  update: async (id: number, data: Partial<CharacterCreate>): Promise<Character> => {
    const response = await api.put<Character>(`/characters/${id}`, data)
    return response.data
  },

  delete: async (id: number): Promise<void> => {
    await api.delete(`/characters/${id}`)
  },
}
