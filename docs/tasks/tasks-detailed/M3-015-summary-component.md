# M3-015: 实现 Summary 渲染组件

**任务ID**: M3-015
**标题**: 实现 Summary 渲染组件
**类型**: frontend (前端开发)
**预估工时**: 4h
**依赖**: M3-014, M3-001

---

## 任务描述

实现 Session 总结的前端渲染组件，将后端生成的结构化摘要以美观、易读的方式呈现给用户。组件需要支持叙事摘要、关键事件、状态变化、线索发现等多种内容的展示。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M3-015-01 | 设计 Summary 类型定义 | TypeScript 接口 | 20min |
| M3-015-02 | 实现叙事摘要组件 | NarrativeSummary | 45min |
| M3-015-03 | 实现关键事件组件 | KeyEventsList | 45min |
| M3-015-04 | 实现状态变化组件 | StateChangesPanel | 45min |
| M3-015-05 | 实现线索发现组件 | DiscoveriesPanel | 30min |
| M3-015-06 | 实现统计信息组件 | StatisticsCards | 30min |
| M3-015-07 | 实现 Summary 主容器 | SummaryViewer | 30min |
| M3-015-08 | 编写组件测试 | 单元测试 | 15min |

---

## 前端代码示例

### 类型定义

```typescript
// frontend/src/types/summary.ts

/** Session 总结 */
export interface SessionSummary {
  summary_id: string;
  session_id: string;
  created_at: string;
  updated_at: string;

  session_info: SessionInfo;
  narrative_summary: NarrativeSummary;
  key_events: KeyEvent[];
  state_changes: StateChanges;
  leads: LeadsInfo;
  promises: Promise[];
  statistics: Statistics;
  visibility: VisibilityInfo;
}

/** Session 信息 */
export interface SessionInfo {
  started_at: string;
  ended_at: string | null;
  duration_seconds: number | null;
  scene_id: string;
  scene_title: string;
}

/** 叙事摘要 */
export interface NarrativeSummary {
  brief: string;
  detailed: string;
  mood: 'calm' | 'tense' | 'horror' | 'mystery' | 'action';
  tone: string;
}

/** 关键事件 */
export interface KeyEvent {
  event_id: string;
  timestamp: string;
  type: EventType;
  title: string;
  description: string;
  participants: Participant[];
  outcome?: EventOutcome;
  related_clues: string[];
  visibility: 'public' | 'kp' | 'player:*';
}

/** 事件类型 */
export type EventType =
  | 'clue_discovered'
  | 'combat_occurred'
  | 'san_check_failed'
  | 'madness_triggered'
  | 'character_injured'
  | 'character_died'
  | 'scene_transition'
  | 'puzzle_solved'
  | 'mystery_revealed'
  | 'critical_failure';

/** 参与者 */
export interface Participant {
  user_id: string;
  character_id?: string;
  role: 'active' | 'passive' | 'witness';
}

/** 事件结果 */
export interface EventOutcome {
  success: boolean;
  description: string;
  consequences?: string[];
}

/** 状态变化 */
export interface StateChanges {
  characters: CharacterStateChange[];
  discoveries: Discovery[];
  consequences: Consequence[];
}

/** 角色状态变化 */
export interface CharacterStateChange {
  character_id: string;
  character_name: string;
  changes: CharacterChanges;
  status_changes: StatusChange[];
  skill_changes: SkillChange[];
  inventory_changes: InventoryChanges;
}

/** 角色数值变化 */
export interface CharacterChanges {
  hp: ValueChange;
  san: ValueChangeWithEvents;
  luck: ValueChange;
  mp?: ValueChange;
}

/** 数值变化 */
export interface ValueChange {
  old: number;
  new: number;
  delta: number;
}

/** 带事件的数值变化 */
export interface ValueChangeWithEvents extends ValueChange {
  events: string[];
}

/** 状态变化 */
export interface StatusChange {
  old: CharacterStatus;
  new: CharacterStatus;
  reason: string;
}

/** 角色状态 */
export type CharacterStatus =
  | 'healthy'
  | 'injured'
  | 'wounded'
  | 'critical'
  | 'unconscious'
  | 'dying'
  | 'dead'
  | 'insane'
  | 'temporary_madness'
  | 'indefinite_madness';

/** 技能变化 */
export interface SkillChange {
  skill_id: string;
  old_value: number;
  new_value: number;
  reason: 'growth' | 'injury' | 'other';
}

/** 物品变化 */
export interface InventoryChanges {
  added: string[];
  removed: string[];
  used: string[];
}

/** 发现 */
export interface Discovery {
  discovery_id: string;
  timestamp: string;
  type: DiscoveryType;
  content: DiscoveryContent;
  discoverer: Discoverer;
  visibility: 'public' | 'party' | 'private' | 'kp';
}

/** 发现类型 */
export type DiscoveryType = 'clue' | 'information' | 'item' | 'location' | 'npc_secret';

/** 发现内容 */
export interface DiscoveryContent {
  title: string;
  description: string;
  evidence?: string[];
}

/** 发现者 */
export interface Discoverer {
  user_id: string;
  character_id?: string;
}

/** 后果 */
export interface Consequence {
  consequence_id: string;
  timestamp: string;
  type: ConsequenceType;
  description: string;
  severity: ConsequenceSeverity;
  cause: Cause;
  affected: Affected;
  status: 'active' | 'resolved' | 'ongoing';
}

/** 后果类型 */
export type ConsequenceType =
  | 'injury'
  | 'san_loss'
  | 'madness'
  | 'resource_loss'
  | 'story_branch';

/** 后果严重程度 */
export type ConsequenceSeverity = 'minor' | 'moderate' | 'major' | 'critical';

/** 原因 */
export interface Cause {
  event_id: string;
  description: string;
}

/** 影响范围 */
export interface Affected {
  characters: string[];
  party: boolean;
}

/** 线索信息 */
export interface LeadsInfo {
  discovered: string[];
  resolved: string[];
  pending: string[];
}

/** 承诺 */
export interface Promise {
  description: string;
  source_event_id: string;
  status: 'pending' | 'fulfilled' | 'broken';
}

/** 统计信息 */
export interface Statistics {
  message_count: number;
  roll_count: number;
  combat_count: number;
  san_check_count: number;
  injury_count: number;
  clue_discovery_count: number;
}

/** 可见性信息 */
export interface VisibilityInfo {
  public: string[];
  kp_only: string[];
  players: Record<string, string[]>;
}
```

