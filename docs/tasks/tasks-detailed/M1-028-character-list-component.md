# M1-028 实现角色卡列表组件 CharacterList

## 概述
实现角色卡列表 React 组件,展示用户的所有角色卡,支持选择、编辑、删除等操作。

## 验收标准
- [ ] 实现角色卡列表布局
- [ ] 实现角色卡卡片展示
- [ ] 支持选择模式
- [ ] 支持搜索和筛选
- [ ] 支持分页
- [ ] 支持加载状态

## 技术方案

### 组件结构

```tsx
import React, { useState, useEffect } from 'react';
import { CharacterCard } from './CharacterCard';
import { CharacterFilters } from './CharacterFilters';
import { Pagination } from '@/components/ui/Pagination';

interface CharacterListProps {
  onSelect?: (characterId: string) => void;
  onEdit?: (characterId: string) => void;
  onDelete?: (characterId: string) => void;
  multiSelect?: boolean;
}

export const CharacterList: React.FC<CharacterListProps> = ({
  onSelect,
  onEdit,
  onDelete,
  multiSelect = false
}) => {
  const [characters, setCharacters] = useState<Character[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

  // 分页
  const [page, setPage] = useState(1);
  const [limit] = useState(12);
  const [total, setTotal] = useState(0);

  // 筛选
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState<string>('all');
  const [sortBy, setSortBy] = useState('created_at');
  const [order, setOrder] = useState<'asc' | 'desc'>('desc');

  // 获取角色列表
  useEffect(() => {
    fetchCharacters();
  }, [page, search, status, sortBy, order]);

  const fetchCharacters = async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({
        page: page.toString(),
        limit: limit.toString(),
        sort_by: sortBy,
        order: order
      });

      if (search) params.append('search', search);
      if (status !== 'all') params.append('status', status);

      const response = await fetch(`/api/characters?${params}`);
      const data = await response.json();

      setCharacters(data.characters);
      setTotal(data.pagination.total);
    } catch (error) {
      console.error('获取角色列表失败:', error);
    } finally {
      setLoading(false);
    }
  };

  // 选择处理
  const handleSelect = (characterId: string) => {
    if (multiSelect) {
      const newSelected = new Set(selectedIds);
      if (newSelected.has(characterId)) {
        newSelected.delete(characterId);
      } else {
        newSelected.add(characterId);
      }
      setSelectedIds(newSelected);
    } else {
      onSelect?.(characterId);
    }
  };

  // 全选
  const handleSelectAll = () => {
    if (selectedIds.size === characters.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(characters.map(c => c.id)));
    }
  };

  return (
    <div className="character-list">
      {/* 筛选栏 */}
      <CharacterFilters
        search={search}
        onSearchChange={setSearch}
        status={status}
        onStatusChange={setStatus}
        sortBy={sortBy}
        onSortByChange={setSortBy}
        order={order}
        onOrderChange={setOrder}
      />

      {/* 全选 */}
      {multiSelect && characters.length > 0 && (
        <div className="select-all-bar">
          <label className="checkbox-label">
            <input
              type="checkbox"
              checked={selectedIds.size === characters.length}
              onChange={handleSelectAll}
            />
            <span>全选 ({selectedIds.size}/{characters.length})</span>
          </label>
        </div>
      )}

      {/* 加载状态 */}
      {loading ? (
        <div className="loading-state">
          <div className="spinner" />
          <p>加载中...</p>
        </div>
      ) : (
        <>
          {/* 角色卡片网格 */}
          <div className="character-grid">
            {characters.map(character => (
              <CharacterCard
                key={character.id}
                character={character}
                selected={selectedIds.has(character.id)}
                onSelect={handleSelect}
                onEdit={onEdit}
                onDelete={onDelete}
              />
            ))}
          </div>

          {/* 空状态 */}
          {characters.length === 0 && (
            <div className="empty-state">
              <div className="empty-icon">📋</div>
              <h3>暂无角色卡</h3>
              <p>创建你的第一个角色卡吧!</p>
            </div>
          )}

          {/* 分页 */}
          <Pagination
            page={page}
            total={total}
            limit={limit}
            onPageChange={setPage}
          />
        </>
      )}
    </div>
  );
};
```

