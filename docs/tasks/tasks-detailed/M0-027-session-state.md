# M0-027: SessionState 结构定义

**任务类型**: spec
**预估工时**: 2h
**依赖**: 无
**状态**: [ ]

---

## 子任务拆解

### 1.1 SessionState 核心结构设计 (30min)

| ID | 任务 | 预估时间 | 状态 |
|----|------|----------|------|
| M0-027-01 | [ ] 设计会话元信息字段 | 10min | [ ] |
| M0-027-02 | [ ] 设计会话状态枚举 | 10min | [ ] |
| M0-027-03 | [ ] 创建基础数据结构 | 10min | [ ] |

```typescript
// src/core/states/session-state.ts

/**
 * 会话状态枚举
 */
export enum SessionStatus {
  /** 等待开始 */
  WAITING = 'waiting',
  /** 进行中 */
  IN_PROGRESS = 'in_progress',
  /** 暂停 */
  PAUSED = 'paused',
  /** 已结束 */
  ENDED = 'ended',
}

/**
 * 角色类型枚举
 */
export enum RoleType {
  /** Keeper of Arcane Lore - 守密人/KP */
  KP = 'kp',
  /** 调查员/玩家角色 */
  PLAYER = 'player',
}

/**
 * 参与者基础信息
 */
export interface Participant {
  /** 用户ID */
  userId: string;
  /** 显示名称 */
  displayName: string;
  /** 角色类型 */
  role: RoleType;
  /** 关联的角色ID（玩家） */
  characterId?: string;
}

/**
 * 会话配置
 */
export interface SessionConfig {
  /** 是否允许推骰 */
  allowPushRoll: boolean;
  /** 是否允许使用幸运 */
  allowLuckSpend: boolean;
  /** 默认难度 */
  defaultDifficulty: number;
  /** 疯狂检定难度阈值 */
  insanityThreshold: number;
  /** 最大奖励骰数量 */
  maxBonusDice: number;
  /** 最大惩罚骰数量 */
  maxPenaltyDice: number;
}

/**
 * 会话状态 - 核心数据结构
 *
 * 包含会话的完整状态信息，用于管理整个游戏会话的生命周期
 */
export interface SessionState {
  /** 会话唯一ID */
  sessionId: string;

  /** 会话名称 */
  name: string;

  /** 当前状态 */
  status: SessionStatus;

  /** 会话配置 */
  config: SessionConfig;

  /** 参与者列表 */
  participants: Participant[];

  /** 主持人ID */
  keeperId: string;

  /** 当前场景ID */
  currentSceneId: string;

  /** 创建时间 */
  createdAt: Date;

  /** 最后更新时间 */
  updatedAt: Date;

  /** 开始时间 */
  startedAt?: Date;

  /** 结束时间 */
  endedAt?: Date;
}

/**
 * SessionState 默认配置
 */
export const DEFAULT_SESSION_CONFIG: SessionConfig = {
  allowPushRoll: true,
  allowLuckSpend: true,
  defaultDifficulty: 50,
  insanityThreshold: 99,
  maxBonusDice: 2,
  maxPenaltyDice: 2,
};

/**
 * 创建空的 SessionState
 */
export function createEmptySession(
  sessionId: string,
  name: string,
  keeperId: string
): SessionState {
  return {
    sessionId,
    name,
    status: SessionStatus.WAITING,
    config: DEFAULT_SESSION_CONFIG,
    participants: [],
    keeperId,
    currentSceneId: '',
    createdAt: new Date(),
    updatedAt: new Date(),
  };
}
```

---

### 1.2 Session 状态管理方法 (30min)

| ID | 任务 | 预估时间 | 状态 |
|----|------|----------|------|
| M0-027-04 | [ ] 实现状态转换方法 | 10min | [ ] |
| M0-027-05 | [ ] 实现参与者管理方法 | 10min | [ ] |
| M0-027-06 | [ ] 实现场景切换方法 | 10min | [ ] |

