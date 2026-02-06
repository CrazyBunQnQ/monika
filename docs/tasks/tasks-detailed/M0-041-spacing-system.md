# M0-041 定义间距系统

## 概述
定义基于 4px 网格的间距系统,确保整个 UI 的间距一致性和视觉协调性。

## 验收标准
- [ ] 定义基础间距单位(4px 网格)
- [ ] 定义间距比例
- [ ] 定义间距命名
- [ ] 定义响应式间距
- [ ] 定义特殊场景间距

## 技术方案

### 间距系统

```typescript
type SpacingScale =
  | '0'
  | 'px'
  | '0.5'
  | '1'
  | '1.5'
  | '2'
  | '2.5'
  | '3'
  | '3.5'
  | '4'
  | '5'
  | '6'
  | '7'
  | '8'
  | '9'
  | '10'
  | '11'
  | '12'
  | '14'
  | '16'
  | '20'
  | '24'
  | '28'
  | '32'
  | '36'
  | '40'
  | '44'
  | '48'
  | '52'
  | '56'
  | '60'
  | '64'
  | '72'
  | '80'
  | '96';

interface SpacingSystem {
  // 间距值(单位: rem, 1rem = 16px)
  spacing: Record<SpacingScale, string>;

  // 间距值(像素)
  spacingPx: Record<SpacingScale, number>;
}

const SPACING_SYSTEM: SpacingSystem = {
  spacing: {
    '0': '0',
    'px': '1px',
    '0.5': '0.125rem',   // 2px
    '1': '0.25rem',      // 4px
    '1.5': '0.375rem',   // 6px
    '2': '0.5rem',       // 8px
    '2.5': '0.625rem',   // 10px
    '3': '0.75rem',      // 12px
    '3.5': '0.875rem',   // 14px
    '4': '1rem',         // 16px
    '5': '1.25rem',      // 20px
    '6': '1.5rem',       // 24px
    '7': '1.75rem',      // 28px
    '8': '2rem',         // 32px
    '9': '2.25rem',      // 36px
    '10': '2.5rem',      // 40px
    '11': '2.75rem',     // 44px
    '12': '3rem',        // 48px
    '14': '3.5rem',      // 56px
    '16': '4rem',        // 64px
    '20': '5rem',        // 80px
    '24': '6rem',        // 96px
    '28': '7rem',        // 112px
    '32': '8rem',        // 128px
    '36': '9rem',        // 144px
    '40': '10rem',       // 160px
    '44': '11rem',       // 176px
    '48': '12rem',       // 192px
    '52': '13rem',       // 208px
    '56': '14rem',       // 224px
    '60': '15rem',       // 240px
    '64': '16rem',       // 256px
    '72': '18rem',       // 288px
    '80': '20rem',       // 320px
    '96': '24rem'        // 384px
  },

  spacingPx: {
    '0': 0,
    'px': 1,
    '0.5': 2,
    '1': 4,
    '1.5': 6,
    '2': 8,
    '2.5': 10,
    '3': 12,
    '3.5': 14,
    '4': 16,
    '5': 20,
    '6': 24,
    '7': 28,
    '8': 32,
    '9': 36,
    '10': 40,
    '11': 44,
    '12': 48,
    '14': 56,
    '16': 64,
    '20': 80,
    '24': 96,
    '28': 112,
    '32': 128,
    '36': 144,
    '40': 160,
    '44': 176,
    '48': 192,
    '52': 208,
    '56': 224,
    '60': 240,
    '64': 256,
    '72': 288,
    '80': 320,
    '96': 384
  }
};
```

### 常用间距语义

```typescript
interface SemanticSpacing {
  // 内边距
  padding: {
    xs: SpacingScale;
    sm: SpacingScale;
    md: SpacingScale;
    lg: SpacingScale;
    xl: SpacingScale;
  };

  // 外边距
  margin: {
    xs: SpacingScale;
    sm: SpacingScale;
    md: SpacingScale;
    lg: SpacingScale;
    xl: SpacingScale;
  };

  // 组件间距
  gap: {
    xs: SpacingScale;
    sm: SpacingScale;
    md: SpacingScale;
    lg: SpacingScale;
    xl: SpacingScale;
  };
}

const SEMANTIC_SPACING: SemanticSpacing = {
  padding: {
    xs: '2',   // 8px - 紧凑
    sm: '3',   // 12px - 小
    md: '4',   // 16px - 中等
    lg: '6',   // 24px - 大
    xl: '8'    // 32px - 超大
  },

  margin: {
    xs: '1',   // 4px - 紧凑
    sm: '2',   // 8px - 小
    md: '4',   // 16px - 中等
    lg: '6',   // 24px - 大
    xl: '8'    // 32px - 超大
  },

  gap: {
    xs: '2',   // 8px - 紧凑
    sm: '3',   // 12px - 小
    md: '4',   // 16px - 中等
    lg: '6',   // 24px - 大
    xl: '8'    // 32px - 超大
  }
};
```

