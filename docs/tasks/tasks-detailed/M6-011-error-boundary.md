# M6-011: 实现错误边界

**任务ID**: M6-011
**标题**: 实现错误边界
**类型**: frontend (前端开发)
**预估工时**: 5h
**依赖**: M1-040

---

## 任务描述

实现 React 错误边界组件，捕获组件树中的 JavaScript 错误，提供友好的错误界面，支持错误恢复和上报。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-011-01 | 设计错误边界策略 | 捕获层级和范围 | 30min |
| M6-011-02 | 实现全局错误边界 | 顶层错误捕获 | 1h |
| M6-011-03 | 实现组件级错误边界 | 局部错误隔离 | 1h |
| M6-011-04 | 实现错误 UI 组件 | 错误展示界面 | 1h |
| M6-011-05 | 实现错误上报服务 | 错误日志收集 | 45min |
| M6-011-06 | 实现错误恢复机制 | 重试和重置 | 45min |
| M6-011-07 | 编写错误处理文档 | 开发者指南 | 30min |

---

## 前端实现

### 错误边界基础组件

```tsx
// frontend/src/components/error-boundary/ErrorBoundary.tsx
import React, { Component, ErrorInfo, ReactNode } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { AlertTriangle, RefreshCw, Home, Bug } from 'lucide-react'
import { reportError } from '@/lib/error-reporting'

interface Props {
  children: ReactNode
  fallback?: ReactNode
  onError?: (error: Error, errorInfo: ErrorInfo) => void
  isolate?: boolean  // 是否隔离错误，不影响父组件
}

interface State {
  hasError: boolean
  error: Error | null
  errorInfo: ErrorInfo | null
  errorId: string | null
}

export class ErrorBoundary extends Component<Props, State> {
  private retryCount = 0
  private maxRetries = 3

  constructor(props: Props) {
    super(props)
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null,
      errorId: null
    }
  }

  static getDerivedStateFromError(error: Error): Partial<State> {
    return {
      hasError: true,
      error
    }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    const errorId = this.generateErrorId()

    this.setState({
      errorInfo,
      errorId
    })

    // 上报错误
    reportError(error, {
      errorInfo,
      errorId,
      componentStack: errorInfo.componentStack,
      props: this.props
    })

    // 调用自定义错误处理
    this.props.onError?.(error, errorInfo)

    // 打印到控制台（开发环境）
    if (process.env.NODE_ENV === 'development') {
      console.error('ErrorBoundary caught an error:', error)
      console.error('Component stack:', errorInfo.componentStack)
    }
  }

  private generateErrorId(): string {
    return `err_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
  }

  private handleRetry = () => {
    if (this.retryCount < this.maxRetries) {
      this.retryCount++
      this.setState({
        hasError: false,
        error: null,
        errorInfo: null,
        errorId: null
      })
    }
  }

  private handleReset = () => {
    this.retryCount = 0
    this.setState({
      hasError: false,
      error: null,
      errorInfo: null,
      errorId: null
    })
  }

  private handleGoHome = () => {
    window.location.href = '/'
  }

  render() {
    if (this.state.hasError) {
      // 使用自定义 fallback
      if (this.props.fallback) {
        return this.props.fallback
      }

      // 默认错误 UI
      return <ErrorFallback
        error={this.state.error}
        errorId={this.state.errorId}
        onRetry={this.handleRetry}
        onReset={this.handleReset}
        onGoHome={this.handleGoHome}
        canRetry={this.retryCount < this.maxRetries}
      />
    }

    return this.props.children
  }
}

/**
 * 默认错误回退组件
 */
interface ErrorFallbackProps {
  error: Error | null
  errorId: string | null
  onRetry: () => void
  onReset: () => void
  onGoHome: () => void
  canRetry: boolean
}

