import os
import json
import aiofiles
from pydantic import BaseModel, RootModel

from discord_bot.utils.failed_reason_code import FailedReasonCode

class Message(BaseModel):
    """メッセージ"""

    ja: str
    en: str

    def get_message(self, lang: str) -> str:
        """
        メッセージを取得する

        Args:
            lang: 言語

        Returns:
            メッセージ
        """

        if lang == "ja":
            return self.ja
        elif lang == "en":
            return self.en
        else:
            raise ValueError(f"Invalid language: {lang}")


class Messages(RootModel):
    """メッセージ一覧"""

    root: dict[str, Message]

    def __getitem__(self, key: str) -> Message:
        return self.root[key]

    async def get_message_locale(self) -> str:
        """
        メッセージのロケールを取得する

        Returns:
            メッセージのロケール
        """

        return os.environ.get("MESSAGE_LANGUAGE", "en")

    async def get_log_locale(self) -> str:
        """
        ログのロケールを取得する

        Returns:
            ログのロケール
        """

        return os.environ.get("LOG_LANGUAGE", "en")

    async def get_message(self, key: str, lang: str) -> str:
        """
        メッセージを取得する

        Args:
            key: メッセージのキー

        Returns:
            メッセージ
        """

        if key not in self.root:
            raise ValueError(f"Message not found: {key}")

        message = self[key]
        try:
            return message.get_message(lang)
        except ValueError:
            raise ValueError(f"Language not found: {lang}")

    async def get_log_and_display_message(self, code: FailedReasonCode, display_locale: str) -> tuple[str, str]:
        """
        ログメッセージを出力する

        Args:
            code: エラーコード
            display_locale: 表示言語

        Returns:
            ログメッセージと表示メッセージ
        """

        log_locale = await self.get_log_locale()
        log_message = await self.get_message(code.value, log_locale)
        display_message = await self.get_message(code.value, display_locale)

        return log_message, display_message


class SingletonMessages:
    """メッセージ一覧のシングルトン"""

    _instance: Messages | None = None

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
