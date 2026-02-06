# M6-014: 实现数据可视化

**任务ID**: M6-014
**标题**: 实现数据可视化
**类型**: frontend (前端开发)
**预估工时**: 9h
**依赖**: M5-001

---

## 任务描述

实现游戏数据可视化功能，包括角色属性雷达图、骰子检定历史、SAN 值变化曲线、游戏统计仪表板等。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-014-01 | 选择可视化库 | 技术选型 | 30min |
| M6-014-02 | 实现角色属性雷达图 | 六维属性展示 | 1.5h |
| M6-014-03 | 实现骰子检定统计 | 检定分布图表 | 1.5h |
| M6-014-04 | 实现 SAN 值曲线 | 理智变化趋势 | 1.5h |
| M6-014-05 | 实现游戏统计仪表板 | 数据概览 | 2h |
| M6-014-06 | 实现战斗数据可视化 | 伤害统计 | 1h |
| M6-014-07 | 实现导出功能 | 图表导出 | 45min |
| M6-014-08 | 实现暗色模式适配 | 主题适配 | 30min |

---

## 前端实现

### 可视化库配置

```typescript
// frontend/src/lib/chart-config.ts
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  RadialLinearScale,
  PointElement,
  LineElement,
  BarElement,
  ArcElement,
  Title,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'

// 注册 Chart.js 组件
ChartJS.register(
  CategoryScale,
  LinearScale,
  RadialLinearScale,
  PointElement,
  LineElement,
  BarElement,
  ArcElement,
  Title,
  Tooltip,
  Legend,
  Filler
)

// 默认配置
export const chartDefaultOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      labels: {
        color: 'rgba(255, 255, 255, 0.7)',
        font: {
          size: 12
        }
      }
    },
    tooltip: {
      backgroundColor: 'rgba(0, 0, 0, 0.8)',
      titleColor: '#fff',
      bodyColor: '#fff',
      borderColor: 'rgba(139, 92, 246, 0.5)',
      borderWidth: 1,
      padding: 12,
      displayColors: true,
      boxPadding: 4
    }
  },
  scales: {
    x: {
      ticks: {
        color: 'rgba(255, 255, 255, 0.5)',
        font: {
          size: 11
        }
      },
      grid: {
        color: 'rgba(255, 255, 255, 0.05)'
      }
    },
    y: {
      ticks: {
        color: 'rgba(255, 255, 255, 0.5)',
        font: {
          size: 11
        }
      },
      grid: {
        color: 'rgba(255, 255, 255, 0.05)'
      }
    }
  }
}

// 颜色主题
export const chartColors = {
  primary: 'rgba(139, 92, 246, 0.8)',
  primaryLight: 'rgba(139, 92, 246, 0.2)',
  secondary: 'rgba(236, 72, 153, 0.8)',
  secondaryLight: 'rgba(236, 72, 153, 0.2)',
  success: 'rgba(34, 197, 94, 0.8)',
  warning: 'rgba(234, 179, 8, 0.8)',
  danger: 'rgba(239, 68, 68, 0.8)',
  info: 'rgba(59, 130, 246, 0.8)'
}

// 渐变色生成器
export function createGradient(ctx: CanvasRenderingContext2D, color: string) {
  const gradient = ctx.createLinearGradient(0, 0, 0, 400)
  gradient.addColorStop(0, color.replace('0.8', '0.5'))
  gradient.addColorStop(1, color.replace('0.8', '0.05'))
  return gradient
}
```

### 角色属性雷达图

