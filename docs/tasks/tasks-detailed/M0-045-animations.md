# M0-045 定义动画/过渡效果规范

## 概述
定义 CoC 跑团平台的动画和过渡效果规范,包括页面切换、组件交互、状态变化等动画效果。

## 验收标准
- [ ] 定义过渡时长规范
- [ ] 定义缓动函数库
- [ ] 定义常用动画预设
- [ ] 定义性能优化规则
- [ ] 定义可访问性考虑

## 技术方案

### 时长规范

```typescript
type Duration = 'instant' | 'fast' | 'normal' | 'slow' | 'slower';

interface DurationSpec {
  ms: number;
  css: string;
}

const DURATIONS: Record<Duration, DurationSpec> = {
  instant: { ms: 0, css: '0ms' },
  fast: { ms: 150, css: '150ms' },
  normal: { ms: 300, css: '300ms' },
  slow: { ms: 500, css: '500ms' },
  slower: { ms: 800, css: '800ms' }
};
```

### 缓动函数

```typescript
type EasingFunction =
  | 'linear'
  | 'ease'
  | 'ease-in'
  | 'ease-out'
  | 'ease-in-out'
  | 'ease-sine'
  | 'ease-cubic'
  | 'ease-bounce'
  | 'ease-elastic';

interface EasingSpec {
  css: string;
  bezier: [number, number, number, number];
}

const EASING: Record<EasingFunction, EasingSpec> = {
  linear: { css: 'linear', bezier: [0, 0, 1, 1] },
  ease: { css: 'ease', bezier: [0.25, 0.1, 0.25, 1] },
  'ease-in': { css: 'ease-in', bezier: [0.42, 0, 1, 1] },
  'ease-out': { css: 'ease-out', bezier: [0, 0, 0.58, 1] },
  'ease-in-out': { css: 'ease-in-out', bezier: [0.42, 0, 0.58, 1] },
  'ease-sine': { css: 'cubic-bezier(0.4, 0, 0.6, 1)', bezier: [0.4, 0, 0.6, 1] },
  'ease-cubic': { css: 'cubic-bezier(0.64, 0, 0.78, 0)', bezier: [0.64, 0, 0.78, 0] },
  'ease-bounce': { css: 'cubic-bezier(0.68, -0.55, 0.265, 1.55)', bezier: [0.68, -0.55, 0.265, 1.55] },
  'ease-elastic': { css: 'cubic-bezier(0.87, 0, 0.13, 1)', bezier: [0.87, 0, 0.13, 1] }
};
```

### 动画预设

