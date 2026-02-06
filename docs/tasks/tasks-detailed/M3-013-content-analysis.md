# M3-013: 实现内容关联分析

**任务ID**: M3-013
**标题**: 实现内容关联分析
**类型**: backend + frontend (全栈开发)
**预估工时**: 5h
**依赖**: M3-001, M3-027, M3-037

---

## 任务描述

实现游戏内容的关联分析功能，自动发现和展示事件之间的关联关系，包括：
- 事件因果链分析
- 角色关系网络
- 线索关联图
- 场景切换路径
- 时间线事件聚类

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M3-013-01 | 设计关联数据结构 | 关系图和节点定义 | 30min |
| M3-013-02 | 实现事件因果链分析 | 识别因果关系 | 1h |
| M3-013-03 | 实现角色关系网络 | 角色互动分析 | 1h |
| M3-013-04 | 实现线索关联图 | 线索关系映射 | 45min |
| M3-013-05 | 实现场景路径分析 | 场景切换追踪 | 30min |
| M3-013-06 | 实现关联分析 API | 分析接口 | 30min |
| M3-013-07 | 实现关系图可视化组件 | 力导向图展示 | 1h |
| M3-013-08 | 编写分析测试 | 测试覆盖 | 15min |

---

## 后端代码示例

### 关联分析服务