```typescript
// src/core/states/session-manager.ts

import {
  SessionState,
  SessionStatus,
  Participant,
  RoleType,
} from './session-state';

/**
 * Session 状态管理类
 */
export class SessionManager {
  private state: SessionState;

  constructor(session: SessionState) {
    this.state = session;
  }

  /**
   * 获取当前状态
   */
  getState(): Readonly<SessionState> {
    return this.state;
  }

  /**
   * 开始会话
   */
  start(): void {
    if (this.state.status !== SessionStatus.WAITING) {
      throw new Error('只能开始等待中的会话');
    }
    this.state.status = SessionStatus.IN_PROGRESS;
    this.state.startedAt = new Date();
    this.state.updatedAt = new Date();
  }

  /**
   * 暂停会话
   */
  pause(): void {
    if (this.state.status !== SessionStatus.IN_PROGRESS) {
      throw new Error('只能暂停进行中的会话');
    }
    this.state.status = SessionStatus.PAUSED;
    this.state.updatedAt = new Date();
  }

  /**
   * 恢复会话
   */
  resume(): void {
    if (this.state.status !== SessionStatus.PAUSED) {
      throw new Error('只能恢复暂停的会话');
    }
    this.state.status = SessionStatus.IN_PROGRESS;
    this.state.updatedAt = new Date();
  }

  /**
   * 结束会话
   */
  end(): void {
    if (this.state.status === SessionStatus.ENDED) {
      throw new Error('会话已经结束');
    }
    this.state.status = SessionStatus.ENDED;
    this.state.endedAt = new Date();
    this.state.updatedAt = new Date();
  }

  /**
   * 添加参与者
   */
  addParticipant(participant: Participant): void {
    // 检查是否已存在
    const exists = this.state.participants.some(
      (p) => p.userId === participant.userId
    );
    if (exists) {
      throw new Error(`用户 ${participant.userId} 已在会话中`);
    }
    this.state.participants.push(participant);
    this.state.updatedAt = new Date();
  }

  /**
   * 移除参与者
   */
  removeParticipant(userId: string): void {
    const index = this.state.participants.findIndex(
      (p) => p.userId === userId
    );
    if (index === -1) {
      throw new Error(`用户 ${userId} 不在会话中`);
    }
    this.state.participants.splice(index, 1);
    this.state.updatedAt = new Date();
  }

  /**
   * 切换当前场景
   */
  switchScene(sceneId: string): void {
    this.state.currentSceneId = sceneId;
    this.state.updatedAt = new Date();
  }

  /**
   * 检查会话是否可操作
   */
  isOperable(): boolean {
    return this.state.status === SessionStatus.IN_PROGRESS;
  }

  /**
   * 获取所有玩家
   */
  getPlayers(): Participant[] {
    return this.state.participants.filter(
      (p) => p.role === RoleType.PLAYER
    );
  }

  /**
   * 获取主持人
   */
  getKeeper(): Participant | undefined {
    return this.state.participants.find(
      (p) => p.role === RoleType.KP
    );
  }

  /**
   * 根据用户ID获取参与者
   */
  getParticipantByUserId(userId: string): Participant | undefined {
    return this.state.participants.find(
      (p) => p.userId === userId
    );
  }
}
```

---

### 1.3 序列化和反序列化 (30min)

| ID | 任务 | 预估时间 | 状态 |
|----|------|----------|------|
| M0-027-07 | [ ] 实现 toJSON 方法 | 10min | [ ] |
| M0-027-08 | [ ] 实现 fromJSON 方法 | 10min | [ ] |
| M0-027-09 | [ ] 实现状态快照方法 | 10min | [ ] |

