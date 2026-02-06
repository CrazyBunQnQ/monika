# M6-015: 实现性能监控

**任务ID**: M6-015
**标题**: 实现性能监控
**类型**: fullstack (全栈开发)
**预估工时**: 7h
**依赖**: M1-040

---

## 任务描述

实现前后端性能监控系统，收集关键性能指标，分析瓶颈，提供优化建议，确保应用流畅运行。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-015-01 | 设计性能指标体系 | KPI 定义 | 45min |
| M6-015-02 | 实现前端性能监控 | Core Web Vitals | 1.5h |
| M6-015-03 | 实现后端性能监控 | API 响应时间 | 1.5h |
| M6-015-04 | 实现用户行为追踪 | 交互分析 | 1h |
| M6-015-05 | 实现性能分析面板 | 数据可视化 | 1.5h |
| M6-015-06 | 实现性能告警机制 | 阈值通知 | 45min |
| M6-015-07 | 实现性能报告生成 | 定期报告 | 45min |

---

## 前端实现

### 性能监控 SDK

```typescript
// frontend/src/lib/performance/monitor.ts
interface PerformanceMetric {
  name: string
  value: number
  rating: 'good' | 'needs-improvement' | 'poor'
  timestamp: number
}

interface NavigationTiming {
  // 网络相关
  dnsLookup: number
  tcpConnection: number
  tlsHandshake: number
  requestTime: number
  responseTime: number

  // 资源加载
  domLoading: number
  domInteractive: number
  domComplete: number
  loadEvent: number

  // 关键指标
  ttfb: number  // Time to First Byte
  domContentLoaded: number
  windowLoad: number
}

interface WebVitals {
  // Core Web Vitals
  LCP: number  // Largest Contentful Paint
  FID: number  // First Input Delay
  CLS: number  // Cumulative Layout Shift

  // 其他重要指标
  FCP: number  // First Contentful Paint
  TTI: number  // Time to Interactive
  TBT: number  // Total Blocking Time
  SI: number   // Speed Index
}

export class PerformanceMonitor {
  private metrics: PerformanceMetric[] = []
  private observers: PerformanceObserver[] = []
  private sessionId: string

  constructor() {
    this.sessionId = this.generateSessionId()
    this.init()
  }

  private generateSessionId(): string {
    return `perf_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
  }

  private async init() {
    // 等待页面加载完成
    if (document.readyState === 'loading') {
      await new Promise(resolve => {
        window.addEventListener('load', resolve, { once: true })
      })
    }

    // 收集导航时序
    this.collectNavigationTiming()

    // 观察 Core Web Vitals
    this.observeWebVitals()

    // 观察资源加载
    this.observeResources()

    // 观察长任务
    this.observeLongTasks()

    // 定期上报
    this.startReporting()
  }

  /**
   * 收集导航时序
   */
  private collectNavigationTiming() {
    const navigation = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming

    if (!navigation) return

    const timing: NavigationTiming = {
      dnsLookup: navigation.domainLookupEnd - navigation.domainLookupStart,
      tcpConnection: navigation.connectEnd - navigation.connectStart,
      tlsHandshake: navigation.secureConnectionStart > 0
        ? navigation.connectEnd - navigation.secureConnectionStart
        : 0,
      requestTime: navigation.responseStart - navigation.requestStart,
      responseTime: navigation.responseEnd - navigation.responseStart,
      domLoading: navigation.domInteractive - navigation.responseEnd,
      domInteractive: navigation.domInteractive - navigation.fetchStart,
      domComplete: navigation.domComplete - navigation.fetchStart,
      loadEvent: navigation.loadEventEnd - navigation.loadEventStart,
      ttfb: navigation.responseStart - navigation.fetchStart,
      domContentLoaded: navigation.domContentLoadedEventEnd - navigation.fetchStart,
      windowLoad: navigation.loadEventEnd - navigation.fetchStart
    }

    this.recordMetric('navigation_timing', timing)
  }

  /**
   * 观察 Web Vitals
   */
  private observeWebVitals() {
    // LCP
    this.observeEntry('largest-contentful-paint', (entries) => {
      const lastEntry = entries[entries.length - 1]
      const lcp = lastEntry.startTime
      const rating = this.rateLCP(lcp)
      this.recordMetric('LCP', lcp, rating)
    })

    // FID
    this.observeEntry('first-input', (entries) => {
      const fid = entries[0].processingStart - entries[0].startTime
      const rating = this.rateFID(fid)
      this.recordMetric('FID', fid, rating)
    })

    // CLS
    let clsValue = 0
    this.observeEntry('layout-shift', (entries) => {
      for (const entry of entries) {
        if (!entry.hadRecentInput) {
          clsValue += entry.value
        }
      }
      const rating = this.rateCLS(clsValue)
      this.recordMetric('CLS', clsValue, rating)
    }, true)
  }

  /**
   * 观察资源加载
   */
  private observeResources() {
    const resources = performance.getEntriesByType('resource') as PerformanceResourceTiming[]

    const byType = resources.reduce((acc, resource) => {
      const type = this.getResourceType(resource.name)
      if (!acc[type]) {
        acc[type] = { count: 0, totalDuration: 0, totalSize: 0 }
      }
      acc[type].count++
      acc[type].totalDuration += resource.duration - resource.redirectStart - resource.fetchStart
      acc[type].totalSize += resource.transferSize || 0
      return acc
    }, {} as Record<string, { count: number; totalDuration: number; totalSize: number }>)

    this.recordMetric('resources', byType)
  }

  /**
   * 观察长任务
   */
  private observeLongTasks() {
    try {
      const observer = new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) {
          this.recordMetric('long_task', {
            duration: entry.duration,
            startTime: entry.startTime,
            name: entry.name
          })
        }
      })

      observer.observe({ entryTypes: ['longtask'] })
      this.observers.push(observer)
    } catch {
      // Long tasks not supported
    }
  }

  /**
   * 通用观察方法
   */
  private observeEntry(
    type: string,
    callback: (entries: any[]) => void,
    buffered: boolean = false
  ) {
    try {
      const observer = new PerformanceObserver((list) => {
        callback(list.getEntries())
      })

      observer.observe({ type, buffered })
      this.observers.push(observer)
    } catch (error) {
      console.warn(`Failed to observe ${type}:`, error)
    }
  }

  /**
   * 记录指标
   */
  private recordMetric(name: string, value: any, rating?: 'good' | 'needs-improvement' | 'poor') {
    const metric: PerformanceMetric = {
      name,
      value: typeof value === 'number' ? value : JSON.stringify(value),
      rating: rating || 'good',
      timestamp: Date.now()
    }

    this.metrics.push(metric)

    // 发送到分析服务
    this.sendToAnalytics(metric)
  }

  /**
   * 发送到分析服务
   */
  private async sendToAnalytics(metric: PerformanceMetric) {
    try {
      await fetch('/api/analytics/performance', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          session_id: this.sessionId,
          url: window.location.href,
          user_agent: navigator.userAgent,
          ...metric
        })
      })
    } catch (error) {
      // 静默失败
    }
  }

  /**
   * 开始定期上报
   */
  private startReporting() {
    // 每 30 秒汇总上报一次
    setInterval(() => {
      this.flushMetrics()
    }, 30000)

    // 页面卸载时上报
    window.addEventListener('beforeunload', () => {
      this.flushMetrics()
    })
  }

  /**
   * 汇总上报所有指标
   */
  private flushMetrics() {
    if (this.metrics.length === 0) return

    // 使用 sendBeacon 进行最后上报
    if (navigator.sendBeacon) {
      const data = new Blob(
        [JSON.stringify({
          session_id: this.sessionId,
          metrics: this.metrics
        })],
        { type: 'application/json' }
      )
      navigator.sendBeacon('/api/analytics/performance/batch', data)
    }

    this.metrics = []
  }

  /**
   * 资源类型判断
   */
  private getResourceType(url: string): string {
    if (url.endsWith('.js')) return 'script'
    if (url.endsWith('.css')) return 'stylesheet'
    if (url.match(/\.(png|jpg|jpeg|gif|svg|webp)$/)) return 'image'
    if (url.endsWith('.woff') || url.endsWith('.woff2')) return 'font'
    if (url.includes('/api/')) return 'api'
    return 'other'
  }

  /**
   * 评级 LCP
   */
  private rateLCP(lcp: number): 'good' | 'needs-improvement' | 'poor' {
    if (lcp < 2500) return 'good'
    if (lcp < 4000) return 'needs-improvement'
    return 'poor'
  }

  /**
   * 评级 FID
   */
  private rateFID(fid: number): 'good' | 'needs-improvement' | 'poor' {
    if (fid < 100) return 'good'
    if (fid < 300) return 'needs-improvement'
    return 'poor'
  }

  /**
   * 评级 CLS
   */
  private rateCLS(cls: number): 'good' | 'needs-improvement' | 'poor' {
    if (cls < 0.1) return 'good'
    if (cls < 0.25) return 'needs-improvement'
    return 'poor'
  }

  /**
   * 手动标记性能事件
   */
  mark(name: string) {
    performance.mark(name)
  }

  /**
   * 测量两个标记之间的时间
   */
  measure(name: string, startMark: string, endMark: string) {
    try {
      performance.measure(name, startMark, endMark)
      const measure = performance.getEntriesByName(name)[0]
      this.recordMetric(name, measure.duration)
    } catch (error) {
      console.warn('Failed to measure:', error)
    }
  }

  /**
   * 获取当前指标摘要
   */
  getSummary() {
    return {
      sessionId: this.sessionId,
      metrics: this.metrics,
      url: window.location.href,
      userAgent: navigator.userAgent
    }
  }

  /**
   * 清理
   */
  destroy() {
    this.observers.forEach(observer => observer.disconnect())
    this.observers = []
  }
}

