# M1-014: 实现角色物品栏功能

**任务ID**: M1-014
**标题**: 实现角色物品栏功能
**类型**: fullstack (全栈开发)
**预估工时**: 2.5h
**依赖**: M1-001

---

## 任务描述

实现角色的物品栏系统，包括物品添加、移除、装备、使用等操作，支持物品分类、重量计算和容量限制。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-014-01 | 设计物品数据模型 | Item Model | 25min |
| M1-014-02 | 实现物品栏服务 | Inventory Service | 35min |
| M1-014-03 | 实现装备系统 | Equipment System | 35min |
| M1-014-04 | 实现物品使用 | Item Usage | 30min |
| M1-014-05 | 实现重量计算 | Weight Calculation | 20min |
| M1-014-06 | 实现前端物品栏 UI | Inventory UI | 30min |
| M1-014-07 | 编写测试 | 测试覆盖 | 25min |

---

## 物品数据模型

```python
# app/db/models/item.py
from sqlalchemy import Column, String, Integer, Float, Text, ForeignKey, Boolean, JSON
from sqlalchemy.orm import relationship
from app.db.database import Base

class Item(Base):
    """物品"""
    __tablename__ = 'items'

    id = Column(String, primary_key=True, index=True)

    # 基本信息
    name = Column(String, nullable=False)
    name_en = Column(String)
    description = Column(Text)
    icon = Column(String)  # emoji 或图标 URL

    # 分类
    category = Column(String)  # weapon, armor, tool, consumable, misc, key_item
    subcategory = Column(String)

    # 数值
    weight = Column(Float, default=0)  # 重量（单位：千克）
    value = Column(Integer, default=0)  # 价值（美元）

    # 装备属性
    is_equippable = Column(Boolean, default=False)
    slot = Column(String)  # weapon, armor, accessory, etc.

    # 战斗属性
    damage = Column(Integer)  # 伤害
    defense = Column(Integer)  # 防御
    damage_bonus = Column(Integer, default=0)
    defense_bonus = Column(Integer, default=0)

    # 使用效果
    is_consumable = Column(Boolean, default=False)
    effects = Column(JSON)  # 使用效果

    # 特殊属性
    is_magical = Column(Boolean, default=False)
    is_cursed = Column(Boolean, default=False)
    magical_effects = Column(JSON)

    # 限制
    required_strength = Column(Integer)
    required_size = Column(String)  # tiny, small, medium, large, huge

    # 描述
    flavor_text = Column(Text)  # 趣味描述
    lore = Column(Text)  # 背景故事

    # 创建者
    created_by = Column(String, ForeignKey('users.id'))
    is_custom = Column(Boolean, default=False)

    def __repr__(self):
        return f"<Item {self.name}>"

class CharacterItem(Base):
    """角色物品"""
    __tablename__ = 'character_items'

    id = Column(String, primary_key=True, index=True)
    character_id = Column(String, ForeignKey('characters.id'), nullable=False, index=True)
    item_id = Column(String, ForeignKey('items.id'), nullable=False)

    # 数量和状态
    quantity = Column(Integer, default=1)
    is_equipped = Column(Boolean, default=False)
    equipped_slot = Column(String)  # 装备槽位

    # 耐久度（可选项）
    durability = Column(Integer)  # 当前耐久
    max_durability = Column(Integer)  # 最大耐久

    # 自定义属性
    custom_name = Column(String)
    custom_description = Column(Text)
    custom_effects = Column(JSON)

    # 获取信息
    obtained_at = Column(String)  # 获取地点
    obtained_from = Column(String)  # 获取来源

    # 关系
    character = relationship("Character", back_populates="items")
    item = relationship("Item")

    def __repr__(self):
        return f"<CharacterItem character={self.character_id} item={self.item_id}>"
```

---

## 物品栏服务

