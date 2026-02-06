# M0-046 定义响应式断点规范

## 概述
定义 CoC 跑团平台的响应式断点系统,确保在不同设备尺寸上提供良好的用户体验。

## 验收标准
- [ ] 定义断点等级
- [ ] 定义断点范围
- [ ] 定义容器查询
- [ ] 定义响应式工具类
- [ ] 定义移动优先策略

## 技术方案

### 断点等级

```typescript
type Breakpoint = 'xs' | 'sm' | 'md' | 'lg' | 'xl' | '2xl';

interface BreakpointSpec {
  // 最小宽度
  minWidth: number;

  // 典型设备
  devices: string[];

  // 容器宽度
  container?: {
    max: number;
    padding: number;
  };
}

const BREAKPOINTS: Record<Breakpoint, BreakpointSpec> = {
  xs: {
    minWidth: 0,
    devices: ['手机竖屏'],
    container: { max: 100, padding: 16 }
  },
  sm: {
    minWidth: 640,
    devices: ['手机横屏', '小平板'],
    container: { max: 640, padding: 20 }
  },
  md: {
    minWidth: 768,
    devices: ['平板竖屏'],
    container: { max: 768, padding: 24 }
  },
  lg: {
    minWidth: 1024,
    devices: ['平板横屏', '小笔记本'],
    container: { max: 1024, padding: 32 }
  },
  xl: {
    minWidth: 1280,
    devices: ['桌面'],
    container: { max: 1280, padding: 40 }
  },
  '2xl': {
    minWidth: 1536,
    devices: ['大屏桌面'],
    container: { max: 1536, padding: 48 }
  }
};
```

### 媒体查询

```typescript
// 生成媒体查询
function mediaQuery(breakpoint: Breakpoint): string {
  const spec = BREAKPOINTS[breakpoint];
  return `@media (min-width: ${spec.minWidth}px)`;
}

// 向下查询
function mediaMaxQuery(breakpoint: Breakpoint): string {
  const spec = BREAKPOINTS[breakpoint];
  return `@media (max-width: ${spec.minWidth - 1}px)`;
}

// 范围查询
function mediaRangeQuery(min: Breakpoint, max: Breakpoint): string {
  const minSpec = BREAKPOINTS[min];
  const maxSpec = BREAKPOINTS[max];
  return `@media (min-width: ${minSpec.minWidth}px) and (max-width: ${maxSpec.minWidth - 1}px)`;
}
```

### 响应式工具

```typescript
interface ResponsiveValue<T> {
  xs?: T;
  sm?: T;
  md?: T;
  lg?: T;
  xl?: T;
  '2xl'?: T;
  default: T;
}

// 获取响应式值
function getResponsiveValue<T>(
  value: ResponsiveValue<T>,
  currentBreakpoint: Breakpoint
): T {
  const breakpointOrder: Breakpoint[] = ['xs', 'sm', 'md', 'lg', 'xl', '2xl'];
  const currentIndex = breakpointOrder.indexOf(currentBreakpoint);

  // 从当前断点向下查找
  for (let i = currentIndex; i >= 0; i--) {
    const bp = breakpointOrder[i];
    if (value[bp] !== undefined) {
      return value[bp]!;
    }
  }

  return value.default;
}

// 示例: 响应式字体大小
const responsiveFontSize: ResponsiveValue<string> = {
  xs: '0.875rem',
  sm: '1rem',
  md: '1.125rem',
  lg: '1.25rem',
  xl: '1.5rem',
  default: '1rem'
};
```

### 容器查询

```typescript
// 容器断点
const CONTAINER_BREAKPOINTS = {
  sm: '640px',
  md: '768px',
  lg: '1024px',
  xl: '1280px'
};

// 容器查询 CSS
const containerQueryCSS = `
@container (min-width: ${CONTAINER_BREAKPOINTS.sm}) {
  .container-component {
    /* 小容器样式 */
  }
}

@container (min-width: ${CONTAINER_BREAKPOINTS.md}) {
  .container-component {
    /* 中容器样式 */
  }
}

@container (min-width: ${CONTAINER_BREAKPOINTS.lg}) {
  .container-component {
    /* 大容器样式 */
  }
}
`;
```

### 移动优先策略

```typescript
// 移动优先: 从小到大编写样式
const mobileFirstCSS = `
/* 基础样式 (移动端) */
.component {
  padding: 16px;
  font-size: 14px;
}

/* 平板及以上 */
@media (min-width: 768px) {
  .component {
    padding: 24px;
    font-size: 16px;
  }
}

