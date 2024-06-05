from typing import Optional
from datetime import datetime
from result import Result, Ok, Err
from sqlalchemy import BigInteger, Column, Interval, and_, select, text
from sqlalchemy.ext.asyncio import AsyncSession
from discord_bot.models.model_base import ModelBase


class MonitoringChannels(ModelBase):
    """削除監視対象のチャンネル一覧

    Args:
        Base (_type_): _description_
    """
    __tablename__ = 'monitoring_channels'
    __table_args__ = (
        {'comment': '削除対象のチャンネル一覧'}
    )
    
    guild_id = Column(BigInteger, primary_key=True, comment="削除監視するサーバーID")
    channel_id = Column(BigInteger, primary_key=True, comment="削除監視するチャンネルID")
    interval = Column(Interval, comment="削除期間")

    @classmethod
    async def select(
        cls,
        session: AsyncSession,
        guild_id: int,
        channel_id: int
    ) -> Optional['MonitoringChannels']:
        result = await session.execute(
            select(MonitoringChannels).filter(
                and_(
                    cls.guild_id == guild_id,
                    cls.channel_id == channel_id
                )
            )
        )
        return result.scalars().first()

    @classmethod
    async def select_with_guild(
        cls,
        session: AsyncSession,
        guild_id: int
    ):
        result = await session.execute(
            select(MonitoringChannels).filter(
                MonitoringChannels.guild_id == guild_id
            )
        )
        return result.scalars().all()

    @classmethod
    async def select_remove_range(
        cls,
        session: AsyncSession,
        guild_id: int,
        channel_id: int
    ) -> Result[datetime, str]:

        range_name = "remove_range_datetime"

        query = text(f"""
                SELECT NOW() - {cls.interval.name} as {range_name}
                FROM {cls.__tablename__}
                WHERE {cls.guild_id.name} = :{cls.guild_id.name}
                AND {cls.channel_id.name} = :{cls.channel_id.name}
            """)
        params = {
            cls.guild_id.name: guild_id,
            cls.channel_id.name: channel_id
        }

        result = await session.execute(query, params)
        data = result.fetchone()
        if data is None or len(data) == 0:
            return Err("設定が見つかりません。")
        else:
            return Ok(data[0])
        

    async def merge(self, session: AsyncSession):
        await session.merge(self)

    async def delete(self, session: AsyncSession):
        await session.delete(self)