```typescript
// src/core/states/session-serialization.ts

import {
  SessionState,
  SessionStatus,
  Participant,
  SessionConfig,
} from './session-state';

/**
 * 会话状态快照
 */
export interface SessionSnapshot {
  sessionId: string;
  name: string;
  status: SessionStatus;
  participantCount: number;
  currentSceneId: string;
  duration: number;
  createdAt: string;
  updatedAt: string;
}

/**
 * 序列化会话状态为 JSON
 */
export function serializeSession(state: SessionState): string {
  const data = {
    ...state,
    createdAt: state.createdAt.toISOString(),
    updatedAt: state.updatedAt.toISOString(),
    startedAt: state.startedAt?.toISOString() ?? null,
    endedAt: state.endedAt?.toISOString() ?? null,
  };
  return JSON.stringify(data, null, 2);
}

/**
 * 从 JSON 反序列化会话状态
 */
export function deserializeSession(json: string): SessionState {
  const data = JSON.parse(json);

  return {
    ...data,
    createdAt: new Date(data.createdAt),
    updatedAt: new Date(data.updatedAt),
    startedAt: data.startedAt ? new Date(data.startedAt) : undefined,
    endedAt: data.endedAt ? new Date(data.endedAt) : undefined,
  };
}

/**
 * 创建会话状态快照
 */
export function createSnapshot(state: SessionState): SessionSnapshot {
  const duration = state.startedAt
    ? Math.floor(
        (state.endedAt?.getTime() ?? Date.now()) - state.startedAt.getTime()
      )
    : 0;

  return {
    sessionId: state.sessionId,
    name: state.name,
    status: state.status,
    participantCount: state.participants.length,
    currentSceneId: state.currentSceneId,
    duration,
    createdAt: state.createdAt.toISOString(),
    updatedAt: state.updatedAt.toISOString(),
  };
}
```

---

## 单元测试

### 1.1 SessionState 基本测试 (20min)

