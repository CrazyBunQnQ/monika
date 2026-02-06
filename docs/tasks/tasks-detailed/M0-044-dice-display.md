# M0-044 定义骰子结果展示规范

## 概述
定义掷骰结果的视觉展示规范,包括骰子动画、结果样式、成功/失败标识、奖惩骰展示等。

## 验收标准
- [ ] 定义骰子动画规范
- [ ] 定义成功等级视觉标识
- [ ] 定义奖惩骰展示方式
- [ ] 定义骰子结果卡片样式
- [ ] 定义历史记录展示

## 技术方案

### 骰子类型

```typescript
type DiceType = 'd100' | 'd20' | 'd12' | 'd10' | 'd8' | 'd6' | 'd4' | 'd100x10';

type SuccessLevel =
  | 'critical'  // 大成功
  | 'extreme'   // 极难成功
  | 'hard'      // 困难成功
  | 'regular'   // 普通成功
  | 'failure'   // 失败
  | 'fumble';   // 大失败
```

### 骰子动画

```typescript
interface DiceAnimation {
  // 持续时间
  duration: number;

  // 动画类型
  type: 'shake' | 'roll' | 'spin' | 'fade';

  // 关键帧
  keyframes: {
    [key: string]: {
      transform?: string;
      opacity?: number;
      scale?: number;
    };
  };
}

const DICE_ANIMATIONS: Record<DiceType, DiceAnimation> = {
  d100: {
    duration: 1500,
    type: 'spin',
    keyframes: {
      '0%': { transform: 'rotateY(0deg) scale(0.8)', opacity: 0 },
      '50%': { transform: 'rotateY(180deg) scale(1.2)', opacity: 1 },
      '100%': { transform: 'rotateY(360deg) scale(1)', opacity: 1 }
    }
  },
  d20: {
    duration: 1200,
    type: 'roll',
    keyframes: {
      '0%': { transform: 'rotate(0deg) scale(0.5)', opacity: 0 },
      '50%': { transform: 'rotate(180deg) scale(1.1)', opacity: 1 },
      '100%': { transform: 'rotate(360deg) scale(1)', opacity: 1 }
    }
  },
  d6: {
    duration: 800,
    type: 'shake',
    keyframes: {
      '0%': { transform: 'translate(0, 0) rotate(0deg)' },
      '25%': { transform: 'translate(-5px, 5px) rotate(-5deg)' },
      '50%': { transform: 'translate(5px, -5px) rotate(5deg)' },
      '75%': { transform: 'translate(-5px, -5px) rotate(-5deg)' },
      '100%': { transform: 'translate(0, 0) rotate(0deg)' }
    }
  }
};
```

### 成功等级样式

```typescript
interface SuccessLevelStyle {
  // 颜色
  colors: {
    primary: string;
    secondary: string;
    text: string;
  };

  // 图标
  icon: string;

  // 标签
  label: string;

  // 特效
  effect?: {
    type: 'glow' | 'pulse' | 'sparkle';
    color: string;
  };
}

const SUCCESS_LEVEL_STYLES: Record<SuccessLevel, SuccessLevelStyle> = {
  critical: {
    colors: {
      primary: '#ffd700',
      secondary: '#ffecb3',
      text: '#f57f17'
    },
    icon: '⭐',
    label: '大成功',
    effect: {
      type: 'sparkle',
      color: '#ffd700'
    }
  },

  extreme: {
    colors: {
      primary: '#66bb6a',
      secondary: '#c8e6c9',
      text: '#2e7d32'
    },
    icon: '✨',
    label: '极难成功'
  },

  hard: {
    colors: {
      primary: '#42a5f5',
      secondary: '#bbdefb',
      text: '#1565c0'
    },
    icon: '🎯',
    label: '困难成功'
  },

  regular: {
    colors: {
      primary: '#78909c',
      secondary: '#cfd8dc',
      text: '#455a64'
    },
    icon: '✓',
    label: '普通成功'
  },

  failure: {
    colors: {
      primary: '#ef5350',
      secondary: '#ffcdd2',
      text: '#c62828'
    },
    icon: '✗',
    label: '失败'
  },

  fumble: {
    colors: {
      primary: '#d32f2f',
      secondary: '#ef9a9a',
      text: '#b71c1c'
    },
    icon: '💀',
    label: '大失败',
    effect: {
      type: 'pulse',
      color: '#d32f2f'
    }
  }
};
```