// 导出单例
export const performanceMonitor = new PerformanceMonitor()
```

### React 集成 Hook

```typescript
// frontend/src/hooks/usePerformanceMonitor.ts
import { useEffect } from 'react'
import { performanceMonitor } from '@/lib/performance/monitor'

export function usePerformanceMonitor(componentName: string) {
  useEffect(() => {
    const mountMark = `${componentName}_mount`
    const renderMark = `${componentName}_render`

    performanceMonitor.mark(mountMark)

    return () => {
      const unmountMark = `${componentName}_unmount`
      performanceMonitor.mark(unmountMark)
      performanceMonitor.measure(
        `${componentName}_lifecycle`,
        mountMark,
        unmountMark
      )
    }
  }, [componentName])

  const measureRender = (callback: () => void) => {
    const startMark = `${componentName}_render_start`
    const endMark = `${componentName}_render_end`

    performanceMonitor.mark(startMark)
    callback()
    requestAnimationFrame(() => {
      performanceMonitor.mark(endMark)
      performanceMonitor.measure(`${componentName}_render`, startMark, endMark)
    })
  }

  return {
    monitor: performanceMonitor,
    measureRender
  }
}
```

---

## 后端实现

### 性能监控中间件

```python
# backend/middleware/performance.py
import time
from typing import Callable
from fastapi import Request, Response
from starlette.middleware.base import BaseHTTPMiddleware
import logging