```python
# app/services/inventory.py
from typing import Dict, Any, List, Optional
from sqlalchemy.orm import Session

from app.db.models.item import Item, CharacterItem
from app.db.models.character import Character
from app.core.security import generate_id

class InventoryService:
    """物品栏服务"""

    def __init__(self, db: Session):
        self.db = db

    def add_item(
        self,
        character_id: str,
        item_id: str,
        quantity: int = 1,
        obtained_at: str = None,
        obtained_from: str = None,
    ) -> CharacterItem:
        """添加物品"""
        # 检查是否已存在该物品
        existing = self.db.query(CharacterItem)\
            .filter(
                CharacterItem.character_id == character_id,
                CharacterItem.item_id == item_id,
            )\
            .first()

        if existing:
            # 增加数量
            existing.quantity += quantity
            self.db.commit()
            self.db.refresh(existing)
            return existing

        # 创建新的物品记录
        character_item = CharacterItem(
            id=generate_id('char_item'),
            character_id=character_id,
            item_id=item_id,
            quantity=quantity,
            obtained_at=obtained_at,
            obtained_from=obtained_from,
        )

        self.db.add(character_item)
        self.db.commit()
        self.db.refresh(character_item)

        return character_item

    def remove_item(
        self,
        character_id: str,
        character_item_id: str,
        quantity: int = 1,
    ) -> bool:
        """移除物品"""
        char_item = self.db.query(CharacterItem)\
            .filter(
                CharacterItem.id == character_item_id,
                CharacterItem.character_id == character_id,
            )\
            .first()

        if not char_item:
            return False

        if char_item.quantity <= quantity:
            # 完全移除
            self.db.delete(char_item)
        else:
            # 减少数量
            char_item.quantity -= quantity

        self.db.commit()
        return True

    def equip_item(
        self,
        character_id: str,
        character_item_id: str,
        slot: str = None,
    ) -> Optional[CharacterItem]:
        """装备物品"""
        char_item = self.db.query(CharacterItem)\
            .filter(
                CharacterItem.id == character_item_id,
                CharacterItem.character_id == character_id,
            )\
            .first()

        if not char_item:
            return None

        item = char_item.item
        if not item.is_equippable:
            raise ValueError("该物品不可装备")

        # 检查是否满足要求
        character = self.db.query(Character).filter(Character.id == character_id).first()
        if not character:
            return None

        if item.required_strength and character.strength < item.required_strength:
            raise ValueError(f"需要力量 {item.required_strength}")

        if item.required_size:
            size_map = {'tiny': 1, 'small': 2, 'medium': 3, 'large': 4, 'huge': 5}
            char_size = size_map.get(character.size, 3)
            req_size = size_map.get(item.required_size, 3)
            if char_size < req_size:
                raise ValueError(f"需要体型 {item.required_size}")

        # 确定装备槽位
        equip_slot = slot or item.slot

        # 卸下同槽位的物品
        self.db.query(CharacterItem)\
            .filter(
                CharacterItem.character_id == character_id,
                CharacterItem.is_equipped == True,
                CharacterItem.equipped_slot == equip_slot,
            )\
            .update({'is_equipped': False, 'equipped_slot': None})

        # 装备物品
        char_item.is_equipped = True
        char_item.equipped_slot = equip_slot

        self.db.commit()
        self.db.refresh(char_item)

        return char_item

    def unequip_item(
        self,
        character_id: str,
        character_item_id: str,
    ) -> Optional[CharacterItem]:
        """卸下装备"""
        char_item = self.db.query(CharacterItem)\
            .filter(
                CharacterItem.id == character_item_id,
                CharacterItem.character_id == character_id,
            )\
            .first()

        if not char_item or not char_item.is_equipped:
            return None

        char_item.is_equipped = False
        char_item.equipped_slot = None

        self.db.commit()
        self.db.refresh(char_item)

        return char_item

    def use_item(
        self,
        character_id: str,
        character_item_id: str,
    ) -> Dict[str, Any]:
        """使用物品"""
        char_item = self.db.query(CharacterItem)\
            .filter(
                CharacterItem.id == character_item_id,
                CharacterItem.character_id == character_id,
            )\
            .first()

        if not char_item:
            return {"success": False, "error": "物品不存在"}

        item = char_item.item

        if not item.is_consumable and not item.is_equippable:
            return {"success": False, "error": "该物品不可使用"}

        # 应用效果
        effects = item.effects or {}

        character = self.db.query(Character).filter(Character.id == character_id).first()
        if not character:
            return {"success": False, "error": "角色不存在"}

        result = {
            "success": True,
            "item_name": item.name,
            "effects_applied": [],
        }

        # 处理各种效果
        if 'heal' in effects:
            heal_amount = effects['heal']
            character.hp = min(character.hp + heal_amount, character.max_hp)
            result["effects_applied"].append(f"恢复 {heal_amount} HP")

        if 'restore_san' in effects:
            san_amount = effects['restore_san']
            character.san = min(character.san + san_amount, character.max_san)
            result["effects_applied"].append(f"恢复 {san_amount} SAN")

        if 'temp_boost' in effects:
            for stat, bonus in effects['temp_boost'].items():
                # TODO: 实现临时加成
                result["effects_applied"].append(f"{stat} +{bonus} (临时)")

        # 消耗品减少数量
        if item.is_consumable:
            char_item.quantity -= 1
            if char_item.quantity <= 0:
                self.db.delete(char_item)

        self.db.commit()

        return result

    def get_inventory(
        self,
        character_id: str,
    ) -> List[Dict[str, Any]]:
        """获取物品栏"""
        char_items = self.db.query(CharacterItem)\
            .filter(CharacterItem.character_id == character_id)\
            .join(Item)\
            .all()

        inventory = []
        for char_item in char_items:
            item = char_item.item
            inventory.append({
                "id": char_item.id,
                "item_id": item.id,
                "name": item.name,
                "description": item.description,
                "icon": item.icon,
                "category": item.category,
                "weight": item.weight,
                "value": item.value,
                "quantity": char_item.quantity,
                "is_equipped": char_item.is_equipped,
                "equipped_slot": char_item.equipped_slot,
                "is_equippable": item.is_equippable,
                "is_consumable": item.is_consumable,
                "slot": item.slot,
            })

        return inventory

    def get_equipped_items(
        self,
        character_id: str,
    ) -> Dict[str, Dict[str, Any]]:
        """获取已装备物品"""
        char_items = self.db.query(CharacterItem)\
            .filter(
                CharacterItem.character_id == character_id,
                CharacterItem.is_equipped == True,
            )\
            .join(Item)\
            .all()

        equipped = {}
        for char_item in char_items:
            item = char_item.item
            slot = char_item.equipped_slot or item.slot
            if slot:
                equipped[slot] = {
                    "id": char_item.id,
                    "item_id": item.id,
                    "name": item.name,
                    "icon": item.icon,
                    "damage": item.damage,
                    "defense": item.defense,
                    "damage_bonus": item.damage_bonus,
                    "defense_bonus": item.defense_bonus,
                }

        return equipped

    def calculate_total_weight(
        self,
        character_id: str,
    ) -> float:
        """计算总重量"""
        char_items = self.db.query(CharacterItem)\
            .filter(CharacterItem.character_id == character_id)\
            .join(Item)\
            .all()

        total_weight = 0.0
        for char_item in char_items:
            total_weight += char_item.item.weight * char_item.quantity

        return total_weight

    def calculate_carry_capacity(
        self,
        character_id: str,
    ) -> float:
        """计算负重能力"""
        character = self.db.query(Character).filter(Character.id == character_id).first()
        if not character:
            return 0

        # CoC 7e 负重规则
        # 负重能力 = 力量 × (体型 + 力量) / 10
        capacity = character.strength * (character.size + character.strength) / 10
        return capacity

    def check_overencumbered(
        self,
        character_id: str,
    ) -> Dict[str, Any]:
        """检查是否超重"""
        total_weight = self.calculate_total_weight(character_id)
        capacity = self.calculate_carry_capacity(character_id)

        return {
            "current_weight": total_weight,
            "max_capacity": capacity,
            "is_overencumbered": total_weight > capacity,
            "overencumberance_penalty": int(max(0, (total_weight - capacity) / 2)),
        }
```

