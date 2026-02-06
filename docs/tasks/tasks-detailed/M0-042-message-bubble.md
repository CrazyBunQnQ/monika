# M0-042 定义消息气泡样式

## 概述
定义游戏台消息气泡的视觉样式,包括 KP/玩家消息区分、不同类型消息的样式变体、状态指示器等。

## 验收标准
- [ ] 定义基础消息气泡样式
- [ ] 定义 KP 消息样式变体
- [ ] 定义玩家消息样式变体
- [ ] 定义系统消息样式
- [ ] 定义特殊消息类型(检定/战斗/SAN)样式
- [ ] 定义消息分组和连接样式

## 技术方案

### 消息类型

```typescript
type MessageType =
  | 'narrative'   // 叙述文本
  | 'dialogue'    // 对话
  | 'action'      // 行动声明
  | 'roll'        // 掷骰结果
  | 'system'      // 系统消息
  | 'combat'      // 战斗事件
  | 'san'         // SAN 检定
  | 'ooc';        // OOC (Out of Character)

type MessageRole = 'kp' | 'player' | 'system';
```

### 基础气泡样式

```typescript
interface MessageBubbleStyle {
  // 布局
  layout: {
    maxWidth: string;
    padding: string;
    borderRadius: string;
    marginBottom: string;
  };

  // 颜色
  colors: {
    background: string;
    text: string;
    textSecondary: string;
    border?: string;
  };

  // 阴影
  shadow?: string;

  // 动画
  animation?: {
    enter: string;
    duration: number;
  };
}

const BASE_BUBBLE_STYLE: MessageBubbleStyle = {
  layout: {
    maxWidth: '70%',
    padding: '12px 16px',
    borderRadius: '12px',
    marginBottom: '8px'
  },
  colors: {
    background: '#f1f3f5',
    text: '#212529',
    textSecondary: '#6c757d'
  },
  shadow: '0 1px 2px rgba(0, 0, 0, 0.05)',
  animation: {
    enter: 'fade-in',
    duration: 200
  }
};
```

### 角色样式变体

```typescript
interface MessageBubbleVariants {
  // KP 消息
  kp: MessageBubbleStyle;

  // 玩家消息
  player: MessageBubbleStyle;

  // 系统消息
  system: MessageBubbleStyle;
}

const MESSAGE_BUBBLE_VARIANTS: MessageBubbleVariants = {
  // KP 气泡
  kp: {
    layout: {
      maxWidth: '80%',
      padding: '12px 16px',
      borderRadius: '12px',
      marginBottom: '12px'
    },
    colors: {
      background: 'linear-gradient(135deg, #7e57c2 0%, #5e35b1 100%)',
      text: '#ffffff',
      textSecondary: 'rgba(255, 255, 255, 0.8)'
    },
    shadow: '0 2px 8px rgba(126, 87, 194, 0.3)',
    animation: {
      enter: 'slide-in-left',
      duration: 300
    }
  },

  // 玩家气泡
  player: {
    layout: {
      maxWidth: '70%',
      padding: '10px 14px',
      borderRadius: '12px 12px 2px 12px',
      marginBottom: '8px'
    },
    colors: {
      background: '#26a69a',
      text: '#ffffff',
      textSecondary: 'rgba(255, 255, 255, 0.8)'
    },
    shadow: '0 1px 4px rgba(38, 166, 154, 0.2)',
    animation: {
      enter: 'slide-in-right',
      duration: 250
    }
  },

  // 系统消息
  system: {
    layout: {
      maxWidth: '90%',
      padding: '8px 12px',
      borderRadius: '8px',
      marginBottom: '8px'
    },
    colors: {
      background: '#f8f9fa',
      text: '#6c757d',
      textSecondary: '#adb5bd',
      border: '1px solid #dee2e6'
    },
    animation: {
      enter: 'fade-in',
      duration: 200
    }
  }
};
```

### 暗色主题

```typescript
const DARK_THEME_VARIANTS: MessageBubbleVariants = {
  kp: {
    layout: MESSAGE_BUBBLE_VARIANTS.kp.layout,
    colors: {
      background: 'linear-gradient(135deg, #9575cd 0%, #7e57c2 100%)',
      text: '#ffffff',
      textSecondary: 'rgba(255, 255, 255, 0.8)'
    },
    shadow: '0 2px 8px rgba(149, 117, 205, 0.3)',
    animation: MESSAGE_BUBBLE_VARIANTS.kp.animation
  },

  player: {
    layout: MESSAGE_BUBBLE_VARIANTS.player.layout,
    colors: {
      background: '#4db6ac',
      text: '#ffffff',
      textSecondary: 'rgba(255, 255, 255, 0.8)'
    },
    shadow: '0 1px 4px rgba(77, 182, 172, 0.2)',
    animation: MESSAGE_BUBBLE_VARIANTS.player.animation
  },

  system: {
    layout: MESSAGE_BUBBLE_VARIANTS.system.layout,
    colors: {
      background: '#2d3748',
      text: '#a0aec0',
      textSecondary: '#718096',
      border: '1px solid #4a5568'
    },
    animation: MESSAGE_BUBBLE_VARIANTS.system.animation
  }
};
```

### 特殊消息样式

