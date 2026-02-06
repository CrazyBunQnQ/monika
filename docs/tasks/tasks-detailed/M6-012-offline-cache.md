# M6-012: 实现离线缓存

**任务ID**: M6-012
**标题**: 实现离线缓存
**类型**: fullstack (全栈开发)
**预估工时**: 7h
**依赖**: M3-001

---

## 任务描述

实现 Service Worker 离线缓存系统，支持离线访问、后台同步、缓存策略管理，提升应用的可靠性和性能。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-012-01 | 设计缓存策略 | 缓存规则定义 | 1h |
| M6-012-02 | 实现 Service Worker | SW 核心逻辑 | 2h |
| M6-012-03 | 实现缓存管理 API | 缓存控制接口 | 1h |
| M6-012-04 | 实现离线 UI 提示 | 离线状态显示 | 45min |
| M6-012-05 | 实现后台同步 | 离线操作同步 | 1h |
| M6-012-06 | 实现缓存版本控制 | 缓存更新机制 | 45min |
| M6-012-07 | 编写缓存管理工具 | 开发者工具 | 45min |

---

## 前端实现

### Service Worker 注册

```typescript
// frontend/src/lib/service-worker/register.ts
const SW_VERSION = '1.0.0'
const SW_URL = `/sw.js?v=${SW_VERSION}`

interface RegisterOptions {
  onUpdate?: (registration: ServiceWorkerRegistration) => void
  onSuccess?: (registration: ServiceWorkerRegistration) => void
}

/**
 * 注册 Service Worker
 */
export async function registerServiceWorker(options: RegisterOptions = {}): Promise<boolean> {
  if (!('serviceWorker' in navigator)) {
    console.warn('Service Worker not supported')
    return false
  }

  try {
    const registration = await navigator.serviceWorker.register(SW_URL, {
      scope: '/'
    })

    // 检测更新
    registration.addEventListener('updatefound', () => {
      const newWorker = registration.installing
      if (newWorker) {
        newWorker.addEventListener('statechange', () => {
          if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
            // 新 SW 已安装，触发更新
            options.onUpdate?.(registration)
          } else if (newWorker.state === 'activated') {
            // SW 首次安装成功
            options.onSuccess?.(registration)
          }
        })
      }
    })

    // 监听控制器变化
    navigator.serviceWorker.addEventListener('controllerchange', () => {
      window.location.reload()
    })

    console.log('Service Worker registered:', registration)
    return true
  } catch (error) {
    console.error('Service Worker registration failed:', error)
    return false
  }
}

/**
 * 检查 Service Worker 更新
 */
export async function checkForUpdates(): Promise<boolean> {
  if (!('serviceWorker' in navigator)) {
    return false
  }

  try {
    const registration = await navigator.serviceWorker.getRegistration()
    if (!registration) {
      return false
    }

    await registration.update()
    return true
  } catch (error) {
    console.error('Failed to check for updates:', error)
    return false
  }
}

/**
 * 获取等待中的 SW
 */
export async function getWaitingSW(): Promise<ServiceWorker | null> {
  if (!('serviceWorker' in navigator)) {
    return null
  }

  try {
    const registration = await navigator.serviceWorker.getRegistration()
    return registration?.waiting || null
  } catch {
    return null
  }
}

/**
 * 跳过等待，立即激活新 SW
 */
export async function skipWaiting(): Promise<void> {
  const waiting = await getWaitingSW()
  if (waiting) {
    waiting.postMessage({ type: 'SKIP_WAITING' })
  }
}

/**
 * 手动取消 Service Worker 注册
 */
export async function unregisterServiceWorker(): Promise<boolean> {
  if (!('serviceWorker' in navigator)) {
    return false
  }

  try {
    const registrations = await navigator.serviceWorker.getRegistrations()
    await Promise.all(registrations.map(r => r.unregister()))
    return true
  } catch (error) {
    console.error('Failed to unregister Service Worker:', error)
    return false
  }
}
```

