# M4-015: 实现场景包评分

**任务ID**: M4-015
**任务名称**: 实现场景包评分
**预估时间**: 3 小时
**优先级**: P1
**依赖**: M4-014 (场景包分享)
**状态**: 待开始

---

## 任务概述

实现场景包的评分功能，允许用户对场景包进行星级评分（1-5星），计算平均分、评分分布，支持按评分排序和筛选。防止重复评分，提供评分统计功能。

---

## 子任务拆解

| ID | 子任务描述 | 预估时间 | 依赖 | 状态 |
|----|-----------|----------|------|------|
| M4-015-01 | 设计评分数据模型和数据库表结构 | 0.5h | M4-014 | 待开始 |
| M4-015-02 | 实现评分创建和更新服务 | 1h | M4-015-01 | 待开始 |
| M4-015-03 | 实现评分统计和聚合计算 | 0.5h | M4-015-02 | 待开始 |
| M4-015-04 | 实现评分查询和排序API | 0.5h | M4-015-03 | 待开始 |
| M4-015-05 | 实现前端评分组件和展示 | 0.5h | M4-015-04 | 待开始 |

**总预估时间**: 3 小时

---

## Python 后端实现

### 1. 数据库模型

```python
# backend/app/models/rating.py
from sqlalchemy import Column, Integer, String, DateTime, ForeignKey, Index, UniqueConstraint
from sqlalchemy.orm import relationship
from datetime import datetime
import uuid

from app.db.base_class import Base

class ScenarioRating(Base):
    """场景包评分记录"""
    __tablename__ = "scenario_ratings"

    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    scenario_id = Column(String(36), ForeignKey("scenarios.id"), nullable=False, index=True)
    user_id = Column(Integer, ForeignKey("users.id"), nullable=False, index=True)

    # 评分（1-5星）
    rating = Column(Integer, nullable=False)

    # 评论（可选）
    comment = Column(String(2000), nullable=True)

    # 时间戳
    created_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # 关系
    scenario = relationship("Scenario", back_populates="ratings")
    user = relationship("User", back_populates="scenario_ratings")

    # 唯一约束：每个用户对每个场景包只能评分一次
    __table_args__ = (
        UniqueConstraint('scenario_id', 'user_id', name='uq_scenario_user_rating'),
        Index('ix_scenario_ratings_scenario_rating', 'scenario_id', 'rating'),
    )

    def __repr__(self):
        return f"<ScenarioRating(scenario={self.scenario_id}, user={self.user_id}, rating={self.rating})>"

    @property
    def stars(self) -> str:
        """返回星星字符串"""
        return "★" * self.rating + "☆" * (5 - self.rating)
```

### 2. 评分服务