function ErrorFallback({
  error,
  errorId,
  onRetry,
  onReset,
  onGoHome,
  canRetry
}: ErrorFallbackProps) {
  const isDev = process.env.NODE_ENV === 'development'

  return (
    <div className="min-h-screen flex items-center justify-center bg-background p-4">
      <Card className="w-full max-w-lg">
        <CardHeader>
          <div className="flex items-center space-x-2">
            <AlertTriangle className="h-6 w-6 text-destructive" />
            <CardTitle>出错了</CardTitle>
          </div>
        </CardHeader>

        <CardContent className="space-y-4">
          {/* 用户友好的错误消息 */}
          <p className="text-muted-foreground">
            抱歉，应用遇到了意外错误。你可以尝试以下操作：
          </p>

          {/* 操作按钮 */}
          <div className="flex flex-col sm:flex-row gap-2">
            {canRetry && (
              <Button onClick={onRetry} className="flex-1">
                <RefreshCw className="h-4 w-4 mr-2" />
                重试
              </Button>
            )}
            <Button onClick={onReset} variant="outline" className="flex-1">
              重置
            </Button>
            <Button onClick={onGoHome} variant="outline" className="flex-1">
              <Home className="h-4 w-4 mr-2" />
              返回首页
            </Button>
          </div>

          {/* 错误 ID（用于反馈） */}
          {errorId && (
            <div className="text-xs text-muted-foreground bg-muted p-2 rounded">
              错误代码: <code className="font-mono">{errorId}</code>
            </div>
          )}

          {/* 开发环境：显示详细错误信息 */}
          {isDev && error && (
            <details className="mt-4">
              <summary className="cursor-pointer text-sm font-semibold mb-2">
                错误详情（仅开发环境）
              </summary>
              <div className="bg-destructive/10 p-3 rounded text-xs font-mono overflow-auto max-h-48">
                <div className="font-bold text-destructive mb-2">
                  {error.name}: {error.message}
                </div>
                <div className="whitespace-pre-wrap text-muted-foreground">
                  {error.stack}
                </div>
              </div>
            </details>
          )}

          {/* 反馈按钮 */}
          <div className="pt-4 border-t">
            <Button
              variant="ghost"
              size="sm"
              className="w-full"
              onClick={() => window.open(`/feedback?error_id=${errorId}`, '_blank')}
            >
              <Bug className="h-4 w-4 mr-2" />
              报告这个问题
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
```

### 特定组件错误边界

```tsx
// frontend/src/components/error-boundary/ComponentErrorBoundary.tsx
import React, { Component, ErrorInfo, ReactNode } from 'react'
import { AlertCircle } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'

interface Props {
  children: ReactNode
  fallbackMessage?: string
  onError?: (error: Error) => void
}

interface State {
  hasError: boolean
}

/**
 * 组件级错误边界
 * 用于隔离单个组件的错误，不影响整个应用
 */
export class ComponentErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false }
  }

  static getDerivedStateFromError() {
    return { hasError: true }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    // 上报错误
    reportError(error, {
      componentStack: errorInfo.componentStack,
      level: 'warning'  // 组件级错误，不是致命的
    })

    this.props.onError?.(error)
  }

  render() {
    if (this.state.hasError) {
      return (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>
            {this.props.fallbackMessage || '此组件暂时无法加载'}
          </AlertDescription>
        </Alert>
      )
    }

    return this.props.children
  }
}
```

### 异步错误边界

```tsx
// frontend/src/components/error-boundary/AsyncErrorBoundary.tsx
import React, { Component, ReactNode } from 'react'

interface Props {
  children: ReactNode
  fallback?: ReactNode
}

interface State {
  hasError: boolean
}

/**
 * 异步错误边界
 * 捕获异步操作中的错误
 */
