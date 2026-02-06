# M6-013: 实现渐进式 Web 应用

**任务ID**: M6-013
**标题**: 实现渐进式 Web 应用
**类型**: frontend (前端开发)
**预估工时**: 8h
**依赖**: M6-012

---

## 任务描述

实现完整的 PWA 功能，包括 Web App Manifest、安装提示、应用图标、启动画面等，提供类似原生应用的体验。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-013-01 | 创建 Web App Manifest | 应用清单文件 | 1h |
| M6-013-02 | 设计应用图标 | 多尺寸图标生成 | 1.5h |
| M6-013-03 | 实现安装提示 UI | 安装横幅/按钮 | 1.5h |
| M6-013-04 | 实现启动画面 | Splash Screen | 1h |
| M6-013-05 | 优化主题颜色 | 主题色配置 | 30min |
| M6-013-06 | 实现应用更新机制 | 更新提示和跳过 | 1h |
| M6-013-07 | 实现应用快捷方式 | Quick Links | 45min |
| M6-013-08 | 配置 PWA 元标签 | SEO 和展示 | 30min |

---

## 前端实现

### Web App Manifest

```json
// frontend/public/manifest.json
{
  "name": "CoC 跑团平台",
  "short_name": "CoC TRPG",
  "description": "基于克苏鲁的呼唤 7 版的在线跑团平台",
  "start_url": "/",
  "scope": "/",
  "display": "standalone",
  "orientation": "portrait-primary",
  "background_color": "#0a0a0a",
  "theme_color": "#8b5cf6",
  "lang": "zh-CN",
  "dir": "ltr",

  "icons": [
    {
      "src": "/icons/icon-72x72.png",
      "sizes": "72x72",
      "type": "image/png",
      "purpose": "any"
    },
    {
      "src": "/icons/icon-96x96.png",
      "sizes": "96x96",
      "type": "image/png",
      "purpose": "any"
    },
    {
      "src": "/icons/icon-128x128.png",
      "sizes": "128x128",
      "type": "image/png",
      "purpose": "any"
    },
    {
      "src": "/icons/icon-144x144.png",
      "sizes": "144x144",
      "type": "image/png",
      "purpose": "any"
    },
    {
      "src": "/icons/icon-152x152.png",
      "sizes": "152x152",
      "type": "image/png",
      "purpose": "any"
    },
    {
      "src": "/icons/icon-192x192.png",
      "sizes": "192x192",
      "type": "image/png",
      "purpose": "any"
    },
    {
      "src": "/icons/icon-384x384.png",
      "sizes": "384x384",
      "type": "image/png",
      "purpose": "any"
    },
    {
      "src": "/icons/icon-512x512.png",
      "sizes": "512x512",
      "type": "image/png",
      "purpose": "any"
    },
    {
      "src": "/icons/maskable-icon-512x512.png",
      "sizes": "512x512",
      "type": "image/png",
      "purpose": "maskable"
    }
  ],

  "screenshots": [
    {
      "src": "/screenshots/mobile1.png",
      "sizes": "540x720",
      "type": "image/png",
      "form_factor": "narrow"
    },
    {
      "src": "/screenshots/desktop1.png",
      "sizes": "1280x720",
      "type": "image/png",
      "form_factor": "wide"
    }
  ],

  "shortcuts": [
    {
      "name": "新建游戏",
      "short_name": "新建",
      "description": "创建新的跑团游戏",
      "url": "/game/new",
      "icons": [
        {
          "src": "/icons/shortcut-new.png",
          "sizes": "96x96"
        }
      ]
    },
    {
      "name": "我的角色",
      "short_name": "角色",
      "description": "查看我的角色卡",
      "url": "/characters",
      "icons": [
        {
          "src": "/icons/shortcut-character.png",
          "sizes": "96x96"
        }
      ]
    },
    {
      "name": "快速投骰",
      "short_name": "投骰",
      "description": "快速进行骰子检定",
      "url": "/roll",
      "icons": [
        {
          "src": "/icons/shortcut-roll.png",
          "sizes": "96x96"
        }
      ]
    },
    {
      "name": "规则查询",
      "short_name": "规则",
      "description": "查询 CoC 规则",
      "url": "/rules",
      "icons": [
        {
          "src": "/icons/shortcut-rules.png",
          "sizes": "96x96"
        }
      ]
    }
  ],

  "categories": ["games", "entertainment"],
  "prefer_related_applications": false,

  "related_applications": [],
  "features": [
    "Cross-origin-Opener-Policy: same-origin",
    "Cross-Origin-Embedder-Policy: require-corp"
  ]
}
```

