# M1-029 实现角色卡表单 CharacterForm

## 概述
实现角色卡创建/编辑的 React 表单组件,包含属性输入、技能管理、状态设置等功能。

## 验收标准
- [ ] 实现基础信息表单
- [ ] 实现 8 个属性输入
- [ ] 实现派生属性自动计算
- [ ] 实现技能管理
- [ ] 实现表单验证
- [ ] 支持保存和取消

## 技术方案

### 表单组件

```tsx
import React, { useState, useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Label } from '@/components/ui/Label';

// 表单验证 schema
const characterSchema = z.object({
  // 基本信息
  name: z.string().min(1, '角色名称必填').max(100, '名称过长'),
  age: z.number().min(15, '年龄最小 15 岁').max(90, '年龄最大 90 岁'),
  occupation: z.string().min(1, '职业必填').max(100, '职业名称过长'),
  player: z.string().min(1, '玩家名必填').max(100, '玩家名过长'),

  // 属性
  attributes: z.object({
    STR: z.number().min(0).max(100),
    CON: z.number().min(0).max(100),
    DEX: z.number().min(0).max(100),
    APP: z.number().min(0).max(100),
    POW: z.number().min(0).max(100),
    INT: z.number().min(0).max(100),
    SIZ: z.number().min(0).max(100),
    EDU: z.number().min(0).max(100),
  }),

  // 派生属性(自动计算)
  derived: z.object({
    HP: z.number(),
    HP_max: z.number(),
    MP: z.number(),
    MP_max: z.number(),
    SAN: z.number(),
    SAN_max: z.number(),
    Luck: z.number(),
    Luck_max: z.number(),
    Move: z.number(),
    DB: z.string(),
    Build: z.number(),
  }),

  // 其他
  status: z.enum(['alive', 'unconscious', 'dying', 'dead', 'insane']),
  inventory: z.array(z.string()).default([]),
  notes: z.string().optional(),
});

type CharacterFormData = z.infer<typeof characterSchema>;

interface CharacterFormProps {
  character?: Character;
  onSave: (data: CharacterFormData) => Promise<void>;
  onCancel: () => void;
}

export const CharacterForm: React.FC<CharacterFormProps> = ({
  character,
  onSave,
  onCancel
}) => {
  const [calculating, setCalculating] = useState(false);

  // 表单
  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors, isDirty }
  } = useForm<CharacterFormData>({
    resolver: zodResolver(characterSchema),
    defaultValues: character ? {
      name: character.name,
      age: character.age,
      occupation: character.occupation,
      player: character.player,
      attributes: character.attributes,
      derived: character.derived,
      status: character.status,
      inventory: character.inventory,
      notes: character.notes,
    } : {
      name: '',
      age: 25,
      occupation: '',
      player: '',
      attributes: {
        STR: 50,
        CON: 50,
        DEX: 50,
        APP: 50,
        POW: 50,
        INT: 50,
        SIZ: 50,
        EDU: 50,
      },
      derived: {
        HP: 10,
        HP_max: 10,
        MP: 10,
        MP_max: 10,
        SAN: 50,
        SAN_max: 99,
        Luck: 50,
        Luck_max: 50,
        Move: 8,
        DB: '0',
        Build: 0,
      },
      status: 'alive',
      inventory: [],
      notes: '',
    }
  });

  // 监听属性变化,自动计算派生属性
  const attributes = watch('attributes');

  useEffect(() => {
    if (attributes) {
      calculateDerived(attributes);
    }
  }, [attributes]);

  // 计算派生属性
  const calculateDerived = async (attrs: typeof attributes) => {
    setCalculating(true);
    try {
      const response = await fetch('/api/characters/calculate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ attributes: attrs })
      });

      const derived = await response.json();

      setValue('derived.HP', derived.HP);
      setValue('derived.HP_max', derived.HP_max);
      setValue('derived.MP', derived.MP);
      setValue('derived.MP_max', derived.MP_max);
      setValue('derived.Luck', derived.Luck);
      setValue('derived.Luck_max', derived.Luck_max);
      setValue('derived.Move', derived.Move);
      setValue('derived.DB', derived.DB);
      setValue('derived.Build', derived.Build);
    } catch (error) {
      console.error('计算派生属性失败:', error);
    } finally {
      setCalculating(false);
    }
  };

  // 提交
  const onSubmit = async (data: CharacterFormData) => {
    await onSave(data);
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="character-form">
      {/* 基本信息 */}
      <section className="form-section">
        <h3>基本信息</h3>

        <div className="form-grid">
          <div>
            <Label htmlFor="name">角色名称 *</Label>
            <Input
              id="name"
              {...register('name')}
              error={errors.name?.message}
            />
          </div>

          <div>
            <Label htmlFor="age">年龄 *</Label>
            <Input
              id="age"
              type="number"
              {...register('age', { valueAsNumber: true })}
              error={errors.age?.message}
            />
          </div>

          <div>
            <Label htmlFor="occupation">职业 *</Label>
            <Input
              id="occupation"
              {...register('occupation')}
              error={errors.occupation?.message}
            />
          </div>

          <div>
            <Label htmlFor="player">玩家 *</Label>
            <Input
              id="player"
              {...register('player')}
              error={errors.player?.message}
            />
          </div>
        </div>
      </section>

      {/* 属性 */}
      <section className="form-section">
        <h3>属性</h3>

        <AttributeGrid
          attributes={attributes}
          register={register}
          errors={errors.attributes}
        />
      </section>

      {/* 派生属性 */}
      <section className="form-section">
        <h3>派生属性</h3>
        {calculating && (
          <div className="calculating-indicator">
            计算中...
          </div>
        )}

        <DerivedStatsGrid
          derived={watch('derived')}
          register={register}
          readonly
        />
      </section>

      {/* 技能 */}
      <section className="form-section">
        <h3>技能</h3>

        <SkillsManager
          skills={watch('skills') || {}}
          onChange={(skills) => setValue('skills', skills)}
        />
      </section>

      {/* 状态和物品 */}
      <section className="form-section">
        <h3>状态</h3>

        <div className="form-grid">
          <div>
            <Label htmlFor="status">状态</Label>
            <select
              id="status"
              {...register('status')}
              className="select"
            >
              <option value="alive">存活</option>
              <option value="unconscious">昏迷</option>
              <option value="dying">濒死</option>
              <option value="dead">死亡</option>
              <option value="insane">疯狂</option>
            </select>
          </div>
        </div>

        <InventoryManager
          inventory={watch('inventory') || []}
          onChange={(items) => setValue('inventory', items)}
        />
      </section>

      {/* 备注 */}
      <section className="form-section">
        <h3>备注</h3>

        <textarea
          {...register('notes')}
          className="textarea"
          rows={5}
          placeholder="角色背景、经历等..."
        />
      </section>

      {/* 操作按钮 */}
      <div className="form-actions">
        <Button
          type="button"
          variant="ghost"
          onClick={onCancel}
        >
          取消
        </Button>
        <Button
          type="submit"
          disabled={!isDirty || calculating}
        >
          {calculating ? '计算中...' : '保存'}
        </Button>
      </div>
    </form>
  );
};
```

