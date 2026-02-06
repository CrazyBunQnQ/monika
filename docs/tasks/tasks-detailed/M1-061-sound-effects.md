# M1-061: 实现音效系统

**任务ID**: M1-061
**标题**: 实现音效系统
**类型**: frontend (前端开发)
**预估工时**: 1.5h
**依赖**: 无

---

## 任务描述

实现游戏音效系统，包括掷骰声、按钮点击、通知提示等音效，以及背景音乐播放功能。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-061-01 | 设计音效架构 | Sound Architecture | 15min |
| M1-061-02 | 实现音效管理器 | Sound Manager | 30min |
| M1-061-03 | 实现音效播放 | Sound Playback | 25min |
| M1-061-04 | 实现音量控制 | Volume Control | 20min |
| M1-061-05 | 实现音效设置 | Sound Settings | 20min |
| M1-061-06 | 添加默认音效 | Default Sounds | 15min |
| M1-061-07 | 编写音效测试 | 测试覆盖 | 10min |

---

## 音效管理器

```typescript
// frontend/src/lib/sounds/sound-manager.ts
export interface SoundEffect {
  id: string
  name: string
  src: string
  volume?: number
  loop?: boolean
}

export interface SoundConfig {
  masterVolume: number
  sfxVolume: number
  musicVolume: number
  enabled: boolean
}

class SoundManager {
  private audioContext: AudioContext | null = null
  private sounds: Map<string, HTMLAudioElement> = new Map()
  private music: HTMLAudioElement | null = null
  private config: SoundConfig = {
    masterVolume: 1,
    sfxVolume: 0.8,
    musicVolume: 0.5,
    enabled: true,
  }

  constructor() {
    this.loadConfig()
  }

  private loadConfig() {
    const saved = localStorage.getItem('sound-config')
    if (saved) {
      this.config = { ...this.config, ...JSON.parse(saved) }
    }
  }

  private saveConfig() {
    localStorage.setItem('sound-config', JSON.stringify(this.config))
  }

  init() {
    if (!this.audioContext) {
      this.audioContext = new (window.AudioContext || (window as any).webkitAudioContext)()
    }
  }

  // 播放音效
  play(soundId: string, volume?: number) {
    if (!this.config.enabled) return

    this.init()

    const sound = this.sounds.get(soundId)
    if (sound) {
      const audio = sound.cloneNode() as HTMLAudioElement
      audio.volume = (volume ?? this.config.sfxVolume) * this.config.masterVolume
      audio.play().catch(console.error)
    }
  }

  // 播放音乐
  playMusic(soundId: string, loop = true) {
    if (!this.config.enabled) return

    this.stopMusic()

    const sound = this.sounds.get(soundId)
    if (sound) {
      this.music = sound.cloneNode() as HTMLAudioElement
      this.music.loop = loop
      this.music.volume = this.config.musicVolume * this.config.masterVolume
      this.music.play().catch(console.error)
    }
  }

  // 停止音乐
  stopMusic() {
    if (this.music) {
      this.music.pause()
      this.music = null
    }
  }

  // 预加载音效
  preload(sounds: SoundEffect[]) {
    sounds.forEach(sound => {
      const audio = new Audio(sound.src)
      audio.volume = sound.volume ?? 1
      audio.loop = sound.loop ?? false
      this.sounds.set(sound.id, audio)
    })
  }

  // 获取配置
  getConfig(): SoundConfig {
    return { ...this.config }
  }

  // 更新配置
  updateConfig(updates: Partial<SoundConfig>) {
    this.config = { ...this.config, ...updates }
    this.saveConfig()

    // 应用音量设置
    if (this.music) {
      this.music.volume = this.config.musicVolume * this.config.masterVolume
    }
  }

  // 切换静音
  toggleMute() {
    this.config.enabled = !this.config.enabled
    this.saveConfig()

    if (!this.config.enabled) {
      this.stopMusic()
    }

    return this.config.enabled
  }
}

// 单例
export const soundManager = new SoundManager()
```

---

## 音效定义