/* 桌面及以上 */
@media (min-width: 1024px) {
  .component {
    padding: 32px;
    font-size: 18px;
  }
}
`;
```

### 响应式布局

```typescript
// 游戏台响应式布局
const gameConsoleLayout = {
  // 移动端
  xs: {
    main: 'column',
    chat: {
      width: '100%',
      height: '60vh'
    },
    sidebar: {
      width: '100%',
      height: '40vh'
    }
  },

  // 平板
  md: {
    main: 'row',
    chat: {
      width: '60%',
      height: '100%'
    },
    sidebar: {
      width: '40%',
      height: '100%'
    }
  },

  // 桌面
  lg: {
    main: 'row',
    chat: {
      width: '70%',
      height: '100%'
    },
    sidebar: {
      width: '30%',
      height: '100%'
    }
  }
};

// 角色卡响应式布局
const characterCardLayout = {
  xs: {
    grid: '1fr',
    gap: '16px'
  },
  md: {
    grid: 'repeat(2, 1fr)',
    gap: '20px'
  },
  lg: {
    grid: 'repeat(3, 1fr)',
    gap: '24px'
  }
};
```

### 隐藏和显示

```typescript
// 响应式显示
const displayUtilities = {
  // 移动端隐藏,其他显示
  'hidden-mobile': {
    xs: 'none',
    sm: 'block',
    md: 'block',
    lg: 'block',
    xl: 'block'
  },

  // 平板及以上隐藏
  'hidden-tablet-up': {
    xs: 'block',
    sm: 'block',
    md: 'none',
    lg: 'none',
    xl: 'none'
  },

  // 桌面及以上隐藏
  'hidden-desktop-up': {
    xs: 'block',
    sm: 'block',
    md: 'block',
    lg: 'none',
    xl: 'none'
  }
};
```

### Tailwind CSS 配置

```javascript
// tailwind.config.js
module.exports = {
  theme: {
    screens: {
      'xs': '0px',
      'sm': '640px',
      'md': '768px',
      'lg': '1024px',
      'xl': '1280px',
      '2xl': '1536px'
    },

    extend: {
      // 容器查询
      containers: {
        'sm': '640px',
        'md': '768px',
        'lg': '1024px',
        'xl': '1280px'
      },

      // 响应式间距
      spacing: {
        'xs': '16px',
        'sm': '20px',
        'md': '24px',
        'lg': '32px',
        'xl': '40px'
      }
    }
  },

  // 移动优先
  variants: {
    extend: {
      display: ['xs', 'sm', 'md', 'lg', 'xl', '2xl']
    }
  }
};
```

### JavaScript 检测

```typescript
// 当前断点检测
function getCurrentBreakpoint(): Breakpoint {
  const width = window.innerWidth;

  if (width < 640) return 'xs';
  if (width < 768) return 'sm';
  if (width < 1024) return 'md';
  if (width < 1280) return 'lg';
  if (width < 1536) return 'xl';
  return '2xl';
}

// 断点变化监听
function onBreakpointChange(callback: (bp: Breakpoint) => void): () => void {
  let currentBp = getCurrentBreakpoint();

  const handler = () => {
    const newBp = getCurrentBreakpoint();
    if (newBp !== currentBp) {
      currentBp = newBp;
      callback(currentBp);
    }
  };

  window.addEventListener('resize', handler);

  // 返回清理函数
  return () => window.removeEventListener('resize', handler);
}

// React Hook
function useBreakpoint(): Breakpoint {
  const [breakpoint, setBreakpoint] = React.useState<Breakpoint>(getCurrentBreakpoint);

  React.useEffect(() => {
    return onBreakpointChange(setBreakpoint);
  }, []);

  return breakpoint;
}
```

### 最佳实践

```typescript
// 响应式设计最佳实践
const RESPONSIVE_GUIDELINES = {
  // 移动优先
  mobileFirst: true,

  // 触控友好(最小 44x44px)
  touchTarget: {
    minSize: 44,
    spacing: 8
  },

  // 字体缩放
  fontScaling: {
    min: 14,
    max: 18
  },

  // 图片响应式
  images: {
    formats: ['webp', 'jpg'],
    sizes: [640, 768, 1024, 1280, 1536]
  },

  // 性能优化
  performance: {
    lazyLoad: true,
    debounceMs: 150
  }
};
```

## 依赖关系
- 前置任务: M0-039 定义配色方案
- 被依赖: M1-109-M1-112 响应式布局实现

## 预估工时
2h
