from sqlalchemy import BigInteger, Column, DateTime, Index, String, delete, func, select
from sqlalchemy.ext.asyncio import AsyncSession
from discord_bot.models.model_base import ModelBase


class AccessFailures(ModelBase):
    """サーバーにアクセスできなかったサーバーID、チャンネルIDを記憶する場所

    Args:
        Base (_type_): _description_
    """
    __tablename__ = 'access_failures'
    __table_args__ = (
        {'comment': 'サーバーにアクセスできなかったサーバーID、チャンネルIDを記憶する場所'},
        Index('ix_access_failures_guild', 'guild_id', 'failed_at', 'failed_reason_code'),
        Index('ix_access_failures_channel', 'guild_id', 'channel_id', 'failed_at', 'failed_reason_code'),
    )

    # サーバーID
    id = Column(BigInteger, primary_key=True, comment="ID")
    guild_id = Column(BigInteger, comment="サーバーID", nullable=False)
    channel_id = Column(BigInteger, comment="チャンネルID", nullable=True)
    failed_at = Column(DateTime, comment="失敗した日時", nullable=False)
    failed_reason_code = Column(BigInteger, comment="失敗理由コード", nullable=False)
    failed_reason = Column(String, comment="失敗理由", nullable=False)



    async def insert(self, session: AsyncSession):
        """サーバーにアクセスできなかったサーバーIDを記憶する

        Args:
            session (AsyncSession): セッション
        """
        session.add(self)

    @classmethod
    async def count_guild(
        cls,
        session: AsyncSession,
        guild_id: int
    ):
        """サーバーにアクセスできなかったサーバーIDの数を取得する

        Args:
            session (AsyncSession): セッション
            guild_id (int): 対象サーバーID

        Returns:
            int: 引数のサーバーがアクセスできなかった回数
        """
        result = await session.execute(
            select(func.count()).where(
                AccessFailures.guild_id == guild_id,
                AccessFailures.channel_id.is_(None)
            )
        )
        return result.scalar()

    @classmethod
    async def count_channel(
        cls,
        session: AsyncSession,
        guild_id: int,
        channel_id: int
    ):
        """チャンネルにアクセスできなかったチャンネルIDの数を取得する

        Args:
            session (AsyncSession): セッション
            guild_id (int): 対象サーバーID
            channel_id (int): 対象チャンネルID

        Returns:
            int: 引数のチャンネルがアクセスできなかった回数
        """
        result = await session.execute(
            select(func.count()).where(
                AccessFailures.guild_id == guild_id,
                AccessFailures.channel_id == channel_id
            )
        )
        return result.scalar()

    @classmethod
    async def reset_guild(
        cls,
        session: AsyncSession,
        guild_id: int
    ):
        """サーバーにアクセスできなかったサーバーIDの数をリセットする

        Args:
            session (AsyncSession): セッション
            guild_id (int): 対象サーバーID
        """
        await session.execute(
            delete(AccessFailures).where(
                AccessFailures.guild_id == guild_id,
                AccessFailures.channel_id.is_(None)
            )
        )

    @classmethod
    async def reset_channel(
        cls,
        session: AsyncSession,
        guild_id: int,
        channel_id: int
    ):
        """サーバーにアクセスできなかったサーバーIDの数をリセットする

        Args:
            session (AsyncSession): セッション
            guild_id (int): 対象サーバーID
            channel_id (int): 対象チャンネルID
        """
        await session.execute(
            delete(AccessFailures).where(
                AccessFailures.guild_id == guild_id,
                AccessFailures.channel_id == channel_id
            )
        )