```python
# app/services/association_analysis.py
from typing import List, Dict, Any, Optional, Tuple
from datetime import datetime
from collections import defaultdict, Counter
import networkx as nx

from sqlalchemy.orm import Session
from app.db.models.gamestate import GameEvent
from app.core.logger import EventLogger

class AssociationAnalysisService:
    """内容关联分析服务"""

    def __init__(self, db: Session):
        self.db = db
        self.logger = EventLogger()

    async def analyze_event_causality(
        self,
        campaign_id: str,
        start_time: Optional[datetime] = None,
        end_time: Optional[datetime] = None,
    ) -> List[Dict[str, Any]]:
        """分析事件因果链

        Args:
            campaign_id: 战役 ID
            start_time: 开始时间
            end_time: 结束时间

        Returns:
            因果链列表
        """
        # 获取事件
        events = await self.logger.get_events(
            campaign_id=campaign_id,
            start_time=start_time,
            end_time=end_time,
        )

        # 构建因果图
        graph = nx.DiGraph()

        for event in events:
            graph.add_node(
                event.id,
                type=event.type,
                description=event.description,
                timestamp=event.timestamp,
                data=event.data,
            )

        # 分析因果关系
        for i, event in enumerate(events):
            for j in range(i + 1, min(i + 10, len(events))):  # 只看后续10个事件
                next_event = events[j]

                # 检查是否有因果关系
                if self._is_causal(event, next_event):
                    graph.add_edge(
                        event.id,
                        next_event.id,
                        strength=self._calculate_causal_strength(event, next_event),
                    )

        # 提取因果链
        chains = []
        for node in graph.nodes():
            # 找到最长的路径
            try:
                predecessors = list(graph.predecessors(node))
                successors = list(graph.successors(node))

                if predecessors and successors:
                    chain = {
                        'chain_id': f'chain_{node}',
                        'events': [
                            graph.nodes[n]['description']
                            for n in nx.shortest_path(graph, predecessors[0], successors[0])
                        ],
                        'start_event': predecessors[0],
                        'end_event': successors[0],
                        'length': len(list(nx.all_simple_paths(graph, predecessors[0], successors[0]))),
                    }
                    chains.append(chain)
            except:
                continue

        return chains[:20]  # 返回前20条因果链

    async def analyze_character_network(
        self,
        campaign_id: str,
    ) -> Dict[str, Any]:
        """分析角色关系网络

        Args:
            campaign_id: 战役 ID

        Returns:
            角色关系网络
        """
        # 获取所有事件
        events = await self.logger.get_events(campaign_id=campaign_id)

        # 提取角色互动
        interactions = defaultdict(lambda: {'count': 0, 'types': Counter()})

        for event in events:
            participants = self._extract_participants(event)
            if len(participants) > 1:
                # 记录角色之间的互动
                for i in range(len(participants)):
                    for j in range(i + 1, len(participants)):
                        char1 = participants[i]
                        char2 = participants[j]

                        # 确保唯一键
                        key = tuple(sorted([char1, char2]))
                        interactions[key]['count'] += 1
                        interactions[key]['types'][event.type] += 1

        # 构建网络图
        graph = nx.Graph()
        for (char1, char2), data in interactions.items():
            graph.add_edge(
                char1,
                char2,
                weight=data['count'],
                types=dict(data['types']),
            )

        # 计算中心性指标
        centrality = nx.degree_centrality(graph)
        betweenness = nx.betweenness_centrality(graph)

        # 提取社群
        communities = list(nx.community.greedy_modularity_communities(graph))

        return {
            'nodes': [
                {
                    'id': node,
                    'centrality': centrality.get(node, 0),
                    'betweenness': betweenness.get(node, 0),
                    'community': next(
                        (i for i, comm in enumerate(communities) if node in comm),
                        -1
                    ),
                }
                for node in graph.nodes()
            ],
            'edges': [
                {
                    'source': u,
                    'target': v,
                    'weight': data['weight'],
                    'types': data['types'],
                }
                for u, v, data in graph.edges(data=True)
            ],
            'communities': [
                {'id': i, 'members': list(comm)}
                for i, comm in enumerate(communities)
            ],
        }

    async def analyze_clue_associations(
        self,
        campaign_id: str,
    ) -> Dict[str, Any]:
        """分析线索关联

        Args:
            campaign_id: 战役 ID

        Returns:
            线索关联图
        """
        # 获取所有线索相关事件
        events = await self.logger.get_events(
            campaign_id=campaign_id,
            event_types=['clue_discovered', 'clue_analyzed', 'clue_connected'],
        )

        # 构建线索关联图
        graph = nx.Graph()

        for event in events:
            clue_id = event.data.get('clue_id')
            if clue_id:
                graph.add_node(
                    clue_id,
                    type='clue',
                    description=event.description,
                    discovered_at=event.timestamp,
                )

            # 关联的线索
            related_clues = event.data.get('related_clues', [])
            for related_id in related_clues:
                graph.add_edge(
                    clue_id,
                    related_id,
                    relationship=event.data.get('relationship', 'related'),
                    event_id=event.id,
                )

        # 计算线索重要性（中心度）
        centrality = nx.degree_centrality(graph)

        # 找到关键线索（高中心度）
        key_clues = sorted(
            centrality.items(),
            key=lambda x: x[1],
            reverse=True
        )[:10]

        return {
            'nodes': [
                {
                    'id': node,
                    'description': graph.nodes[node]['description'],
                    'centrality': centrality[node],
                    'discovered_at': graph.nodes[node]['discovered_at'].isoformat(),
                }
                for node in graph.nodes()
            ],
            'edges': [
                {
                    'source': u,
                    'target': v,
                    'relationship': data['relationship'],
                    'event_id': data['event_id'],
                }
                for u, v, data in graph.edges(data=True)
            ],
            'key_clues': [
                {'id': clue, 'centrality': score}
                for clue, score in key_clues
            ],
        }

    async def analyze_scene_transitions(
        self,
        campaign_id: str,
    ) -> List[Dict[str, Any]]:
        """分析场景切换路径

        Args:
            campaign_id: 战役 ID

        Returns:
            场景路径
        """
        # 获取场景切换事件
        events = await self.logger.get_events(
            campaign_id=campaign_id,
            event_types=['scene_transition'],
        )

        # 构建场景切换路径
        paths = []
        current_path = []

        for event in events:
            from_scene = event.data.get('from_scene')
            to_scene = event.data.get('to_scene')

            if from_scene and to_scene:
                current_path.append({
                    'from': from_scene,
                    'to': to_scene,
                    'timestamp': event.timestamp.isoformat(),
                    'event_id': event.id,
                })

            # 检测路径中断（Session 结束等）
            if event.type == 'session_ended' and current_path:
                paths.append(current_path)
                current_path = []

        if current_path:
            paths.append(current_path)

        # 统计场景切换频率
        transition_counts = Counter()
        for path in paths:
            for transition in path:
                key = f"{transition['from']} -> {transition['to']}"
                transition_counts[key] += 1

        return {
            'paths': paths,
            'frequent_transitions': [
                {'transition': trans, 'count': count}
                for trans, count in transition_counts.most_common(10)
            ],
        }

    def _is_causal(self, event1: GameEvent, event2: GameEvent) -> bool:
        """判断两个事件是否有因果关系"""
        # 检查时间顺序
        if event1.timestamp >= event2.timestamp:
            return False

        # 检查是否有明确的因果类型
        causal_pairs = [
            ('clue_discovered', 'clue_analyzed'),
            ('combat_started', 'damage_dealt'),
            ('san_check_failed', 'madness_triggered'),
            ('character_injured', 'hp_changed'),
            ('scene_started', 'npc_encountered'),
        ]

        if (event1.type, event2.type) in causal_pairs:
            return True

        # 检查是否有共同的角色/物品
        participants1 = self._extract_participants(event1)
        participants2 = self._extract_participants(event2)

        if set(participants1) & set(participants2):
            return True

        return False

    def _calculate_causal_strength(self, event1: GameEvent, event2: GameEvent) -> float:
        """计算因果关系强度"""
        strength = 0.5  # 基础强度

        # 如果有明确的因果类型
        if event2.type in ['clue_analyzed', 'damage_dealt', 'madness_triggered']:
            strength += 0.3

        # 如果时间很接近
        time_diff = (event2.timestamp - event1.timestamp).total_seconds()
        if time_diff < 60:  # 1分钟内
            strength += 0.2

        return min(strength, 1.0)

    def _extract_participants(self, event: GameEvent) -> List[str]:
        """提取事件参与者"""
        participants = []

        if event.character_id:
            participants.append(event.character_id)

        # 从事件数据中提取
        data = event.data or {}
        if 'target' in data:
            participants.append(data['target'])
        if 'source' in data:
            participants.append(data['source'])

        return participants
```