```python
# backend/app/services/rating_service.py
from typing import Optional, List, Dict, Any
from datetime import datetime
from sqlalchemy.orm import Session
from sqlalchemy import func, case

from app.models.rating import ScenarioRating
from app.core.exceptions import ParseError

class RatingService:
    """评分服务"""

    def __init__(self, db: Session):
        self.db = db

    def create_or_update_rating(
        self,
        scenario_id: str,
        user_id: int,
        rating: int,
        comment: Optional[str] = None
    ) -> ScenarioRating:
        """
        创建或更新评分

        Args:
            scenario_id: 场景包ID
            user_id: 用户ID
            rating: 评分（1-5）
            comment: 评论（可选）

        Returns:
            ScenarioRating: 评分记录

        Raises:
            ParseError: 评分无效时抛出
        """
        # 验证评分范围
        if not 1 <= rating <= 5:
            raise ParseError("评分必须在1-5之间")

        # 查找现有评分
        existing_rating = self.db.query(ScenarioRating).filter(
            ScenarioRating.scenario_id == scenario_id,
            ScenarioRating.user_id == user_id
        ).first()

        if existing_rating:
            # 更新现有评分
            existing_rating.rating = rating
            existing_rating.comment = comment
            existing_rating.updated_at = datetime.utcnow()
            self.db.commit()
            self.db.refresh(existing_rating)
            return existing_rating
        else:
            # 创建新评分
            new_rating = ScenarioRating(
                scenario_id=scenario_id,
                user_id=user_id,
                rating=rating,
                comment=comment
            )
            self.db.add(new_rating)
            self.db.commit()
            self.db.refresh(new_rating)
            return new_rating

    def get_user_rating(
        self,
        scenario_id: str,
        user_id: int
    ) -> Optional[ScenarioRating]:
        """获取用户对场景包的评分"""
        return self.db.query(ScenarioRating).filter(
            ScenarioRating.scenario_id == scenario_id,
            ScenarioRating.user_id == user_id
        ).first()

    def delete_rating(
        self,
        scenario_id: str,
        user_id: int
    ) -> bool:
        """删除评分"""
        rating = self.db.query(ScenarioRating).filter(
            ScenarioRating.scenario_id == scenario_id,
            ScenarioRating.user_id == user_id
        ).first()

        if not rating:
            return False

        self.db.delete(rating)
        self.db.commit()
        return True

    def get_scenario_ratings(
        self,
        scenario_id: str,
        skip: int = 0,
        limit: int = 20
    ) -> List[ScenarioRating]:
        """获取场景包的评分列表"""
        return self.db.query(ScenarioRating).filter(
            ScenarioRating.scenario_id == scenario_id
        ).order_by(
            ScenarioRating.created_at.desc()
        ).offset(skip).limit(limit).all()

    def get_rating_statistics(
        self,
        scenario_id: str
    ) -> Dict[str, Any]:
        """
        获取场景包评分统计

        Returns:
            Dict: {
                "average": 4.5,           # 平均分
                "total_count": 100,        # 总评分数
                "distribution": {          # 评分分布
                    "1": 5,
                    "2": 10,
                    "3": 15,
                    "4": 30,
                    "5": 40
                }
            }
        """
        # 获取总评分数和平均分
        stats = self.db.query(
            func.count(ScenarioRating.id).label('total_count'),
            func.avg(ScenarioRating.rating).label('average')
        ).filter(
            ScenarioRating.scenario_id == scenario_id
        ).first()

        total_count = stats.total_count or 0
        average = round(float(stats.average), 2) if stats.average else 0.0

        # 获取评分分布
        distribution = {}
        for star in range(1, 6):
            count = self.db.query(func.count(ScenarioRating.id)).filter(
                ScenarioRating.scenario_id == scenario_id,
                ScenarioRating.rating == star
            ).scalar()
            distribution[str(star)] = count or 0

        return {
            "average": average,
            "total_count": total_count,
            "distribution": distribution
        }

    def get_top_rated_scenarios(
        self,
        limit: int = 10,
        min_ratings: int = 5
    ) -> List[Dict[str, Any]]:
        """
       获取评分最高的场景包

        Args:
            limit: 返回数量
            min_ratings: 最小评分数门槛

        Returns:
            List: [{"scenario_id": "...", "average": 4.5, "count": 100}, ...]
        """
        # 子查询：获取每个场景包的评分统计
        stats = self.db.query(
            ScenarioRating.scenario_id,
            func.count(ScenarioRating.id).label('count'),
            func.avg(ScenarioRating.rating).label('average')
        ).group_by(
            ScenarioRating.scenario_id
        ).having(
            func.count(ScenarioRating.id) >= min_ratings
        ).subquery()

        # 查询并排序
        results = self.db.query(
            stats.c.scenario_id,
            stats.c.average,
            stats.c.count
        ).order_by(
            stats.c.average.desc(),
            stats.c.count.desc()
        ).limit(limit).all()

        return [
            {
                "scenario_id": row.scenario_id,
                "average": round(float(row.average), 2),
                "count": row.count
            }
            for row in results
        ]

    def get_recent_ratings(
        self,
        limit: int = 20
    ) -> List[ScenarioRating]:
        """获取最近的评分记录"""
        return self.db.query(ScenarioRating).order_by(
            ScenarioRating.created_at.desc()
        ).limit(limit).all()

    def get_user_rating_history(
        self,
        user_id: int,
        skip: int = 0,
        limit: int = 20
    ) -> List[ScenarioRating]:
        """获取用户的评分历史"""
        return self.db.query(ScenarioRating).filter(
            ScenarioRating.user_id == user_id
        ).order_by(
            ScenarioRating.created_at.desc()
        ).offset(skip).limit(limit).all()

    def calculate_percentile(
        self,
        scenario_id: str
    ) -> float:
        """
        计算场景包评分的百分位数

        Returns:
            float: 百分位数（0-100），表示该场景包超越了多少其他场景包
        """
        # 获取该场景包的平均分
        stats = self.get_rating_statistics(scenario_id)
        average = stats["average"]

        if stats["total_count"] == 0:
            return 0.0

        # 获取所有有评分的场景包的平均分
        all_averages = self.db.query(
            func.avg(ScenarioRating.rating)
        ).group_by(
            ScenarioRating.scenario_id
        ).all()

        if not all_averages:
            return 0.0

        # 计算百分位数
        all_values = [float(row[0]) for row in all_averages]
        better_count = sum(1 for v in all_values if v < average)
        percentile = (better_count / len(all_values)) * 100

        return round(percentile, 2)
```

