
from typing import Optional, Sequence
import uuid
from sqlalchemy import UUID, BigInteger, Column, select, and_
from sqlalchemy.ext.asyncio import AsyncSession
from discord_bot.models.model_base import ModelBase


class ExclusionMessage(ModelBase):
    """削除対象から除外するメッセージ一覧

    Args:
        Base (_type_): _description_
    """
    __tablename__ = 'exclusion_message'
    __table_args__ = (
        {'comment': '削除対象から除外するメッセージ一覧'}
    )
    
    guild_id = Column(BigInteger, primary_key=True,comment="削除監視するサーバーID")
    channel_id = Column(BigInteger, primary_key=True,comment="削除監視するチャンネルID")
    message_id = Column(BigInteger, primary_key=True,comment="削除対象から除外するメッセージID")

    @classmethod
    async def select(
        cls,
        session: AsyncSession,
        guild_id: int,
        channel_id: int,
        message_id: Optional[int]
    ) -> Optional['ExclusionMessage']:
        result = await session.execute(
            select(ExclusionMessage).filter(
                and_(
                    cls.guild_id == guild_id,
                    cls.channel_id == channel_id,
                    cls.message_id == message_id
                )
            )
        )
        return result.scalars().first()

    @classmethod
    async def select_with_guild(
        cls,
        session: AsyncSession,
        guild_id: int,
    ) -> Sequence['ExclusionMessage']:
        result = await session.execute(
            select(ExclusionMessage).filter(
                ExclusionMessage.guild_id == guild_id
            )
        )
        return result.scalars().all()

    @classmethod
    async def select_with_guild_channel(
        cls,
        session: AsyncSession,
        guild_id: int,
        channel_id: int
    ) -> Sequence['ExclusionMessage']:
        result = await session.execute(
            select(ExclusionMessage).filter(
                and_(
                    ExclusionMessage.guild_id == guild_id,
                    ExclusionMessage.channel_id == channel_id
                )
            )
        )
        return result.scalars().all()

    async def merge(self, session: AsyncSession):
        await session.merge(self)

    async def delete(self, session: AsyncSession):
        await session.delete(self)