### 角色卡片组件

```tsx
interface CharacterCardProps {
  character: Character;
  selected?: boolean;
  onSelect?: (characterId: string) => void;
  onEdit?: (characterId: string) => void;
  onDelete?: (characterId: string) => void;
}

export const CharacterCard: React.FC<CharacterCardProps> = ({
  character,
  selected = false,
  onSelect,
  onEdit,
  onDelete
}) => {
  const [showActions, setShowActions] = useState(false);

  // 状态颜色
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'alive': return 'success';
      case 'unconscious': return 'warning';
      case 'dying': return 'danger';
      case 'dead': return 'dark';
      case 'insane': return 'info';
      default: return 'secondary';
    }
  };

  return (
    <div
      className={cn(
        'character-card',
        selected && 'character-card--selected'
      )}
      onMouseEnter={() => setShowActions(true)}
      onMouseLeave={() => setShowActions(false)}
    >
      {/* 选择框 */}
      {onSelect && (
        <div className="character-card__select">
          <input
            type="checkbox"
            checked={selected}
            onChange={() => onSelect(character.id)}
          />
        </div>
      )}

      {/* 卡片内容 */}
      <div className="character-card__content">
        {/* 头像 */}
        <div className="character-card__avatar">
          <img
            src={character.avatar || '/default-avatar.png'}
            alt={character.name}
          />
          <span
            className={cn(
              'status-badge',
              `status-badge--${getStatusColor(character.status)}`
            )}
          >
            {character.status}
          </span>
        </div>

        {/* 信息 */}
        <div className="character-card__info">
          <h3 className="character-card__name">{character.name}</h3>
          <p className="character-card__occupation">
            {character.occupation}
          </p>
          <p className="character-card__player">
            玩家: {character.player}
          </p>

          {/* 属性预览 */}
          <div className="character-card__stats">
            <div className="stat-item">
              <span className="stat-label">HP</span>
              <span className="stat-value">
                {character.summary.hp}/{character.summary.hp_max}
              </span>
            </div>
            <div className="stat-item">
              <span className="stat-label">SAN</span>
              <span className="stat-value">
                {character.summary.san}/{character.summary.san_max}
              </span>
            </div>
            <div className="stat-item">
              <span className="stat-label">Luck</span>
              <span className="stat-value">
                {character.summary.luck}
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* 操作按钮 */}
      {showActions && (
        <div className="character-card__actions">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onEdit?.(character.id)}
          >
            编辑
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm">
                <MoreVertical />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent>
              <DropdownMenuItem onClick={() => onEdit?.(character.id)}>
                <Edit /> 编辑
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => handleExport(character.id)}>
                <Download /> 导出
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => handleDuplicate(character.id)}>
                <Copy /> 复制
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={() => onDelete?.(character.id)}
                className="text-danger"
              >
                <Trash /> 删除
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      )}

      {/* 更新时间 */}
      <div className="character-card__timestamp">
        更新于 {formatDate(character.updated_at)}
      </div>
    </div>
  );
};
```

### 筛选组件

