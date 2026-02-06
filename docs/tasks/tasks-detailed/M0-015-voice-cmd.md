# M0-015: 编写语音命令规范

**任务ID**: M0-015
**标题**: 编写语音命令规范
**类型**: spec (规范定义)
**预估工时**: 1h
**依赖**: 无

---

## 任务描述

定义语音输入命令的规范，支持语音识别和命令执行，让玩家可以通过语音进行掷骰、检定等操作。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M0-015-01 | 定义语音命令格式 | Voice Format | 15min |
| M0-015-02 | 定义命令映射 | Command Mapping | 15min |
| M0-015-03 | 定义模糊匹配 | Fuzzy Matching | 20min |
| M0-015-04 | 编写语音示例 | Examples | 15min |
| M0-015-05 | 编写错误处理 | Error Handling | 15min |
| M0-015-06 | 编写输出格式 | Output Format | 10min |

---

## 语音命令格式

```
<命令关键词> [参数1] [参数2] ...
```

**说明**:
- 命令关键词：核心命令名称，如 "掷骰"、"检定"
- 参数：命令参数，用自然语言表达

---

## 命令映射

### 掷骰类

| 语音输入 | 对应命令 | 说明 |
|----------|----------|------|
| "掷骰" / "roll" | `/roll d100` | 默认 d100 |
| "掷二十面" / "d20" | `/roll d20` | 掷 d20 |
| "掷十个骰子" | `/roll 10d10` | 多个骰子 |
| "大成功" | `/roll 100` | 特殊结果 |
| "掷五加二" | `/roll 5d6+2` | 带修正值 |

### 检定类

| 语音输入 | 对应命令 | 说明 |
|----------|----------|------|
| "侦查检定" | `/check 侦查` | 技能检定 |
| "侦查加二十" | `/check 侦查 +20` | 带修正 |
| "幸运检定" | `/check luck` | 幸运检定 |
| "暗骰侦查" | `/roll 1d100 侦查 bonus` | 暗骰检定 |

### 战斗类

| 语音输入 | 对应命令 | 说明 |
|----------|----------|------|
| "攻击僵尸" | `/attack 僵尸` | 攻击目标 |
| "造成五点伤害" | `/damage 僵尸 5` | 造成伤害 |
| "恢复三点生命" | `/heal 3` | 治疗恢复 |
| "下一回合" | `/combat turn` | 战斗推进 |

### 角色管理

| 语音输入 | 对应命令 | 说明 |
|----------|----------|------|
| "查看角色卡" | `/character` | 查看角色 |
| "切换到张三" | `/switch 张三` | 切换角色 |
| "查看张三的状态" | `/status 张三` | 查看状态 |

---

## 模糊匹配

### 相似度计算

```typescript
// frontend/src/lib/voice/fuzzy-matcher.ts
export function calculateSimilarity(str1: string, str2: string): number {
  // 编辑距离算法
  const matrix = []
  const len1 = str1.length
  const len2 = str2.length

  for (let i = 0; i <= len2; i++) {
    matrix[i] = [i]
  }

  for (let j = 0; j <= len1; j++) {
    matrix[0][j] = j
  }

  for (let i = 1; i <= len2; i++) {
    for (let j = 1; j <= len1; j++) {
      if (str2.charAt(i - 1) === str1.charAt(j - 1)) {
        matrix[i][j] = matrix[i - 1][j - 1]
      } else {
        matrix[i][j] = Math.min(
          matrix[i][j - 1] + 1,
          matrix[i - 1][j] + 1,
          matrix[i][j - 1] + 1
        )
      }
    }
  }

  const maxLen = Math.max(len1, len2)
  return (maxLen - matrix[len2][len1]) / maxLen
}

export function matchCommand(
  input: string,
  commands: string[]
): { command: string; confidence: number } | null {
  const threshold = 0.6 // 相似度阈值

  let bestMatch: { command: string; confidence: number } | null = null

  for (const cmd of commands) {
    const similarity = calculateSimilarity(input.toLowerCase(), cmd.toLowerCase())

    if (similarity >= threshold) {
      if (!bestMatch || similarity > bestMatch.confidence) {
        bestMatch = { command: cmd, confidence: similarity }
      }
    }
  }

  return bestMatch
}
```

### 命令同义词