```typescript
interface SpecialMessageStyles {
  // 掷骰结果
  roll: {
    background: string;
    border: string;
    icon: string;
    color: string;
  };

  // 战斗事件
  combat: {
    background: string;
    border: string;
    icon: string;
    color: string;
  };

  // SAN 检定
  san: {
    background: string;
    border: string;
    icon: string;
    color: string;
  };

  // 成功/失败
  success: {
    background: string;
    border: string;
    color: string;
  };
  failure: {
    background: string;
    border: string;
    color: string;
  };
}

const SPECIAL_MESSAGE_STYLES: SpecialMessageStyles = {
  roll: {
    background: '#f8f9fa',
    border: '2px solid #5c6bc0',
    icon: '🎲',
    color: '#5c6bc0'
  },

  combat: {
    background: '#fff5f5',
    border: '2px solid #ef5350',
    icon: '⚔️',
    color: '#ef5350'
  },

  san: {
    background: '#faf5ff',
    border: '2px solid #ab47bc',
    icon: '🧠',
    color: '#ab47bc'
  },

  success: {
    background: '#f0f9ff',
    border: '2px solid #66bb6a',
    color: '#66bb6a'
  },

  failure: {
    background: '#fff5f5',
    border: '2px solid #ef5350',
    color: '#ef5350'
  }
};
```

### 消息组件结构

```typescript
interface MessageBubbleProps {
  id: string;
  type: MessageType;
  role: MessageRole;

  // 内容
  content: string;
  sender?: string;

  // 元数据
  timestamp?: string;
  metadata?: {
    skill?: string;      // 检定技能
    roll?: number;       // 掷骰结果
    difficulty?: number; // 难度
    success?: boolean;   // 是否成功
  };

  // 样式
  variant?: 'default' | 'success' | 'failure' | 'warning';
  size?: 'sm' | 'md' | 'lg';

  // 状态
  grouped?: boolean;     // 是否分组
    avatar?: string;     // 头像 URL
}

// 组件示例
const MessageBubble: React.FC<MessageBubbleProps> = ({
  type,
  role,
  content,
  sender,
  timestamp,
  metadata,
  variant = 'default',
  size = 'md',
  grouped = false,
  avatar
}) => {
  const style = getBubbleStyle(role, type, variant);

  return (
    <div
      className={cn(
        'message-bubble',
        `message-bubble--${role}`,
        `message-bubble--${type}`,
        `message-bubble--${size}`,
        grouped && 'message-bubble--grouped'
      )}
      style={style}
    >
      {avatar && <img src={avatar} alt={sender} className="message-avatar" />}

      <div className="message-content">
        {sender && <div className="message-sender">{sender}</div>}

        <div className="message-text">
          {metadata?.icon && <span className="message-icon">{metadata.icon}</span>}
          {content}
        </div>

        {metadata && <MessageMetadata metadata={metadata} />}

        {timestamp && <div className="message-timestamp">{timestamp}</div>}
      </div>
    </div>
  );
};
```

### 消息分组

```typescript
interface MessageGroup {
  // 同一发送者的连续消息
  sender: string;
  messages: MessageBubbleProps[];
}

// 分组逻辑
function groupMessages(messages: MessageBubbleProps[]): MessageGroup[] {
  const groups: MessageGroup[] = [];
  let currentGroup: MessageGroup | null = null;

  messages.forEach(message => {
    if (!currentGroup || currentGroup.sender !== message.sender) {
      currentGroup = {
        sender: message.sender,
        messages: [message]
      };
      groups.push(currentGroup);
    } else {
      currentGroup.messages.push(message);
    }
  });

  return groups;
}

// 分组样式
const GROUPED_STYLES = {
  // 第一条消息
  first: {
    borderRadius: '12px 12px 2px 12px',
    marginBottom: '2px'
  },

  // 中间消息
  middle: {
    borderRadius: '2px 12px 2px 12px',
    marginBottom: '2px'
  },

  // 最后一条消息
  last: {
    borderRadius: '2px 12px 12px 12px',
    marginBottom: '12px'
  }
};
```

### 动画定义

```typescript
const MESSAGE_ANIMATIONS = {
  'fade-in': {
    from: { opacity: 0 },
    to: { opacity: 1 },
    duration: 200
  },

  'slide-in-left': {
    from: {
      opacity: 0,
      transform: 'translateX(-20px)'
    },
    to: {
      opacity: 1,
      transform: 'translateX(0)'
    },
    duration: 300
  },

  'slide-in-right': {
    from: {
      opacity: 0,
      transform: 'translateX(20px)'
    },
    to: {
      opacity: 1,
      transform: 'translateX(0)'
    },
    duration: 250
  }
};

// CSS 动画
const cssAnimations = `
@keyframes fade-in {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes slide-in-left {
  from {
    opacity: 0;
    transform: translateX(-20px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

@keyframes slide-in-right {
  from {
    opacity: 0;
    transform: translateX(20px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

.message-bubble {
  animation-name: fade-in;
  animation-duration: 0.2s;
  animation-timing-function: ease-out;
}

.message-bubble--kp {
  animation-name: slide-in-left;
  animation-duration: 0.3s;
}

.message-bubble--player {
  animation-name: slide-in-right;
  animation-duration: 0.25s;
}
`;
```

### Tailwind CSS 配置

```javascript
// tailwind.config.js
module.exports = {
  theme: {
    extend: {
      colors: {
        // KP 颜色
        kp: {
          light: '#7e57c2',
          DEFAULT: '#5e35b1',
          dark: '#4527a0'
        },

        // 玩家颜色
        player: {
          light: '#4db6ac',
          DEFAULT: '#26a69a',
          dark: '#00897b'
        }
      },

      borderRadius: {
        'message': '12px',
        'message-tl': '12px 12px 2px 12px',
        'message-tr': '12px 12px 12px 2px',
        'message-bl': '2px 12px 12px 12px',
        'message-br': '12px 2px 12px 12px'
      }
    }
  }
};
```

## 依赖关系
- 前置任务: M0-039 定义配色方案, M0-041 定义间距系统
- 被依赖: M1-038 实现 MessageBubble 气泡组件

## 预估工时
2h