### 叙事摘要组件

```typescript
// frontend/src/components/summary/NarrativeSummary.tsx
import React from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { NarrativeSummary as NarrativeSummaryType } from '@/types/summary';

interface NarrativeSummaryProps {
  summary: NarrativeSummaryType;
}

const MOOD_COLORS: Record<NarrativeSummaryType['mood'], string> = {
  calm: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
  tense: 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200',
  horror: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
  mystery: 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200',
  action: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
};

const MOOD_LABELS: Record<NarrativeSummaryType['mood'], string> = {
  calm: '平静',
  tense: '紧张',
  horror: '恐怖',
  mystery: '神秘',
  action: '动作',
};

export function NarrativeSummary({ summary }: NarrativeSummaryProps) {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle>叙事摘要</CardTitle>
          <Badge className={MOOD_COLORS[summary.mood]}>
            {MOOD_LABELS[summary.mood]}
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* 简述 */}
        <div>
          <p className="text-lg font-medium leading-relaxed">
            {summary.brief}
          </p>
        </div>

        <Separator />

        {/* 详细叙述 */}
        <div className="space-y-3">
          <h4 className="font-medium text-sm text-muted-foreground">详细叙述</h4>
          <div className="prose prose-sm dark:prose-invert max-w-none">
            {summary.detailed.split('\n\n').map((paragraph, index) => (
              <p key={index} className="mb-3 last:mb-0">
                {paragraph}
              </p>
            ))}
          </div>
        </div>

        {/* 氛围描述 */}
        {summary.tone && (
          <>
            <Separator />
            <div>
              <h4 className="font-medium text-sm text-muted-foreground mb-2">
                氛围基调
              </h4>
              <p className="text-sm italic text-muted-foreground">
                {summary.tone}
              </p>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}
```

### 关键事件组件

