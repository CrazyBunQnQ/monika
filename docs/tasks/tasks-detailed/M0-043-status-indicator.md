# M0-043 定义状态指示器样式

## 概述
定义游戏台中的状态指示器样式,包括 HP/SAN/Luck 进度条、状态标志、伤害指示等。

## 验收标准
- [ ] 定义进度条样式
- [ ] 定义状态标志样式
- [ ] 定义颜色语义(健康/危险/警告)
- [ ] 定义动画效果
- [ ] 定义响应式尺寸
- [ ] 定义可访问性

## 技术方案

### 状态指示器类型

```typescript
type StatusIndicatorType =
  | 'progress'   // 进度条
  | 'badge'      // 徽章
  | 'icon'       // 图标
  | 'text'       // 文本
  | 'meter';     // 仪表盘

type StatusLevel = 'critical' | 'warning' | 'normal' | 'good';
```

### 进度条样式

```typescript
interface ProgressBarStyle {
  // 尺寸
  size: {
    height: string;
    width: string;
    borderRadius: string;
  };

  // 颜色
  colors: {
    background: string;
    fill: string;
    text: string;
  };

  // 动画
  animation?: {
    duration: number;
    easing: string;
  };

  // 标签
  label: {
    show: boolean;
    position: 'top' | 'bottom' | 'inside';
    format: 'percentage' | 'value' | 'both';
  };
}

const PROGRESS_BAR_STYLES: Record<string, ProgressBarStyle> = {
  // HP 进度条
  hp: {
    size: {
      height: '24px',
      width: '100%',
      borderRadius: '12px'
    },
    colors: {
      background: '#e9ecef',
      fill: '#66bb6a',
      text: '#212529'
    },
    animation: {
      duration: 300,
      easing: 'ease-out'
    },
    label: {
      show: true,
      position: 'inside',
      format: 'both'
    }
  },

  // SAN 进度条
  san: {
    size: {
      height: '20px',
      width: '100%',
      borderRadius: '10px'
    },
    colors: {
      background: '#f3e5f5',
      fill: '#ab47bc',
      text: '#4a148c'
    },
    animation: {
      duration: 300,
      easing: 'ease-out'
    },
    label: {
      show: true,
      position: 'top',
      format: 'value'
    }
  },

  // Luck 进度条
  luck: {
    size: {
      height: '16px',
      width: '100%',
      borderRadius: '8px'
    },
    colors: {
      background: '#fff8e1',
      fill: '#ffa726',
      text: '#212529'
    },
    animation: {
      duration: 200,
      easing: 'ease-out'
    },
    label: {
      show: true,
      position: 'bottom',
      format: 'value'
    }
  }
};
```

### 状态级别颜色

```typescript
const STATUS_LEVEL_COLORS = {
  // 危急级 (红色)
  critical: {
    light: '#ef5350',
    DEFAULT: '#e53935',
    dark: '#c62828',
    contrast: '#ffffff'
  },

  // 警告级 (橙色)
  warning: {
    light: '#ffa726',
    DEFAULT: '#fb8c00',
    dark: '#ef6c00',
    contrast: '#212529'
  },

  // 正常级 (绿色)
  normal: {
    light: '#66bb6a',
    DEFAULT: '#43a047',
    dark: '#2e7d32',
    contrast: '#ffffff'
  },

  // 良好级 (蓝色)
  good: {
    light: '#42a5f5',
    DEFAULT: '#1e88e5',
    dark: '#1565c0',
    contrast: '#ffffff'
  }
};

// HP 阈值
const HP_THRESHOLDS = {
  critical: 0.25,    // < 25% - 红色
  warning: 0.5,      // < 50% - 橙色
  normal: 0.75,      // < 75% - 绿色
  good: 1.0          // >= 75% - 蓝色
};

// 获取 HP 颜色
function getHPColor(current: number, max: number): string {
  const ratio = current / max;

  if (ratio <= HP_THRESHOLDS.critical) {
    return STATUS_LEVEL_COLORS.critical.DEFAULT;
  } else if (ratio <= HP_THRESHOLDS.warning) {
    return STATUS_LEVEL_COLORS.warning.DEFAULT;
  } else if (ratio <= HP_THRESHOLDS.normal) {
    return STATUS_LEVEL_COLORS.normal.DEFAULT;
  } else {
    return STATUS_LEVEL_COLORS.good.DEFAULT;
  }
}
```

### 状态徽章

```typescript
interface StatusBadgeStyle {
  // 尺寸
  size: {
    height: string;
    padding: string;
    fontSize: string;
    borderRadius: string;
  };

  // 颜色
  colors: {
    background: string;
    text: string;
    border?: string;
  };

  // 图标
  icon?: string;
}

const STATUS_BADGE_STYLES: Record<string, StatusBadgeStyle> = {
  // 活跃
  alive: {
    size: {
      height: '24px',
      padding: '4px 12px',
      fontSize: '12px',
      borderRadius: '12px'
    },
    colors: {
      background: '#e8f5e9',
      text: '#2e7d32'
    },
    icon: '❤️'
  },

  // 昏迷
  unconscious: {
    size: {
      height: '24px',
      padding: '4px 12px',
      fontSize: '12px',
      borderRadius: '12px'
    },
    colors: {
      background: '#fff3e0',
      text: '#ef6c00'
    },
    icon: '😵'
  },

  // 濒死
  dying: {
    size: {
      height: '24px',
      padding: '4px 12px',
      fontSize: '12px',
      borderRadius: '12px'
    },
    colors: {
      background: '#ffebee',
      text: '#c62828'
    },
    icon: '💀'
  },

  // 死亡
  dead: {
    size: {
      height: '24px',
      padding: '4px 12px',
      fontSize: '12px',
      borderRadius: '12px'
    },
    colors: {
      background: '#212529',
      text: '#ffffff'
    },
    icon: '☠️'
  },

  // 疯狂
  insane: {
    size: {
      height: '24px',
      padding: '4px 12px',
      fontSize: '12px',
      borderRadius: '12px'
    },
    colors: {
      background: '#f3e5f5',
      text: '#7b1fa2'
    },
    icon: '🤪'
  }
};
```

