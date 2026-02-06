# M6-017: 完善越界拒绝模板

**任务ID**: M6-017
**标题**: 完善越界拒绝模板
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M1-050

---

## 任务描述

完善越界请求的拒绝响应模板，提供友好的错误提示和替代建议，引导玩家正确游戏。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-017-01 | 分析越界场景 | 确定常见越界请求 | 20min |
| M6-017-02 | 设计拒绝模板结构 | 标准化响应格式 | 25min |
| M6-017-03 | 编写常见拒绝模板 | 分类模板 | 45min |
| M6-017-04 | 实现模板引擎 | 模板渲染 | 20min |
| M6-017-05 | 实现建议生成 | 替代行动建议 | 20min |
| M6-017-06 | 编写模板测试 | 验证响应质量 | 15min |
| M6-017-07 | 编写模板文档 | 供 KP 扩展参考 | 10min |

---

## 拒绝模板结构

```typescript
interface RefusalTemplate {
  // 模板 ID
  template_id: string;

  // 拒绝原因分类
  category: RefusalCategory;

  // 拒绝内容
  refusal: {
    tone: 'polite' | 'firm' | 'humorous' | 'dramatic';
    message: string;           // 拒绝消息
    explanation?: string;      // 原因说明
  };

  // 替代方案
  alternatives: {
    count: number;
    suggestions: AlternativeSuggestion[];
  };

  // 引导
  guidance?: {
    next_action?: string;      // 建议下一步行动
    prompt?: string;           // 引导性提示
  };

  // 条件
  conditions?: {
    show_alternatives?: boolean;
    show_explanation?: boolean;
    tone?: RefusalTone;
  };
}

type RefusalCategory =
  | 'out_of_scope'         // 超出游戏范围
  | 'not_allowed'          // 规则不允许
  | 'missing_context'      // 缺少上下文
  | 'invalid_command'      // 无效命令
  | 'wrong_timing'         // 错误时机
  | 'requires_kp'          // 需要 KP 操作
  | 'character_limit'      // 角色限制
  | 'scene_constraint';    // 场景约束

interface AlternativeSuggestion {
  action: string;          // 建议行动
  description: string;     // 描述
  example?: string;        // 示例命令
  icon?: string;           // 图标
}

type RefusalTone =
  | 'polite'               // 礼貌
  | 'mysterious'           // 神秘
  | 'dramatic'             // 戏剧化
  | 'helpful'              // 乐于助人
  | 'neutral';             // 中立
```

---

## 常见拒绝模板