```typescript
// frontend/src/components/summary/KeyEventsList.tsx
import React, { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { ChevronDown, ChevronUp } from 'lucide-react';
import { KeyEvent } from '@/types/summary';
import { formatDistanceToNow } from 'date-fns';
import { zhCN } from 'date-fns/locale';

interface KeyEventsListProps {
  events: KeyEvent[];
}

const EVENT_TYPE_ICONS: Record<KeyEvent['type'], string> = {
  clue_discovered: '🔍',
  combat_occurred: '⚔️',
  san_check_failed: '😱',
  madness_triggered: '🧠',
  character_injured: '🩹',
  character_died: '💀',
  scene_transition: '🚪',
  puzzle_solved: '🧩',
  mystery_revealed: '💡',
  critical_failure: '❌',
};

const EVENT_TYPE_LABELS: Record<KeyEvent['type'], string> = {
  clue_discovered: '发现线索',
  combat_occurred: '战斗',
  san_check_failed: 'SAN 检定失败',
  madness_triggered: '触发疯狂',
  character_injured: '角色受伤',
  character_died: '角色死亡',
  scene_transition: '场景转换',
  puzzle_solved: '解开谜题',
  mystery_revealed: '真相揭露',
  critical_failure: '关键失败',
};

const EVENT_TYPE_COLORS: Record<KeyEvent['type'], string> = {
  clue_discovered: 'default',
  combat_occurred: 'destructive',
  san_check_failed: 'outline',
  madness_triggered: 'secondary',
  character_injured: 'destructive',
  character_died: 'destructive',
  scene_transition: 'outline',
  puzzle_solved: 'default',
  mystery_revealed: 'default',
  critical_failure: 'destructive',
};

export function KeyEventsList({ events }: KeyEventsListProps) {
  const [expandedEvents, setExpandedEvents] = useState<Set<string>>(new Set());

  const toggleExpand = (eventId: string) => {
    setExpandedEvents(prev => {
      const newSet = new Set(prev);
      if (newSet.has(eventId)) {
        newSet.delete(eventId);
      } else {
        newSet.add(eventId);
      }
      return newSet;
    });
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>关键事件</CardTitle>
        <p className="text-sm text-muted-foreground">
          共 {events.length} 个关键事件
        </p>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {events.map((event, index) => (
            <div key={event.event_id} className="border rounded-lg p-4 space-y-3">
              {/* 事件头部 */}
              <div className="flex items-start gap-3">
                <div className="text-2xl">{EVENT_TYPE_ICONS[event.type]}</div>
                <div className="flex-1 space-y-1">
                  <div className="flex items-center gap-2 flex-wrap">
                    <Badge variant={EVENT_TYPE_COLORS[event.type] as any}>
                      {EVENT_TYPE_LABELS[event.type]}
                    </Badge>
                    <span className="text-xs text-muted-foreground">
                      #{index + 1}
                    </span>
                    <span className="text-xs text-muted-foreground">
                      {formatDistanceToNow(new Date(event.timestamp), {
                        addSuffix: true,
                        locale: zhCN,
                      })}
                    </span>
                  </div>
                  <h4 className="font-medium">{event.title}</h4>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => toggleExpand(event.event_id)}
                >
                  {expandedEvents.has(event.event_id) ? (
                    <ChevronUp className="h-4 w-4" />
                  ) : (
                    <ChevronDown className="h-4 w-4" />
                  )}
                </Button>
              </div>

              {/* 事件描述 */}
              <p className="text-sm text-muted-foreground">
                {event.description}
              </p>

              {/* 展开详情 */}
              {expandedEvents.has(event.event_id) && (
                <div className="space-y-3 pt-3 border-t">
                  {/* 参与者 */}
                  {event.participants.length > 0 && (
                    <div>
                      <h5 className="text-xs font-medium text-muted-foreground mb-2">
                        参与者
                      </h5>
                      <div className="flex flex-wrap gap-2">
                        {event.participants.map((participant, i) => (
                          <Badge key={i} variant="outline">
                            {participant.character_id || participant.user_id}
                            <span className="ml-1 text-xs opacity-70">
                              ({participant.role})
                            </span>
                          </Badge>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* 结果 */}
                  {event.outcome && (
                    <div>
                      <h5 className="text-xs font-medium text-muted-foreground mb-2">
                        结果
                      </h5>
                      <div className="text-sm">
                        <Badge
                          variant={event.outcome.success ? 'default' : 'destructive'}
                          className="mb-2"
                        >
                          {event.outcome.success ? '成功' : '失败'}
                        </Badge>
                        <p className="text-muted-foreground">
                          {event.outcome.description}
                        </p>
                        {event.outcome.consequences && event.outcome.consequences.length > 0 && (
                          <ul className="mt-2 space-y-1 text-xs text-muted-foreground">
                            {event.outcome.consequences.map((consequence, i) => (
                              <li key={i}>• {consequence}</li>
                            ))}
                          </ul>
                        )}
                      </div>
                    </div>
                  )}

                  {/* 相关线索 */}
                  {event.related_clues.length > 0 && (
                    <div>
                      <h5 className="text-xs font-medium text-muted-foreground mb-2">
                        相关线索
                      </h5>
                      <div className="flex flex-wrap gap-2">
                        {event.related_clues.map((clueId, i) => (
                          <Badge key={i} variant="secondary">
                            {clueId}
                          </Badge>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
```