logger = logging.getLogger(__name__)


class PerformanceMiddleware(BaseHTTPMiddleware):
    """性能监控中间件"""

    async def dispatch(self, request: Request, call_next: Callable) -> Response:
        # 记录开始时间
        start_time = time.perf_counter()

        # 处理请求
        response = await call_next(request)

        # 计算处理时间
        process_time = time.perf_counter() - start_time

        # 添加响应头
        response.headers["X-Process-Time"] = str(process_time)

        # 记录慢请求
        if process_time > 1.0:  # 超过1秒
            logger.warning(
                f"Slow request: {request.method} {request.url.path} "
                f"took {process_time:.3f}s"
            )

        # 存储性能指标
        await self.record_metrics(request, response, process_time)

        return response

    async def record_metrics(
        self,
        request: Request,
        response: Response,
        process_time: float
    ):
        """记录性能指标"""
        from utils.performance_store import PerformanceStore

        metrics = {
            "method": request.method,
            "path": request.url.path,
            "status_code": response.status_code,
            "process_time": process_time,
            "timestamp": time.time()
        }

        PerformanceStore.record(metrics)
```

### 性能数据存储

```python
# backend/utils/performance_store.py
from collections import deque
from datetime import datetime, timedelta
from typing import Dict, List, Any
import threading