```python
# app/templates/refusals.py

REFUSAL_TEMPLATES = {
    # === 超出游戏范围 ===
    "out_of_scope_modern": {
        "category": "out_of_scope",
        "refusal": {
            "tone": "polite",
            "message": "我理解你想了解这个话题，但在 CoC 跑团中，我们专注于 1920 年代的调查员故事。",
            "explanation": "现代科技、当代事件等超出了游戏的时间背景。"
        },
        "alternatives": {
            "suggestions": [
                {
                    "action": "inquiry_1920s",
                    "description": "用 1920 年代的方式调查",
                    "example": "我在图书馆查阅关于这个主题的旧资料"
                },
                {
                    "action": "ask_kp",
                    "description": "询问 KP 是否有特殊设定",
                    "example": "/rule 这个时代有相关设定吗"
                }
            ]
        },
        "guidance": {
            "next_action": "尝试用符合时代背景的方式行动"
        }
    },

    # === 无法理解 ===
    "not_understood": {
        "category": "missing_context",
        "refusal": {
            "tone": "helpful",
            "message": "我不太确定你想做什么。能换种说法吗？",
            "explanation": "需要更具体的描述才能理解你的意图。"
        },
        "alternatives": {
            "suggestions": [
                {
                    "action": "clarify",
                    "description": "更详细地描述你的行动",
                    "example": "我想[动词][目标]，目的是[目的]"
                },
                {
                    "action": "use_command",
                    "description": "使用标准命令",
                    "example": "/help 查看可用命令"
                }
            ]
        }
    },

    # === 检定不可用 ===
    "check_not_available": {
        "category": "wrong_timing",
        "refusal": {
            "tone": "firm",
            "message": "现在不能进行检定。",
            "explanation": "{reason}"
        },
        "alternatives": {
            "suggestions": [
                {
                    "action": "wait",
                    "description": "等待合适的时机",
                    "example": "/status 查看当前状态"
                },
                {
                    "action": "alternative_action",
                    "description": "尝试其他行动",
                    "example": "/leads 查看可选行动"
                }
            ]
        }
    },

    # === 战斗外使用战斗技能 ===
    "combat_out_of_combat": {
        "category": "scene_constraint",
        "refusal": {
            "tone": "dramatic",
            "message": "现在并没有发生战斗，你无法使用战斗技能。",
            "explanation": "战斗技能只能在战斗场景中使用。"
        },
        "alternatives": {
            "suggestions": [
                {
                    "action": "start_combat",
                    "description": "开始战斗",
                    "example": "/combat start"
                },
                {
                    "action": "use_normal_check",
                    "description": "使用普通检定",
                    "example": "/roll 格斗"
                }
            ]
        }
    },

    # === 需要 KP 操作 ===
    "requires_kp": {
        "category": "requires_kp",
        "refusal": {
            "tone": "polite",
            "message": "这个操作需要 KP 来执行。",
            "explanation": "只有 KP 能执行此操作，请联系 KP。"
        },
        "alternatives": {
            "suggestions": [
                {
                    "action": "request_kp",
                    "description": "向 KP 提出请求",
                    "example": "KP，可以帮我处理这个吗？"
                }
            ]
        }
    },

    # === 角色状态限制 ===
    "character_unconscious": {
        "category": "character_limit",
        "refusal": {
            "tone": "dramatic",
            "message": "{name} 已经失去意识，无法行动。",
            "explanation": "角色当前处于 {status} 状态，需要先恢复。"
        },
        "alternatives": {
            "suggestions": [
                {
                    "action": "wait_help",
                    "description": "等待队友救援",
                    "example": "/status 查看状态"
                },
                {
                    "action": "roll_vitality",
                    "description": "尝试恢复意识",
                    "example": "/roll 体质 (恢复检定)"
                }
            ]
        }
    },
}

# 模板渲染器
class RefusalTemplateRenderer:
    def render(
        self,
        template_id: str,
        context: dict
    ) -> dict:
        """渲染拒绝模板"""
        template = REFUSAL_TEMPLATES.get(template_id)

        if not template:
            return self._render_default(context)

        # 渲染消息中的变量
        refusal = template["refusal"]
        message = self._render_template(refusal["message"], context)

        # 渲染建议
        alternatives = self._render_alternatives(
            template.get("alternatives", {}),
            context
        )

        return {
            "refusal": {
                "message": message,
                "tone": refusal["tone"],
            },
            "alternatives": alternatives,
            "guidance": template.get("guidance")
        }

    def _render_template(self, template: str, context: dict) -> str:
        """渲染模板字符串"""
        return template.format(**context)

    def _render_alternatives(
        self,
        alternatives: dict,
        context: dict
    ) -> list:
        """渲染替代建议"""
        suggestions = alternatives.get("suggestions", [])
        return [
            {
                "action": s["action"],
                "description": s["description"],
                "example": s.get("example"),
            }
            for s in suggestions
        ]

    def _render_default(self, context: dict) -> dict:
        """默认拒绝响应"""
        return {
            "refusal": {
                "message": "我无法执行这个操作。",
                "tone": "neutral",
            },
            "alternatives": [
                {
                    "action": "check_command",
                    "description": "检查命令是否正确",
                    "example": "/help"
                }
            ]
        }
```

---

## 使用示例

```python
# 在 LLM 服务中使用
from app.templates.refusals import RefusalTemplateRenderer

class GameService:
    def __init__(self):
        self.refusal_renderer = RefusalTemplateRenderer()

    def handle_out_of_scope(self, user_input: str) -> str:
        """处理越界请求"""
        context = {
            "user_input": user_input,
            "setting": "1920s",
        }

        response = self.refusal_renderer.render(
            "out_of_scope_modern",
            context
        )

        # 格式化为 LLM 响应
        return self._format_response(response)
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/templates/refusals.py` | 创建 | 拒绝模板 |
| `app/services/refusal.py` | 创建 | 拒绝服务 |
| `tests/test_refusals.py` | 创建 | 模板测试 |

---

## 验收标准

- [ ] 常见越界场景有模板
- [ ] 拒绝消息友好礼貌
- [ ] 替代建议有用
- [ ] 模板渲染正确
- [ ] 易于扩展

---

## 参考文档

- M1-050: 拒绝模板响应
- M6-018: 无法理解模板

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