### PWA 安装 Hook

```typescript
// frontend/src/hooks/usePWAInstall.ts
import { useEffect, useState, useCallback } from 'react'

interface InstallPromptEvent extends Event {
  prompt: () => Promise<void>
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>
}

export interface PWAInstallState {
  canInstall: boolean
  isInstalled: boolean
  isInstalling: boolean
  install: () => Promise<void>
  dismiss: () => void
}

export function usePWAInstall(): PWAInstallState {
  const [deferredPrompt, setDeferredPrompt] = useState<InstallPromptEvent | null>(null)
  const [isInstalled, setIsInstalled] = useState(false)
  const [isInstalling, setIsInstalling] = useState(false)
  const [dismissed, setDismissed] = useState(false)

  useEffect(() => {
    // 检查是否已安装
    const checkInstalled = () => {
      const isStandalone = window.matchMedia('(display-mode: standalone)').matches
      setIsInstalled(isStandalone || (navigator as any).standalone === true)
    }

    checkInstalled()

    // 监听安装提示事件
    const handleBeforeInstallPrompt = (e: Event) => {
      e.preventDefault()
      setDeferredPrompt(e as InstallPromptEvent)
    }

    // 监听应用安装事件
    const handleAppInstalled = () => {
      setIsInstalled(true)
      setDeferredPrompt(null)
    }

    window.addEventListener('beforeinstallprompt', handleBeforeInstallPrompt)
    window.addEventListener('appinstalled', handleAppInstalled)

    return () => {
      window.removeEventListener('beforeinstallprompt', handleBeforeInstallPrompt)
      window.removeEventListener('appinstalled', handleAppInstalled)
    }
  }, [])

  const install = useCallback(async () => {
    if (!deferredPrompt || isInstalled) {
      return
    }

    setIsInstalling(true)

    try {
      await deferredPrompt.prompt()
      const { outcome } = await deferredPrompt.userChoice

      if (outcome === 'accepted') {
        setDeferredPrompt(null)
      }
    } catch (error) {
      console.error('Install prompt failed:', error)
    } finally {
      setIsInstalling(false)
    }
  }, [deferredPrompt, isInstalled])

  const dismiss = useCallback(() => {
    setDeferredPrompt(null)
    setDismissed(true)
    // 记录用户选择，7天内不再显示
    localStorage.setItem('pwa-install-dismissed', Date.now().toString())
  }, [])

  const canInstall = deferredPrompt !== null && !isInstalled && !dismissed

  return {
    canInstall,
    isInstalled,
    isInstalling,
    install,
    dismiss
  }
}
```

### 安装提示组件

```tsx
// frontend/src/components/pwa/InstallPrompt.tsx
import { usePWAInstall } from '@/hooks/usePWAInstall'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { X, Download, Rocket } from 'lucide-react'
import { useEffect, useState } from 'react'

export function InstallPrompt() {
  const { canInstall, isInstalling, install, dismiss } = usePWAInstall()
  const [show, setShow] = useState(false)

  useEffect(() => {
    if (canInstall) {
      // 检查是否在7天内被用户关闭过
      const dismissed = localStorage.getItem('pwa-install-dismissed')
      if (!dismissed || Date.now() - parseInt(dismissed) > 7 * 24 * 60 * 60 * 1000) {
        // 延迟显示，不打扰用户
        const timer = setTimeout(() => setShow(true), 30000)
        return () => clearTimeout(timer)
      }
    }
  }, [canInstall])

  if (!show || !canInstall) {
    return null
  }

  return (
    <Card className="fixed bottom-4 left-4 right-4 md:left-auto md:right-4 md:w-96 z-50 animate-slide-up">
      <CardContent className="p-4">
        <button
          onClick={dismiss}
          className="absolute top-2 right-2 text-muted-foreground hover:text-foreground"
        >
          <X className="h-4 w-4" />
        </button>

        <div className="flex items-start space-x-4">
          <div className="h-12 w-12 rounded-xl bg-gradient-to-br from-purple-500 to-pink-500 flex items-center justify-center flex-shrink-0">
            <Rocket className="h-6 w-6 text-white" />
          </div>

          <div className="flex-1 space-y-2">
            <div>
              <h3 className="font-semibold">安装 CoC 跑团平台</h3>
              <p className="text-sm text-muted-foreground">
                安装到桌面，获得更好的体验
              </p>
            </div>

            <div className="flex gap-2">
              <Button
                size="sm"
                onClick={install}
                disabled={isInstalling}
                className="flex-1"
              >
                <Download className="h-4 w-4 mr-2" />
                {isInstalling ? '安装中...' : '立即安装'}
              </Button>
              <Button
                size="sm"
                variant="ghost"
                onClick={dismiss}
              >
                暂不
              </Button>
            </div>

            <div className="text-xs text-muted-foreground space-y-1">
              <div>• 离线可用，随时随地游戏</div>
              <div>• 更快的启动速度</div>
              <div>• 原生应用般的体验</div>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
```

