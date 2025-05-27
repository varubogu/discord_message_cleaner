from enum import Enum


class FailedReasonCode(Enum):
    """失敗理由コード"""

    # 不明な理由
    UNKNOWN = 0

    # Discord API トークンが無効
    INVALID_TOKEN = 1

    # Discord API レートリミット
    RATE_LIMIT = 2

    # サーバーが見つからない
    GUILD_NOT_FOUND = 3

    # サーバーにアクセスできない
    GUILD_ACCESS_DENIED = 4

    # チャンネルが見つからない
    CHANNEL_NOT_FOUND = 5

    # チャンネルにアクセスできない
    CHANNEL_ACCESS_DENIED = 6

    # 権限的にメッセージを削除できない
    MESSAGE_DELETE_PERMISSION_DENIED = 7

    def __str__(self):
        return self.name