class PerformanceStore:
    """内存中的性能数据存储"""

    # 存储最近10000条记录
    _metrics: deque = deque(maxlen=10000)
    _lock = threading.Lock()

    @classmethod
    def record(cls, metrics: Dict[str, Any]):
        """记录性能指标"""
        with cls._lock:
            cls._metrics.append({
                **metrics,
                "recorded_at": datetime.utcnow()
            })

    @classmethod
    def get_metrics(
        cls,
        path: str = None,
        status_code: int = None,
        minutes: int = 60
    ) -> List[Dict[str, Any]]:
        """获取性能指标"""
        cutoff = datetime.utcnow() - timedelta(minutes=minutes)

        with cls._lock:
            metrics = list(cls._metrics)

        # 过滤
        if path:
            metrics = [m for m in metrics if m["path"] == path]
        if status_code:
            metrics = [m for m in metrics if m["status_code"] == status_code]
        metrics = [m for m in metrics if m["recorded_at"] >= cutoff]

        return metrics

    @classmethod
    def get_summary(cls, minutes: int = 60) -> Dict[str, Any]:
        """获取性能摘要"""
        metrics = cls.get_metrics(minutes=minutes)

        if not metrics:
            return {}

        total_requests = len(metrics)
        total_time = sum(m["process_time"] for m in metrics)
        avg_time = total_time / total_requests

        # 按路径分组
        by_path: Dict[str, List[float]] = {}
        for m in metrics:
            if m["path"] not in by_path:
                by_path[m["path"]] = []
            by_path[m["path"]].append(m["process_time"])

        path_stats = {}
        for path, times in by_path.items():
            path_stats[path] = {
                "count": len(times),
                "avg": sum(times) / len(times),
                "min": min(times),
                "max": max(times),
                "p95": sorted(times)[int(len(times) * 0.95)] if len(times) > 0 else 0
            }

        # 状态码分布
        status_codes: Dict[int, int] = {}
        for m in metrics:
            status_codes[m["status_code"]] = status_codes.get(m["status_code"], 0) + 1

        return {
            "total_requests": total_requests,
            "avg_response_time": avg_time,
            "requests_per_minute": total_requests / minutes,
            "path_stats": path_stats,
            "status_codes": status_codes,
            "slowest_paths": sorted(
                path_stats.items(),
                key=lambda x: x[1]["avg"],
                reverse=True
            )[:10]
        }
```

### 性监控 API

```python
# backend/api/routes/performance.py
from fastapi import APIRouter, Depends
from sqlalchemy.orm import Session

from database import get_db
from utils.performance_store import PerformanceStore
from middleware.auth import get_current_user

router = APIRouter(prefix="/performance", tags=["performance"])


@router.get("/summary")
async def get_performance_summary(
    minutes: int = 60,
    current_user = Depends(get_current_user)
):
    """获取性能摘要（管理员）"""
    if current_user.role != "admin":
        raise HTTPException(status_code=403, detail="Admin only")

    return PerformanceStore.get_summary(minutes)


@router.get("/paths")
async def get_path_performance(
    minutes: int = 60,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_user)
):
    """获取各路径性能（管理员）"""
    if current_user.role != "admin":
        raise HTTPException(status_code=403, detail="Admin only")

    summary = PerformanceStore.get_summary(minutes)
    return summary.get("path_stats", {})