### Service Worker 实现

```javascript
// frontend/public/sw.js
const CACHE_VERSION = 'v1.0.0'
const CACHE_NAME = `coc-trpg-${CACHE_VERSION}`

// 缓存策略配置
const CACHE_STRATEGIES = {
  // 静态资源：缓存优先
  static: {
    pattern: /\.(js|css|png|jpg|jpeg|svg|gif|woff|woff2|ttf|eot)$/,
    strategy: 'cacheFirst'
  },

  // API 请求：网络优先，短期缓存
  api: {
    pattern: /^\/api\//,
    strategy: 'networkFirst',
    cacheTime: 5 * 60 * 1000  // 5 分钟
  },

  // HTML 文档：网络优先
  document: {
    pattern: /\.html$/,
    strategy: 'networkFirst'
  },

  // 其他：网络优先
  default: {
    strategy: 'networkFirst'
  }
}

// 预缓存资源列表
const PRECACHE_URLS = [
  '/',
  '/index.html',
  '/offline.html',
  '/manifest.json'
]

// 安装事件
self.addEventListener('install', (event) => {
  console.log('[SW] Install:', CACHE_VERSION)

  event.waitUntil(
    (async () => {
      // 预缓存核心资源
      const cache = await caches.open(CACHE_NAME)
      await cache.addAll(PRECACHE_URLS)

      // 立即激活
      await self.skipWaiting()
    })()
  )
})

// 激活事件
self.addEventListener('activate', (event) => {
  console.log('[SW] Activate:', CACHE_VERSION)

  event.waitUntil(
    (async () => {
      // 清理旧缓存
      const cacheNames = await caches.keys()
      await Promise.all(
        cacheNames
          .filter(name => name.startsWith('coc-trpg-') && name !== CACHE_NAME)
          .map(name => caches.delete(name))
      )

      // 立即控制所有客户端
      await self.clients.claim()
    })()
  )
})

// 消息事件
self.addEventListener('message', (event) => {
  const { type, data } = event.data

  switch (type) {
    case 'SKIP_WAITING':
      self.skipWaiting()
      break

    case 'CLEAR_CACHE':
      clearCache(data?.pattern)
      break

    case 'GET_CACHE_SIZE':
      getCacheSize().then(size => {
        event.ports[0].postMessage({ type: 'CACHE_SIZE', data: size })
      })
      break
  }
})

// 拦截网络请求
self.addEventListener('fetch', (event) => {
  const { request } = event
  const url = new URL(request.url)

  // 只处理同源请求
  if (url.origin !== self.location.origin) {
    return
  }

  // 跳过 chrome-extension 等
  if (!url.protocol.startsWith('http')) {
    return
  }

  // 确定缓存策略
  const strategy = determineStrategy(url.pathname, request.destination)

  // 应用策略
  event.respondWith(handleRequest(request, strategy))
})

/**
 * 确定缓存策略
 */
function determineStrategy(pathname, destination) {
  // API 请求
  if (pathname.startsWith('/api/')) {
    return CACHE_STRATEGIES.api
  }

  // 静态资源
  for (const [name, config] of Object.entries(CACHE_STRATEGIES)) {
    if (config.pattern && config.pattern.test(pathname)) {
      return config
    }
  }

  // HTML 文档
  if (destination === 'document') {
    return { strategy: 'networkFirst' }
  }

  return CACHE_STRATEGIES.default
}

/**
 * 处理请求
 */
async function handleRequest(request, strategyConfig) {
  const { strategy } = strategyConfig

  switch (strategy) {
    case 'cacheFirst':
      return cacheFirst(request, strategyConfig)
    case 'networkFirst':
      return networkFirst(request, strategyConfig)
    case 'staleWhileRevalidate':
      return staleWhileRevalidate(request, strategyConfig)
    case 'networkOnly':
      return networkOnly(request)
    case 'cacheOnly':
      return cacheOnly(request)
    default:
      return networkFirst(request, strategyConfig)
  }
}

/**
 * 缓存优先策略
 */
async function cacheFirst(request, config) {
  const cache = await caches.open(CACHE_NAME)
  const cached = await cache.match(request)

  if (cached) {
    // 后台更新缓存
    fetch(request).then(response => {
      if (response.ok) {
        cache.put(request, response.clone())
      }
    }).catch(() => {})

    return cached
  }

  try {
    const response = await fetch(request)
    if (response.ok) {
      cache.put(request, response.clone())
    }
    return response
  } catch {
    // 返回离线页面
    return getOfflineResponse(request)
  }
}

/**
 * 网络优先策略
 */
async function networkFirst(request, config) {
  const cache = await caches.open(CACHE_NAME)

  try {
    const response = await fetch(request)

    // 缓存成功的响应
    if (response.ok && request.method === 'GET') {
      cache.put(request, response.clone())
    }

    return response
  } catch {
    // 网络失败，尝试缓存
    const cached = await cache.match(request)

    if (cached) {
      // 添加离线标记
      const headers = new Headers(cached.headers)
      headers.append('X-Offline', 'true')
      return new Response(cached.body, {
        status: cached.status,
        statusText: cached.statusText,
        headers
      })
    }

    // 返回离线页面
    return getOfflineResponse(request)
  }
}

/**
 * 过时但可重用策略
 */
async function staleWhileRevalidate(request, config) {
  const cache = await caches.open(CACHE_NAME)
  const cached = await cache.match(request)

  // 后台获取并更新
  const fetchPromise = fetch(request).then(response => {
    if (response.ok) {
      cache.put(request, response.clone())
    }
    return response
  })

  // 返回缓存或等待网络
  return cached || fetchPromise
}

/**
 * 仅网络
 */
async function networkOnly(request) {
  return fetch(request)
}

/**
 * 仅缓存
 */
async function cacheOnly(request) {
  const cache = await caches.open(CACHE_NAME)
  const cached = await cache.match(request)
  return cached || getOfflineResponse(request)
}

/**
 * 获取离线响应
 */
function getOfflineResponse(request) {
  // API 请求返回错误响应
  if (request.url.includes('/api/')) {
    return new Response(
      JSON.stringify({ error: 'Offline', message: 'No network connection' }),
      {
        status: 503,
        headers: { 'Content-Type': 'application/json' }
      }
    )
  }

  // 导航请求返回离线页面
  if (request.destination === 'document') {
    return caches.match('/offline.html')
  }

  // 其他请求返回网络错误
  return new Response('Network error', { status: 503 })
}

/**
 * 清除缓存
 */
async function clearCache(pattern) {
  const cache = await caches.open(CACHE_NAME)
  const keys = await cache.keys()

  for (const request of keys) {
    if (!pattern || new RegExp(pattern).test(request.url)) {
      await cache.delete(request)
    }
  }
}

/**
 * 获取缓存大小
 */
async function getCacheSize() {
  const cache = await caches.open(CACHE_NAME)
  const keys = await cache.keys()

  let totalSize = 0
  for (const request of keys) {
    const response = await cache.match(request)
    if (response) {
      const blob = await response.blob()
      totalSize += blob.size
    }
  }

  return totalSize
}

// 后台同步
self.addEventListener('sync', (event) => {
  console.log('[SW] Background sync:', event.tag)

  if (event.tag === 'sync-commands') {
    event.waitUntil(syncCommands())
  } else if (event.tag === 'sync-feedback') {
    event.waitUntil(syncFeedback())
  }
})

/**
 * 同步离线命令
 */
async function syncCommands() {
  // 从 IndexedDB 获取待同步的命令
  const commands = await getOfflineCommands()

  for (const command of commands) {
    try {
      const response = await fetch('/api/commands', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(command)
      })

      if (response.ok) {
        await removeOfflineCommand(command.id)
      }
    } catch (error) {
      console.error('Failed to sync command:', error)
    }
  }
}

/**
 * 同步离线反馈
 */
async function syncFeedback() {
  // 从 IndexedDB 获取待同步的反馈
  const feedbacks = await getOfflineFeedbacks()

  for (const feedback of feedbacks) {
    try {
      const response = await fetch('/api/feedback', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(feedback)
      })

      if (response.ok) {
        await removeOfflineFeedback(feedback.id)
      }
    } catch (error) {
      console.error('Failed to sync feedback:', error)
    }
  }
}

// 推送通知（可选）
self.addEventListener('push', (event) => {
  const options = {
    body: event.data?.text() || 'New update available',
    icon: '/icons/icon-192.png',
    badge: '/icons/badge.png',
    vibrate: [200, 100, 200],
    data: {
      url: '/'
    }
  }

  event.waitUntil(
    self.registration.showNotification('CoC TRPG', options)
  )
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  event.waitUntil(
    clients.openWindow(event.notification.data.url)
  )
})
```