```tsx
// frontend/src/components/charts/AttributeRadar.tsx
import { Radar } from 'react-chartjs-2'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { chartDefaultOptions, chartColors } from '@/lib/chart-config'
import { Brain, Muscle, Eye, Hand, Mouth, Ear } from 'lucide-react'

interface Attribute {
  name: string
  value: number
  max: number
}

interface AttributeRadarProps {
  attributes: Attribute[]
  className?: string
}

const ATTRIBUTE_ICONS = {
  '力量': Muscle,
  '体质': Muscle,
  '敏捷': Hand,
  '外貌': Eye,
  '智力': Brain,
  '教育': Brain,
  '意志': Brain,
  '幸运': Star
}

export function AttributeRadar({ attributes, className }: AttributeRadarProps) {
  const labels = attributes.map(a => a.name)
  const data = attributes.map(a => ((a.value / a.max) * 100).toFixed(0))

  const chartData = {
    labels,
    datasets: [{
      label: '属性值',
      data,
      backgroundColor: 'rgba(139, 92, 246, 0.2)',
      borderColor: 'rgba(139, 92, 246, 1)',
      borderWidth: 2,
      pointBackgroundColor: 'rgba(139, 92, 246, 1)',
      pointBorderColor: '#fff',
      pointHoverBackgroundColor: '#fff',
      pointHoverBorderColor: 'rgba(139, 92, 246, 1)',
      pointRadius: 4,
      pointHoverRadius: 6
    }]
  }

  const options = {
    ...chartDefaultOptions,
    scales: {
      r: {
        beginAtZero: true,
        max: 100,
        ticks: {
          stepSize: 20,
          color: 'rgba(255, 255, 255, 0.5)',
          backdropColor: 'transparent'
        },
        grid: {
          color: 'rgba(255, 255, 255, 0.1)'
        },
        angleLines: {
          color: 'rgba(255, 255, 255, 0.1)'
        },
        pointLabels: {
          color: 'rgba(255, 255, 255, 0.7)',
          font: {
            size: 12,
            weight: 'bold'
          }
        }
      }
    },
    plugins: {
      legend: {
        display: false
      }
    }
  }

  return (
    <Card className={className}>
      <CardHeader>
        <CardTitle>角色属性</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="h-80">
          <Radar data={chartData} options={options} />
        </div>

        {/* 属性详情 */}
        <div className="mt-6 grid grid-cols-2 gap-4">
          {attributes.map((attr) => {
            const Icon = ATTRIBUTE_ICONS[attr.name] || Star
            const percentage = (attr.value / attr.max) * 100

            return (
              <div key={attr.name} className="flex items-center space-x-3">
                <Icon className="h-5 w-5 text-purple-500" />
                <div className="flex-1">
                  <div className="flex items-center justify-between text-sm mb-1">
                    <span className="font-medium">{attr.name}</span>
                    <span className="text-muted-foreground">
                      {attr.value}/{attr.max}
                    </span>
                  </div>
                  <div className="h-2 bg-secondary rounded-full overflow-hidden">
                    <div
                      className="h-full bg-gradient-to-r from-purple-500 to-pink-500 transition-all"
                      style={{ width: `${percentage}%` }}
                    />
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      </CardContent>
    </Card>
  )
}
```

### 骰子检定统计