```typescript
// frontend/src/lib/sounds/sounds.ts
import { SoundEffect } from './sound-manager'

export const SOUND_EFFECTS: SoundEffect[] = [
  // 掷骰声
  {
    id: 'dice-roll',
    name: '掷骰',
    src: '/sounds/dice-roll.mp3',
    volume: 0.6,
  },
  {
    id: 'dice-hit',
    name: '骰子碰撞',
    src: '/sounds/dice-hit.mp3',
    volume: 0.5,
  },

  // 按钮
  {
    id: 'button-click',
    name: '按钮点击',
    src: '/sounds/button-click.mp3',
    volume: 0.3,
  },
  {
    id: 'button-hover',
    name: '按钮悬停',
    src: '/sounds/button-hover.mp3',
    volume: 0.2,
  },

  // 通知
  {
    id: 'notification',
    name: '通知',
    src: '/sounds/notification.mp3',
    volume: 0.5,
  },
  {
    id: 'success',
    name: '成功',
    src: '/sounds/success.mp3',
    volume: 0.5,
  },
  {
    id: 'error',
    name: '错误',
    src: '/sounds/error.mp3',
    volume: 0.5,
  },
  {
    id: 'warning',
    name: '警告',
    src: '/sounds/warning.mp3',
    volume: 0.5,
  },

  // 游戏
  {
    id: 'turn-start',
    name: '回合开始',
    src: '/sounds/turn-start.mp3',
    volume: 0.6,
  },
  {
    id: 'combat-start',
    name: '战斗开始',
    src: '/sounds/combat-start.mp3',
    volume: 0.7,
  },
  {
    id: 'level-up',
    name: '升级',
    src: '/sounds/level-up.mp3',
    volume: 0.7,
  },

  // SAN 检定
  {
    id: 'san-check',
    name: 'SAN检定',
    src: '/sounds/san-check.mp3',
    volume: 0.6,
  },
  {
    id: 'san-fail',
    name: 'SAN失败',
    src: '/sounds/san-fail.mp3',
    volume: 0.7,
  },
  {
    id: 'madness',
    name: '疯狂',
    src: '/sounds/madness.mp3',
    volume: 0.8,
  },
]

export const BACKGROUND_MUSIC: SoundEffect[] = [
  {
    id: 'ambient-main',
    name: '主场景',
    src: '/music/ambient-main.mp3',
    volume: 0.4,
    loop: true,
  },
  {
    id: 'ambient-combat',
    name: '战斗',
    src: '/music/ambient-combat.mp3',
    volume: 0.5,
    loop: true,
  },
  {
    id: 'ambient-investigation',
    name: '调查',
    src: '/music/ambient-investigation.mp3',
    volume: 0.4,
    loop: true,
  },
  {
    id: 'ambient-horror',
    name: '恐怖',
    src: '/music/ambient-horror.mp3',
    volume: 0.5,
    loop: true,
  },
]
```

---

## 音效 Context

```tsx
// frontend/src/contexts/SoundContext.tsx
import { createContext, useContext, ReactNode } from 'react'
import { soundManager, SoundConfig } from '@/lib/sounds/sound-manager'
import { SOUND_EFFECTS, BACKGROUND_MUSIC } from '@/lib/sounds/sounds'

interface SoundContextValue {
  play: (soundId: string, volume?: number) => void
  playMusic: (musicId: string, loop?: boolean) => void
  stopMusic: () => void
  config: SoundConfig
  updateConfig: (updates: Partial<SoundConfig>) => void
  toggleMute: () => boolean
}

const SoundContext = createContext<SoundContextValue | null>(null)

export function useSound() {
  const context = useContext(SoundContext)
  if (!context) {
    throw new Error('useSound must be used within SoundProvider')
  }
  return context
}

interface SoundProviderProps {
  children: ReactNode
}

export function SoundProvider({ children }: SoundProviderProps) {
  // 预加载音效
  soundManager.preload([...SOUND_EFFECTS, ...BACKGROUND_MUSIC])

  const value: SoundContextValue = {
    play: (soundId, volume) => soundManager.play(soundId, volume),
    playMusic: (musicId, loop) => soundManager.playMusic(musicId, loop),
    stopMusic: () => soundManager.stopMusic(),
    config: soundManager.getConfig(),
    updateConfig: (updates) => soundManager.updateConfig(updates),
    toggleMute: () => soundManager.toggleMute(),
  }

  return (
    <SoundContext.Provider value={value}>
      {children}
    </SoundContext.Provider>
  )
}
```