### 启动画面

```html
<!-- frontend/public/splash.html -->
<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>CoC 跑团平台</title>
  <style>
    * {
      margin: 0;
      padding: 0;
      box-sizing: border-box;
    }

    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
      background: linear-gradient(135deg, #0a0a0a 0%, #1a1a2e 100%);
      display: flex;
      align-items: center;
      justify-content: center;
      min-height: 100vh;
      color: white;
    }

    .splash-container {
      text-align: center;
      animation: fadeIn 0.5s ease-in;
    }

    @keyframes fadeIn {
      from { opacity: 0; transform: scale(0.9); }
      to { opacity: 1; transform: scale(1); }
    }

    .logo {
      width: 120px;
      height: 120px;
      margin: 0 auto 24px;
      background: linear-gradient(135deg, #8b5cf6 0%, #ec4899 100%);
      border-radius: 32px;
      display: flex;
      align-items: center;
      justify-content: center;
      box-shadow: 0 20px 40px rgba(139, 92, 246, 0.3);
      animation: pulse 2s infinite;
    }

    @keyframes pulse {
      0%, 100% { transform: scale(1); }
      50% { transform: scale(1.05); }
    }

    .logo svg {
      width: 64px;
      height: 64px;
    }

    h1 {
      font-size: 24px;
      font-weight: 700;
      margin-bottom: 8px;
      background: linear-gradient(135deg, #8b5cf6 0%, #ec4899 100%);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      background-clip: text;
    }

    .subtitle {
      font-size: 14px;
      color: rgba(255, 255, 255, 0.6);
      margin-bottom: 32px;
    }

    .loader {
      width: 40px;
      height: 40px;
      margin: 0 auto;
      position: relative;
    }

    .loader::before,
    .loader::after {
      content: '';
      position: absolute;
      top: 0;
      left: 0;
      width: 100%;
      height: 100%;
      border-radius: 50%;
      border: 3px solid transparent;
    }

    .loader::before {
      border-top-color: #8b5cf6;
      animation: spin 1s linear infinite;
    }

    .loader::after {
      border-bottom-color: #ec4899;
      animation: spin 1s linear infinite reverse;
    }

    @keyframes spin {
      from { transform: rotate(0deg); }
      to { transform: rotate(360deg); }
    }

    .tips {
      margin-top: 32px;
      padding: 16px;
      background: rgba(255, 255, 255, 0.05);
      border-radius: 12px;
      max-width: 280px;
      margin-left: auto;
      margin-right: auto;
    }

    .tips-title {
      font-size: 12px;
      font-weight: 600;
      margin-bottom: 8px;
      color: rgba(255, 255, 255, 0.8);
    }

    .tips-text {
      font-size: 12px;
      color: rgba(255, 255, 255, 0.5);
      line-height: 1.5;
    }

    .version {
      position: fixed;
      bottom: 16px;
      font-size: 12px;
      color: rgba(255, 255, 255, 0.3);
    }
  </style>
</head>
<body>
  <div class="splash-container">
    <div class="logo">
      <svg viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2">
        <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/>
      </svg>
    </div>

    <h1>CoC 跑团平台</h1>
    <p class="subtitle">克苏鲁的呼唤 7 版</p>

    <div class="loader"></div>

    <div class="tips">
      <div class="tips-title">💡 提示</div>
      <p class="tips-text" id="tip-text">正在准备你的冒险...</p>
    </div>
  </div>

  <div class="version">v1.0.0</div>

  <script>
    const tips = [
      '在 CoC 跑团中，失败不是终点，而是新的开始',
      '保护你的理智值（SAN），它比生命值更重要',
      '团队合作是生存的关键',
      '不要相信所有 NPC，他们可能有秘密',
      '记录重要线索，它们会派上用场'
    ]

    document.getElementById('tip-text').textContent =
      tips[Math.floor(Math.random() * tips.length)]

    // 跳转到主页
    setTimeout(() => {
      window.location.href = '/'
    }, 2000)
  </script>
</body>
</html>
```