export class AsyncErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false }
  }

  componentDidMount() {
    // 捕获未处理的 Promise rejection
    window.addEventListener('unhandledrejection', this.handlePromiseRejection)

    // 捕获全局错误
    window.addEventListener('error', this.handleGlobalError)
  }

  componentWillUnmount() {
    window.removeEventListener('unhandledrejection', this.handlePromiseRejection)
    window.removeEventListener('error', this.handleGlobalError)
  }

  handlePromiseRejection = (event: PromiseRejectionEvent) => {
    event.preventDefault()

    const error = event.reason instanceof Error
      ? event.reason
      : new Error(String(event.reason))

    reportError(error, {
      type: 'unhandled_promise_rejection',
      reason: event.reason
    })

    // 在开发环境显示错误
    if (process.env.NODE_ENV === 'development') {
      console.error('Unhandled promise rejection:', event.reason)
    }
  }

  handleGlobalError = (event: ErrorEvent) => {
    event.preventDefault()

    const error = event.error || new Error(event.message)

    reportError(error, {
      type: 'global_error',
      filename: event.filename,
      lineno: event.lineno,
      colno: event.colno
    })

    if (process.env.NODE_ENV === 'development') {
      console.error('Global error:', error)
    }
  }

  render() {
    if (this.state.hasError && this.props.fallback) {
      return this.props.fallback
    }

    return this.props.children
  }
}
```

### Hooks 形式的错误边界

```tsx
// frontend/src/hooks/useErrorHandler.ts
import { useState, useCallback } from 'react'
import { reportError } from '@/lib/error-reporting'

export interface ErrorHandler {
  handleError: (error: Error) => void
  error: Error | null
  clearError: () => void
}

/**
 * 错误处理 Hook
 * 用于函数组件中手动处理错误
 */
export function useErrorHandler(): ErrorHandler {
  const [error, setError] = useState<Error | null>(null)

  const handleError = useCallback((error: Error) => {
    setError(error)
    reportError(error)
  }, [])

  const clearError = useCallback(() => {
    setError(null)
  }, [])

  return {
    handleError,
    error,
    clearError
  }
}

/**
 * 异步操作错误处理 Hook
 */
export function useAsyncError() {
  const handleError = useErrorHandler().handleError

  return (error: Error) => {
    // 延迟到下一个事件循环，确保可以被 ErrorBoundary 捕获
    setTimeout(() => {
      handleError(error)
    }, 0)
  }
}
```

### 错误上报服务

```typescript
// frontend/src/lib/error-reporting.ts
interface ErrorContext {
  errorInfo?: {
    componentStack: string
  }
  errorId?: string
  componentStack?: string
  props?: any
  type?: string
  level?: 'error' | 'warning'
  [key: string]: any
}

interface ErrorReport {
  message: string
  stack?: string
  name?: string
  context: ErrorContext
  userAgent: string
  url: string
  timestamp: string
  userId?: string
}

/**
 * 上报错误到服务器
 */
export async function reportError(
  error: Error,
  context: ErrorContext = {}
): Promise<void> {
  // 获取用户信息
  const userId = getUserId()

  const report: ErrorReport = {
    message: error.message,
    stack: error.stack,
    name: error.name,
    context,
    userAgent: navigator.userAgent,
    url: window.location.href,
    timestamp: new Date().toISOString(),
    userId: userId || undefined
  }

  // 开发环境打印到控制台
  if (process.env.NODE_ENV === 'development') {
    console.group('Error Report')
    console.error('Error:', error)
    console.log('Context:', context)
    console.log('Report:', report)
    console.groupEnd()
    return
  }

  // 生产环境上报到服务器
  try {
    await fetch('/api/errors', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(report)
    })
  } catch (reportingError) {
    // 上报失败，至少在本地存储
    console.error('Failed to report error:', reportingError)
    storeErrorLocally(report)
  }
}

/**
 * 获取用户 ID
 */
function getUserId(): string | null {
  try {
    const userData = localStorage.getItem('user')
    if (userData) {
      const user = JSON.parse(userData)
      return user.id || null
    }
  } catch {
    return null
  }
  return null
}

/**
 * 本地存储错误（上报失败时的降级方案）
 */