### 骰子结果卡片

```typescript
interface DiceResultCard {
  // 布局
  layout: {
    width: string;
    padding: string;
    borderRadius: string;
    gap: string;
  };

  // 骰子显示
  dice: {
    size: string;
    fontSize: string;
    fontWeight: string;
  };

  // 结果显示
  result: {
    fontSize: string;
    fontWeight: string;
    textAlign: 'center';
  };

  // 详情
  details: {
    fontSize: string;
    color: string;
  };
}

const DICE_RESULT_CARD: DiceResultCard = {
  layout: {
    width: '100%',
    padding: '16px',
    borderRadius: '12px',
    gap: '12px'
  },
  dice: {
    size: '64px',
    fontSize: '32px',
    fontWeight: 'bold'
  },
  result: {
    fontSize: '24px',
    fontWeight: 'bold',
    textAlign: 'center'
  },
  details: {
    fontSize: '14px',
    color: '#6c757d'
  }
};
```

### 奖惩骰展示

```typescript
interface BonusPenaltyDice {
  // 奖励骰
  bonus: {
    color: string;
    label: string;
    position: 'left' | 'right';
  };

  // 惩罚骰
  penalty: {
    color: string;
    label: string;
    position: 'left' | 'right';
  };

  // 骰子排列
  layout: {
    dice: 'horizontal' | 'vertical';
    spacing: string;
  };

  // 选中的骰子
  selected: {
    border: string;
    scale: number;
  };
}

const BONUS_PENALTY_DICE: BonusPenaltyDice = {
  bonus: {
    color: '#66bb6a',
    label: '奖励骰',
    position: 'left'
  },
  penalty: {
    color: '#ef5350',
    label: '惩罚骰',
    position: 'right'
  },
  layout: {
    dice: 'horizontal',
    spacing: '8px'
  },
  selected: {
    border: '3px solid #ffd700',
    scale: 1.1
  }
};
```

### 骰子组件

```typescript
interface DiceResultProps {
  // 掷骰数据
  diceType: DiceType;
  rolls: number[];
  selectedRoll: number;
  successLevel: SuccessLevel;

  // 检定信息
  skill?: string;
  difficulty?: number;

  // 奖惩骰
  bonusDice?: number;
  penaltyDice?: number;

  // 尺寸
  size?: 'sm' | 'md' | 'lg';
}

const DiceResult: React.FC<DiceResultProps> = ({
  diceType,
  rolls,
  selectedRoll,
  successLevel,
  skill,
  difficulty,
  bonusDice = 0,
  penaltyDice = 0,
  size = 'md'
}) => {
  const style = SUCCESS_LEVEL_STYLES[successLevel];
  const animation = DICE_ANIMATIONS[diceType];

  return (
    <div
      className={cn(
        'dice-result',
        `dice-result--${size}`,
        `dice-result--${successLevel}`
      )}
      style={{
        backgroundColor: style.colors.secondary,
        border: `2px solid ${style.colors.primary}`
      }}
    >
      {/* 奖惩骰指示 */}
      {(bonusDice > 0 || penaltyDice > 0) && (
        <div className="dice-modifiers">
          {bonusDice > 0 && (
            <span className="dice-bonus">
              +{bonusDice} {BONUS_PENALTY_DICE.bonus.label}
            </span>
          )}
          {penaltyDice > 0 && (
            <span className="dice-penalty">
              -{penaltyDice} {BONUS_PENALTY_DICE.penalty.label}
            </span>
          )}
        </div>
      )}

      {/* 骰子显示 */}
      <div className="dice-container">
        {rolls.map((roll, index) => (
          <div
            key={index}
            className={cn(
              'dice',
              roll === selectedRoll && 'dice--selected'
            )}
            style={{
              animation: `${animation.type} ${animation.duration}ms ease-out`,
              borderColor: roll === selectedRoll
                ? BONUS_PENALTY_DICE.selected.border
                : 'transparent',
              transform: roll === selectedRoll
                ? `scale(${BONUS_PENALTY_DICE.selected.scale})`
                : 'scale(1)'
            }}
          >
            {roll}
          </div>
        ))}
      </div>

      {/* 结果显示 */}
      <div className="dice-result-value">
        <div className="dice-number">{selectedRoll}</div>
        <div className="dice-label" style={{ color: style.colors.text }}>
          {style.icon} {style.label}
        </div>
      </div>

      {/* 检定详情 */}
      {(skill || difficulty) && (
        <div className="dice-details">
          {skill && <span className="dice-skill">{skill}</span>}
          {difficulty && (
            <span className="dice-difficulty">
              难度: {difficulty}
            </span>
          )}
        </div>
      )}
    </div>
  );
};
```