---

## 音效设置组件

```tsx
// frontend/src/components/game/SoundSettings.tsx
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Slider } from '@/components/ui/slider'
import { Button } from '@/components/ui/button'
import { Volume2, VolumeX, Music, Bell } from 'lucide-react'
import { useSound } from '@/contexts/SoundContext'
import { useState } from 'react'

export function SoundSettings() {
  const { config, updateConfig, toggleMute, play } = useSound()
  const [previewPlaying, setPreviewPlaying] = useState(false)

  const handlePreview = () => {
    setPreviewPlaying(true)
    play('dice-roll')
    setTimeout(() => setPreviewPlaying(false), 500)
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base flex items-center justify-between">
          <span>音效设置</span>
          <Button
            size="sm"
            variant={config.enabled ? 'default' : 'outline'}
            onClick={toggleMute}
          >
            {config.enabled ? (
              <Volume2 className="h-4 w-4" />
            ) : (
              <VolumeX className="h-4 w-4" />
            )}
          </Button>
        </CardTitle>
      </CardHeader>

      <CardContent className="space-y-6">
        {/* 主音量 */}
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">主音量</span>
            <span className="text-xs text-muted-foreground">
              {Math.round(config.masterVolume * 100)}%
            </span>
          </div>
          <Slider
            value={[config.masterVolume * 100]}
            onValueChange={([value]) => updateConfig({ masterVolume: value / 100 })}
            max={100}
            step={1}
            disabled={!config.enabled}
          />
        </div>

        {/* 音效音量 */}
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium flex items-center">
              <Bell className="h-4 w-4 mr-2" />
              音效
            </span>
            <span className="text-xs text-muted-foreground">
              {Math.round(config.sfxVolume * 100)}%
            </span>
          </div>
          <Slider
            value={[config.sfxVolume * 100]}
            onValueChange={([value]) => updateConfig({ sfxVolume: value / 100 })}
            max={100}
            step={1}
            disabled={!config.enabled}
          />
        </div>

        {/* 音乐音量 */}
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium flex items-center">
              <Music className="h-4 w-4 mr-2" />
              音乐
            </span>
            <span className="text-xs text-muted-foreground">
              {Math.round(config.musicVolume * 100)}%
            </span>
          </div>
          <Slider
            value={[config.musicVolume * 100]}
            onValueChange={([value]) => updateConfig({ musicVolume: value / 100 })}
            max={100}
            step={1}
            disabled={!config.enabled}
          />
        </div>

        {/* 测试按钮 */}
        <Button
          size="sm"
          variant="outline"
          className="w-full"
          onClick={handlePreview}
          disabled={!config.enabled || previewPlaying}
        >
          {previewPlaying ? '播放中...' : '测试音效'}
        </Button>
      </CardContent>
    </Card>
  )
}
```

---

## 使用示例

```tsx
// 在组件中使用音效
import { useSound } from '@/contexts/SoundContext'
import { useEffect } from 'react'

export function DiceRoller() {
  const { play } = useSound()

  const handleRoll = () => {
    play('dice-roll')
    // ... 掷骰逻辑
  }

  return (
    <button onClick={handleRoll}>
      掷骰
    </button>
  )
}

// 播放背景音乐
import { useSound } from '@/contexts/SoundContext'

export function GameRoom() {
  const { playMusic } = useSound()

  useEffect(() => {
    playMusic('ambient-main')
    return () => {
      // cleanup
    }
  }, [])

  // ...
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `frontend/src/lib/sounds/sound-manager.ts` | 创建 | 音效管理器 |
| `frontend/src/lib/sounds/sounds.ts` | 创建 | 音效定义 |
| `frontend/src/contexts/SoundContext.tsx` | 创建 | 音效 Context |
| `frontend/src/components/game/SoundSettings.tsx` | 创建 | 音效设置组件 |
| `frontend/public/sounds/` | 创建 | 音效文件目录 |
| `frontend/public/music/` | 创建 | 音乐文件目录 |

---

## 验收标准

- [ ] 音效播放正常
- [ ] 音量控制有效
- [ ] 音乐循环播放
- [ ] 静音功能正常
- [ ] 配置持久化
- [ ] 性能表现良好

---

## 参考文档

- Web Audio API
- HTMLAudioElement

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
