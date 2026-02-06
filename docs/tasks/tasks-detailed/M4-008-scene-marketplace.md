# M4-008: 实现场景包市场

**任务ID**: M4-008
**标题**: 实现场景包市场
**类型**: fullstack (全栈开发)
**预估工时**: 3h
**依赖**: M4-001

---

## 任务描述

实现场景包市场功能，让用户可以分享、浏览和下载其他人创建的场景包，包括搜索、评分、评论等功能。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M4-008-01 | 设计市场数据模型 | Marketplace Model | 30min |
| M4-008-02 | 实现发布系统 | Publishing System | 35min |
| M4-008-03 | 实现浏览和搜索 | Browse & Search | 35min |
| M4-008-04 | 实现评分系统 | Rating System | 30min |
| M4-008-05 | 实现评论系统 | Comment System | 30min |
| M4-008-06 | 实现下载统计 | Download Stats | 20min |
| M4-008-07 | 编写测试 | 测试覆盖 | 20min |

---

## 市场数据模型

```python
# app/db/models/marketplace.py
from sqlalchemy import Column, String, Integer, Text, ForeignKey, Boolean, JSON, DateTime, Float
from sqlalchemy.orm import relationship
from app.db.database import Base
from datetime import datetime

class MarketplaceListing(Base):
    """市场列表"""
    __tablename__ = 'marketplace_listings'

    id = Column(String, primary_key=True, index=True)
    scene_package_id = Column(String, ForeignKey('scene_packages.id'), nullable=False, index=True)

    # 发布信息
    publisher_id = Column(String, ForeignKey('users.id'), nullable=False)
    published_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # 状态
    status = Column(String, default='pending')  # pending, approved, rejected, removed
    reviewed_by = Column(String, ForeignKey('users.id'))
    reviewed_at = Column(DateTime)
    rejection_reason = Column(Text)

    # 统计
    view_count = Column(Integer, default=0)
    download_count = Column(Integer, default=0)
    favorite_count = Column(Integer, default=0)

    # 评分
    average_rating = Column(Float, default=0)
    rating_count = Column(Integer, default=0)

    # 标签和分类
    tags = Column(JSON, default=list)
    category = Column(String)  # short_scenario, full_campaign, one_shot, etc.
    difficulty = Column(String)  # easy, medium, hard
    player_count_min = Column(Integer)
    player_count_max = Column(Integer)
    duration_hours = Column(Integer)

    # 语言
    language = Column(String, default='zh')

    # 版本
    version = Column(String, default='1.0.0')

    # 许可证
    license = Column(String, default='CC-BY-NC-SA-4.0')

    # 关系
    scene_package = relationship("ScenePackage", back_populates="marketplace_listing")
    publisher = relationship("User", foreign_keys=[publisher_id])
    reviewer = relationship("User", foreign_keys=[reviewed_by])

    def __repr__(self):
        return f"<MarketplaceListing {self.scene_package.name}>"

class ListingRating(Base):
    """列表评分"""
    __tablename__ = 'listing_ratings'

    id = Column(String, primary_key=True, index=True)
    listing_id = Column(String, ForeignKey('marketplace_listings.id'), nullable=False, index=True)
    user_id = Column(String, ForeignKey('users.id'), nullable=False)

    # 评分 (1-5)
    rating = Column(Integer, nullable=False)

    # 评论
    comment = Column(Text)

    # 是否推荐
    is_recommended = Column(Boolean, default=False)

    # 时间戳
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # 唯一约束（一个用户对一个列表只能评分一次）
    __table_args__ = (
        Base.__table_args__,
    )

    def __repr__(self):
        return f"<ListingRating listing={self.listing_id} rating={self.rating}>"

class UserFavorite(Base):
    """用户收藏"""
    __tablename__ = 'user_favorites'

    id = Column(String, primary_key=True, index=True)
    user_id = Column(String, ForeignKey('users.id'), nullable=False, index=True)
    listing_id = Column(String, ForeignKey('marketplace_listings.id'), nullable=False, index=True)

    created_at = Column(DateTime, default=datetime.utcnow)

    # 关系
    user = relationship("User", back_populates="favorites")
    listing = relationship("MarketplaceListing")

class DownloadRecord(Base):
    """下载记录"""
    __tablename__ = 'download_records'

    id = Column(String, primary_key=True, index=True)
    listing_id = Column(String, ForeignKey('marketplace_listings.id'), nullable=False)
    user_id = Column(String, ForeignKey('users.id'))

    downloaded_at = Column(DateTime, default=datetime.utcnow)

    # 关系
    listing = relationship("MarketplaceListing")
    user = relationship("User")
```