### 关联分析 API

```python
# app/api/analysis.py
from fastapi import APIRouter, Depends, Query
from sqlalchemy.orm import Session
from typing import Optional
from datetime import datetime

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.association_analysis import AssociationAnalysisService

router = APIRouter(prefix="/analysis", tags=["analysis"])

@router.get("/causality")
async def get_causal_chains(
    campaign_id: str,
    start_time: Optional[str] = None,
    end_time: Optional[str] = None,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取事件因果链"""
    service = AssociationAnalysisService(db)

    chains = await service.analyze_event_causality(
        campaign_id=campaign_id,
        start_time=datetime.fromisoformat(start_time) if start_time else None,
        end_time=datetime.fromisoformat(end_time) if end_time else None,
    )

    return {"chains": chains}

@router.get("/character-network")
async def get_character_network(
    campaign_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取角色关系网络"""
    service = AssociationAnalysisService(db)

    network = await service.analyze_character_network(
        campaign_id=campaign_id,
    )

    return network

@router.get("/clue-associations")
async def get_clue_associations(
    campaign_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取线索关联图"""
    service = AssociationAnalysisService(db)

    associations = await service.analyze_clue_associations(
        campaign_id=campaign_id,
    )

    return associations

@router.get("/scene-transitions")
async def get_scene_transitions(
    campaign_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取场景切换路径"""
    service = AssociationAnalysisService(db)

    transitions = await service.analyze_scene_transitions(
        campaign_id=campaign_id,
    )

    return transitions
```

---

## 前端代码示例

### 关系图可视化组件