@router.get("/slow-requests")
async def get_slow_requests(
    threshold: float = 1.0,
    minutes: int = 60,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_user)
):
    """获取慢请求列表（管理员）"""
    if current_user.role != "admin":
        raise HTTPException(status_code=403, detail="Admin only")

    metrics = PerformanceStore.get_metrics(minutes=minutes)
    slow_requests = [
        m for m in metrics
        if m["process_time"] > threshold
    ]

    return sorted(
        slow_requests,
        key=lambda x: x["process_time"],
        reverse=True
    )[:100]
```

---

## 性能监控面板

```tsx
// frontend/src/components/performance/PerformanceDashboard.tsx
import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { Activity, Clock, AlertTriangle, CheckCircle } from 'lucide-react'

export function PerformanceDashboard() {
  const [metrics, setMetrics] = useState<any>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchMetrics()
    const interval = setInterval(fetchMetrics, 30000)
    return () => clearInterval(interval)
  }, [])

  const fetchMetrics = async () => {
    try {
      const response = await fetch('/api/performance/summary')
      if (response.ok) {
        const data = await response.json()
        setMetrics(data)
      }
    } finally {
      setLoading(false)
    }
  }

  if (loading) return <div>加载中...</div>

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold">性能监控</h2>
        <Badge variant="outline">实时</Badge>
      </div>

      {/* 关键指标 */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <MetricCard
          title="总请求数"
          value={metrics?.total_requests || 0}
          icon={Activity}
        />
        <MetricCard
          title="平均响应"
          value={`${(metrics?.avg_response_time * 1000).toFixed(0)}ms`}
          icon={Clock}
        />
        <MetricCard
          title="QPS"
          value={metrics?.requests_per_minute?.toFixed(1) || 0}
          icon={Activity}
          suffix="/min"
        />
        <MetricCard
          title="慢请求"
          value={metrics?.slow_requests || 0}
          icon={AlertTriangle}
          variant="warning"
        />
      </div>

      {/* 路径性能 */}
      <Card>
        <CardHeader>
          <CardTitle>API 性能</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {metrics?.slowest_paths?.map(([path, stats]) => (
              <div key={path} className="space-y-2">
                <div className="flex items-center justify-between text-sm">
                  <code className="text-xs">{path}</code>
                  <span className="text-muted-foreground">
                    {stats.avg.toFixed(0)}ms (P95: {stats.p95.toFixed(0)}ms)
                  </span>
                </div>
                <Progress
                  value={Math.min((stats.avg / 1000) * 100, 100)}
                  className={stats.avg > 500 ? "bg-red-500" : ""}
                />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

function MetricCard({
  title,
  value,
  icon: Icon,
  suffix = '',
  variant = 'default'
}: any) {
  return (
    <Card>
      <CardContent className="p-4">
        <div className="flex items-center space-x-3">
          <Icon className={`h-8 w-8 ${
            variant === 'warning' ? 'text-yellow-500' : 'text-purple-500'
          }`} />
          <div>
            <div className="text-sm text-muted-foreground">{title}</div>
            <div className="text-2xl font-bold">
              {value}{suffix}
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `frontend/src/lib/performance/monitor.ts` | 创建 | 前端性能监控 SDK |
| `frontend/src/hooks/usePerformanceMonitor.ts` | 创建 | React 集成 Hook |
| `backend/middleware/performance.py` | 创建 | 后端性能中间件 |
| `backend/utils/performance_store.py` | 创建 | 性能数据存储 |
| `backend/api/routes/performance.py` | 创建 | 性监控 API |
| `frontend/src/components/performance/PerformanceDashboard.tsx` | 创建 | 性监控面板 |

---

## 验收标准

- [ ] Core Web Vitals 正确测量
- [ ] API 响应时间准确记录
- [ ] 慢请求正确识别和告警
- [ ] 性能面板数据实时更新
- [ ] 支持历史数据查询
- [ ] 监控对性能影响可忽略
- [ ] 移动端监控正常工作

---

## 参考文档

- Web Vitals 官方文档
- Performance API 文档
- M6-011: 错误边界
- M6-014: 数据可视化

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
