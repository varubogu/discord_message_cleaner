from discord_bot.models.model_base import ModelBase
from sqlalchemy import Column


async def get_column_names(table_cls: ModelBase) -> list[str]:
    return [column.name for column in table_cls.__table__.columns]


async def get_columns(table_cls: ModelBase) -> list[Column]:
    return table_cls.__table__.columns