```tsx
// frontend/src/components/charts/RollHistoryChart.tsx
import { Line, Bar } from 'react-chartjs-2'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { chartDefaultOptions, chartColors, createGradient } from '@/lib/chart-config'
import { Dice } from 'lucide-react'

interface RollRecord {
  id: string
  skill: string
  value: number
  difficulty: number
  success: boolean
  timestamp: Date
}

interface RollHistoryChartProps {
  rolls: RollRecord[]
  className?: string
}

export function RollHistoryChart({ rolls, className }: RollHistoryChartProps) {
  // 按技能分组统计
  const skillStats = rolls.reduce((acc, roll) => {
    if (!acc[roll.skill]) {
      acc[roll.skill] = { total: 0, success: 0, failure: 0, values: [] }
    }
    acc[roll.skill].total++
    acc[roll.skill].values.push(roll.value)
    if (roll.success) {
      acc[roll.skill].success++
    } else {
      acc[roll.skill].failure++
    }
    return acc
  }, {} as Record<string, { total: number; success: number; failure: number; values: number[] }>)

  // 技能成功率数据
  const successRateData = {
    labels: Object.keys(skillStats),
    datasets: [{
      label: '成功率 (%)',
      data: Object.values(skillStats).map(s =>
        ((s.success / s.total) * 100).toFixed(1)
      ),
      backgroundColor: Object.values(skillStats).map(s =>
        s.success / s.total > 0.5 ? chartColors.success : chartColors.danger
      ),
      borderRadius: 8
    }]
  }

  // 时间序列数据（最近20次）
  const recentRolls = rolls.slice(-20)
  const timeSeriesData = {
    labels: recentRolls.map((_, i) => `#${i + 1}`),
    datasets: [
      {
        label: '检定值',
        data: recentRolls.map(r => r.value),
        borderColor: chartColors.primary,
        backgroundColor: (context: any) => {
          const ctx = context.chart.ctx
          return createGradient(ctx, chartColors.primary)
        },
        fill: true,
        tension: 0.4
      },
      {
        label: '难度',
        data: recentRolls.map(r => r.difficulty),
        borderColor: chartColors.secondary,
        borderDash: [5, 5],
        fill: false,
        tension: 0.4
      }
    ]
  }

  // 分布统计
  const distributionData = {
    labels: ['大成功', '困难成功', '成功', '失败', '大失败'],
    datasets: [{
      label: '次数',
      data: [
        rolls.filter(r => r.value === 1).length,
        rolls.filter(r => r.value > 1 && r.value <= r.difficulty / 2).length,
        rolls.filter(r => r.value > r.difficulty / 2 && r.value <= r.difficulty).length,
        rolls.filter(r => r.value > r.difficulty && r.value < 100).length,
        rolls.filter(r => r.value === 100).length
      ],
      backgroundColor: [
        chartColors.success,
        'rgba(34, 197, 94, 0.6)',
        chartColors.info,
        'rgba(239, 68, 68, 0.6)',
        chartColors.danger
      ],
      borderWidth: 0
    }]
  }

  const barOptions = {
    ...chartDefaultOptions,
    plugins: {
      ...chartDefaultOptions.plugins,
      legend: {
        display: false
      }
    }
  }

  return (
    <Card className={className}>
      <CardHeader>
        <CardTitle className="flex items-center">
          <Dice className="h-5 w-5 mr-2 text-purple-500" />
          检定统计
        </CardTitle>
      </CardHeader>

      <CardContent>
        <Tabs defaultValue="timeline" className="w-full">
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="timeline">时间线</TabsTrigger>
            <TabsTrigger value="success">成功率</TabsTrigger>
            <TabsTrigger value="distribution">分布</TabsTrigger>
          </TabsList>

          <TabsContent value="timeline" className="space-y-4">
            <div className="h-64">
              <Line data={timeSeriesData} options={chartDefaultOptions} />
            </div>
            <div className="text-sm text-muted-foreground text-center">
              最近 {recentRolls.length} 次检定的数值变化
            </div>
          </TabsContent>

          <TabsContent value="success" className="space-y-4">
            <div className="h-64">
              <Bar data={successRateData} options={barOptions} />
            </div>
            <div className="text-sm text-muted-foreground text-center">
              各技能的检定成功率
            </div>
          </TabsContent>

          <TabsContent value="distribution" className="space-y-4">
            <div className="h-64">
              <Bar data={distributionData} options={barOptions} />
            </div>
            <div className="grid grid-cols-5 gap-2 text-center text-xs">
              <div>
                <div className="font-bold text-green-500">
                  {rolls.filter(r => r.value === 1).length}
                </div>
                <div className="text-muted-foreground">大成功</div>
              </div>
              <div>
                <div className="font-bold text-green-400">
                  {rolls.filter(r => r.value > 1 && r.value <= 50).length}
                </div>
                <div className="text-muted-foreground">困难成功</div>
              </div>
              <div>
                <div className="font-bold text-blue-500">
                  {rolls.filter(r => r.value > 50 && r.value <= 75).length}
                </div>
                <div className="text-muted-foreground">成功</div>
              </div>
              <div>
                <div className="font-bold text-red-400">
                  {rolls.filter(r => r.value > 75 && r.value < 100).length}
                </div>
                <div className="text-muted-foreground">失败</div>
              </div>
              <div>
                <div className="font-bold text-red-500">
                  {rolls.filter(r => r.value === 100).length}
                </div>
                <div className="text-muted-foreground">大失败</div>
              </div>
            </div>
          </TabsContent>
        </Tabs>

        {/* 汇总统计 */}
        <div className="mt-6 pt-6 border-t grid grid-cols-4 gap-4 text-center">
          <div>
            <div className="text-2xl font-bold text-purple-500">{rolls.length}</div>
            <div className="text-xs text-muted-foreground">总检定</div>
          </div>
          <div>
            <div className="text-2xl font-bold text-green-500">
              {((rolls.filter(r => r.success).length / rolls.length) * 100).toFixed(0)}%
            </div>
            <div className="text-xs text-muted-foreground">成功率</div>
          </div>
          <div>
            <div className="text-2xl font-bold">
              {(rolls.reduce((sum, r) => sum + r.value, 0) / rolls.length).toFixed(1)}
            </div>
            <div className="text-xs text-muted-foreground">平均值</div>
          </div>
          <div>
            <div className="text-2xl font-bold text-pink-500">
              {Math.max(...rolls.map(r => r.value))}
            </div>
            <div className="text-xs text-muted-foreground">最高值</div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
```

### SAN 值变化曲线

```tsx
// frontend/src/components/charts/SanityTrendChart.tsx
import { Line } from 'react-chartjs-2'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { chartDefaultOptions, chartColors, createGradient } from '@/lib/chart-config'
import { Brain, TrendingDown, TrendingUp, Minus } from 'lucide-react'

interface SanityRecord {
  timestamp: Date
  value: number
  change: number
  reason: string
}

interface SanityTrendChartProps {
  records: SanityRecord[]
  maxSan: number
  className?: string
}

export function SanityTrendChart({ records, maxSan, className }: SanityTrendChartProps) {
  const labels = records.map((r, i) => {
    const time = new Date(r.timestamp)
    return `${time.getHours()}:${time.getMinutes().toString().padStart(2, '0')}`
  })

  const chartData = {
    labels,
    datasets: [
      {
        label: 'SAN 值',
        data: records.map(r => r.value),
        borderColor: chartColors.primary,
        backgroundColor: (context: any) => {
          const ctx = context.chart.ctx
          return createGradient(ctx, chartColors.primary)
        },
        fill: true,
        tension: 0.4,
        pointRadius: 4,
        pointHoverRadius: 6
      },
      {
        label: '警告线',
        data: records.map(() => maxSan * 0.2),
        borderColor: chartColors.danger,
        borderDash: [5, 5],
        pointRadius: 0,
        fill: false
      }
    ]
  }

  const options = {
    ...chartDefaultOptions,
    scales: {
      ...chartDefaultOptions.scales,
      y: {
        ...chartDefaultOptions.scales.y,
        min: 0,
        max: maxSan
      }
    },
    plugins: {
      ...chartDefaultOptions.plugins,
      tooltip: {
        ...chartDefaultOptions.plugins.tooltip,
        callbacks: {
          afterLabel: (context: any) => {
            const record = records[context.dataIndex]
            return `原因: ${record.reason}`
          }
        }
      }
    }
  }

  // 统计数据
  const totalLoss = records.reduce((sum, r) => sum + (r.change < 0 ? Math.abs(r.change) : 0), 0)
  const totalGain = records.reduce((sum, r) => sum + (r.change > 0 ? r.change : 0), 0)
  const currentSan = records[records.length - 1]?.value || maxSan

  return (
    <Card className={className}>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <span className="flex items-center">
            <Brain className="h-5 w-5 mr-2 text-purple-500" />
            SAN 值变化
          </span>
          <div className="text-2xl font-bold">
            {currentSan}
            <span className="text-sm font-normal text-muted-foreground"> / {maxSan}</span>
          </div>
        </CardTitle>
      </CardHeader>

      <CardContent className="space-y-6">
        {/* 图表 */}
        <div className="h-64">
          <Line data={chartData} options={options} />
        </div>

        {/* 统计卡片 */}
        <div className="grid grid-cols-3 gap-4">
          <div className="p-4 rounded-lg bg-red-500/10 border border-red-500/20">
            <div className="flex items-center space-x-2 mb-1">
              <TrendingDown className="h-4 w-4 text-red-500" />
              <span className="text-sm text-muted-foreground">总损失</span>
            </div>
            <div className="text-2xl font-bold text-red-500">
              -{totalLoss}
            </div>
          </div>

          <div className="p-4 rounded-lg bg-green-500/10 border border-green-500/20">
            <div className="flex items-center space-x-2 mb-1">
              <TrendingUp className="h-4 w-4 text-green-500" />
              <span className="text-sm text-muted-foreground">总恢复</span>
            </div>
            <div className="text-2xl font-bold text-green-500">
              +{totalGain}
            </div>
          </div>

          <div className="p-4 rounded-lg bg-blue-500/10 border border-blue-500/20">
            <div className="flex items-center space-x-2 mb-1">
              <Minus className="h-4 w-4 text-blue-500" />
              <span className="text-sm text-muted-foreground">净变化</span>
            </div>
            <div className="text-2xl font-bold text-blue-500">
              {totalGain - totalLoss > 0 ? '+' : ''}{totalGain - totalLoss}
            </div>
          </div>
        </div>

        {/* 变化记录 */}
        <div className="space-y-2">
          <h4 className="text-sm font-semibold">最近变化</h4>
          <div className="space-y-1 max-h-48 overflow-y-auto">
            {records.slice(-10).reverse().map((record, index) => (
              <div
                key={index}
                className="flex items-center justify-between text-sm p-2 rounded bg-secondary/50"
              >
                <div className="flex items-center space-x-2">
                  {record.change < 0 ? (
                    <TrendingDown className="h-3 w-3 text-red-500" />
                  ) : record.change > 0 ? (
                    <TrendingUp className="h-3 w-3 text-green-500" />
                  ) : (
                    <Minus className="h-3 w-3 text-gray-500" />
                  )}
                  <span className="text-muted-foreground">{record.reason}</span>
                </div>
                <div className={record.change < 0 ? 'text-red-500' : 'text-green-500'}>
                  {record.change > 0 ? '+' : ''}{record.change}
                </div>
              </div>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
```

### 游戏统计仪表板

```tsx
// frontend/src/components/charts/GameDashboard.tsx
import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { AttributeRadar } from './AttributeRadar'
import { RollHistoryChart } from './RollHistoryChart'
import { SanityTrendChart } from './SanityTrendChart'
import { Button } from '@/components/ui/button'
import { Download, RefreshCw } from 'lucide-react'

interface DashboardProps {
  gameId: string
  className?: string
}

export function GameDashboard({ gameId, className }: DashboardProps) {
  const [data, setData] = useState<any>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchGameData()
  }, [gameId])

  const fetchGameData = async () => {
    setLoading(true)
    try {
      const response = await fetch(`/api/games/${gameId}/stats`)
      if (response.ok) {
        const gameData = await response.json()
        setData(gameData)
      }
    } finally {
      setLoading(false)
    }
  }

  const handleExport = async () => {
    // 导出图表为图片
    // TODO: 实现导出功能
  }

  if (loading) {
    return <div>加载中...</div>
  }

  if (!data) {
    return <div>暂无数据</div>
  }

  return (
    <div className={className}>
      {/* 操作栏 */}
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold">游戏统计</h2>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={fetchGameData}>
            <RefreshCw className="h-4 w-4 mr-2" />
            刷新
          </Button>
          <Button size="sm" onClick={handleExport}>
            <Download className="h-4 w-4 mr-2" />
            导出
          </Button>
        </div>
      </div>

      {/* 仪表板 */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* 角色属性 */}
        <AttributeRadar
          attributes={data.character.attributes}
          className="md:col-span-2 lg:col-span-1"
        />

        {/* SAN 值变化 */}
        <SanityTrendChart
          records={data.sanity.history}
          maxSan={data.character.maxSan}
        />

        {/* 检定统计 */}
        <RollHistoryChart
          rolls={data.rolls}
          className="md:col-span-2"
        />

        {/* 战斗统计 */}
        {data.combat && (
          <Card>
            <CardHeader>
              <CardTitle>战斗统计</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div className="text-center p-4 rounded-lg bg-secondary">
                    <div className="text-2xl font-bold">{data.combat.totalDamage}</div>
                    <div className="text-xs text-muted-foreground">总伤害</div>
                  </div>
                  <div className="text-center p-4 rounded-lg bg-secondary">
                    <div className="text-2xl font-bold">{data.combat.hits}</div>
                    <div className="text-xs text-muted-foreground">命中次数</div>
                  </div>
                  <div className="text-center p-4 rounded-lg bg-secondary">
                    <div className="text-2xl font-bold text-red-500">{data.combat.damageTaken}</div>
                    <div className="text-xs text-muted-foreground">承受伤害</div>
                  </div>
                  <div className="text-center p-4 rounded-lg bg-secondary">
                    <div className="text-2xl font-bold">{data.combat.kills}</div>
                    <div className="text-xs text-muted-foreground">击败敌人</div>
                  </div>
                </div>

                <div className="space-y-2">
                  <div className="text-sm font-semibold">伤害来源</div>
                  {data.combat.damageBySource.map((source: any) => (
                    <div key={source.name} className="flex items-center justify-between">
                      <span className="text-sm">{source.name}</span>
                      <div className="flex items-center space-x-2">
                        <div className="w-32 h-2 bg-secondary rounded-full overflow-hidden">
                          <div
                            className="h-full bg-red-500"
                            style={{ width: `${(source.damage / data.combat.totalDamage) * 100}%` }}
                          />
                        </div>
                        <span className="text-sm text-muted-foreground">{source.damage}</span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </CardContent>
          </Card>
        )}

        {/* 时间统计 */}
        <Card>
          <CardHeader>
            <CardTitle>时间统计</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">游戏时长</span>
                <span className="font-medium">{formatDuration(data.stats.duration)}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">回合数</span>
                <span className="font-medium">{data.stats.rounds}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">检定次数</span>
                <span className="font-medium">{data.stats.totalRolls}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">战斗次数</span>
                <span className="font-medium">{data.stats.combats}</span>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

function formatDuration(seconds: number): string {
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  return `${hours}h ${minutes}m`
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `frontend/src/lib/chart-config.ts` | 创建 | 图表配置 |
| `frontend/src/components/charts/AttributeRadar.tsx` | 创建 | 属性雷达图 |
| `frontend/src/components/charts/RollHistoryChart.tsx` | 创建 | 检定统计图 |
| `frontend/src/components/charts/SanityTrendChart.tsx` | 创建 | SAN 曲线图 |
| `frontend/src/components/charts/GameDashboard.tsx` | 创建 | 仪表板组件 |
| `frontend/src/types/charts.ts` | 创建 | 图表类型定义 |

---

## 验收标准

- [ ] 雷达图正确显示属性
- [ ] 检定统计准确无误
- [ ] SAN 曲线正确显示趋势
- [ ] 图表支持响应式适配
- [ ] 暗色模式显示正常
- [ ] 图表可导出为图片
- [ ] 交互提示信息完整
- [ ] 性能良好，无卡顿

---

## 参考文档

- Chart.js 官方文档
- React-Chartjs-2 文档
- M5-001: 角色系统
- M0-009: SAN 检定

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