### 状态指示器组件

```typescript
interface StatusIndicatorProps {
  type: StatusIndicatorType;
  level: StatusLevel;

  // 进度条专用
  value?: number;
  max?: number;

  // 徽章专用
  status?: string;
  icon?: string;

  // 尺寸
  size?: 'sm' | 'md' | 'lg';

  // 动画
  animated?: boolean;

  // 可访问性
  ariaLabel?: string;
}

// 进度条组件
const ProgressBar: React.FC<{
  value: number;
  max: number;
  type: 'hp' | 'san' | 'luck';
  size?: 'sm' | 'md' | 'lg';
}> = ({ value, max, type, size = 'md' }) => {
  const style = PROGRESS_BAR_STYLES[type];
  const percentage = Math.round((value / max) * 100);
  const color = type === 'hp' ? getHPColor(value, max) : style.colors.fill;

  return (
    <div className="progress-bar" style={{ width: style.size.width }}>
      {style.label.show && style.label.position === 'top' && (
        <div className="progress-label">
          {type.toUpperCase()}: {value}/{max}
        </div>
      )}

      <div
        className={cn(
          'progress-track',
          `progress-track--${size}`
        )}
        style={{
          height: style.size.height,
          borderRadius: style.size.radius,
          backgroundColor: style.colors.background
        }}
      >
        <div
          className={cn(
            'progress-fill',
            `progress-fill--${size}`,
            'animated' && 'progress-fill--animated'
          )}
          style={{
            width: `${percentage}%`,
            backgroundColor: color,
            transition: `width ${style.animation?.duration}ms ${style.animation?.easing}`
          }}
        >
          {style.label.show && style.label.position === 'inside' && (
            <span className="progress-text">
              {style.label.format === 'percentage'
                ? `${percentage}%`
                : style.label.format === 'value'
                ? `${value}/${max}`
                : `${percentage}% (${value}/${max})`}
            </span>
          )}
        </div>
      </div>

      {style.label.show && style.label.position === 'bottom' && (
        <div className="progress-label">
          {value}/{max}
        </div>
      )}
    </div>
  );
};

// 状态徽章组件
const StatusBadge: React.FC<{
  status: 'alive' | 'unconscious' | 'dying' | 'dead' | 'insane';
  size?: 'sm' | 'md' | 'lg';
}> = ({ status, size = 'md' }) => {
  const style = STATUS_BADGE_STYLES[status];

  return (
    <div
      className={cn(
        'status-badge',
        `status-badge--${status}`,
        `status-badge--${size}`
      )}
      style={{
        height: style.size.height,
        padding: style.size.padding,
        fontSize: style.size.fontSize,
        borderRadius: style.size.borderRadius,
        backgroundColor: style.colors.background,
        color: style.colors.text
      }}
    >
      {style.icon && <span className="status-icon">{style.icon}</span>}
      <span className="status-text">{status}</span>
    </div>
  );
};
```

### 动画效果

```typescript
const STATUS_ANIMATIONS = {
  // 进度条填充动画
  progressFill: {
    keyframes: `
      @keyframes progress-fill {
        from { width: 0; }
        to { width: var(--progress); }
      }
    `,
    className: 'progress-fill--animated'
  },

  // 伤害闪烁
  damageFlash: {
    keyframes: `
      @keyframes damage-flash {
        0%, 100% { opacity: 1; }
        50% { opacity: 0.5; background-color: #ef5350; }
      }
    `,
    className: 'damage-flash'
  },

  // 状态脉冲
  statusPulse: {
    keyframes: `
      @keyframes status-pulse {
        0%, 100% { transform: scale(1); }
        50% { transform: scale(1.1); }
      }
    `,
    className: 'status-pulse'
  }
};
```

### 响应式尺寸

```typescript
const RESPONSIVE_SIZES = {
  sm: {
    height: '16px',
    fontSize: '11px',
    padding: '2px 8px'
  },
  md: {
    height: '20px',
    fontSize: '12px',
    padding: '4px 12px'
  },
  lg: {
    height: '24px',
    fontSize: '14px',
    padding: '6px 16px'
  }
};
```

### Tailwind CSS 配置

```javascript
// tailwind.config.js
module.exports = {
  theme: {
    extend: {
      colors: {
        // 状态颜色
        critical: {
          light: '#ef5350',
          DEFAULT: '#e53935',
          dark: '#c62828'
        },
        warning: {
          light: '#ffa726',
          DEFAULT: '#fb8c00',
          dark: '#ef6c00'
        },
        success: {
          light: '#66bb6a',
          DEFAULT: '#43a047',
          dark: '#2e7d32'
        },
        info: {
          light: '#42a5f5',
          DEFAULT: '#1e88e5',
          dark: '#1565c0'
        }
      },

      animation: {
        'progress-fill': 'progress-fill 0.3s ease-out',
        'damage-flash': 'damage-flash 0.5s ease-in-out',
        'status-pulse': 'status-pulse 1s ease-in-out infinite'
      }
    }
  }
};
```

## 依赖关系
- 前置任务: M0-039 定义配色方案
- 被依赖: M1-044 实现状态数值展示

## 预估工时
2h