### 3. API 路由

```python
# backend/app/api/v1/endpoints/rating.py
from fastapi import APIRouter, Depends, HTTPException, Query
from typing import Optional, List
from sqlalchemy.orm import Session

from app.api.deps import get_current_user, get_db
from app.models.user import User
from app.services.rating_service import RatingService
from pydantic import BaseModel

router = APIRouter()

class RatingCreate(BaseModel):
    """创建评分请求"""
    rating: int
    comment: Optional[str] = None

    class Config:
        schema_extra = {
            "example": {
                "rating": 5,
                "comment": "非常棒的场景包！"
            }
        }

@router.post("/scenarios/{scenario_id}/rating")
async def create_or_update_rating(
    scenario_id: str,
    rating_data: RatingCreate,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """
    创建或更新场景包评分

    - **scenario_id**: 场景包ID
    - **rating**: 评分（1-5星）
    - **comment**: 评论（可选）
    """
    rating_service = RatingService(db)

    try:
        rating = rating_service.create_or_update_rating(
            scenario_id=scenario_id,
            user_id=current_user.id,
            rating=rating_data.rating,
            comment=rating_data.comment
        )

        return {
            "success": True,
            "rating": {
                "id": rating.id,
                "rating": rating.rating,
                "comment": rating.comment,
                "created_at": rating.created_at,
                "updated_at": rating.updated_at
            }
        }

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/scenarios/{scenario_id}/rating")
async def get_scenario_rating(
    scenario_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """
    获取用户对场景包的评分

    - **scenario_id**: 场景包ID
    """
    rating_service = RatingService(db)
    rating = rating_service.get_user_rating(
        scenario_id=scenario_id,
        user_id=current_user.id
    )

    if not rating:
        raise HTTPException(status_code=404, detail="未找到评分")

    return {
        "id": rating.id,
        "rating": rating.rating,
        "comment": rating.comment,
        "created_at": rating.created_at,
        "updated_at": rating.updated_at
    }

@router.delete("/scenarios/{scenario_id}/rating")
async def delete_rating(
    scenario_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """
    删除场景包评分

    - **scenario_id**: 场景包ID
    """
    rating_service = RatingService(db)
    success = rating_service.delete_rating(
        scenario_id=scenario_id,
        user_id=current_user.id
    )

    if not success:
        raise HTTPException(status_code=404, detail="未找到评分")

    return {"success": True}

@router.get("/scenarios/{scenario_id}/ratings")
async def get_scenario_ratings(
    scenario_id: str,
    skip: int = Query(0, ge=0),
    limit: int = Query(20, ge=1, le=100),
    db: Session = Depends(get_db)
):
    """
    获取场景包的评分列表

    - **scenario_id**: 场景包ID
    - **skip**: 跳过记录数
    - **limit**: 返回记录数
    """
    rating_service = RatingService(db)
    ratings = rating_service.get_scenario_ratings(
        scenario_id=scenario_id,
        skip=skip,
        limit=limit
    )

    return {
        "ratings": [
            {
                "id": rating.id,
                "user_id": rating.user_id,
                "rating": rating.rating,
                "comment": rating.comment,
                "created_at": rating.created_at
            }
            for rating in ratings
        ]
    }

@router.get("/scenarios/{scenario_id}/rating/statistics")
async def get_rating_statistics(
    scenario_id: str,
    db: Session = Depends(get_db)
):
    """
    获取场景包评分统计

    - **scenario_id**: 场景包ID
    """
    rating_service = RatingService(db)
    stats = rating_service.get_rating_statistics(scenario_id)
    percentile = rating_service.calculate_percentile(scenario_id)

    return {
        **stats,
        "percentile": percentile
    }

@router.get("/scenarios/top-rated")
async def get_top_rated_scenarios(
    limit: int = Query(10, ge=1, le=50),
    min_ratings: int = Query(5, ge=1),
    db: Session = Depends(get_db)
):
    """
    获取评分最高的场景包

    - **limit**: 返回数量
    - **min_ratings**: 最小评分数门槛
    """
    rating_service = RatingService(db)
    results = rating_service.get_top_rated_scenarios(
        limit=limit,
        min_ratings=min_ratings
    )

    return {"scenarios": results}

@router.get("/users/me/ratings")
async def get_my_rating_history(
    skip: int = Query(0, ge=0),
    limit: int = Query(20, ge=1, le=100),
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """
    获取我的评分历史

    - **skip**: 跳过记录数
    - **limit**: 返回记录数
    """
    rating_service = RatingService(db)
    ratings = rating_service.get_user_rating_history(
        user_id=current_user.id,
        skip=skip,
        limit=limit
    )

    return {
        "ratings": [
            {
                "id": rating.id,
                "scenario_id": rating.scenario_id,
                "rating": rating.rating,
                "comment": rating.comment,
                "created_at": rating.created_at
            }
            for rating in ratings
        ]
    }

@router.get("/ratings/recent")
async def get_recent_ratings(
    limit: int = Query(20, ge=1, le=100),
    db: Session = Depends(get_db)
):
    """
    获取最近的评分记录

    - **limit**: 返回记录数
    """
    rating_service = RatingService(db)
    ratings = rating_service.get_recent_ratings(limit=limit)

    return {
        "ratings": [
            {
                "id": rating.id,
                "scenario_id": rating.scenario_id,
                "user_id": rating.user_id,
                "rating": rating.rating,
                "comment": rating.comment,
                "created_at": rating.created_at
            }
            for rating in ratings
        ]
    }
```

