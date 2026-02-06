# M0-026 编写循环引用检测规则

## 概述
定义场景包中循环引用的检测规则,防止无限循环导致的问题,如场景 A 跳转到 B,B 又跳转回 A 的死循环。

## 验收标准
- [ ] 定义场景跳转循环检测
- [ ] 定义 NPC 关系循环检测(如 A 的父级是 B,B 的父级是 A)
- [ ] 定义线索链循环检测
- [ ] 检测自引用(如场景跳转到自己)
- [ ] 定义最大深度限制
- [ ] 生成循环路径报告
- [ ] 提供循环修复建议

## 技术方案

### 循环类型定义

```typescript
interface CircularReference {
  type: 'scene_transition' | 'npc_relationship' | 'clue_chain' | 'self_reference';
  path: string[];
  detected: boolean;
  severity: 'error' | 'warning';
}

interface CycleDetectionRule {
  type: string;
  field: string;
  maxDepth: number;
  detectSelf: boolean;
}
```

### 场景跳转循环检测

```typescript
function detectSceneTransitionCycles(data: any): CircularReference[] {
  const cycles: CircularReference[] = [];
  const scenes = data.scenes || {};
  const visited = new Set<string>();
  const path: string[] = [];

  // DFS 检测循环
  function dfs(sceneId: string, depth: number): boolean {
    if (depth > 20) {
      cycles.push({
        type: 'scene_transition',
        path: [...path, sceneId, '...(depth limit)'],
        detected: true,
        severity: 'error'
      });
      return false;
    }

    if (path.includes(sceneId)) {
      // 找到循环
      const cycleStart = path.indexOf(sceneId);
      const cyclePath = [...path.slice(cycleStart), sceneId];
      cycles.push({
        type: 'scene_transition',
        path: cyclePath,
        detected: true,
        severity: cyclePath.length === 1 ? 'warning' : 'error' // 自引用为警告
      });
      return true;
    }

    if (visited.has(sceneId)) {
      return false;
    }

    visited.add(sceneId);
    path.push(sceneId);

    const scene = scenes[sceneId];
    if (scene?.transitions) {
      for (const transition of scene.transitions) {
        if (transition.target) {
          dfs(transition.target, depth + 1);
        }
        if (transition.choices) {
          for (const choice of transition.choices) {
            if (choice.target) {
              dfs(choice.target, depth + 1);
            }
          }
        }
        if (transition.random) {
          for (const random of transition.random) {
            if (random.target) {
              dfs(random.target, depth + 1);
            }
          }
        }
      }
    }

    path.pop();
    return false;
  }

  // 从所有场景开始检测
  Object.keys(scenes).forEach(sceneId => {
    dfs(sceneId, 0);
  });

  return cycles;
}
```

### NPC 关系循环检测

```typescript
function detectNPCRelationshipCycles(data: any): CircularReference[] {
  const cycles: CircularReference[] = [];
  const npcs = data.shared?.npcs || {};

  // 构建关系图
  const graph: Record<string, string> = {};

  Object.entries(npcs).forEach(([npcId, npc]: [string, any]) => {
    if (npc.relationships) {
      npc.relationships.forEach((rel: any) => {
        if (rel.type === 'parent' || rel.type === 'partner') {
          graph[npcId] = rel.target;
        }
      });
    }
  });

  // 检测循环
  const visited = new Set<string>();
  const path: string[] = [];

  function dfs(npcId: string): boolean {
    if (path.includes(npcId)) {
      const cycleStart = path.indexOf(npcId);
      cycles.push({
        type: 'npc_relationship',
        path: [...path.slice(cycleStart), npcId],
        detected: true,
        severity: 'error'
      });
      return true;
    }

    if (visited.has(npcId) || !graph[npcId]) {
      return false;
    }

    visited.add(npcId);
    path.push(npcId);

    dfs(graph[npcId]);

    path.pop();
    return false;
  }

  Object.keys(npcs).forEach(npcId => dfs(npcId));

  return cycles;
}
```

### 自引用检测

```typescript
function detectSelfReferences(data: any): CircularReference[] {
  const selfRefs: CircularReference[] = [];

  // 场景跳转到自己
  Object.entries(data.scenes || {}).forEach(([sceneId, scene]: [string, any]) => {
    if (scene.transitions) {
      scene.transitions.forEach((transition: any, idx: number) => {
        if (transition.target === sceneId) {
          selfRefs.push({
            type: 'self_reference',
            path: [sceneId, `transitions[${idx}]`, sceneId],
            detected: true,
            severity: 'warning' // 自引用可能是故意的
          });
        }
      });
    }
  });

  // NPC 自己的亲属
  Object.entries(data.shared?.npcs || {}).forEach(([npcId, npc]: [string, any]) => {
    if (npc.relationships) {
      npc.relationships.forEach((rel: any, idx: number) => {
        if (rel.target === npcId) {
          selfRefs.push({
            type: 'self_reference',
            path: [npcId, `relationships[${idx}]`, npcId],
            detected: true,
            severity: 'error'
          });
        }
      });
    }
  });

  return selfRefs;
}
```

### 深度限制检测

```typescript
function detectDepthExceeded(data: any, maxDepth: number = 10): CircularReference[] {
  const violations: CircularReference[] = [];

  // 检测嵌套过深的结构
  function checkDepth(obj: any, currentPath: string[], depth: number) {
    if (depth > maxDepth) {
      violations.push({
        type: 'scene_transition',
        path: [...currentPath, '...(too deep)'],
        detected: true,
        severity: 'warning'
      });
      return;
    }

    if (typeof obj !== 'object' || obj === null) {
      return;
    }

    if (Array.isArray(obj)) {
      obj.forEach((item, idx) => {
        checkDepth(item, [...currentPath, `[${idx}]`], depth + 1);
      });
    } else {
      Object.entries(obj).forEach(([key, value]) => {
        checkDepth(value, [...currentPath, key], depth + 1);
      });
    }
  }

  checkDepth(data, [], 0);

  return violations;
}
```

### 检测报告

```typescript
interface CycleDetectionReport {
  hasCycles: boolean;
  cycles: CircularReference[];
  summary: {
    byType: Record<string, number>;
    bySeverity: Record<string, number>;
  };
  suggestions: string[];
}

function generateCycleReport(cycles: CircularReference[]): CycleDetectionReport {
  const summary = {
    byType: {} as Record<string, number>,
    bySeverity: {} as Record<string, number>
  };

  cycles.forEach(cycle => {
    summary.byType[cycle.type] = (summary.byType[cycle.type] || 0) + 1;
    summary.bySeverity[cycle.severity] = (summary.bySeverity[cycle.severity] || 0) + 1;
  });

  const suggestions: string[] = [];

  if (summary.byType['scene_transition'] > 0) {
    suggestions.push('考虑在跳转链中添加结束场景或条件退出');
  }

  if (summary.byType['self_reference'] > 0) {
    suggestions.push('自引用可能导致死循环,请确认是否为预期行为');
  }

  return {
    hasCycles: cycles.length > 0,
    cycles,
    summary,
    suggestions
  };
}
```

## 依赖关系
- 前置任务: M0-025 编写引用完整性校验
- 被依赖: 无

## 预估工时
2h
