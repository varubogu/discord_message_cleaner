import json
import aiofiles
from pydantic import BaseModel, RootModel

class Message(BaseModel):
    """メッセージ"""

    ja: str
    en: str


class Messages(RootModel):
    """メッセージ一覧"""

    root: dict[str, Message]

    def __getitem__(self, key: str) -> Message:
        return self.root[key]


class SingletonMessages:
    """メッセージ一覧のシングルトン"""

    _instance = None

    @classmethod
    async def get_instance(cls) -> Messages:
        if cls._instance is None:
            cls._instance = await cls.load()
        return cls._instance

    @classmethod
    async def load(cls) -> Messages:
        """メッセージを読み込む"""

        async with aiofiles.open("messages.json", "r", encoding="utf-8") as f:
            raw = await f.read()
            return Messages.model_validate(json.loads(raw))