---

## 市场服务

```python
# app/services/marketplace.py
from typing import Dict, Any, List, Optional
from sqlalchemy.orm import Session
from sqlalchemy import and_, or_, func, desc

from app.db.models.marketplace import (
    MarketplaceListing,
    ListingRating,
    UserFavorite,
    DownloadRecord,
)
from app.db.models.scene import ScenePackage
from app.core.security import generate_id

class MarketplaceService:
    """市场服务"""

    def __init__(self, db: Session):
        self.db = db

    def publish_to_marketplace(
        self,
        scene_package_id: str,
        publisher_id: str,
        metadata: Dict[str, Any],
    ) -> MarketplaceListing:
        """发布到市场"""
        # 检查是否已发布
        existing = self.db.query(MarketplaceListing)\
            .filter(MarketplaceListing.scene_package_id == scene_package_id)\
            .first()

        if existing:
            raise ValueError("该场景包已发布")

        # 创建市场列表
        listing = MarketplaceListing(
            id=generate_id('listing'),
            scene_package_id=scene_package_id,
            publisher_id=publisher_id,
            status='pending',
            tags=metadata.get('tags', []),
            category=metadata.get('category'),
            difficulty=metadata.get('difficulty'),
            player_count_min=metadata.get('player_count_min'),
            player_count_max=metadata.get('player_count_max'),
            duration_hours=metadata.get('duration_hours'),
            language=metadata.get('language', 'zh'),
            version=metadata.get('version', '1.0.0'),
            license=metadata.get('license', 'CC-BY-NC-SA-4.0'),
        )

        self.db.add(listing)
        self.db.commit()
        self.db.refresh(listing)

        return listing

    def approve_listing(
        self,
        listing_id: str,
        reviewer_id: str,
    ) -> Optional[MarketplaceListing]:
        """批准发布"""
        listing = self.db.query(MarketplaceListing)\
            .filter(MarketplaceListing.id == listing_id)\
            .first()

        if not listing:
            return None

        listing.status = 'approved'
        listing.reviewed_by = reviewer_id
        listing.reviewed_at = datetime.utcnow()

        self.db.commit()
        self.db.refresh(listing)

        return listing

    def reject_listing(
        self,
        listing_id: str,
        reviewer_id: str,
        reason: str,
    ) -> Optional[MarketplaceListing]:
        """拒绝发布"""
        listing = self.db.query(MarketplaceListing)\
            .filter(MarketplaceListing.id == listing_id)\
            .first()

        if not listing:
            return None

        listing.status = 'rejected'
        listing.reviewed_by = reviewer_id
        listing.reviewed_at = datetime.utcnow()
        listing.rejection_reason = reason

        self.db.commit()
        self.db.refresh(listing)

        return listing

    def get_approved_listings(
        self,
        category: str = None,
        difficulty: str = None,
        language: str = None,
        tags: list = None,
        sort_by: str = 'updated',
        limit: int = 50,
        offset: int = 0,
    ) -> List[Dict[str, Any]]:
        """获取已批准的列表"""
        query = self.db.query(MarketplaceListing)\
            .filter(MarketplaceListing.status == 'approved')\
            .join(ScenePackage)

        if category:
            query = query.filter(MarketplaceListing.category == category)

        if difficulty:
            query = query.filter(MarketplaceListing.difficulty == difficulty)

        if language:
            query = query.filter(MarketplaceListing.language == language)

        if tags:
            query = query.filter(MarketplaceListing.tags.overlap(tags))

        # 排序
        if sort_by == 'updated':
            query = query.order_by(desc(MarketplaceListing.updated_at))
        elif sort_by == 'downloads':
            query = query.order_by(desc(MarketplaceListing.download_count))
        elif sort_by == 'rating':
            query = query.order_by(desc(MarketplaceListing.average_rating))
        elif sort_by == 'newest':
            query = query.order_by(desc(MarketplaceListing.published_at))

        listings = query.offset(offset).limit(limit).all()

        return [
            {
                "id": listing.id,
                "scene_package_id": listing.scene_package_id,
                "name": listing.scene_package.metadata.get('name'),
                "description": listing.scene_package.metadata.get('description'),
                "thumbnail": listing.scene_package.thumbnail_path,
                "author": listing.publisher.username,
                "category": listing.category,
                "difficulty": listing.difficulty,
                "tags": listing.tags,
                "player_count_min": listing.player_count_min,
                "player_count_max": listing.player_count_max,
                "duration_hours": listing.duration_hours,
                "average_rating": listing.average_rating,
                "rating_count": listing.rating_count,
                "download_count": listing.download_count,
                "view_count": listing.view_count,
                "published_at": listing.published_at.isoformat(),
                "updated_at": listing.updated_at.isoformat(),
            }
            for listing in listings
        ]

    def get_listing_details(
        self,
        listing_id: str,
        increment_view: bool = True,
    ) -> Optional[Dict[str, Any]]:
        """获取列表详情"""
        listing = self.db.query(MarketplaceListing)\
            .filter(MarketplaceListing.id == listing_id)\
            .first()

        if not listing:
            return None

        # 增加浏览计数
        if increment_view:
            listing.view_count += 1
            self.db.commit()

        # 获取评分分布
        rating_distribution = self.db.query(
            ListingRating.rating,
            func.count(ListingRating.id),
        )\
            .filter(ListingRating.listing_id == listing_id)\
            .group_by(ListingRating.rating)\
            .all()

        distribution = {1: 0, 2: 0, 3: 0, 4: 0, 5: 0}
        for rating, count in rating_distribution:
            distribution[rating] = count

        return {
            "id": listing.id,
            "scene_package_id": listing.scene_package_id,
            "name": listing.scene_package.metadata.get('name'),
            "description": listing.scene_package.metadata.get('description'),
            "long_description": listing.scene_package.metadata.get('long_description'),
            "thumbnail": listing.scene_package.thumbnail_path,
            "author": listing.publisher.username,
            "author_id": listing.publisher_id,
            "category": listing.category,
            "difficulty": listing.difficulty,
            "tags": listing.tags,
            "player_count_min": listing.player_count_min,
            "player_count_max": listing.player_count_max,
            "duration_hours": listing.duration_hours,
            "language": listing.language,
            "version": listing.version,
            "license": listing.license,
            "average_rating": listing.average_rating,
            "rating_count": listing.rating_count,
            "rating_distribution": distribution,
            "download_count": listing.download_count,
            "view_count": listing.view_count,
            "favorite_count": listing.favorite_count,
            "published_at": listing.published_at.isoformat(),
            "updated_at": listing.updated_at.isoformat(),
        }

    def rate_listing(
        self,
        listing_id: str,
        user_id: str,
        rating: int,
        comment: str = None,
        is_recommended: bool = False,
    ) -> ListingRating:
        """评分列表"""
        # 检查是否已评分
        existing = self.db.query(ListingRating)\
            .filter(
                ListingRating.listing_id == listing_id,
                ListingRating.user_id == user_id,
            )\
            .first()

        if existing:
            # 更新评分
            existing.rating = rating
            existing.comment = comment
            existing.is_recommended = is_recommended
            existing.updated_at = datetime.utcnow()
            self.db.commit()
            self.db.refresh(existing)
            return existing

        # 创建新评分
        new_rating = ListingRating(
            id=generate_id('rating'),
            listing_id=listing_id,
            user_id=user_id,
            rating=rating,
            comment=comment,
            is_recommended=is_recommended,
        )

        self.db.add(new_rating)

        # 更新列表评分
        self._update_listing_rating(listing_id)

        self.db.commit()
        self.db.refresh(new_rating)

        return new_rating

    def _update_listing_rating(self, listing_id: str):
        """更新列表评分统计"""
        avg_rating = self.db.query(
            func.avg(ListingRating.rating),
        )\
            .filter(ListingRating.listing_id == listing_id)\
            .scalar()

        rating_count = self.db.query(
            func.count(ListingRating.id),
        )\
            .filter(ListingRating.listing_id == listing_id)\
            .scalar()

        listing = self.db.query(MarketplaceListing)\
            .filter(MarketplaceListing.id == listing_id)\
            .first()

        if listing:
            listing.average_rating = round(avg_rating or 0, 1)
            listing.rating_count = rating_count or 0

    def toggle_favorite(
        self,
        listing_id: str,
        user_id: str,
    ) -> bool:
        """切换收藏状态"""
        existing = self.db.query(UserFavorite)\
            .filter(
                UserFavorite.listing_id == listing_id,
                UserFavorite.user_id == user_id,
            )\
            .first()

        listing = self.db.query(MarketplaceListing)\
            .filter(MarketplaceListing.id == listing_id)\
            .first()

        if existing:
            # 取消收藏
            self.db.delete(existing)
            if listing:
                listing.favorite_count -= 1
            self.db.commit()
            return False
        else:
            # 添加收藏
            favorite = UserFavorite(
                id=generate_id('favorite'),
                listing_id=listing_id,
                user_id=user_id,
            )
            self.db.add(favorite)
            if listing:
                listing.favorite_count += 1
            self.db.commit()
            return True

    def record_download(
        self,
        listing_id: str,
        user_id: str = None,
    ):
        """记录下载"""
        record = DownloadRecord(
            id=generate_id('download'),
            listing_id=listing_id,
            user_id=user_id,
        )

        self.db.add(record)

        # 更新下载计数
        listing = self.db.query(MarketplaceListing)\
            .filter(MarketplaceListing.id == listing_id)\
            .first()

        if listing:
            listing.download_count += 1

        self.db.commit()

    def get_user_favorites(
        self,
        user_id: str,
    ) -> List[Dict[str, Any]]:
        """获取用户收藏"""
        favorites = self.db.query(UserFavorite)\
            .filter(UserFavorite.user_id == user_id)\
            .join(MarketplaceListing)\
            .join(ScenePackage)\
            .order_by(UserFavorite.created_at.desc())\
            .all()

        return [
            {
                "id": fav.listing.id,
                "scene_package_id": fav.listing.scene_package_id,
                "name": fav.listing.scene_package.metadata.get('name'),
                "thumbnail": fav.listing.scene_package.thumbnail_path,
                "author": fav.listing.publisher.username,
                "average_rating": fav.listing.average_rating,
                "favorited_at": fav.created_at.isoformat(),
            }
            for fav in favorites
        ]

    def get_user_listings(
        self,
        user_id: str,
        include_all: bool = False,
    ) -> List[Dict[str, Any]]:
        """获取用户发布的列表"""
        query = self.db.query(MarketplaceListing)\
            .filter(MarketplaceListing.publisher_id == user_id)

        if not include_all:
            query = query.filter(MarketplaceListing.status == 'approved')

        listings = query.order_by(MarketplaceListing.published_at.desc()).all()

        return [
            {
                "id": listing.id,
                "scene_package_id": listing.scene_package_id,
                "name": listing.scene_package.metadata.get('name'),
                "status": listing.status,
                "download_count": listing.download_count,
                "average_rating": listing.average_rating,
                "published_at": listing.published_at.isoformat(),
            }
            for listing in listings
        ]
```