---

## TypeScript/React 前端实现

### 1. 评分服务

```typescript
// frontend/src/services/api/rating.ts
import api from './client';

export interface RatingCreate {
  rating: number;
  comment?: string;
}

export interface Rating {
  id: string;
  scenario_id: string;
  user_id: number;
  rating: number;
  comment: string | null;
  created_at: string;
  updated_at: string;
}

export interface RatingStatistics {
  average: number;
  total_count: number;
  distribution: Record<string, number>;
  percentile: number;
}

class RatingService {
  /**
   * 创建或更新评分
   */
  async createOrUpdateRating(
    scenarioId: string,
    data: RatingCreate
  ): Promise<{ success: boolean; rating: Rating }> {
    try {
      const response = await api.post(
        `/api/v1/rating/scenarios/${scenarioId}/rating`,
        data
      );
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '评分失败');
    }
  }

  /**
   * 获取我的评分
   */
  async getMyRating(scenarioId: string): Promise<Rating> {
    try {
      const response = await api.get(
        `/api/v1/rating/scenarios/${scenarioId}/rating`
      );
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '获取评分失败');
    }
  }

  /**
   * 删除评分
   */
  async deleteRating(scenarioId: string): Promise<{ success: boolean }> {
    try {
      const response = await api.delete(
        `/api/v1/rating/scenarios/${scenarioId}/rating`
      );
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '删除评分失败');
    }
  }

  /**
   * 获取场景包评分列表
   */
  async getScenarioRatings(
    scenarioId: string,
    params?: { skip?: number; limit?: number }
  ): Promise<{ ratings: Rating[] }> {
    try {
      const response = await api.get(
        `/api/v1/rating/scenarios/${scenarioId}/ratings`,
        { params }
      );
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '获取评分列表失败');
    }
  }

  /**
   * 获取评分统计
   */
  async getRatingStatistics(scenarioId: string): Promise<RatingStatistics> {
    try {
      const response = await api.get(
        `/api/v1/rating/scenarios/${scenarioId}/rating/statistics`
      );
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '获取统计信息失败');
    }
  }

  /**
   * 获取高分场景包
   */
  async getTopRated(params?: {
    limit?: number;
    min_ratings?: number;
  }): Promise<{ scenarios: Array<{ scenario_id: string; average: number; count: number }> }> {
    try {
      const response = await api.get('/api/v1/rating/scenarios/top-rated', {
        params,
      });
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '获取高分场景包失败');
    }
  }

  /**
   * 获取我的评分历史
   */
  async getMyRatingHistory(params?: {
    skip?: number;
    limit?: number;
  }): Promise<{ ratings: Rating[] }> {
    try {
      const response = await api.get('/api/v1/rating/users/me/ratings', {
        params,
      });
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '获取评分历史失败');
    }
  }
}

export default new RatingService();
```