### 更新提示组件

```tsx
// frontend/src/components/pwa/UpdateBanner.tsx
import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Download, X, RefreshCw } from 'lucide-react'
import { skipWaiting } from '@/lib/service-worker/register'

export function UpdateBanner() {
  const [show, setShow] = useState(false)
  const [isUpdating, setIsUpdating] = useState(false)

  useEffect(() => {
    // 监听 SW 更新
    const messageHandler = (event: MessageEvent) => {
      if (event.data.type === 'SW_UPDATE_AVAILABLE') {
        setShow(true)
      }
    }

    navigator.serviceWorker.addEventListener('message', messageHandler)

    return () => {
      navigator.serviceWorker.removeEventListener('message', messageHandler)
    }
  }, [])

  const handleUpdate = async () => {
    setIsUpdating(true)
    await skipWaiting()
    // 页面会被 controllerchange 事件重新加载
  }

  const handleDismiss = () => {
    setShow(false)
    // 提醒用户下次访问时更新
    localStorage.setItem('update-pending', 'true')
  }

  if (!show) {
    return null
  }

  return (
    <Alert className="fixed top-0 left-0 right-0 z-50 rounded-none border-b bg-gradient-to-r from-purple-500/10 to-pink-500/10">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-3">
          <Download className="h-5 w-5 text-purple-500 animate-bounce" />
          <AlertDescription className="flex items-center space-x-2">
            <span className="font-semibold">新版本可用</span>
            <span className="text-muted-foreground">点击更新以获得最新功能和修复</span>
          </AlertDescription>
        </div>

        <div className="flex items-center space-x-2">
          <Button
            size="sm"
            onClick={handleUpdate}
            disabled={isUpdating}
          >
            {isUpdating ? (
              <>
                <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                更新中...
              </>
            ) : (
              <>
                <Download className="h-4 w-4 mr-2" />
                立即更新
              </>
            )}
          </Button>

          <Button
            size="sm"
            variant="ghost"
            onClick={handleDismiss}
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </Alert>
  )
}
```

### PWA 元标签配置

```tsx
// frontend/src/components/pwa/PWAMeta.tsx
import { Helmet } from 'react-helmet-async'

const MANIFEST_URL = '/manifest.json'
const THEME_COLOR = '#8b5cf6'
const BACKGROUND_COLOR = '#0a0a0a'

export function PWAMeta() {
  return (
    <Helmet>
      {/* PWA 核心 */}
      <link rel="manifest" href={MANIFEST_URL} />
      <meta name="theme-color" content={THEME_COLOR} />

      {/* iOS 支持 */}
      <meta name="apple-mobile-web-app-capable" content="yes" />
      <meta name="apple-mobile-web-app-status-bar-style" content="black-translucent" />
      <meta name="apple-mobile-web-app-title" content="CoC TRPG" />
      <link rel="apple-touch-icon" href="/icons/icon-152x152.png" />

      {/* Android Chrome */}
      <meta name="mobile-web-app-capable" content="yes" />
      <meta name="application-name" content="CoC TRPG" />

      {/* Windows Metro */}
      <meta name="msapplication-TileColor" content={THEME_COLOR} />
      <meta name="msapplication-TileImage" content="/icons/icon-144x144.png" />
      <meta name="msapplication-config" content="/browserconfig.xml" />

      {/* Favicons */}
      <link rel="icon" type="image/png" sizes="32x32" href="/icons/favicon-32x32.png" />
      <link rel="icon" type="image/png" sizes="16x16" href="/icons/favicon-16x16.png" />
      <link rel="shortcut icon" href="/favicon.ico" />

      {/* 启动画面 */}
      <link rel="apple-touch-startup-image" href="/splash.png" />

      {/* SEO */}
      <meta name="description" content="基于克苏鲁的呼唤 7 版的在线跑团平台" />
      <meta name="keywords" content="CoC,TRPG,跑团,克苏鲁,在线游戏" />
    </Helmet>
  )
}
```