function storeErrorLocally(report: ErrorReport): void {
  try {
    const errors = JSON.parse(localStorage.getItem('error_reports') || '[]')
    errors.push({
      ...report,
      storedAt: new Date().toISOString()
    })

    // 只保留最近 50 条
    if (errors.length > 50) {
      errors.splice(0, errors.length - 50)
    }

    localStorage.setItem('error_reports', JSON.stringify(errors))
  } catch {
    // 静默失败
  }
}

/**
 * 获取本地存储的错误报告
 */
export function getLocalErrorReports(): ErrorReport[] {
  try {
    return JSON.parse(localStorage.getItem('error_reports') || '[]')
  } catch {
    return []
  }
}

/**
 * 清除本地错误报告
 */
export function clearLocalErrorReports(): void {
  localStorage.removeItem('error_reports')
}
```

### 使用示例

```tsx
// frontend/src/App.tsx
import { ErrorBoundary, AsyncErrorBoundary } from '@/components/error-boundary'
import { BrowserRouter } from 'react-router-dom'

export function App() {
  return (
    <AsyncErrorBoundary>
      <ErrorBoundary>
        <BrowserRouter>
          {/* 应用路由 */}
        </BrowserRouter>
      </ErrorBoundary>
    </AsyncErrorBoundary>
  )
}
```

```tsx
// frontend/src/pages/GamePage.tsx
import { ErrorBoundary } from '@/components/error-boundary'
import { ComponentErrorBoundary } from '@/components/error-boundary/ComponentErrorBoundary'

export function GamePage() {
  return (
    <div className="game-page">
      <ErrorBoundary
        fallback={
          <div>
            游戏界面加载失败，请刷新页面重试
          </div>
        }
      >
        <GameHeader />

        <ComponentErrorBoundary fallbackMessage="聊天功能暂时不可用">
          <ChatPanel />
        </ComponentErrorBoundary>

        <ComponentErrorBoundary fallbackMessage="角色信息暂时无法显示">
          <CharacterPanel />
        </ComponentErrorBoundary>

        <ComponentErrorBoundary fallbackMessage="物品栏暂时无法显示">
          <InventoryPanel />
        </ComponentErrorBoundary>
      </ErrorBoundary>
    </div>
  )
}
```

```tsx
// frontend/src/hooks/useGameData.ts
import { useState, useEffect } from 'react'
import { useAsyncError } from '@/hooks/useErrorHandler'

export function useGameData(gameId: string) {
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)
  const handleError = useAsyncError()

  useEffect(() => {
    fetchGameData()
      .then(setData)
      .catch((error) => {
        // 错误会被 AsyncErrorBoundary 捕获
        handleError(error)
      })
      .finally(() => {
        setLoading(false)
      })
  }, [gameId, handleError])

  return { data, loading }
}
```

### 性能监控错误边界

```tsx
// frontend/src/components/error-boundary/PerformanceErrorBoundary.tsx
import React, { Component, ReactNode } from 'react'

interface Props {
  children: ReactNode
  threshold?: number  // 渲染时间阈值（毫秒）
}

/**
 * 性能监控错误边界
 * 检测渲染性能问题
 */
export class PerformanceErrorBoundary extends Component<Props> {
  componentDidMount() {
    // 监控长任务
    this.observeLongTasks()

    // 监控内存使用（如果支持）
    this.observeMemory()
  }

  observeLongTasks() {
    if ('PerformanceObserver' in window) {
      try {
        const observer = new PerformanceObserver((list) => {
          for (const entry of list.getEntries()) {
            if (entry.duration > (this.props.threshold || 50)) {
              reportError(new Error('Long task detected'), {
                type: 'performance',
                duration: entry.duration,
                name: entry.name,
                startTime: entry.startTime,
                level: 'warning'
              })
            }
          }
        })

        observer.observe({ entryTypes: ['measure', 'longtask'] })
      } catch {
        // PerformanceObserver 不支持 longtask
      }
    }
  }