### 应用示例

```typescript
// 组件间距规范
const componentSpacing = {
  // 按钮内边距
  button: {
    sm: { padding: '2 4' },    // 8px 16px
    md: { padding: '3 6' },    // 12px 24px
    lg: { padding: '4 8' }     // 16px 32px
  },

  // 卡片内边距
  card: {
    sm: { padding: '4' },      // 16px
    md: { padding: '6' },      // 24px
    lg: { padding: '8' }       // 32px
  },

  // 消息气泡
  messageBubble: {
    padding: '4',             // 16px
    marginBottom: '3'         // 12px
  },

  // 面板间距
  panel: {
    headerPadding: '4 6',     // 16px 24px
    bodyPadding: '6',         // 24px
    footerPadding: '4 6'      // 16px 24px
  },

  // 表单
  form: {
    labelMargin: '1',         // 4px
    inputPadding: '3 4',      // 12px 16px
    groupGap: '4'             // 16px
  }
};
```

### 响应式间距

```typescript
interface ResponsiveSpacing {
  mobile: SpacingScale;
  tablet: SpacingScale;
  desktop: SpacingScale;
}

const RESPONSIVE_SPACING: Record<string, ResponsiveSpacing> = {
  container: {
    mobile: '4',   // 16px
    tablet: '6',   // 24px
    desktop: '8'   // 32px
  },

  section: {
    mobile: '6',   // 24px
    tablet: '10',  // 40px
    desktop: '16'  // 64px
  },

  gap: {
    mobile: '3',   // 12px
    tablet: '4',   // 16px
    desktop: '6'   // 24px
  }
};
```

### 特殊场景

```typescript
// 游戏元素间距
const gameSpacing = {
  // 骰子结果
  dice: {
    gap: '2',      // 骰子之间 8px
    padding: '4'   // 容器内边距 16px
  },

  // 战斗追踪器
  combat: {
    rowGap: '2',   // 行间距 8px
    colGap: '4',   // 列间距 16px
    padding: '4'   // 单元格内边距 16px
  },

  // 角色卡
  character: {
    sectionGap: '6',   // 区域间距 24px
    fieldGap: '3',     // 字段间距 12px
    padding: '6'       // 容器内边距 24px
  }
};
```

### Tailwind CSS 配置

```javascript
// tailwind.config.js
module.exports = {
  theme: {
    extend: {
      spacing: {
        '0': '0',
        'px': '1px',
        '0.5': '0.125rem',   // 2px
        '1': '0.25rem',      // 4px
        '1.5': '0.375rem',   // 6px
        '2': '0.5rem',       // 8px
        '2.5': '0.625rem',   // 10px
        '3': '0.75rem',      // 12px
        '3.5': '0.875rem',   // 14px
        '4': '1rem',         // 16px
        '5': '1.25rem',      // 20px
        '6': '1.5rem',       // 24px
        '7': '1.75rem',      // 28px
        '8': '2rem',         // 32px
        '9': '2.25rem',      // 36px
        '10': '2.5rem',      // 40px
        '11': '2.75rem',     // 44px
        '12': '3rem',        // 48px
        '14': '3.5rem',      // 56px
        '16': '4rem',        // 64px
        '20': '5rem',        // 80px
        '24': '6rem',        // 96px
        '28': '7rem',        // 112px
        '32': '8rem',        // 128px
        '36': '9rem',        // 144px
        '40': '10rem',       // 160px
        '44': '11rem',       // 176px
        '48': '12rem',       // 192px
        '52': '13rem',       // 208px
        '56': '14rem',       // 224px
        '60': '15rem',       // 240px
        '64': '16rem',       // 256px
        '72': '18rem',       // 288px
        '80': '20rem',       // 320px
        '96': '24rem'        // 384px
      }
    }
  }
};
```

### 间距使用指南

```typescript
// 间距选择指南
const spacingGuide = {
  // 视觉层次
  hierarchy: {
    // 紧密关联的元素
    tight: '1-2',  // 4-8px

    // 相关元素
    close: '2-4',  // 8-16px

    // 一般间距
    normal: '4-6', // 16-24px

    // 区域分隔
    loose: '6-12', // 24-48px

    // 大区域分隔
    loosest: '12-24' // 48-96px
  },

  // 常见模式
  patterns: {
    // 按钮内边距
    button: {
      height: {
        sm: '8',   // 32px - padding 3 + 3 + border + text
        md: '10',  // 40px
        lg: '12'   // 48px
      }
    },

    // 卡片
    card: {
      padding: '6',     // 24px
      gap: '4',         // 16px
      marginBottom: '6' // 24px
    },

    // 列表
    list: {
      itemPadding: '4',     // 16px
      itemGap: '3',         // 12px
      sectionGap: '8'       // 32px
    }
  }
};
```

## 依赖关系
- 前置任务: M0-040 定义字体层级规范
- 被依赖: M0-042 定义消息气泡样式

## 预估工时
2h