```typescript
// frontend/src/components/analysis/RelationshipGraph.tsx
import React, { useEffect, useRef, useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { ZoomIn, ZoomOut, Download } from 'lucide-react';

interface Node {
  id: string;
  label: string;
  centrality: number;
  community: number;
  [key: string]: any;
}

interface Edge {
  source: string;
  target: string;
  weight: number;
  [key: string]: any;
}

interface GraphData {
  nodes: Node[];
  edges: Edge[];
  communities?: Array<{ id: number; members: string[] }>;
}

interface RelationshipGraphProps {
  title: string;
  data: GraphData;
  onNodeClick?: (node: Node) => void;
}

export function RelationshipGraph({
  title,
  data,
  onNodeClick,
}: RelationshipGraphProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [scale, setScale] = useState(1);
  const [offset, setOffset] = useState({ x: 0, y: 0 });
  const [dragging, setDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });

  // 使用力导向布局
  useEffect(() => {
    if (!canvasRef.current) return;

    const canvas = canvasRef.current;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    // 简单的力导向布局算法
    const nodes = data.nodes.map(n => ({
      ...n,
      x: Math.random() * canvas.width,
      y: Math.random() * canvas.height,
      vx: 0,
      vy: 0,
    }));

    const edges = data.edges.map(e => ({
      ...e,
      sourceObj: nodes.find(n => n.id === e.source),
      targetObj: nodes.find(n => n.id === e.target),
    }));

    // 模拟物理布局
    for (let iteration = 0; iteration < 100; iteration++) {
      // 排斥力
      for (let i = 0; i < nodes.length; i++) {
        for (let j = i + 1; j < nodes.length; j++) {
          const dx = nodes[j].x - nodes[i].x;
          const dy = nodes[j].y - nodes[i].y;
          const dist = Math.sqrt(dx * dx + dy * dy) || 1;
          const force = 1000 / (dist * dist);

          nodes[i].vx -= (dx / dist) * force;
          nodes[i].vy -= (dy / dist) * force;
          nodes[j].vx += (dx / dist) * force;
          nodes[j].vy += (dy / dist) * force;
        }
      }

      // 吸引力（边）
      edges.forEach(edge => {
        if (edge.sourceObj && edge.targetObj) {
          const dx = edge.targetObj.x - edge.sourceObj.x;
          const dy = edge.targetObj.y - edge.sourceObj.y;
          const dist = Math.sqrt(dx * dx + dy * dy) || 1;
          const force = (dist - 100) * 0.01;

          edge.sourceObj.vx += (dx / dist) * force;
          edge.sourceObj.vy += (dy / dist) * force;
          edge.targetObj.vx -= (dx / dist) * force;
          edge.targetObj.vy -= (dy / dist) * force;
        }
      });

      // 更新位置
      nodes.forEach(node => {
        node.x += node.vx;
        node.y += node.vy;
        node.vx *= 0.9;
        node.vy *= 0.9;

        // 边界限制
        node.x = Math.max(20, Math.min(canvas.width - 20, node.x));
        node.y = Math.max(20, Math.min(canvas.height - 20, node.y));
      });
    }

    // 绘制
    const draw = () => {
      ctx.clearRect(0, 0, canvas.width, canvas.height);
      ctx.save();
      ctx.translate(offset.x, offset.y);
      ctx.scale(scale, scale);

      // 绘制边
      edges.forEach(edge => {
        if (edge.sourceObj && edge.targetObj) {
          ctx.beginPath();
          ctx.moveTo(edge.sourceObj.x, edge.sourceObj.y);
          ctx.lineTo(edge.targetObj.x, edge.targetObj.y);
          ctx.strokeStyle = `rgba(100, 100, 100, ${Math.min(edge.weight / 10, 1)})`;
          ctx.lineWidth = Math.min(edge.weight, 5);
          ctx.stroke();
        }
      });

      // 绘制节点
      nodes.forEach(node => {
        const radius = 10 + node.centrality * 20;
        const hue = (node.community * 60) % 360;

        ctx.beginPath();
        ctx.arc(node.x, node.y, radius, 0, Math.PI * 2);
        ctx.fillStyle = `hsl(${hue}, 70%, 60%)`;
        ctx.fill();
        ctx.strokeStyle = '#333';
        ctx.lineWidth = 2;
        ctx.stroke();

        // 绘制标签
        ctx.fillStyle = '#333';
        ctx.font = '12px sans-serif';
        ctx.textAlign = 'center';
        ctx.fillText(node.label, node.x, node.y + radius + 15);
      });

      ctx.restore();
    };

    draw();

    // 处理鼠标交互
    const handleMouseDown = (e: MouseEvent) => {
      const rect = canvas.getBoundingClientRect();
      const x = (e.clientX - rect.left - offset.x) / scale;
      const y = (e.clientY - rect.top - offset.y) / scale;

      // 检查是否点击了节点
      for (const node of nodes) {
        const dx = x - node.x;
        const dy = y - node.y;
        if (dx * dx + dy * dy < (10 + node.centrality * 20) ** 2) {
          onNodeClick?.(node);
          return;
        }
      }

      // 开始拖动画布
      setDragging(true);
      setDragStart({ x: e.clientX - offset.x, y: e.clientY - offset.y });
    };

    const handleMouseMove = (e: MouseEvent) => {
      if (dragging) {
        setOffset({
          x: e.clientX - dragStart.x,
          y: e.clientY - dragStart.y,
        });
      }
    };

    const handleMouseUp = () => {
      setDragging(false);
    };

    canvas.addEventListener('mousedown', handleMouseDown);
    canvas.addEventListener('mousemove', handleMouseMove);
    canvas.addEventListener('mouseup', handleMouseUp);

    return () => {
      canvas.removeEventListener('mousedown', handleMouseDown);
      canvas.removeEventListener('mousemove', handleMouseMove);
      canvas.removeEventListener('mouseup', handleMouseUp);
    };
  }, [data, scale, offset, dragging, dragStart, onNodeClick]);

  const handleZoomIn = () => setScale(s => Math.min(s * 1.2, 3));
  const handleZoomOut = () => setScale(s => Math.max(s / 1.2, 0.3));
  const handleExport = () => {
    // 导出图片
    const canvas = canvasRef.current;
    if (canvas) {
      const link = document.createElement('a');
      link.download = `${title}.png`;
      link.href = canvas.toDataURL();
      link.click();
    }
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle>{title}</CardTitle>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="icon" onClick={handleZoomIn}>
              <ZoomIn className="h-4 w-4" />
            </Button>
            <Button variant="outline" size="icon" onClick={handleZoomOut}>
              <ZoomOut className="h-4 w-4" />
            </Button>
            <Button variant="outline" size="icon" onClick={handleExport}>
              <Download className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <canvas
          ref={canvasRef}
          width={800}
          height={600}
          className="border rounded-lg cursor-move"
        />
        {data.communities && data.communities.length > 0 && (
          <div className="mt-4">
            <p className="text-sm font-medium mb-2">社群分组:</p>
            <div className="flex flex-wrap gap-2">
              {data.communities.map((comm, index) => (
                <Badge key={index} variant="outline">
                  社群 {index + 1}: {comm.members.length} 成员
                </Badge>
              ))}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
```