### CSS 动画

```css
/* 骰子旋转 */
@keyframes spin {
  0% { transform: rotateY(0deg) scale(0.8); opacity: 0; }
  50% { transform: rotateY(180deg) scale(1.2); opacity: 1; }
  100% { transform: rotateY(360deg) scale(1); opacity: 1; }
}

/* 骰子滚动 */
@keyframes roll {
  0% { transform: rotate(0deg) scale(0.5); opacity: 0; }
  50% { transform: rotate(180deg) scale(1.1); opacity: 1; }
  100% { transform: rotate(360deg) scale(1); opacity: 1; }
}

/* 骰子抖动 */
@keyframes shake {
  0%, 100% { transform: translate(0, 0) rotate(0deg); }
  25% { transform: translate(-5px, 5px) rotate(-5deg); }
  50% { transform: translate(5px, -5px) rotate(5deg); }
  75% { transform: translate(-5px, -5px) rotate(-5deg); }
}

/* 成功闪烁 */
@keyframes success-glow {
  0%, 100% { box-shadow: 0 0 5px currentColor; }
  50% { box-shadow: 0 0 20px currentColor, 0 0 30px currentColor; }
}

/* 失败脉冲 */
@keyframes failure-pulse {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.05); }
}

/* 星光特效 */
@keyframes sparkle {
  0%, 100% { opacity: 0; transform: scale(0) rotate(0deg); }
  50% { opacity: 1; transform: scale(1) rotate(180deg); }
}

.dice {
  animation-duration: var(--dice-duration);
  animation-timing-function: ease-out;
  animation-fill-mode: forwards;
}

.dice--critical {
  animation: success-glow 1s ease-in-out infinite;
}

.dice--fumble {
  animation: failure-pulse 0.5s ease-in-out infinite;
}

/* 星光粒子 */
.sparkle {
  position: absolute;
  width: 10px;
  height: 10px;
  background: #ffd700;
  border-radius: 50%;
  animation: sparkle 1s ease-in-out infinite;
}
```

### Tailwind CSS 配置

```javascript
// tailwind.config.js
module.exports = {
  theme: {
    extend: {
      colors: {
        // 成功等级颜色
        critical: '#ffd700',
        extreme: '#66bb6a',
        hard: '#42a5f5',
        regular: '#78909c',
        failure: '#ef5350',
        fumble: '#d32f2f'
      },

      animation: {
        'dice-spin': 'spin 1.5s ease-out',
        'dice-roll': 'roll 1.2s ease-out',
        'dice-shake': 'shake 0.8s ease-out',
        'success-glow': 'success-glow 1s ease-in-out infinite',
        'failure-pulse': 'failure-pulse 0.5s ease-in-out infinite',
        'sparkle': 'sparkle 1s ease-in-out infinite'
      }
    }
  }
};
```

## 依赖关系
- 前置任务: M0-039 定义配色方案
- 被依赖: M1-066 实现 DiceRoll 组件

## 预估工时
2h