```typescript
// frontend/src/lib/voice/synonyms.ts
export const COMMAND_SYNONYMS: Record<string, string[]> = {
  'roll': ['掷骰', 'roll', '投掷', '骰子', 'd100', '二十面', 'd20'],
  'check': ['检定', '检查', '测试', '判定'],
  'attack': ['攻击', '打击', '攻击目标'],
  'damage': ['伤害', '造成伤害', '伤害值'],
  'heal': ['治疗', '恢复', '恢复生命', '加血'],
  'san': ['san检定', '理智检定', '理智'],
  'character': ['角色', '角色卡', '查看角色'],
  'status': ['状态', '查看状态', '当前状态'],
  'help': ['帮助', '帮助命令', '指令'],
}

export function resolveCommand(input: string): string | null {
  for (const [command, synonyms] of Object.entries(COMMAND_SYNONYMS)) {
    if (synonyms.includes(input)) {
      return command
    }
  }
  return null
}
```

---

## 参数提取

```typescript
// frontend/src/lib/voice/param-extractor.ts
export interface ExtractedParams {
  command: string
  params: Record<string, any>
}

export function extractParameters(input: string): ExtractedParams {
  const result: ExtractedParams = {
    command: '',
    params: {}
  }

  // 数字提取
  const numberPattern = /(\d+(?:\.\d+)?)\s*(?:d(\d+))?/g
  const numbers = []
  let match

  while ((match = numberPattern.exec(input)) !== null) {
    if (match[2]) {
      // 骰子格式: NdM
      numbers.push({
        type: 'dice',
        count: parseInt(match[1]),
        sides: parseInt(match[2])
      })
    } else {
      // 纯数字
      numbers.push({
        type: 'number',
        value: parseFloat(match[1])
      })
    }
  }

  // 中文数字转换
  const chineseNumbers = {
    '一': 1, '二': 2, '三': 3, '四': 4, '五': 5,
    '六': 6, '七': 7, '八': 8, '九': 9, '十': 10,
    '两': 2
  }

  // 技能名称提取
  const skills = ['侦查', '聆听', '图书馆', '心理学', '说服', '恐吓',
                 '斗殴', '闪避', '射击', '急救', '侦查', '暗骰']

  for (const skill of skills) {
    if (input.includes(skill)) {
      result.params.skill = skill
      break
    }
  }

  // 目标名称提取
  const targets = input.match(/(?:攻击|伤害)\s+([^\s，。]+)/)
  if (targets) {
    result.params.target = targets[1].trim()
  }

  // 提取骰子信息
  if (numbers.length > 0) {
    result.params.dice = numbers
  }

  return result
}
```

---

## 使用示例

### 示例 1: 简单掷骰

```bash
语音输入: "掷骰"
识别结果: /roll d100
输出: 🎲 掷骰结果: 45
```

### 示例 2: 技能检定

```bash
语音输入: "侦查检定加二十"
识别结果: /check 侦查 +20
输出: 🎲 侦查检定(60): 40 [成功]
```

### 示例 3: 战斗

```bash
语音输入: "攻击僵尸A造成五点伤害"
识别结果:
  1. /attack 僵尸A
  2. /damage 僵尸A 5

输出:
  ⚔️ 攻击检定: 65/60 [失败]
  💥 造成伤害: 5 点
  僵尸A HP: 17/22
```

### 示例 4: 多骰子

```bash
语音输入: "掷三个六面骰子"
识别结果: /roll 3d6
输出: 🎲 掷骰结果: 3d6 = 2 + 5 + 1 = 8
```

---

## 错误处理

### 无法识别

```
❌ 无法识别命令: "巴拉巴拉小魔仙"
请尝试:
- "掷骰"
- "侦查检定"
- "攻击 [目标]"
```

### 参数不完整

```
⚠️ 命令不完整
语音输入: "攻击"
需要指定目标，例如: "攻击僵尸A"
```

### 多个匹配

```
⚠️ 发现多个匹配
语音输入: "攻击"
可能的目标:
1. 僵尸A
2. 僵尸B
3. 管家
请明确指定目标
```

---

## 前端集成