---

## 市场 API

```python
# app/api/marketplace.py
from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy.orm import Session
from pydantic import BaseModel
from typing import List, Optional

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.marketplace import MarketplaceService

router = APIRouter(prefix="/marketplace", tags=["marketplace"])

class PublishRequest(BaseModel):
    scene_package_id: str
    category: str
    difficulty: str
    player_count_min: int
    player_count_max: int
    duration_hours: int
    tags: list = None
    language: str = 'zh'
    version: str = '1.0.0'
    license: str = 'CC-BY-NC-SA-4.0'

class RateRequest(BaseModel):
    rating: int
    comment: str = None
    is_recommended: bool = False

@router.post("/publish")
async def publish_to_marketplace(
    request: PublishRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """发布到市场"""
    service = MarketplaceService(db)

    try:
        listing = service.publish_to_marketplace(
            request.scene_package_id,
            current_user.id,
            request.dict(),
        )
        return {"listing_id": listing.id, "status": listing.status}
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))

@router.get("")
async def browse_marketplace(
    category: Optional[str] = None,
    difficulty: Optional[str] = None,
    language: Optional[str] = None,
    tags: Optional[str] = None,
    sort_by: str = 'updated',
    limit: int = Query(50, ge=1, le=100),
    offset: int = Query(0, ge=0),
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """浏览市场"""
    service = MarketplaceService(db)

    tag_list = tags.split(',') if tags else None
    listings = service.get_approved_listings(
        category=category,
        difficulty=difficulty,
        language=language,
        tags=tag_list,
        sort_by=sort_by,
        limit=limit,
        offset=offset,
    )

    return {"listings": listings}

@router.get("/{listing_id}")
async def get_listing_details(
    listing_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取列表详情"""
    service = MarketplaceService(db)
    listing = service.get_listing_details(listing_id)

    if not listing:
        raise HTTPException(status_code=404, detail="列表不存在")

    return listing

@router.post("/{listing_id}/rate")
async def rate_listing(
    listing_id: str,
    request: RateRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """评分列表"""
    service = MarketplaceService(db)

    if request.rating < 1 or request.rating > 5:
        raise HTTPException(status_code=400, detail="评分必须在1-5之间")

    rating = service.rate_listing(
        listing_id,
        current_user.id,
        request.rating,
        request.comment,
        request.is_recommended,
    )

    return {"message": "评分成功", "rating": rating.rating}

@router.post("/{listing_id}/favorite")
async def toggle_favorite(
    listing_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """切换收藏"""
    service = MarketplaceService(db)
    is_favorited = service.toggle_favorite(listing_id, current_user.id)

    return {"is_favorited": is_favorited}

@router.post("/{listing_id}/download")
async def download_listing(
    listing_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """记录下载"""
    service = MarketplaceService(db)
    service.record_download(listing_id, current_user.id)

    return {"message": "下载已记录"}

@router.get("/favorites/my")
async def get_my_favorites(
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取我的收藏"""
    service = MarketplaceService(db)
    favorites = service.get_user_favorites(current_user.id)

    return {"favorites": favorites}

@router.get("/published/my")
async def get_my_listings(
    include_all: bool = False,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取我发布的列表"""
    service = MarketplaceService(db)
    listings = service.get_user_listings(current_user.id, include_all)

    return {"listings": listings}
```