### 状态变化组件

```typescript
// frontend/src/components/summary/StateChangesPanel.tsx
import React from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { TrendingUp, TrendingDown, Minus } from 'lucide-react';
import { StateChanges } from '@/types/summary';

interface StateChangesPanelProps {
  changes: StateChanges;
}

export function StateChangesPanel({ changes }: StateChangesPanelProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>状态变化</CardTitle>
      </CardHeader>
      <CardContent>
        <Tabs defaultValue="characters" className="w-full">
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="characters">角色</TabsTrigger>
            <TabsTrigger value="discoveries">发现</TabsTrigger>
            <TabsTrigger value="consequences">后果</TabsTrigger>
          </TabsList>

          <TabsContent value="characters" className="space-y-4">
            {changes.characters.length > 0 ? (
              changes.characters.map(character => (
                <CharacterStateCard key={character.character_id} character={character} />
              ))
            ) : (
              <p className="text-center text-sm text-muted-foreground py-8">
                没有角色状态变化
              </p>
            )}
          </TabsContent>

          <TabsContent value="discoveries" className="space-y-3">
            {changes.discoveries.length > 0 ? (
              changes.discoveries.map(discovery => (
                <DiscoveryItem key={discovery.discovery_id} discovery={discovery} />
              ))
            ) : (
              <p className="text-center text-sm text-muted-foreground py-8">
                没有重要发现
              </p>
            )}
          </TabsContent>

          <TabsContent value="consequences" className="space-y-3">
            {changes.consequences.length > 0 ? (
              changes.consequences.map(consequence => (
                <ConsequenceItem key={consequence.consequence_id} consequence={consequence} />
              ))
            ) : (
              <p className="text-center text-sm text-muted-foreground py-8">
                没有产生后果
              </p>
            )}
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  );
}

// 角色状态卡片
function CharacterStateCard({ character }: { character: StateChanges['characters'][0] }) {
  return (
    <div className="border rounded-lg p-4 space-y-3">
      {/* 角色名 */}
      <h4 className="font-medium">{character.character_name}</h4>

      {/* 数值变化 */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <ValueChangeItem
          label="HP"
          old={character.changes.hp.old}
          new={character.changes.hp.new}
          delta={character.changes.hp.delta}
          color="red"
        />
        <ValueChangeItem
          label="SAN"
          old={character.changes.san.old}
          new={character.changes.san.new}
          delta={character.changes.san.delta}
          color="purple"
          events={character.changes.san.events}
        />
        <ValueChangeItem
          label="幸运"
          old={character.changes.luck.old}
          new={character.changes.luck.new}
          delta={character.changes.luck.delta}
          color="yellow"
        />
        {character.changes.mp && (
          <ValueChangeItem
            label="MP"
            old={character.changes.mp.old}
            new={character.changes.mp.new}
            delta={character.changes.mp.delta}
            color="blue"
          />
        )}
      </div>

      {/* 状态变化 */}
      {character.status_changes.length > 0 && (
        <div>
          <h5 className="text-xs font-medium text-muted-foreground mb-2">状态变化</h5>
          <div className="space-y-1">
            {character.status_changes.map((change, i) => (
              <div key={i} className="text-sm flex items-center gap-2">
                <Badge variant="outline">{change.old}</Badge>
                <span>→</span>
                <Badge>{change.new}</Badge>
                <span className="text-xs text-muted-foreground">({change.reason})</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* 技能变化 */}
      {character.skill_changes.length > 0 && (
        <div>
          <h5 className="text-xs font-medium text-muted-foreground mb-2">技能变化</h5>
          <div className="space-y-1">
            {character.skill_changes.map((change, i) => (
              <div key={i} className="text-sm flex items-center gap-2">
                <span>{change.skill_id}</span>
                <Badge variant="outline">{change.old_value}</Badge>
                <span>→</span>
                <Badge>{change.new_value}</Badge>
                <span className="text-xs text-muted-foreground">
                  ({change.reason === 'growth' ? '成长' : change.reason === 'injury' ? '受伤' : '其他'})
                </span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

// 数值变化项
function ValueChangeItem({
  label,
  old,
  new: newValue,
  delta,
  color,
  events,
}: {
  label: string;
  old: number;
  new: number;
  delta: number;
  color: string;
  events?: string[];
}) {
  const Icon = delta > 0 ? TrendingUp : delta < 0 ? TrendingDown : Minus;

  return (
    <div className="text-center space-y-1">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="flex items-center justify-center gap-1">
        <span className={`text-${color}-600 dark:text-${color}-400`}>{old}</span>
        <Icon className="h-3 w-3 text-muted-foreground" />
        <span className={`text-${color}-600 dark:text-${color}-400 font-medium`}>
          {newValue}
        </span>
      </div>
      <Badge
        variant={delta > 0 ? 'default' : delta < 0 ? 'destructive' : 'secondary'}
        className="text-xs"
      >
        {delta > 0 ? '+' : ''}{delta}
      </Badge>
      {events && events.length > 0 && (
        <div className="text-xs text-muted-foreground">
          {events.length} 个事件
        </div>
      )}
    </div>
  );
}

// 发现项
function DiscoveryItem({ discovery }: { discovery: StateChanges['discoveries'][0] }) {
  const typeLabels: Record<StateChanges['discoveries'][number]['type'], string> = {
    clue: '线索',
    information: '信息',
    item: '物品',
    location: '地点',
    npc_secret: 'NPC秘密',
  };

  return (
    <div className="border rounded-lg p-3 space-y-2">
      <div className="flex items-center gap-2">
        <Badge variant="secondary">{typeLabels[discovery.type]}</Badge>
        <span className="text-xs text-muted-foreground">
          {new Date(discovery.timestamp).toLocaleString()}
        </span>
      </div>
      <h5 className="font-medium">{discovery.content.title}</h5>
      <p className="text-sm text-muted-foreground">{discovery.content.description}</p>
    </div>
  );
}

// 后果项
function ConsequenceItem({ consequence }: { consequence: StateChanges['consequences'][number] }) {
  const severityColors: Record<StateChanges['consequences'][number]['severity'], string> = {
    minor: 'default',
    moderate: 'secondary',
    major: 'destructive',
    critical: 'destructive',
  };

  return (
    <div className="border rounded-lg p-3 space-y-2">
      <div className="flex items-center gap-2">
        <Badge variant={severityColors[consequence.severity] as any}>
          {consequence.severity}
        </Badge>
        <Badge variant="outline">{consequence.type}</Badge>
        <Badge variant={consequence.status === 'active' ? 'destructive' : 'secondary'}>
          {consequence.status}
        </Badge>
      </div>
      <p className="text-sm">{consequence.description}</p>
      <div className="text-xs text-muted-foreground">
        原因: {consequence.cause.description}
      </div>
    </div>
  );
}
```