### 离线状态 Hook

```typescript
// frontend/src/hooks/useOfflineStatus.ts
import { useState, useEffect } from 'react'

export interface OfflineStatus {
  isOnline: boolean
  isOffline: boolean
  offlineSince: Date | null
}

export function useOfflineStatus(): OfflineStatus {
  const [status, setStatus] = useState<OfflineStatus>({
    isOnline: navigator.onLine,
    isOffline: !navigator.onLine,
    offlineSince: null
  })

  useEffect(() => {
    const handleOnline = () => {
      setStatus({
        isOnline: true,
        isOffline: false,
        offlineSince: null
      })
    }

    const handleOffline = () => {
      setStatus({
        isOnline: false,
        isOffline: true,
        offlineSince: new Date()
      })
    }

    window.addEventListener('online', handleOnline)
    window.addEventListener('offline', handleOffline)

    return () => {
      window.removeEventListener('online', handleOnline)
      window.removeEventListener('offline', handleOffline)
    }
  }, [])

  return status
}
```

### 离线 UI 组件

```tsx
// frontend/src/components/offline/OfflineBanner.tsx
import { useOfflineStatus } from '@/hooks/useOfflineStatus'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { WifiOff, RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useEffect, useState } from 'react'

export function OfflineBanner() {
  const { isOnline, isOffline, offlineSince } = useOfflineStatus()
  const [showSync, setShowSync] = useState(false)

  useEffect(() => {
    if (isOnline && offlineSince) {
      // 刚从离线恢复，显示同步提示
      setShowSync(true)
      const timer = setTimeout(() => setShowSync(false), 5000)
      return () => clearTimeout(timer)
    }
  }, [isOnline, offlineSince])

  if (!showSync && isOnline) {
    return null
  }

  return (
    <>
      {isOffline && (
        <Alert variant="destructive" className="fixed top-0 left-0 right-0 z-50 rounded-none">
          <WifiOff className="h-4 w-4" />
          <AlertDescription className="flex items-center justify-between">
            <span>
              网络连接已断开，部分功能可能不可用。离线期间的更改将在恢复连接后自动同步。
            </span>
          </AlertDescription>
        </Alert>
      )}

      {showSync && (
        <Alert className="fixed top-0 left-0 right-0 z-50 rounded-none bg-green-500 text-white">
          <RefreshCw className="h-4 w-4 animate-spin" />
          <AlertDescription className="flex items-center justify-between">
            <span>
              网络已恢复，正在同步离线期间的更改...
            </span>
            <Button
              variant="ghost"
              size="sm"
              className="text-white hover:bg-green-600"
              onClick={() => setShowSync(false)}
            >
              关闭
            </Button>
          </AlertDescription>
        </Alert>
      )}
    </>
  )
}
```

