# M1-044: 实现 CharacterSheet 角色卡组件

**任务ID**: M1-044
**标题**: 实现 CharacterSheet 角色卡组件
**类型**: frontend (前端开发)
**预估工时**: 2.5h
**依赖**: M1-019

---

## 任务描述

实现角色卡显示组件，包括属性查看、技能列表、装备管理等。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-044-01 | 设计角色卡布局 | UI 设计 | 25min |
| M1-044-02 | 实现属性显示 | 属性面板 | 30min |
| M1-044-03 | 实现技能列表 | 技能展示 | 30min |
| M1-044-04 | 实现装备栏 | 装备显示 | 25min |
| M1-044-05 | 实现物品栏 | 物品管理 | 30min |
| M1-044-06 | 实现编辑功能 | 编辑模式 | 25min |
| M1-044-07 | 实现导出功能 | 导出PDF | 10min |

---

## 角色卡组件

```tsx
// frontend/src/components/game/CharacterSheet.tsx
import { useState } from 'react'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Edit, Download } from 'lucide-react'
import type { Character } from '@/types/character'
import { AttributesPanel } from './character/AttributesPanel'
import { SkillsPanel } from './character/SkillsPanel'
import { EquipmentPanel } from './character/EquipmentPanel'
import { InventoryPanel } from './character/InventoryPanel'

interface CharacterSheetProps {
  character: Character
  onUpdate?: (updates: Partial<Character>) => void
  readonly?: boolean
}

export function CharacterSheet({
  character,
  onUpdate,
  readonly = false,
}: CharacterSheetProps) {
  const { user } = useAuth()
  const [editing, setEditing] = useState(false)
  const [activeTab, setActiveTab] = useState('attributes')

  const isOwner = user?.id === character.user_id
  const isKp = user?.role === 'kp'
  const canEdit = !readonly && (isOwner || isKp)

  const handleExport = async () => {
    // 导出为 PDF
    const response = await fetch(`/api/characters/${character.id}/export`, {
      method: 'POST',
    })
    const blob = await response.blob()
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${character.name}_character_sheet.pdf`
    a.click()
  }

  return (
    <div className="space-y-4">
      {/* 头部 */}
      <div className="flex items-center justify-between">
        <div className="space-y-1">
          <h2 className="text-2xl font-bold">{character.name}</h2>
          <p className="text-sm text-muted-foreground">
            {character.occupation} • {character.age}岁
          </p>
        </div>

        <div className="flex items-center space-x-2">
          {canEdit && (
            <Button
              size="sm"
              variant="outline"
              onClick={() => setEditing(!editing)}
            >
              <Edit className="h-4 w-4 mr-1" />
              {editing ? '完成' : '编辑'}
            </Button>
          )}
          <Button
            size="sm"
            variant="outline"
            onClick={handleExport}
          >
            <Download className="h-4 w-4 mr-1" />
            导出
          </Button>
        </div>
      </div>

      {/* 快速状态 */}
      <div className="grid grid-cols-3 gap-4">
        <Card>
          <CardContent className="p-4 text-center">
            <div className="text-2xl font-bold text-red-500">
              {character.status?.hp || 0}
            </div>
            <div className="text-xs text-muted-foreground">HP</div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4 text-center">
            <div className="text-2xl font-bold text-blue-500">
              {character.status?.mp || 0}
            </div>
            <div className="text-xs text-muted-foreground">MP</div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4 text-center">
            <div className="text-2xl font-bold text-purple-500">
              {character.status?.san || 0}
            </div>
            <div className="text-xs text-muted-foreground">SAN</div>
          </CardContent>
        </Card>
      </div>

      {/* 标签页 */}
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="attributes">属性</TabsTrigger>
          <TabsTrigger value="skills">技能</TabsTrigger>
          <TabsTrigger value="equipment">装备</TabsTrigger>
          <TabsTrigger value="inventory">物品</TabsTrigger>
        </TabsList>

        <TabsContent value="attributes" className="mt-4">
          <AttributesPanel
            attributes={character.attributes || {}}
            editing={editing}
            onUpdate={(attrs) => onUpdate?.({ attributes: attrs })}
          />
        </TabsContent>

        <TabsContent value="skills" className="mt-4">
          <SkillsPanel
            skills={character.skills || []}
            editing={editing}
            onUpdate={(skills) => onUpdate?.({ skills })}
          />
        </TabsContent>

        <TabsContent value="equipment" className="mt-4">
          <EquipmentPanel
            equipment={character.equipment || {}}
            editing={editing}
            onUpdate={(equipment) => onUpdate?.({ equipment })}
          />
        </TabsContent>

        <TabsContent value="inventory" className="mt-4">
          <InventoryPanel
            inventory={character.inventory || []}
            editing={editing}
            onUpdate={(inventory) => onUpdate?.({ inventory })}
          />
        </TabsContent>
      </Tabs>

      {/* 背景信息 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">背景故事</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm whitespace-pre-wrap">
            {character.backstory || '暂无背景故事'}
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
```

---

## 属性面板

```tsx
// frontend/src/components/game/character/AttributesPanel.tsx
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'

interface AttributesPanelProps {
  attributes: Record<string, number>
  editing?: boolean
  onUpdate?: (attrs: Record<string, number>) => void
}

const ATTRIBUTES = [
  { key: 'str', name: '力量', short: 'STR' },
  { key: 'con', name: '体质', short: 'CON' },
  { key: 'siz', name: '体型', short: 'SIZ' },
  { key: 'dex', name: '敏捷', short: 'DEX' },
  { key: 'app', name: '外貌', short: 'APP' },
  { key: 'int', name: '智力', short: 'INT' },
  { key: 'pow', name: '意志', short: 'POW' },
  { key: 'edu', name: '教育', short: 'EDU' },
]

export function AttributesPanel({
  attributes,
  editing = false,
  onUpdate,
}: AttributesPanelProps) {
  const handleUpdate = (key: string, value: number) => {
    onUpdate?.({ ...attributes, [key]: value })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">核心属性</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-4 gap-4">
          {ATTRIBUTES.map((attr) => {
            const value = attributes[attr.key] || 0
            const derived = Math.floor(value / 10)

            return (
              <div key={attr.key} className="text-center space-y-2">
                <div className="text-xs text-muted-foreground">
                  {attr.short}
                </div>
                <div className="text-2xl font-bold">
                  {editing ? (
                    <Input
                      type="number"
                      value={value}
                      onChange={(e) => handleUpdate(attr.key, parseInt(e.target.value) || 0)}
                      className="w-16 text-center"
                      min={0}
                      max={100}
                    />
                  ) : (
                    value
                  )}
                </div>
                <div className="flex items-center justify-center space-x-1">
                  <span className="text-xs text-muted-foreground">{attr.name}</span>
                  {attr.key !== 'siz' && attr.key !== 'edu' && (
                    <Badge variant="secondary" className="text-xs">
                      {derived}
                    </Badge>
                  )}
                </div>
              </div>
            )
          })}
        </div>

        {/* 衍生属性说明 */}
        <div className="mt-4 pt-4 border-t">
          <div className="text-xs text-muted-foreground">
            括号内数字为衍生属性值
          </div>
          <div className="grid grid-cols-2 gap-2 mt-2 text-xs">
            <div>HP: {(attributes.con || 0 + attributes.siz || 0) / 10}</div>
            <div>MP: {attributes.pow || 0 / 5}</div>
            <div>SAN: {attributes.pow || 0}</div>
            <div>幸运: {attributes.pow || 0 * 5}</div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
```

---

## 技能面板

```tsx
// frontend/src/components/game/character/SkillsPanel.tsx
import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'

interface SkillsPanelProps {
  skills: Array<{ name: string; value: number; category: string }>
  editing?: boolean
  onUpdate?: (skills: Array<{ name: string; value: number; category: string }>) => void
}

export function SkillsPanel({
  skills,
  editing = false,
  onUpdate,
}: SkillsPanelProps) {
  const [filter, setFilter] = useState('all')

  const categories = ['all', ...Array.from(new Set(skills.map(s => s.category)))]

  const filteredSkills = filter === 'all'
    ? skills
    : skills.filter(s => s.category === filter)

  const handleUpdate = (index: number, value: number) => {
    const newSkills = [...skills]
    newSkills[index].value = value
    onUpdate?.(newSkills)
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">技能</CardTitle>
      </CardHeader>
      <CardContent>
        {/* 分类筛选 */}
        <div className="flex flex-wrap gap-2 mb-4">
          {categories.map(cat => (
            <Badge
              key={cat}
              variant={filter === cat ? 'default' : 'outline'}
              className="cursor-pointer"
              onClick={() => setFilter(cat)}
            >
              {cat === 'all' ? '全部' : cat}
            </Badge>
          ))}
        </div>

        {/* 技能列表 */}
        <div className="space-y-2">
          {filteredSkills.map((skill, index) => (
            <div
              key={index}
              className="flex items-center justify-between p-2 rounded hover:bg-muted"
            >
              <span className="flex-1">{skill.name}</span>
              <Badge variant="secondary" className="mr-2">
                {skill.category}
              </Badge>
              {editing ? (
                <Input
                  type="number"
                  value={skill.value}
                  onChange={(e) => handleUpdate(index, parseInt(e.target.value) || 0)}
                  className="w-20 text-center"
                  min={0}
                  max={100}
                />
              ) : (
                <span className="font-mono w-20 text-center">{skill.value}</span>
              )}
            </div>
          ))}
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
| `frontend/src/components/game/CharacterSheet.tsx` | 创建 | 角色卡主组件 |
| `frontend/src/components/game/character/AttributesPanel.tsx` | 创建 | 属性面板 |
| `frontend/src/components/game/character/SkillsPanel.tsx` | 创建 | 技能面板 |
| `frontend/src/components/game/character/EquipmentPanel.tsx` | 创建 | 装备面板 |
| `frontend/src/components/game/character/InventoryPanel.tsx` | 创建 | 物品面板 |

---

## 验收标准

- [ ] 角色卡布局合理
- [ ] 属性显示正确
- [ ] 技能列表完整
- [ ] 装备栏准确
- [ ] 编辑功能可用
- [ ] 导出功能正常

---

## 参考文档

- M1-003: 角色卡数据模型
- M1-019: StatePanel 组件

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