### 主容器组件

```typescript
// frontend/src/components/summary/SummaryViewer.tsx
import React from 'react';
import { NarrativeSummary } from './NarrativeSummary';
import { KeyEventsList } from './KeyEventsList';
import { StateChangesPanel } from './StateChangesPanel';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { SessionSummary } from '@/types/summary';

interface SummaryViewerProps {
  summary: SessionSummary;
}

export function SummaryViewer({ summary }: SummaryViewerProps) {
  return (
    <div className="space-y-6">
      {/* 头部信息 */}
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <h2 className="text-2xl font-bold">Session 总结</h2>
          <Badge variant="outline">{summary.session_id.slice(0, 8)}</Badge>
        </div>
        <div className="flex items-center gap-4 text-sm text-muted-foreground">
          <span>场景: {summary.session_info.scene_title}</span>
          <span>
            时间: {new Date(summary.session_info.started_at).toLocaleString()}
          </span>
          {summary.session_info.duration_seconds && (
            <span>
              时长: {Math.floor(summary.session_info.duration_seconds / 60)} 分钟
            </span>
          )}
        </div>
      </div>

      {/* 叙事摘要 */}
      <NarrativeSummary summary={summary.narrative_summary} />

      {/* 关键事件 */}
      <KeyEventsList events={summary.key_events} />

      {/* 状态变化 */}
      <StateChangesPanel changes={summary.state_changes} />

      {/* 线索和承诺 */}
      {(summary.leads.discovered.length > 0 ||
        summary.leads.resolved.length > 0 ||
        summary.leads.pending.length > 0 ||
        summary.promises.length > 0) && (
        <Card>
          <CardContent className="pt-6">
            <div className="grid md:grid-cols-2 gap-6">
              {/* 线索 */}
              <div className="space-y-3">
                <h3 className="font-semibold">线索账本</h3>
                {summary.leads.discovered.length > 0 && (
                  <div>
                    <span className="text-sm text-muted-foreground">新发现:</span>
                    <div className="flex flex-wrap gap-1 mt-1">
                      {summary.leads.discovered.map(id => (
                        <Badge key={id} variant="default">{id}</Badge>
                      ))}
                    </div>
                  </div>
                )}
                {summary.leads.resolved.length > 0 && (
                  <div>
                    <span className="text-sm text-muted-foreground">已解决:</span>
                    <div className="flex flex-wrap gap-1 mt-1">
                      {summary.leads.resolved.map(id => (
                        <Badge key={id} variant="secondary">{id}</Badge>
                      ))}
                    </div>
                  </div>
                )}
                {summary.leads.pending.length > 0 && (
                  <div>
                    <span className="text-sm text-muted-foreground">待处理:</span>
                    <div className="flex flex-wrap gap-1 mt-1">
                      {summary.leads.pending.map(id => (
                        <Badge key={id} variant="outline">{id}</Badge>
                      ))}
                    </div>
                  </div>
                )}
              </div>

              {/* 承诺 */}
              {summary.promises.length > 0 && (
                <div className="space-y-3">
                  <h3 className="font-semibold">待兑现承诺</h3>
                  <div className="space-y-2">
                    {summary.promises.map((promise, i) => (
                      <div key={i} className="border rounded p-3">
                        <p className="text-sm">{promise.description}</p>
                        <Badge
                          variant={
                            promise.status === 'pending'
                              ? 'outline'
                              : promise.status === 'fulfilled'
                              ? 'default'
                              : 'destructive'
                          }
                          className="mt-2"
                        >
                          {promise.status === 'pending'
                            ? '待兑现'
                            : promise.status === 'fulfilled'
                            ? '已兑现'
                            : '已违约'}
                        </Badge>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* 统计信息 */}
      <Card>
        <CardContent className="pt-6">
          <h3 className="font-semibold mb-4">Session 统计</h3>
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
            <StatItem label="消息" value={summary.statistics.message_count} />
            <StatItem label="检定" value={summary.statistics.roll_count} />
            <StatItem label="战斗" value={summary.statistics.combat_count} />
            <StatItem label="SAN检定" value={summary.statistics.san_check_count} />
            <StatItem label="受伤" value={summary.statistics.injury_count} />
            <StatItem label="发现线索" value={summary.statistics.clue_discovery_count} />
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function StatItem({ label, value }: { label: string; value: number }) {
  return (
    <div className="text-center">
      <div className="text-2xl font-bold">{value}</div>
      <div className="text-sm text-muted-foreground">{label}</div>
    </div>
  );
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `frontend/src/types/summary.ts` | 创建 | Summary 类型定义 |
| `frontend/src/components/summary/NarrativeSummary.tsx` | 创建 | 叙事摘要组件 |
| `frontend/src/components/summary/KeyEventsList.tsx` | 创建 | 关键事件组件 |
| `frontend/src/components/summary/StateChangesPanel.tsx` | 创建 | 状态变化组件 |
| `frontend/src/components/summary/SummaryViewer.tsx` | 创建 | Summary 主容器 |
| `frontend/src/__tests__/components/summary/SummaryViewer.test.tsx` | 创建 | 组件测试 |

---

## 验收标准

- [ ] 所有组件类型定义完整
- [ ] 叙事摘要格式正确且美观
- [ ] 关键事件展开/收起流畅
- [ ] 状态变化数值变化清晰可见
- [ ] 线索和承诺正确显示
- [ ] 统计信息准确
- [ ] 响应式布局正常
- [ ] 组件可复用且易于扩展

---

## 参考文档

- M3-014: Summary 数据结构设计
- M3-001: AI 总结服务
- shadcn/ui 组件文档
- date-fns 文档

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