### 离线队列管理

```typescript
// frontend/src/lib/offline-queue.ts
import { openDB } from 'idb'

interface QueueItem {
  id: string
  type: 'command' | 'feedback' | 'analytics'
  data: any
  timestamp: number
  retries: number
}

const DB_NAME = 'coc-trpg-offline'
const DB_VERSION = 1
const STORE_NAME = 'queue'

class OfflineQueue {
  private db: any

  async init() {
    this.db = await openDB(DB_NAME, DB_VERSION, {
      upgrade(db) {
        if (!db.objectStoreNames.contains(STORE_NAME)) {
          const store = db.createObjectStore(STORE_NAME, { keyPath: 'id' })
          store.createIndex('type', 'type')
          store.createIndex('timestamp', 'timestamp')
        }
      }
    })
  }

  async add(item: Omit<QueueItem, 'id' | 'timestamp' | 'retries'>): Promise<void> {
    await this.init()

    const queueItem: QueueItem = {
      id: `${item.type}_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
      timestamp: Date.now(),
      retries: 0,
      ...item
    }

    await this.db.add(STORE_NAME, queueItem)
  }

  async getAll(type?: QueueItem['type']): Promise<QueueItem[]> {
    await this.init()

    if (type) {
      return await this.db.getAllFromIndex(STORE_NAME, 'type', type)
    }

    return await this.db.getAll(STORE_NAME)
  }

  async remove(id: string): Promise<void> {
    await this.init()
    await this.db.delete(STORE_NAME, id)
  }

  async clear(): Promise<void> {
    await this.init()
    await this.db.clear(STORE_NAME)
  }

  async count(type?: QueueItem['type']): Promise<number> {
    await this.init()

    if (type) {
      const items = await this.getAll(type)
      return items.length
    }

    return await this.db.count(STORE_NAME)
  }
}