---

## 物品栏 API

```python
# app/api/inventory.py
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session
from pydantic import BaseModel
from typing import List, Optional

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.inventory import InventoryService

router = APIRouter(prefix="/inventory", tags=["inventory"])

class AddItemRequest(BaseModel):
    character_id: str
    item_id: str
    quantity: int = 1
    obtained_at: str = None
    obtained_from: str = None

class RemoveItemRequest(BaseModel):
    character_id: str
    character_item_id: str
    quantity: int = 1

class EquipItemRequest(BaseModel):
    character_id: str
    character_item_id: str
    slot: str = None

@router.post("/add")
async def add_item(
    request: AddItemRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """添加物品"""
    service = InventoryService(db)
    char_item = service.add_item(
        request.character_id,
        request.item_id,
        request.quantity,
        request.obtained_at,
        request.obtained_from,
    )

    return {"message": "物品已添加", "item_id": char_item.id}

@router.delete("/remove")
async def remove_item(
    request: RemoveItemRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """移除物品"""
    service = InventoryService(db)
    success = service.remove_item(
        request.character_id,
        request.character_item_id,
        request.quantity,
    )

    if not success:
        raise HTTPException(status_code=404, detail="物品不存在")

    return {"message": "物品已移除"}

@router.post("/equip")
async def equip_item(
    request: EquipItemRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """装备物品"""
    service = InventoryService(db)

    try:
        char_item = service.equip_item(
            request.character_id,
            request.character_item_id,
            request.slot,
        )
        return {"message": "物品已装备", "slot": char_item.equipped_slot}
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))

@router.post("/unequip")
async def unequip_item(
    character_id: str,
    character_item_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """卸下装备"""
    service = InventoryService(db)
    char_item = service.unequip_item(character_id, character_item_id)

    if not char_item:
        raise HTTPException(status_code=404, detail="装备不存在")

    return {"message": "装备已卸下"}

@router.post("/use")
async def use_item(
    character_id: str,
    character_item_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """使用物品"""
    service = InventoryService(db)
    result = service.use_item(character_id, character_item_id)

    if not result["success"]:
        raise HTTPException(status_code=400, detail=result["error"])

    return result

@router.get("/character/{character_id}")
async def get_inventory(
    character_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取物品栏"""
    service = InventoryService(db)
    inventory = service.get_inventory(character_id)

    return {"inventory": inventory}

@router.get("/character/{character_id}/equipped")
async def get_equipped_items(
    character_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取已装备物品"""
    service = InventoryService(db)
    equipped = service.get_equipped_items(character_id)

    return {"equipped": equipped}

@router.get("/character/{character_id}/weight")
async def check_weight(
    character_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """检查负重"""
    service = InventoryService(db)
    weight_info = service.check_overencumbered(character_id)

    return weight_info
```