```typescript
// tests/unit/core/states/session-state.test.ts

import {
  SessionState,
  SessionStatus,
  RoleType,
  createEmptySession,
  DEFAULT_SESSION_CONFIG,
} from '@/core/states/session-state';
import { SessionManager } from '@/core/states/session-manager';

describe('SessionState', () => {
  describe('createEmptySession', () => {
    it('应该创建带有正确初始值的会话', () => {
      const session = createEmptySession(
        'session-001',
        '测试剧本',
        'user-kp-001'
      );

      expect(session.sessionId).toBe('session-001');
      expect(session.name).toBe('测试剧本');
      expect(session.status).toBe(SessionStatus.WAITING);
      expect(session.keeperId).toBe('user-kp-001');
      expect(session.participants).toHaveLength(0);
      expect(session.config).toEqual(DEFAULT_SESSION_CONFIG);
    });

    it('应该创建有效的 Date 对象', () => {
      const session = createEmptySession(
        'session-002',
        '测试剧本',
        'user-kp-001'
      );

      expect(session.createdAt).toBeInstanceOf(Date);
      expect(session.updatedAt).toBeInstanceOf(Date);
    });
  });

  describe('SessionConfig 默认值', () => {
    it('应该包含正确的默认配置', () => {
      expect(DEFAULT_SESSION_CONFIG.allowPushRoll).toBe(true);
      expect(DEFAULT_SESSION_CONFIG.allowLuckSpend).toBe(true);
      expect(DEFAULT_SESSION_CONFIG.defaultDifficulty).toBe(50);
      expect(DEFAULT_SESSION_CONFIG.insanityThreshold).toBe(99);
      expect(DEFAULT_SESSION_CONFIG.maxBonusDice).toBe(2);
      expect(DEFAULT_SESSION_CONFIG.maxPenaltyDice).toBe(2);
    });
  });
});

describe('SessionManager', () => {
  const createTestSession = (): SessionManager => {
    const session = createEmptySession(
      'session-001',
      '测试剧本',
      'user-kp-001'
    );
    return new SessionManager(session);
  };

  describe('状态转换', () => {
    it('应该能正确开始会话', () => {
      const manager = createTestSession();

      manager.start();

      expect(manager.getState().status).toBe(SessionStatus.IN_PROGRESS);
      expect(manager.getState().startedAt).toBeDefined();
    });

    it('应该能正确暂停会话', () => {
      const manager = createTestSession();
      manager.start();

      manager.pause();

      expect(manager.getState().status).toBe(SessionStatus.PAUSED);
    });

    it('应该能正确恢复会话', () => {
      const manager = createTestSession();
      manager.start();
      manager.pause();

      manager.resume();

      expect(manager.getState().status).toBe(SessionStatus.IN_PROGRESS);
    });

    it('应该能正确结束会话', () => {
      const manager = createTestSession();
      manager.start();

      manager.end();

      expect(manager.getState().status).toBe(SessionStatus.ENDED);
      expect(manager.getState().endedAt).toBeDefined();
    });

    it('不应该允许从未开始状态暂停', () => {
      const manager = createTestSession();

      expect(() => manager.pause()).toThrow('只能暂停进行中的会话');
    });

    it('不应该允许重复结束会话', () => {
      const manager = createTestSession();
      manager.start();
      manager.end();

      expect(() => manager.end()).toThrow('会话已经结束');
    });
  });

  describe('参与者管理', () => {
    it('应该能添加参与者', () => {
      const manager = createTestSession();

      manager.addParticipant({
        userId: 'user-001',
        displayName: '玩家1',
        role: RoleType.PLAYER,
        characterId: 'char-001',
      });

      expect(manager.getState().participants).toHaveLength(1);
    });

    it('不应该添加重复的参与者', () => {
      const manager = createTestSession();

      manager.addParticipant({
        userId: 'user-001',
        displayName: '玩家1',
        role: RoleType.PLAYER,
      });

      expect(() =>
        manager.addParticipant({
          userId: 'user-001',
          displayName: '玩家1-重复',
          role: RoleType.PLAYER,
        })
      ).toThrow('用户 user-001 已在会话中');
    });

    it('应该能移除参与者', () => {
      const manager = createTestSession();

      manager.addParticipant({
        userId: 'user-001',
        displayName: '玩家1',
        role: RoleType.PLAYER,
      });

      manager.removeParticipant('user-001');

      expect(manager.getState().participants).toHaveLength(0);
    });

    it('应该能获取所有玩家', () => {
      const manager = createTestSession();

      manager.addParticipant({
        userId: 'user-kp',
        displayName: 'KP',
        role: RoleType.KP,
      });
      manager.addParticipant({
        userId: 'user-001',
        displayName: '玩家1',
        role: RoleType.PLAYER,
        characterId: 'char-001',
      });
      manager.addParticipant({
        userId: 'user-002',
        displayName: '玩家2',
        role: RoleType.PLAYER,
        characterId: 'char-002',
      });

      const players = manager.getPlayers();

      expect(players).toHaveLength(2);
      expect(players.every((p) => p.role === RoleType.PLAYER)).toBe(true);
    });
  });

  describe('场景切换', () => {
    it('应该能切换当前场景', () => {
      const manager = createTestSession();

      manager.switchScene('scene-002');

      expect(manager.getState().currentSceneId).toBe('scene-002');
    });
  });

  describe('isOperable', () => {
    it('进行中时应返回 true', () => {
      const manager = createTestSession();
      manager.start();

      expect(manager.isOperable()).toBe(true);
    });

    it('等待中时应返回 false', () => {
      const manager = createTestSession();

      expect(manager.isOperable()).toBe(false);
    });

    it('暂停中时应返回 false', () => {
      const manager = createTestSession();
      manager.start();
      manager.pause();

      expect(manager.isOperable()).toBe(false);
    });

    it('已结束时返回 false', () => {
      const manager = createTestSession();
      manager.start();
      manager.end();

      expect(manager.isOperable()).toBe(false);
    });
  });
});
```

---

## 验收标准

- [ ] SessionState 包含所有必需字段（sessionId, name, status, config, participants, keeperId, currentSceneId）
- [ ] SessionStatus 枚举包含所有状态（WAITING, IN_PROGRESS, PAUSED, ENDED）
- [ ] SessionManager 能正确处理状态转换
- [ ] 参与者管理功能完整（添加、移除、查询）
- [ ] 序列化和反序列化正常工作
- [ ] 所有公共方法有完整的类型注解
- [ ] 单元测试覆盖率达到 80% 以上

---

## 涉及文件

| 文件 | 操作 |
|------|------|
| `src/core/states/session-state.ts` | 创建 |
| `src/core/states/session-manager.ts` | 创建 |
| `src/core/states/session-serialization.ts` | 创建 |
| `tests/unit/core/states/session-state.test.ts` | 创建 |

---

## 参考文档

- [01-m0-spec-freeze.md - 状态字段定义](../01-m0-spec-freeze.md)
- CoC 7e 规则书 - 角色创建章节
- JSON Schema 规范