### 分析面板组件

```typescript
// frontend/src/components/analysis/AnalysisPanel.tsx
import React, { useState, useEffect } from 'react';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { RelationshipGraph } from './RelationshipGraph';
import { Card, CardContent } from '@/components/ui/card';

export function AnalysisPanel({ campaignId }: { campaignId: string }) {
  const [characterNetwork, setCharacterNetwork] = useState(null);
  const [clueAssociations, setClueAssociations] = useState(null);
  const [causalChains, setCausalChains] = useState(null);

  useEffect(() => {
    loadAnalyses();
  }, [campaignId]);

  const loadAnalyses = async () => {
    const [network, clues, chains] = await Promise.all([
      fetch(`/api/analysis/character-network?campaign_id=${campaignId}`).then(r => r.json()),
      fetch(`/api/analysis/clue-associations?campaign_id=${campaignId}`).then(r => r.json()),
      fetch(`/api/analysis/causality?campaign_id=${campaignId}`).then(r => r.json()),
    ]);

    setCharacterNetwork(network);
    setClueAssociations(clues);
    setCausalChains(chains);
  };

  const handleNodeClick = (node: any) => {
    console.log('Clicked node:', node);
    // 显示节点详情
  };

  return (
    <Tabs defaultValue="network" className="w-full">
      <TabsList className="grid w-full grid-cols-3">
        <TabsTrigger value="network">角色关系</TabsTrigger>
        <TabsTrigger value="clues">线索关联</TabsTrigger>
        <TabsTrigger value="causality">因果链</TabsTrigger>
      </TabsList>

      <TabsContent value="network">
        {characterNetwork && (
          <RelationshipGraph
            title="角色关系网络"
            data={characterNetwork}
            onNodeClick={handleNodeClick}
          />
        )}
      </TabsContent>

      <TabsContent value="clues">
        {clueAssociations && (
          <RelationshipGraph
            title="线索关联图"
            data={clueAssociations}
            onNodeClick={handleNodeClick}
          />
        )}
      </TabsContent>

      <TabsContent value="causality">
        {causalChains && (
          <Card>
            <CardContent className="pt-6">
              <h3 className="text-lg font-semibold mb-4">事件因果链</h3>
              <div className="space-y-4">
                {causalChains.chains.map((chain: any, index: number) => (
                  <div key={index} className="border-l-2 pl-4">
                    <div className="flex items-center gap-2 mb-2">
                      <span className="font-medium">链 #{index + 1}</span>
                      <span className="text-sm text-muted-foreground">
                        {chain.length} 个事件
                      </span>
                    </div>
                    <div className="space-y-1">
                      {chain.events.map((event: string, i: number) => (
                        <div key={i} className="text-sm pl-4 border-l border-muted">
                          {event}
                        </div>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        )}
      </TabsContent>
    </Tabs>
  );
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/association_analysis.py` | 创建 | 关联分析服务 |
| `app/api/analysis.py` | 创建 | 关联分析 API |
| `frontend/src/components/analysis/RelationshipGraph.tsx` | 创建 | 关系图可视化组件 |
| `frontend/src/components/analysis/AnalysisPanel.tsx` | 创建 | 分析面板组件 |
| `tests/test_association_analysis.py` | 创建 | 关联分析测试 |

---

## 验收标准

- [ ] 因果链分析能识别合理的事件关联
- [ ] 角色关系网络正确显示互动频率
- [ ] 线索关联图准确映射线索关系
- [ ] 场景路径追踪完整
- [ ] 关系图可视化流畅（缩放、拖拽）
- [ ] 节点点击交互正常
- [ ] 社群分组合理
- [ ] 导出功能正常

---

## 参考文档

- M3-001: AI 总结服务
- M3-027: 全文检索功能
- M3-037: 向量检索
- NetworkX 文档
- 力导向布局算法

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
