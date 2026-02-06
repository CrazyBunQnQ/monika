# M0-040 定义字体层级规范

## 概述
定义 CoC 跑团平台的字体层级系统,包括标题、正文、辅助文本等不同层级的字体大小、行高、字重规范。

## 验收标准
- [ ] 定义字体大小层级(1-6 级)
- [ ] 定义行高规范
- [ ] 定义字重层级
- [ ] 定义字母间距
- [ ] 定义暗色/亮色主题差异
- [ ] 定义响应式断点调整

## 技术方案

### 字体层级

```typescript
type FontSize = 'xs' | 'sm' | 'base' | 'lg' | 'xl' | '2xl' | '3xl' | '4xl' | '5xl' | '6xl';

interface FontHierarchy {
  // 字体大小
  size: Record<FontSize, string>;

  // 行高
  lineHeight: Record<FontSize, string>;

  // 字重
  weight: {
    thin: number;
    light: number;
    regular: number;
    medium: number;
    semibold: number;
    bold: number;
  };

  // 字母间距
  letterSpacing: Record<'tighter' | 'tight' | 'normal' | 'wide' | 'wider', string>;
}

const FONT_HIERARCHY: FontHierarchy = {
  size: {
    xs: '0.75rem',    // 12px - 辅助文本
    sm: '0.875rem',   // 14px - 小号文本
    base: '1rem',     // 16px - 正文
    lg: '1.125rem',   // 18px - 大号正文
    xl: '1.25rem',    // 20px - 小标题
    '2xl': '1.5rem',  // 24px - 标题
    '3xl': '1.875rem',// 30px - 大标题
    '4xl': '2.25rem', // 36px - 页面标题
    '5xl': '3rem',    // 48px - 主标题
    '6xl': '3.75rem'  // 60px - 巨型标题
  },

  lineHeight: {
    xs: '1rem',
    sm: '1.25rem',
    base: '1.5rem',
    lg: '1.75rem',
    xl: '1.75rem',
    '2xl': '2rem',
    '3xl': '2.25rem',
    '4xl': '2.5rem',
    '5xl': '1',
    '6xl': '1'
  },

  weight: {
    thin: 100,
    light: 300,
    regular: 400,
    medium: 500,
    semibold: 600,
    bold: 700
  },

  letterSpacing: {
    tighter: '-0.05em',
    tight: '-0.025em',
    normal: '0',
    wide: '0.025em',
    wider: '0.05em'
  }
};
```

### 应用场景

```typescript
interface TextUsage {
  // 标题
  h1: { size: '4xl'; weight: 'bold'; line: 'tight' };
  h2: { size: '3xl'; weight: 'semibold'; line: 'tight' };
  h3: { size: '2xl'; weight: 'semibold'; line: 'tight' };
  h4: { size: 'xl'; weight: 'medium'; line: 'normal' };
  h5: { size: 'lg'; weight: 'medium'; line: 'normal' };
  h6: { size: 'base'; weight: 'medium'; line: 'normal' };

  // 正文
  body: { size: 'base'; weight: 'regular'; line: 'relaxed' };
  bodyLarge: { size: 'lg'; weight: 'regular'; line: 'relaxed' };
  bodySmall: { size: 'sm'; weight: 'regular'; line: 'normal' };

  // 辅助文本
  caption: { size: 'xs'; weight: 'regular'; line: 'normal' };
  overline: { size: 'xs'; weight: 'medium'; line: 'normal'; transform: 'uppercase' };

  // 代码
  code: { size: 'sm'; weight: 'regular'; family: 'monospace' };
  codeInline: { size: 'sm'; weight: 'regular'; family: 'monospace' };

  // 按钮
  button: { size: 'sm'; weight: 'medium' };
  buttonLarge: { size: 'base'; weight: 'medium' };
}

// 使用示例
const typographyStyles = {
  // 页面标题
  pageTitle: {
    fontSize: FONT_HIERARCHY.size['4xl'],
    fontWeight: FONT_HIERARCHY.weight.bold,
    lineHeight: FONT_HIERARCHY.lineHeight['4xl']
  },

  // 消息文本
  messageText: {
    fontSize: FONT_HIERARCHY.size.base,
    fontWeight: FONT_HIERARCHY.weight.regular,
    lineHeight: FONT_HIERARCHY.lineHeight.base
  },

  // 时间戳
  timestamp: {
    fontSize: FONT_HIERARCHY.size.xs,
    fontWeight: FONT_HIERARCHY.weight.regular,
    lineHeight: FONT_HIERARCHY.lineHeight.xs
  }
};
```