---

## 前端物品栏组件

```tsx
// frontend/src/components/character/CharacterInventory.tsx
import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Package, Sword, Shield, Dumbbell } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ItemDetail } from './ItemDetail'

interface InventoryItem {
  id: string
  item_id: string
  name: string
  description: string
  icon: string
  category: string
  weight: number
  value: number
  quantity: number
  is_equipped: boolean
  equipped_slot: string | null
  is_equippable: boolean
  is_consumable: boolean
  slot: string | null
}

interface WeightInfo {
  current_weight: number
  max_capacity: number
  is_overencumbered: boolean
  overencumberance_penalty: number
}

export function CharacterInventory({ characterId }: { characterId: string }) {
  const [inventory, setInventory] = useState<InventoryItem[]>([])
  const [weightInfo, setWeightInfo] = useState<WeightInfo | null>(null)
  const [selectedItem, setSelectedItem] = useState<InventoryItem | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchInventory()
    fetchWeightInfo()
  }, [characterId])

  const fetchInventory = async () => {
    try {
      const response = await fetch(`/api/inventory/character/${characterId}`)
      if (response.ok) {
        const data = await response.json()
        setInventory(data.inventory)
      }
    } catch (error) {
      console.error('Failed to fetch inventory:', error)
    } finally {
      setLoading(false)
    }
  }

  const fetchWeightInfo = async () => {
    try {
      const response = await fetch(`/api/inventory/character/${characterId}/weight`)
      if (response.ok) {
        const data = await response.json()
        setWeightInfo(data)
      }
    } catch (error) {
      console.error('Failed to fetch weight info:', error)
    }
  }

  const handleEquip = async (itemId: string) => {
    try {
      const response = await fetch('/api/inventory/equip', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          character_id: characterId,
          character_item_id: itemId,
        }),
      })

      if (response.ok) {
        fetchInventory()
      }
    } catch (error) {
      console.error('Failed to equip item:', error)
    }
  }

  const handleUnequip = async (itemId: string) => {
    try {
      const response = await fetch('/api/inventory/unequip', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          character_id: characterId,
          character_item_id: itemId,
        }),
      })

      if (response.ok) {
        fetchInventory()
      }
    } catch (error) {
      console.error('Failed to unequip item:', error)
    }
  }

  const handleUse = async (itemId: string) => {
    try {
      const response = await fetch('/api/inventory/use', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          character_id: characterId,
          character_item_id: itemId,
        }),
      })

      if (response.ok) {
        const data = await response.json()
        alert(data.effects_applied.join('\n'))
        fetchInventory()
      }
    } catch (error) {
      console.error('Failed to use item:', error)
    }
  }

  if (loading) {
    return <div>加载中...</div>
  }

  const categoryGroups = inventory.reduce((acc, item) => {
    if (!acc[item.category]) {
      acc[item.category] = []
    }
    acc[item.category].push(item)
    return acc
  }, {} as Record<string, InventoryItem[]>)

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <div className="flex items-center">
            <Package className="h-5 w-5 mr-2" />
            物品栏
          </div>
          {weightInfo && (
            <div className={`flex items-center text-sm ${
              weightInfo.is_overencumbered ? 'text-red-500' : 'text-muted-foreground'
            }`}>
              <Dumbbell className="h-4 w-4 mr-1" />
              {weightInfo.current_weight.toFixed(1)} / {weightInfo.max_capacity.toFixed(1)} kg
            </div>
          )}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <ScrollArea className="h-[400px]">
          <div className="space-y-4">
            {Object.entries(categoryGroups).map(([category, items]) => (
              <div key={category}>
                <h4 className="text-sm font-medium mb-2 capitalize">
                  {category === 'weapon' ? '武器' :
                   category === 'armor' ? '护甲' :
                   category === 'consumable' ? '消耗品' :
                   category === 'tool' ? '工具' : '杂项'}
                </h4>
                <div className="grid grid-cols-1 gap-2">
                  {items.map((item) => (
                    <div
                      key={item.id}
                      className="flex items-center p-2 rounded hover:bg-muted cursor-pointer"
                      onClick={() => setSelectedItem(item)}
                    >
                      <div className="text-2xl mr-3">{item.icon || '📦'}</div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center">
                          <span className="font-medium truncate">{item.name}</span>
                          {item.is_equipped && (
                            <Badge variant="secondary" className="ml-2">
                              {item.equipped_slot}
                            </Badge>
                          )}
                        </div>
                        <div className="text-sm text-muted-foreground">
                          {item.weight > 0 && `${item.weight} kg`}
                          {item.quantity > 1 && ` ×${item.quantity}`}
                        </div>
                      </div>
                      <div className="flex gap-1">
                        {item.is_equippable && !item.is_equipped && (
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={(e) => {
                              e.stopPropagation()
                              handleEquip(item.id)
                            }}
                          >
                            <Sword className="h-4 w-4" />
                          </Button>
                        )}
                        {item.is_equipped && (
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={(e) => {
                              e.stopPropagation()
                              handleUnequip(item.id)
                            }}
                          >
                            <Shield className="h-4 w-4" />
                          </Button>
                        )}
                        {item.is_consumable && (
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={(e) => {
                              e.stopPropagation()
                              handleUse(item.id)
                            }}
                          >
                            使用
                          </Button>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </ScrollArea>

        {selectedItem && (
          <Dialog open={!!selectedItem} onOpenChange={() => setSelectedItem(null)}>
            <DialogContent>
              <DialogHeader>
                <DialogTitle className="flex items-center">
                  <span className="text-3xl mr-3">{selectedItem.icon}</span>
                  {selectedItem.name}
                </DialogTitle>
                <DialogDescription>
                  {selectedItem.description}
                </DialogDescription>
              </DialogHeader>
              <ItemDetail item={selectedItem} />
            </DialogContent>
          </Dialog>
        )}
      </CardContent>
    </Card>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/db/models/item.py` | 创建 | 物品数据模型 |
| `app/services/inventory.py` | 创建 | 物品栏服务 |
| `app/api/inventory.py` | 创建 | 物品栏 API |
| `frontend/src/components/character/CharacterInventory.tsx` | 创建 | 物品栏组件 |
| `frontend/src/components/character/ItemDetail.tsx` | 创建 | 物品详情组件 |

---

## 验收标准

- [ ] 物品添加/移除正常
- [ ] 装备系统工作正常
- [ ] 消耗品使用正确
- [ ] 重量计算准确
- [ ] 负重限制生效
- [ ] UI 交互流畅

---

## 参考文档

- M1-001: 角色系统
- CoC 7e 规则书 - 物品与装备

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
