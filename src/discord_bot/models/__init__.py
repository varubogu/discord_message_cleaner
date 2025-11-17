# モデル定義をimportすることでModelBase.metadata内に反映され、自動でテーブル作成処理が実行される
from sqlalchemy.ext.asyncio import AsyncEngine
from sqlalchemy.ext.asyncio import AsyncConnection

from discord_bot.models.access_failures import AccessFailures
from discord_bot.models.exclusion_message import ExclusionMessage
from discord_bot.models.monitoring_channels import MonitoringChannels
from discord_bot.models.model_base import ModelBase


async def init_db(conn: AsyncConnection):
    # モデル定義に従ってテーブル作成
    await conn.run_sync(ModelBase.metadata.create_all)


async def drop_db(conn: AsyncConnection):
    await conn.run_sync(ModelBase.metadata.drop_all)


async def init_db_from_engine(engine: AsyncEngine):
    async with engine.begin() as conn:
        await init_db(conn)


async def drop_db_from_engine(engine: AsyncEngine):
    async with engine.begin() as conn:
        await drop_db(conn)


def get_metadata():
    return ModelBase.metadata


class TableNameMapping:

    __CLASSES__ = [
            AccessFailures,
            ExclusionMessage,
            MonitoringChannels

    ]

    __MAPPING__ = [
            {
                'table_name_en': table_class.__tablename__,
                'clsobj': table_class
            } for table_class in __CLASSES__
    ]

    @classmethod
    def getClassObject(cls, table_name_en: str) -> ModelBase:
        """
        get table model
        """
        for cls_info in cls.__MAPPING__:
            if cls_info['table_name_en'] == table_name_en:
                return cls_info['clsobj']
        return None