### 属性网格组件

```tsx
interface AttributeGridProps {
  attributes: {
    STR: number;
    CON: number;
    DEX: number;
    APP: number;
    POW: number;
    INT: number;
    SIZ: number;
    EDU: number;
  };
  register: any;
  errors?: any;
}

export const AttributeGrid: React.FC<AttributeGridProps> = ({
  attributes,
  register,
  errors
}) => {
  const attrList = [
    { key: 'STR', label: '力量', desc: 'STR' },
    { key: 'CON', label: '体质', desc: 'CON' },
    { key: 'DEX', label: '敏捷', desc: 'DEX' },
    { key: 'APP', label: '外貌', desc: 'APP' },
    { key: 'POW', label: '意志', desc: 'POW' },
    { key: 'INT', label: '智力', desc: 'INT' },
    { key: 'SIZ', label: '体型', desc: 'SIZ' },
    { key: 'EDU', label: '教育', desc: 'EDU' },
  ];

  return (
    <div className="attribute-grid">
      {attrList.map(attr => (
        <div key={attr.key} className="attribute-item">
          <Label htmlFor={`attributes.${attr.key}`}>
            {attr.label} ({attr.key})
          </Label>
          <Input
            id={`attributes.${attr.key}`}
            type="number"
            min={0}
            max={100}
            {...register(`attributes.${attr.key}`, { valueAsNumber: true })}
            error={errors?.[attr.key]?.message}
          />
          <div className="attribute-desc">
            {attr.desc}
          </div>
        </div>
      ))}
    </div>
  );
};
```