### 应用集成

```tsx
// frontend/src/App.tsx
import { PWAMeta } from '@/components/pwa/PWAMeta'
import { InstallPrompt } from '@/components/pwa/InstallPrompt'
import { UpdateBanner } from '@/components/pwa/UpdateBanner'
import { registerServiceWorker } from '@/lib/service-worker/register'

function App() {
  useEffect(() => {
    // 注册 Service Worker
    registerServiceWorker({
      onUpdate: (registration) => {
        // 通知有更新可用
        window.postMessage({ type: 'SW_UPDATE_AVAILABLE' }, '*')
      },
      onSuccess: (registration) => {
        console.log('Service Worker ready')
      }
    })
  }, [])

  return (
    <>
      <PWAMeta />
      <UpdateBanner />
      <InstallPrompt />

      {/* 应用内容 */}
      <Router>
        {/* ... */}
      </Router>
    </>
  )
}
```

---

## 图标生成脚本

```javascript
// scripts/generate-icons.js
const sharp = require('sharp')
const fs = require('fs')
const path = require('path')

const SIZES = [
  72, 96, 128, 144, 152, 192, 384, 512
]

const SOURCE_ICON = path.join(__dirname, '../assets/icon.png')
const OUTPUT_DIR = path.join(__dirname, '../frontend/public/icons')

async function generateIcons() {
  // 确保输出目录存在
  if (!fs.existsSync(OUTPUT_DIR)) {
    fs.mkdirSync(OUTPUT_DIR, { recursive: true })
  }

  // 生成各尺寸图标
  for (const size of SIZES) {
    await sharp(SOURCE_ICON)
      .resize(size, size, { fit: 'cover', background: { r: 0, g: 0, b: 0, alpha: 0 } })
      .png()
      .toFile(path.join(OUTPUT_DIR, `icon-${size}x${size}.png`))

    console.log(`Generated ${size}x${size} icon`)
  }

  // 生成 maskable 图标（带安全边距）
  const MASKABLE_SIZE = 512
  const SAFE_ZONE = 0.85  // 85% 安全区

  await sharp(SOURCE_ICON)
    .resize(
      MASKABLE_SIZE * SAFE_ZONE,
      MASKABLE_SIZE * SAFE_ZONE,
      { fit: 'cover' }
    )
    .extend({
      top: Math.floor(MASKABLE_SIZE * (1 - SAFE_ZONE) / 2),
      bottom: Math.floor(MASKABLE_SIZE * (1 - SAFE_ZONE) / 2),
      left: Math.floor(MASKABLE_SIZE * (1 - SAFE_ZONE) / 2),
      right: Math.floor(MASKABLE_SIZE * (1 - SAFE_ZONE) / 2),
      background: { r: 139, g: 92, b: 246, alpha: 1 }
    })
    .png()
    .toFile(path.join(OUTPUT_DIR, 'maskable-icon-512x512.png'))

  console.log('Generated maskable icon')

  console.log('All icons generated successfully!')
}

generateIcons().catch(console.error)
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `frontend/public/manifest.json` | 创建 | PWA 清单 |
| `frontend/public/splash.html` | 创建 | 启动画面 |
| `frontend/public/icons/*` | 创建 | 应用图标集 |
| `frontend/src/hooks/usePWAInstall.ts` | 创建 | 安装 Hook |
| `frontend/src/components/pwa/InstallPrompt.tsx` | 创建 | 安装提示 |
| `frontend/src/components/pwa/UpdateBanner.tsx` | 创建 | 更新提示 |
| `frontend/src/components/pwa/PWAMeta.tsx` | 创建 | PWA 元标签 |
| `scripts/generate-icons.js` | 创建 | 图标生成脚本 |

---

## 验收标准

- [ ] 应用可以被安装到桌面
- [ ] 安装后可以独立窗口运行
- [ ] 显示自定义应用图标
- [ ] 显示启动画面
- [ ] 主题色正确显示
- [ ] 快捷方式正常工作
- [ ] 更新提示正确显示
- [ ] iOS 和 Android 兼容

---

## 参考文档

- M6-012: 离线缓存实现
- Web App Manifest 规范
- PWA 最佳实践指南

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