### 响应式调整

```typescript
interface ResponsiveFontSize {
  mobile: string;
  tablet: string;
  desktop: string;
}

const RESPONSIVE_SIZES: Record<FontSize, ResponsiveFontSize> = {
  xs: { mobile: '0.75rem', tablet: '0.75rem', desktop: '0.75rem' },
  sm: { mobile: '0.875rem', tablet: '0.875rem', desktop: '0.875rem' },
  base: { mobile: '1rem', tablet: '1rem', desktop: '1rem' },
  lg: { mobile: '1.125rem', tablet: '1.125rem', desktop: '1.125rem' },
  xl: { mobile: '1.125rem', tablet: '1.25rem', desktop: '1.25rem' },
  '2xl': { mobile: '1.25rem', tablet: '1.5rem', desktop: '1.5rem' },
  '3xl': { mobile: '1.5rem', tablet: '1.875rem', desktop: '1.875rem' },
  '4xl': { mobile: '1.875rem', tablet: '2.25rem', desktop: '2.25rem' },
  '5xl': { mobile: '2.25rem', tablet: '3rem', desktop: '3rem' },
  '6xl': { mobile: '2.75rem', tablet: '3.75rem', desktop: '3.75rem' }
};

// Tailwind CSS 配置
const tailwindConfig = {
  theme: {
    extend: {
      fontSize: {
        xs: RESPONSIVE_SIZES xs,
        sm: RESPONSIVE_SIZES.sm,
        base: RESPONSIVE_SIZES.base,
        lg: RESPONSIVE_SIZES.lg,
        xl: RESPONSIVE_SIZES.xl,
        '2xl': RESPONSIVE_SIZES['2xl'],
        '3xl': RESPONSIVE_SIZES['3xl'],
        '4xl': RESPONSIVE_SIZES['4xl'],
        '5xl': RESPONSIVE_SIZES['5xl'],
        '6xl': RESPONSIVE_SIZES['6xl']
      }
    }
  }
};
```

### 字体族

```typescript
interface FontFamily {
  sans: string[];
  serif: string[];
  mono: string[];
  display?: string[];
}

const FONT_FAMILY: FontFamily = {
  // 无衬线字体(默认)
  sans: [
    '-apple-system',
    'BlinkMacSystemFont',
    'Segoe UI',
    'Roboto',
    'Helvetica Neue',
    'Arial',
    'sans-serif'
  ],

  // 衬线字体(特殊叙事)
  serif: [
    'Georgia',
    'Cambria',
    'Times New Roman',
    'Times',
    'serif'
  ],

  // 等宽字体(代码/骰子)
  mono: [
    'SF Mono',
    'Monaco',
    'Cascadia Code',
    'Roboto Mono',
    'Courier New',
    'monospace'
  ]
};
```

### 主题适配

```typescript
interface ThemeTypography {
  light: {
    text: string;
    textSecondary: string;
    textMuted: string;
    textInverse: string;
  };
  dark: {
    text: string;
    textSecondary: string;
    textMuted: string;
    textInverse: string;
  };
}

const THEME_COLORS: ThemeTypography = {
  light: {
    text: '#212529',      // 主要文本
    textSecondary: '#6c757d',  // 次要文本
    textMuted: '#adb5bd',      // 弱化文本
    textInverse: '#ffffff'     // 反色文本
  },

  dark: {
    text: '#e9ecef',
    textSecondary: '#adb5bd',
    textMuted: '#6c757d',
    textInverse: '#212529'
  }
};
```

## 依赖关系
- 前置任务: M0-039 定义配色方案
- 被依赖: M0-041 定义间距系统, M0-042 定义消息气泡样式

## 预估工时
2h