### 2. 评分组件

```typescript
// frontend/src/components/scenario/RatingStars.tsx
import React, { useState } from 'react';
import { Rate, Input, Button, message, Modal, Statistic, Row, Col, Progress } from 'antd';
import { StarOutlined } from '@ant-design/icons';
import ratingService, { RatingCreate } from '@/services/api/rating';

interface RatingStarsProps {
  scenarioId: string;
  initialRating?: number;
  onRated?: (rating: number) => void;
  readonly?: boolean;
}

const RatingStars: React.FC<RatingStarsProps> = ({
  scenarioId,
  initialRating,
  onRated,
  readonly = false,
}) => {
  const [rating, setRating] = useState(initialRating || 0);
  const [comment, setComment] = useState('');
  const [modalVisible, setModalVisible] = useState(false);
  const [loading, setLoading] = useState(false);

  const handleRate = async (value: number) => {
    if (readonly) return;

    if (value === 0) {
      setModalVisible(true);
      return;
    }

    setRating(value);
    await submitRating(value);
  };

  const submitRating = async (value: number) => {
    setLoading(true);

    try {
      const data: RatingCreate = {
        rating: value,
        comment: comment || undefined,
      };

      await ratingService.createOrUpdateRating(scenarioId, data);
      message.success('评分成功');
      setModalVisible(false);
      setComment('');
      onRated?.(value);
    } catch (error: any) {
      message.error(error.message || '评分失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <Rate
        value={rating}
        onChange={handleRate}
        disabled={readonly}
        character={<StarOutlined />}
        allowHalf={false}
      />

      <Modal
        title="添加评分评论"
        open={modalVisible}
        onOk={() => submitRating(rating)}
        onCancel={() => {
          setModalVisible(false);
          setComment('');
        }}
        confirmLoading={loading}
      >
        <div style={{ marginBottom: 16 }}>
          <Rate
            value={rating}
            onChange={setRating}
            character={<StarOutlined />}
          />
        </div>
        <Input.TextArea
          rows={4}
          placeholder="写下你的评价（可选）"
          value={comment}
          onChange={(e) => setComment(e.target.value)}
          maxLength={2000}
          showCount
        />
      </Modal>
    </>
  );
};

export default RatingStars;
```