```typescript
// frontend/src/components/game/VoiceCommand.tsx
import { useState, useRef, useEffect } from 'react'
import { Mic, MicOff, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useToast } from '@/hooks/use-toast'

export function VoiceCommand() {
  const [isListening, setIsListening] = useState(false)
  const [isProcessing, setIsProcessing] = useState(false)
  const recognitionRef = useRef<any>()

  const { toast } = useToast()

  useEffect(() => {
    // 初始化语音识别
    if ('webkitSpeechRecognition' in window) {
      const SpeechRecognition = (window as any).webkitSpeechRecognition
      recognitionRef.current = new SpeechRecognition()
      recognitionRef.current.continuous = false
      recognitionRef.current.lang = 'zh-CN'

      recognitionRef.current.onresult = (event: any) => {
        const transcript = event.results[0][0].transcript
        handleVoiceCommand(transcript)
      }

      recognitionRef.current.onerror = (event: any) => {
        console.error('Speech recognition error:', event.error)
        setIsListening(false)
        toast({
          title: '语音识别失败',
          description: event.error,
          variant: 'destructive',
        })
      }

      recognitionRef.current.onend = () => {
        setIsListening(false)
      }
    }

    return () => {
      if (recognitionRef.current) {
        recognitionRef.current.abort()
      }
    }
  }, [toast])

  const handleVoiceCommand = async (transcript: string) => {
    setIsProcessing(true)

    try {
      // 发送到后端解析
      const response = await fetch('/api/voice/parse', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ transcript }),
      })

      if (!response.ok) throw new Error('解析失败')

      const result = await response.json()

      if (result.error) {
        toast({
          title: '无法识别命令',
          description: result.error,
          variant: 'destructive',
        })
      } else {
        // 执行命令
        await executeCommand(result.command)
        toast({
          title: '语音命令',
          description: `已执行: ${transcript} → ${result.command}`,
        })
      }
    } catch (error) {
      console.error('Failed to process voice command:', error)
    } finally {
      setIsProcessing(false)
    }
  }

  const executeCommand = async (command: string) => {
    // 通过 WebSocket 发送命令
    // ...
  }

  const toggleListening = () => {
    if (!recognitionRef.current) {
      toast({
        title: '浏览器不支持语音识别',
        variant: 'destructive',
      })
      return
    }

    if (isListening) {
      recognitionRef.current.stop()
      setIsListening(false)
    } else {
      recognitionRef.current.start()
      setIsListening(true)
    }
  }

  return (
    <Button
      size="sm"
      variant={isListening ? 'default' : 'outline'}
      onClick={toggleListening}
      disabled={isProcessing}
    >
      {isProcessing ? (
        <Loader2 className="h-4 w-4 animate-spin" />
      ) : isListening ? (
        <MicOff className="h-4 w-4" />
      ) : (
        <Mic className="h-4 w-4" />
      )}
    </Button>
  )
}
```

---

## 后端解析 API

```python
# app/api/voice.py
from fastapi import APIRouter, Depends
from pydantic import BaseModel

from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.voice import VoiceCommandService

router = APIRouter(prefix="/voice", tags=["voice"])

class ParseRequest(BaseModel):
    transcript: str

@router.post("/parse")
async def parse_voice_command(
    request: ParseRequest,
    current_user: User = Depends(get_current_user),
):
    """解析语音命令"""
    service = VoiceCommandService()

    try:
      result = service.parse(request.transcript)
      return result
    except Exception as e:
        return {
            "error": str(e),
            "suggestions": [
              "掷骰",
              "侦查检定",
              "攻击 [目标]",
            ]
        }
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `docs/specs/commands/voice.md` | 创建 | 命令规范文档 |
| `frontend/src/lib/voice/fuzzy-matcher.ts` | 创建 | 模糊匹配 |
| `frontend/src/lib/voice/synonyms.ts` | 创建 | 同义词映射 |
| `frontend/src/lib/voice/param-extractor.ts` | 创建 | 参数提取 |
| `frontend/src/components/game/VoiceCommand.tsx` | 创建 | 语音命令组件 |
| `app/services/voice.py` | 创建 | 语音解析服务 |
| `app/api/voice.py` | 创建 | 语音 API |

---

## 验收标准

- [ ] 语音格式定义完整
- [ ] 命令映射准确
- [ ] 模糊匹配有效
- [ ] 示例覆盖全面
- [ ] 错误处理友好
- [ ] 集成测试通过

---

## 参考文档

- M0-010: 命令语法 BNF 范式
- Web Speech API

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