### 派生属性网格

```tsx
interface DerivedStatsGridProps {
  derived: {
    HP: number;
    HP_max: number;
    MP: number;
    MP_max: number;
    SAN: number;
    SAN_max: number;
    Luck: number;
    Luck_max: number;
    Move: number;
    DB: string;
    Build: number;
  };
  register?: any;
  readonly?: boolean;
}

export const DerivedStatsGrid: React.FC<DerivedStatsGridProps> = ({
  derived,
  readonly = false
}) => {
  return (
    <div className="derived-stats-grid">
      {/* HP */}
      <div className="derived-item">
        <Label>HP / HP_max</Label>
        <div className="derived-value">
          {derived.HP} / {derived.HP_max}
        </div>
      </div>

      {/* MP */}
      <div className="derived-item">
        <Label>MP / MP_max</Label>
        <div className="derived-value">
          {derived.MP} / {derived.MP_max}
        </div>
      </div>

      {/* SAN */}
      <div className="derived-item">
        <Label>SAN / SAN_max</Label>
        <div className="derived-value">
          {derived.SAN} / {derived.SAN_max}
        </div>
      </div>

      {/* Luck */}
      <div className="derived-item">
        <Label>Luck</Label>
        <div className="derived-value">
          {derived.Luck}
        </div>
      </div>

      {/* Move */}
      <div className="derived-item">
        <Label>Move</Label>
        <div className="derived-value">
          {derived.Move}
        </div>
      </div>

      {/* DB */}
      <div className="derived-item">
        <Label>DB</Label>
        <div className="derived-value">
          {derived.DB}
        </div>
      </div>

      {/* Build */}
      <div className="derived-item">
        <Label>Build</Label>
        <div className="derived-value">
          {derived.Build}
        </div>
      </div>
    </div>
  );
};
```

### 样式

```css
.character-form {
  display: flex;
  flex-direction: column;
  gap: 2rem;
}

.form-section {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 1.5rem;
}

.form-section h3 {
  margin-bottom: 1rem;
  font-size: 1.125rem;
  font-weight: 600;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
}

.attribute-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
  gap: 1rem;
}

.attribute-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.attribute-desc {
  font-size: 0.75rem;
  color: #6c757d;
}

.derived-stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: 1rem;
}

.derived-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.derived-value {
  font-size: 1.125rem;
  font-weight: 600;
  color: #5c6bc0;
}

.calculating-indicator {
  padding: 0.75rem;
  background: #f8f9fa;
  border-radius: 4px;
  text-align: center;
  color: #6c757d;
  font-size: 0.875rem;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  padding-top: 1rem;
  border-top: 1px solid #e5e7eb;
}
```

## 依赖关系
- 前置任务: M1-025 实现属性自动计算
- 被依赖: M1-021 实现角色卡预览 CharacterPreview

## 预估工时
4h