```typescript
// frontend/src/components/scenario/RatingDisplay.tsx
import React, { useEffect, useState } from 'react';
import { Card, Row, Col, Statistic, Progress, Tag } from 'antd';
import { StarFilled, TrophyOutlined } from '@ant-design/icons';
import ratingService, { RatingStatistics } from '@/services/api/rating';

interface RatingDisplayProps {
  scenarioId: string;
}

const RatingDisplay: React.FC<RatingDisplayProps> = ({ scenarioId }) => {
  const [stats, setStats] = useState<RatingStatistics | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadStats();
  }, [scenarioId]);

  const loadStats = async () => {
    setLoading(true);
    try {
      const data = await ratingService.getRatingStatistics(scenarioId);
      setStats(data);
    } catch (error) {
      console.error('Failed to load rating statistics:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading || !stats) {
    return null;
  }

  const renderStars = (rating: number) => {
    return (
      <span style={{ color: '#faad14', fontSize: 16 }}>
        {'★'.repeat(Math.floor(rating))}
        {'☆'.repeat(5 - Math.floor(rating))}
      </span>
    );
  };

  return (
    <Card title="评分统计" loading={loading}>
      <Row gutter={16}>
        <Col span={6}>
          <Statistic
            title="平均评分"
            value={stats.average}
            precision={1}
            prefix={<StarFilled />}
            suffix="/ 5"
            valueStyle={{ color: '#faad14' }}
          />
        </Col>
        <Col span={6}>
          <Statistic
            title="评分数"
            value={stats.total_count}
            suffix="人"
          />
        </Col>
        <Col span={6}>
          <Statistic
            title="超越"
            value={stats.percentile}
            precision={1}
            suffix="%"
            prefix={<TrophyOutlined />}
            valueStyle={{ color: '#52c41a' }}
          />
        </Col>
        <Col span={6}>
          <div style={{ textAlign: 'center' }}>
            <div style={{ fontSize: 14, color: '#999', marginBottom: 8 }}>
              推荐度
            </div>
            {renderStars(stats.average)}
          </div>
        </Col>
      </Row>

      {stats.total_count > 0 && (
        <div style={{ marginTop: 24 }}>
          <div style={{ marginBottom: 8 }}>评分分布</div>
          {[5, 4, 3, 2, 1].map((star) => {
            const count = stats.distribution[star] || 0;
            const percent = stats.total_count > 0
              ? (count / stats.total_count) * 100
              : 0;

            return (
              <div key={star} style={{ marginBottom: 8 }}>
                <Row gutter={8} align="middle">
                  <Col span={2}>
                    <Tag color={star >= 4 ? 'green' : star >= 3 ? 'orange' : 'red'}>
                      {star} 星
                    </Tag>
                  </Col>
                  <Col span={18}>
                    <Progress
                      percent={percent}
                      size="small"
                      showInfo={false}
                      strokeColor={star >= 4 ? '#52c41a' : star >= 3 ? '#faad14' : '#ff4d4f'}
                    />
                  </Col>
                  <Col span={4}>
                    <span style={{ fontSize: 12, color: '#999' }}>
                      {count} 人
                    </span>
                  </Col>
                </Row>
              </div>
            );
          })}
        </div>
      )}
    </Card>
  );
};

export default RatingDisplay;
```

---

## 涉及文件清单

### 新建文件

| 文件路径 | 说明 |
|---------|------|
| `/backend/app/models/rating.py` | 评分数据模型 |
| `/backend/app/services/rating_service.py` | 评分服务 |
| `/backend/app/api/v1/endpoints/rating.py` | 评分API路由 |

| 文件路径 | 说明 |
|---------|------|
| `/frontend/src/services/api/rating.ts` | 评分服务API |
| `/frontend/src/components/scenario/RatingStars.tsx` | 评分组件 |
| `/frontend/src/components/scenario/RatingDisplay.tsx` | 评分展示组件 |

---

## 验收标准

### 功能验收

- [ ] 用户可以对场景包评分（1-5星）
- [ ] 用户只能对每个场景包评分一次
- [ ] 支持修改已有评分
- [ ] 支持删除自己的评分
- [ ] 正确计算平均分和评分分布
- [ ] 支持按评分排序场景包
- [ ] 显示评分统计和百分位数

### 数据一致性验收

- [ ] 评分数据正确持久化
- [ ] 评分统计实时更新
- [ ] 防止重复评分的约束生效

### 性能验收

- [ ] 评分操作响应时间 < 500ms
- [ ] 统计查询响应时间 < 1s

---

## 参考文档

### 内部文档

- [M4-014: 场景包分享](/Users/guochangxi/git/monika/docs/tasks/tasks-detailed/M4-014-package-sharing.md)

### 技术文档

- [SQLAlchemy Aggregate Functions](https://docs.sqlalchemy.org/en/14/core/functions.html)
- [Ant Design Rate Component](https://ant.design/components/rate-cn/)

---

**创建日期**: 2026-02-06
**最后更新**: 2026-02-06