export const offlineQueue = new OfflineQueue()

/**
 * 同步离线队列
 */
export async function syncOfflineQueue() {
  if (!navigator.onLine) {
    return
  }

  const items = await offlineQueue.getAll()

  for (const item of items) {
    try {
      let success = false

      switch (item.type) {
        case 'command':
          success = await syncCommand(item.data)
          break
        case 'feedback':
          success = await syncFeedback(item.data)
          break
        case 'analytics':
          success = await syncAnalytics(item.data)
          break
      }

      if (success) {
        await offlineQueue.remove(item.id)
      } else {
        // 增加重试计数
        item.retries++
        if (item.retries > 3) {
          // 超过最大重试次数，放弃
          await offlineQueue.remove(item.id)
        }
      }
    } catch (error) {
      console.error('Failed to sync item:', item, error)
    }
  }
}

async function syncCommand(data: any): Promise<boolean> {
  const response = await fetch('/api/commands', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data)
  })
  return response.ok
}

async function syncFeedback(data: any): Promise<boolean> {
  const response = await fetch('/api/feedback', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data)
  })
  return response.ok
}

async function syncAnalytics(data: any): Promise<boolean> {
  const response = await fetch('/api/analytics', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data)
  })
  return response.ok
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `frontend/src/lib/service-worker/register.ts` | 创建 | SW 注册工具 |
| `frontend/public/sw.js` | 创建 | Service Worker |
| `frontend/public/offline.html` | 创建 | 离线页面 |
| `frontend/src/hooks/useOfflineStatus.ts` | 创建 | 离线状态 Hook |
| `frontend/src/components/offline/OfflineBanner.tsx` | 创建 | 离线提示组件 |
| `frontend/src/lib/offline-queue.ts` | 创建 | 离线队列管理 |
| `frontend/src/index.tsx` | 修改 | 集成 SW 注册 |

---

## 验收标准

- [ ] Service Worker 正确安装和激活
- [ ] 静态资源被正确缓存
- [ ] API 响应被缓存（网络优先）
- [ ] 离线时显示友好提示
- [ ] 离线操作在恢复后自动同步
- [ ] 缓存更新机制正常工作
- [ ] PWA 安装提示正常显示

---

## 参考文档

- M3-001: 记忆系统
- M6-013: PWA 实现
- Service Worker API 文档
- Workbox 官方文档

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