```typescript
interface AnimationPreset {
  name: string;
  duration: Duration;
  easing: EasingFunction;
  keyframes: KeyframeRule;
}

type KeyframeRule = Record<string, {
  transform?: string;
  opacity?: number;
  scale?: number;
  translate?: string;
  rotate?: string;
}>;

const ANIMATION_PRESETS: Record<string, AnimationPreset> = {
  // 淡入
  fadeIn: {
    name: 'fadeIn',
    duration: 'fast',
    easing: 'ease-out',
    keyframes: {
      from: { opacity: 0 },
      to: { opacity: 1 }
    }
  },

  // 淡出
  fadeOut: {
    name: 'fadeOut',
    duration: 'fast',
    easing: 'ease-in',
    keyframes: {
      from: { opacity: 1 },
      to: { opacity: 0 }
    }
  },

  // 上滑进入
  slideUp: {
    name: 'slideUp',
    duration: 'normal',
    easing: 'ease-cubic',
    keyframes: {
      from: { transform: 'translateY(20px)', opacity: 0 },
      to: { transform: 'translateY(0)', opacity: 1 }
    }
  },

  // 下滑进入
  slideDown: {
    name: 'slideDown',
    duration: 'normal',
    easing: 'ease-cubic',
    keyframes: {
      from: { transform: 'translateY(-20px)', opacity: 0 },
      to: { transform: 'translateY(0)', opacity: 1 }
    }
  },

  // 左滑进入
  slideLeft: {
    name: 'slideLeft',
    duration: 'normal',
    easing: 'ease-cubic',
    keyframes: {
      from: { transform: 'translateX(20px)', opacity: 0 },
      to: { transform: 'translateX(0)', opacity: 1 }
    }
  },

  // 右滑进入
  slideRight: {
    name: 'slideRight',
    duration: 'normal',
    easing: 'ease-cubic',
    keyframes: {
      from: { transform: 'translateX(-20px)', opacity: 0 },
      to: { transform: 'translateX(0)', opacity: 1 }
    }
  },

  // 缩放进入
  scaleIn: {
    name: 'scaleIn',
    duration: 'normal',
    easing: 'ease-bounce',
    keyframes: {
      from: { transform: 'scale(0.9)', opacity: 0 },
      to: { transform: 'scale(1)', opacity: 1 }
    }
  },

  // 缩放退出
  scaleOut: {
    name: 'scaleOut',
    duration: 'fast',
    easing: 'ease-in',
    keyframes: {
      from: { transform: 'scale(1)', opacity: 1 },
      to: { transform: 'scale(0.9)', opacity: 0 }
    }
  },

  // 旋转进入
  rotateIn: {
    name: 'rotateIn',
    duration: 'normal',
    easing: 'ease-elastic',
    keyframes: {
      from: { transform: 'rotate(-180deg) scale(0.5)', opacity: 0 },
      to: { transform: 'rotate(0) scale(1)', opacity: 1 }
    }
  },

  // 弹跳
  bounce: {
    name: 'bounce',
    duration: 'slow',
    easing: 'ease-bounce',
    keyframes: {
      '0%': { transform: 'translateY(0)' },
      '20%': { transform: 'translateY(-20px)' },
      '40%': { transform: 'translateY(0)' },
      '60%': { transform: 'translateY(-10px)' },
      '80%': { transform: 'translateY(0)' },
      '100%': { transform: 'translateY(0)' }
    }
  },

  // 脉冲
  pulse: {
    name: 'pulse',
    duration: 'slow',
    easing: 'ease-in-out',
    keyframes: {
      '0%, 100%': { transform: 'scale(1)', opacity: 1 },
      '50%': { transform: 'scale(1.05)', opacity: 0.8 }
    }
  },

  // 闪烁
  shimmer: {
    name: 'shimmer',
    duration: 'slower',
    easing: 'linear',
    keyframes: {
      from: { backgroundPosition: '-200% 0' },
      to: { backgroundPosition: '200% 0' }
    }
  }
};
```

### 过渡效果

```typescript
interface TransitionSpec {
  property: string | string[];
  duration: Duration;
  easing: EasingFunction;
  delay?: Duration;
}

const TRANSITIONS: Record<string, TransitionSpec> = {
  // 默认过渡
  default: {
    property: 'all',
    duration: 'fast',
    easing: 'ease-out'
  },

  // 颜色过渡
  colors: {
    property: ['background-color', 'border-color', 'color'],
    duration: 'fast',
    easing: 'ease-in-out'
  },

  // 变换过渡
  transform: {
    property: 'transform',
    duration: 'normal',
    easing: 'ease-cubic'
  },

  // 透明度过渡
  opacity: {
    property: 'opacity',
    duration: 'fast',
    easing: 'ease-out'
  },

  // 阴影过渡
  shadow: {
    property: 'box-shadow',
    duration: 'normal',
    easing: 'ease-out'
  },

  // 布局过渡
  layout: {
    property: ['width', 'height', 'margin', 'padding'],
    duration: 'normal',
    easing: 'ease-cubic'
  }
};
```

### 交互动画

```typescript
// 按钮悬停
const buttonHover = {
  scale: 1.05,
  duration: 'fast' as Duration,
  easing: 'ease-out' as EasingFunction
};

// 按钮点击
const buttonActive = {
  scale: 0.95,
  duration: 'instant' as Duration,
  easing: 'ease-out' as EasingFunction
};

// 卡片悬停
const cardHover = {
  translateY: -4,
  shadow: '0 8px 16px rgba(0, 0, 0, 0.1)',
  duration: 'fast' as Duration,
  easing: 'ease-cubic' as EasingFunction
};

// 输入框聚焦
const inputFocus = {
  borderColor: '#5c6bc0',
  boxShadow: '0 0 0 3px rgba(92, 107, 192, 0.1)',
  duration: 'fast' as Duration,
  easing: 'ease-out' as EasingFunction
};
```

### 页面切换动画