---

## 前端市场组件

```tsx
// frontend/src/components/marketplace/MarketplaceBrowser.tsx
import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { Search, Star, Download, Heart, Eye } from 'lucide-react'
import { useToast } from '@/hooks/use-toast'

interface Listing {
  id: string
  scene_package_id: string
  name: string
  description: string
  thumbnail: string
  author: string
  category: string
  difficulty: string
  tags: string[]
  player_count_min: number
  player_count_max: number
  duration_hours: number
  average_rating: number
  rating_count: number
  download_count: number
  view_count: number
  published_at: string
}

export function MarketplaceBrowser() {
  const [listings, setListings] = useState<Listing[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [category, setCategory] = useState<string>('')
  const [difficulty, setDifficulty] = useState<string>('')
  const [sortBy, setSortBy] = useState('updated')
  const { toast } = useToast()

  useEffect(() => {
    fetchListings()
  }, [category, difficulty, sortBy])

  const fetchListings = async () => {
    setLoading(true)
    try {
      const params = new URLSearchParams()
      if (category) params.append('category', category)
      if (difficulty) params.append('difficulty', difficulty)
      params.append('sort_by', sortBy)

      const response = await fetch(`/api/marketplace?${params}`)
      if (response.ok) {
        const data = await response.json()
        setListings(data.listings)
      }
    } catch (error) {
      console.error('Failed to fetch listings:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleDownload = async (listingId: string) => {
    try {
      await fetch(`/api/marketplace/${listingId}/download`, { method: 'POST' })

      // 导入场景包
      // ...

      toast({
        title: '下载成功',
        description: '场景包已添加到你的库中',
      })
    } catch (error) {
      console.error('Failed to download:', error)
    }
  }

  const handleFavorite = async (listingId: string) => {
    try {
      const response = await fetch(`/api/marketplace/${listingId}/favorite`, {
        method: 'POST',
      })

      if (response.ok) {
        const data = await response.json()
        toast({
          title: data.is_favorited ? '已收藏' : '已取消收藏',
        })
      }
    } catch (error) {
      console.error('Failed to favorite:', error)
    }
  }

  const filteredListings = listings.filter(
    (listing) =>
      listing.name.toLowerCase().includes(search.toLowerCase()) ||
      listing.description.toLowerCase().includes(search.toLowerCase()) ||
      listing.tags.some((tag) => tag.toLowerCase().includes(search.toLowerCase()))
  )

  return (
    <div className="space-y-6">
      <div className="flex gap-4">
        <div className="flex-1 relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="搜索场景包..."
            className="pl-10"
          />
        </div>

        <Select value={category} onValueChange={setCategory}>
          <SelectTrigger className="w-[150px]">
            <SelectValue placeholder="分类" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="">全部分类</SelectItem>
            <SelectItem value="short_scenario">短篇</SelectItem>
            <SelectItem value="full_campaign">完整战役</SelectItem>
            <SelectItem value="one_shot">单次冒险</SelectItem>
          </SelectContent>
        </Select>

        <Select value={difficulty} onValueChange={setDifficulty}>
          <SelectTrigger className="w-[150px]">
            <SelectValue placeholder="难度" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="">全部难度</SelectItem>
            <SelectItem value="easy">简单</SelectItem>
            <SelectItem value="medium">中等</SelectItem>
            <SelectItem value="hard">困难</SelectItem>
          </SelectContent>
        </Select>

        <Select value={sortBy} onValueChange={setSortBy}>
          <SelectTrigger className="w-[150px]">
            <SelectValue placeholder="排序" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="updated">最近更新</SelectItem>
            <SelectItem value="downloads">最多下载</SelectItem>
            <SelectItem value="rating">最高评分</SelectItem>
            <SelectItem value="newest">最新发布</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {loading ? (
        <div>加载中...</div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {filteredListings.map((listing) => (
            <Card key={listing.id} className="overflow-hidden">
              <div
                className="h-40 bg-cover bg-center"
                style={{ backgroundImage: `url(${listing.thumbnail})` }}
              />
              <CardHeader className="pb-2">
                <CardTitle className="text-lg line-clamp-1">
                  {listing.name}
                </CardTitle>
                <div className="text-sm text-muted-foreground">
                  by {listing.author}
                </div>
              </CardHeader>
              <CardContent className="space-y-2">
                <p className="text-sm line-clamp-2">{listing.description}</p>

                <div className="flex flex-wrap gap-1">
                  {listing.tags.slice(0, 3).map((tag) => (
                    <Badge key={tag} variant="secondary" className="text-xs">
                      {tag}
                    </Badge>
                  ))}
                </div>

                <div className="flex items-center gap-4 text-sm text-muted-foreground">
                  <div className="flex items-center gap-1">
                    <Star className="h-3 w-3 fill-yellow-400 text-yellow-400" />
                    <span>{listing.average_rating.toFixed(1)}</span>
                    <span className="text-xs">({listing.rating_count})</span>
                  </div>
                  <div className="flex items-center gap-1">
                    <Download className="h-3 w-3" />
                    <span>{listing.download_count}</span>
                  </div>
                  <div className="flex items-center gap-1">
                    <Eye className="h-3 w-3" />
                    <span>{listing.view_count}</span>
                  </div>
                </div>

                <div className="flex gap-2">
                  <Button
                    size="sm"
                    className="flex-1"
                    onClick={() => handleDownload(listing.id)}
                  >
                    <Download className="h-4 w-4 mr-1" />
                    下载
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => handleFavorite(listing.id)}
                  >
                    <Heart className="h-4 w-4" />
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/db/models/marketplace.py` | 创建 | 市场数据模型 |
| `app/services/marketplace.py` | 创建 | 市场服务 |
| `app/api/marketplace.py` | 创建 | 市场 API |
| `frontend/src/components/marketplace/MarketplaceBrowser.tsx` | 创建 | 市场浏览组件 |
| `frontend/src/components/marketplace/ListingDetail.tsx` | 创建 | 列表详情组件 |
| `frontend/src/components/marketplace/PublishForm.tsx` | 创建 | 发布表单组件 |

---

## 验收标准

- [ ] 发布功能正常
- [ ] 浏览和搜索有效
- [ ] 评分系统准确
- [ ] 收藏功能正常
- [ ] 下载统计正确
- [ ] 审核流程完整

---

## 参考文档

- M4-001: 场景包系统
- M4-007: 场景包导入功能
- Steam Workshop 设计

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