  observeMemory() {
    if ('memory' in performance) {
      setInterval(() => {
        const mem = (performance as any).memory
        if (mem.usedJSHeapSize > mem.jsHeapSizeLimit * 0.9) {
          reportError(new Error('High memory usage'), {
            type: 'performance',
            memory: {
              used: mem.usedJSHeapSize,
              total: mem.totalJSHeapSize,
              limit: mem.jsHeapSizeLimit
            },
            level: 'warning'
          })
        }
      }, 30000)  // 每 30 秒检查一次
    }
  }

  render() {
    return this.props.children
  }
}
```

---

## 后端实现

### 错误日志存储

```python
# backend/models/error_log.py
from datetime import datetime
from sqlalchemy import Column, Integer, String, Text, DateTime, JSON, Index

from database import Base


class ErrorLog(Base):
    """错误日志表"""
    __tablename__ = "error_logs"

    id = Column(Integer, primary_key=True, index=True)

    # 错误信息
    message = Column(String, nullable=False)
    stack = Column(Text)
    name = Column(String)

    # 上下文
    context = Column(JSON)

    # 请求信息
    user_agent = Column(String)
    url = Column(String)
    user_id = Column(Integer, index=True)

    # 时间戳
    timestamp = Column(DateTime, default=datetime.utcnow, index=True)

    # 索引
    __table_args__ = (
        Index('idx_error_logs_timestamp_user', 'timestamp', 'user_id'),
        Index('idx_error_logs_name', 'name'),
    )
```

### 错误统计 API

```python
# backend/api/routes/errors.py
from fastapi import APIRouter, Depends
from sqlalchemy.orm import Session
from sqlalchemy import func

from database import get_db
from models.error_log import ErrorLog

router = APIRouter(prefix="/errors", tags=["errors"])


@router.post("")
async def create_error_log(
    data: dict,
    db: Session = Depends(get_db)
):
    """记录前端错误"""
    log = ErrorLog(**data)
    db.add(log)
    db.commit()
    return {"success": True}


@router.get("/stats")
async def get_error_stats(
    days: int = 7,
    db: Session = Depends(get_db)
):
    """获取错误统计"""
    from datetime import timedelta

    cutoff = datetime.utcnow() - timedelta(days=days)

    total = db.query(func.count(ErrorLog.id)).filter(
        ErrorLog.timestamp >= cutoff
    ).scalar()

    by_name = db.query(
        ErrorLog.name,
        func.count(ErrorLog.id).label('count')
    ).filter(
        ErrorLog.timestamp >= cutoff
    ).group_by(ErrorLog.name).order_by(
        func.count(ErrorLog.id).desc()
    ).limit(10).all()

    return {
        "total": total,
        "by_name": [{"name": name, "count": count} for name, count in by_name]
    }
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `frontend/src/components/error-boundary/ErrorBoundary.tsx` | 创建 | 全局错误边界 |
| `frontend/src/components/error-boundary/ComponentErrorBoundary.tsx` | 创建 | 组件错误边界 |
| `frontend/src/components/error-boundary/AsyncErrorBoundary.tsx` | 创建 | 异步错误边界 |
| `frontend/src/components/error-boundary/PerformanceErrorBoundary.tsx` | 创建 | 性能监控边界 |
| `frontend/src/hooks/useErrorHandler.ts` | 创建 | 错误处理 Hooks |
| `frontend/src/lib/error-reporting.ts` | 创建 | 错误上报服务 |
| `backend/models/error_log.py` | 创建 | 错误日志模型 |
| `backend/api/routes/errors.py` | 创建 | 错误 API |

---

## 验收标准

- [ ] 错误边界正确捕获组件错误
- [ ] 提供友好的错误 UI
- [ ] 支持错误重试和恢复
- [ ] 错误日志正确上报
- [ ] 不影响正常功能性能
- [ ] 组件级错误隔离工作正常
- [ ] 异步错误被正确捕获
- [ ] 开发环境显示详细错误信息

---

## 参考文档

- React Error Boundary 文档
- Sentry 错误监控最佳实践
- M6-009: 反馈收集系统

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
