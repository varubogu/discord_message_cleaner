from enum import Enum


class FailedReasonCode(Enum):
    """失敗理由コード"""

    # 不明な理由
    UNKNOWN = "E0001"

    # Discord API トークンが無効
    INVALID_TOKEN = "E0002"

    # Discord API レートリミット
    RATE_LIMIT = "E0003"

    # サーバーが見つからない
    GUILD_NOT_FOUND = "E0004"

    # サーバーにアクセスできない
    GUILD_ACCESS_DENIED = "E0005"

    # チャンネルが見つからない
    CHANNEL_NOT_FOUND = "E0006"

    # チャンネルにアクセスできない
    CHANNEL_ACCESS_DENIED = "E0007"

    # 権限的にメッセージを削除できない
    MESSAGE_DELETE_PERMISSION_DENIED = "E0008"

    def __str__(self):
        return self.name