```typescript
interface PageTransition {
  enter: AnimationPreset;
  exit: AnimationPreset;
}

const PAGE_TRANSITIONS: Record<string, PageTransition> = {
  // 淡入淡出
  fade: {
    enter: ANIMATION_PRESETS.fadeIn,
    exit: ANIMATION_PRESETS.fadeOut
  },

  // 滑动
  slide: {
    enter: ANIMATION_PRESETS.slideUp,
    exit: ANIMATION_PRESETS.fadeOut
  },

  // 缩放
  scale: {
    enter: ANIMATION_PRESETS.scaleIn,
    exit: ANIMATION_PRESETS.scaleOut
  }
};
```

### 性能优化

```typescript
// GPU 加速属性
const GPU_ACCELERATED = [
  'transform',
  'opacity',
  'filter'
];

// 避免动画的属性
const AVOID_ANIMATING = [
  'width',
  'height',
  'margin',
  'padding',
  'border-width',
  'top',
  'left'
];

// 减少重绘的技巧
const PERFORMANCE_TIPS = {
  // 使用 will-change
  willChange: 'transform, opacity',

  // 使用 translate3d 触发 GPU
  translate3d: 'translate3d(0, 0, 0)',

  // 避免布局抖动
  avoidLayout: true,

  // 使用 requestAnimationFrame
  raf: true
};
```

### 可访问性

```typescript
// 减少动画
const REDUCED_MOTION = {
  // 检测用户偏好
  prefersReducedMotion: window.matchMedia('(prefers-reduced-motion: reduce)').matches,

  // 禁用动画
  disable: () => {
    document.documentElement.classList.add('reduce-motion');
  },

  // 替代方案
  fallback: {
    duration: 'instant' as Duration,
    easing: 'linear' as EasingFunction
  }
};

// 可访问性最佳实践
const A11Y_GUIDELINES = {
  // 尊重用户偏好
  respectPreferences: true,

  // 提供禁用选项
  allowDisable: true,

  // 避免闪烁
  avoidFlashing: true,

  // 限制动画次数(每秒最多 4 次)
  maxPerSecond: 4
};
```

### CSS 实现

```css
/* 基础动画类 */
.animate {
  transition-duration: var(--duration);
  transition-timing-function: var(--easing);
}

/* 淡入 */
@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.fade-in {
  animation: fadeIn var(--duration) var(--easing);
}

/* 滑入 */
@keyframes slideIn {
  from {
    transform: translateY(20px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

.slide-in {
  animation: slideIn var(--duration) var(--easing);
}

/* 缩放 */
@keyframes scale {
  from { transform: scale(0.9); opacity: 0; }
  to { transform: scale(1); opacity: 1; }
}

.scale-in {
  animation: scale var(--duration) var(--easing);
}

/* 脉冲 */
@keyframes pulse {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.05); }
}

.pulse {
  animation: pulse var(--duration) ease-in-out infinite;
}

/* 减少动画 */
@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
```

### Tailwind CSS 配置

```javascript
// tailwind.config.js
module.exports = {
  theme: {
    extend: {
      transitionDuration: {
        instant: '0ms',
        fast: '150ms',
        normal: '300ms',
        slow: '500ms',
        slower: '800ms'
      },

      transitionTimingFunction: {
        'ease-sine': 'cubic-bezier(0.4, 0, 0.6, 1)',
        'ease-cubic': 'cubic-bezier(0.64, 0, 0.78, 0)',
        'ease-bounce': 'cubic-bezier(0.68, -0.55, 0.265, 1.55)',
        'ease-elastic': 'cubic-bezier(0.87, 0, 0.13, 1)'
      },

      animation: {
        'fade-in': 'fadeIn 150ms ease-out',
        'fade-out': 'fadeOut 150ms ease-in',
        'slide-up': 'slideUp 300ms cubic-bezier(0.64, 0, 0.78, 0)',
        'slide-down': 'slideDown 300ms cubic-bezier(0.64, 0, 0.78, 0)',
        'slide-left': 'slideLeft 300ms cubic-bezier(0.64, 0, 0.78, 0)',
        'slide-right': 'slideRight 300ms cubic-bezier(0.64, 0, 0.78, 0)',
        'scale-in': 'scaleIn 300ms cubic-bezier(0.68, -0.55, 0.265, 1.55)',
        'scale-out': 'scaleOut 150ms ease-in',
        'pulse': 'pulse 500ms ease-in-out infinite',
        'bounce': 'bounce 500ms cubic-bezier(0.68, -0.55, 0.265, 1.55)'
      }
    }
  }
};
```

## 依赖关系
- 前置任务: M0-039 定义配色方案
- 被依赖: 所有 UI 组件实现

## 预估工时
2h