```tsx
interface CharacterFiltersProps {
  search: string;
  onSearchChange: (value: string) => void;
  status: string;
  onStatusChange: (value: string) => void;
  sortBy: string;
  onSortByChange: (value: string) => void;
  order: 'asc' | 'desc';
  onOrderChange: (value: 'asc' | 'desc') => void;
}

export const CharacterFilters: React.FC<CharacterFiltersProps> = ({
  search,
  onSearchChange,
  status,
  onStatusChange,
  sortBy,
  onSortByChange,
  order,
  onOrderChange
}) => {
  return (
    <div className="character-filters">
      {/* 搜索 */}
      <div className="filter-group">
        <Search />
        <Input
          placeholder="搜索角色名称..."
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
        />
      </div>

      {/* 状态筛选 */}
      <Select value={status} onValueChange={onStatusChange}>
        <SelectTrigger>
          <SelectValue placeholder="状态" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">全部状态</SelectItem>
          <SelectItem value="alive">存活</SelectItem>
          <SelectItem value="unconscious">昏迷</SelectItem>
          <SelectItem value="dying">濒死</SelectItem>
          <SelectItem value="dead">死亡</SelectItem>
          <SelectItem value="insane">疯狂</SelectItem>
        </SelectContent>
      </Select>

      {/* 排序 */}
      <Select value={sortBy} onValueChange={onSortByChange}>
        <SelectTrigger>
          <SelectValue placeholder="排序" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="created_at">创建时间</SelectItem>
          <SelectItem value="updated_at">更新时间</SelectItem>
          <SelectItem value="name">名称</SelectItem>
          <SelectItem value="age">年龄</SelectItem>
        </SelectContent>
      </Select>

      {/* 排序方向 */}
      <Button
        variant="ghost"
        size="icon"
        onClick={() => onOrderChange(order === 'asc' ? 'desc' : 'asc')}
      >
        {order === 'asc' ? <ArrowUp /> : <ArrowDown />}
      </Button>
    </div>
  );
};
```

### 样式

```css
.character-list {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.character-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 1rem;
}

.character-card {
  position: relative;
  background: white;
  border: 2px solid #e5e7eb;
  border-radius: 12px;
  padding: 1rem;
  transition: all 0.2s;
}

.character-card:hover {
  border-color: #5c6bc0;
  box-shadow: 0 4px 12px rgba(92, 107, 192, 0.15);
}

.character-card--selected {
  border-color: #5c6bc0;
  background: #f5f7ff;
}

.character-card__select {
  position: absolute;
  top: 0.75rem;
  left: 0.75rem;
}

.character-card__content {
  display: flex;
  gap: 1rem;
}

.character-card__avatar {
  position: relative;
  width: 64px;
  height: 64px;
}

.character-card__avatar img {
  width: 100%;
  height: 100%;
  border-radius: 50%;
  object-fit: cover;
}

.status-badge {
  position: absolute;
  bottom: -4px;
  right: -4px;
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 10px;
  font-weight: 600;
}

.character-card__info {
  flex: 1;
}

.character-card__name {
  font-size: 1.125rem;
  font-weight: 600;
  margin-bottom: 0.25rem;
}

.character-card__occupation {
  color: #6c757d;
  font-size: 0.875rem;
  margin-bottom: 0.125rem;
}

.character-card__player {
  color: #adb5bd;
  font-size: 0.75rem;
}

.character-card__stats {
  display: flex;
  gap: 0.75rem;
  margin-top: 0.75rem;
}

.stat-item {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.stat-label {
  font-size: 0.625rem;
  color: #6c757d;
  font-weight: 600;
}

.stat-value {
  font-size: 0.875rem;
  font-weight: 600;
}

.character-card__actions {
  position: absolute;
  top: 0.75rem;
  right: 0.75rem;
  display: flex;
  gap: 0.5rem;
}

.character-card__timestamp {
  margin-top: 0.75rem;
  padding-top: 0.75rem;
  border-top: 1px solid #e5e7eb;
  font-size: 0.75rem;
  color: #adb5bd;
}

.empty-state {
  text-align: center;
  padding: 3rem 1rem;
}

.empty-icon {
  font-size: 4rem;
  margin-bottom: 1rem;
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 3rem;
}
```

## 依赖关系
- 前置任务: M1-023 实现列表角色卡 GET /characters
- 被依赖: M1-029 实现角色卡表单 CharacterForm

## 预估工时
2h
