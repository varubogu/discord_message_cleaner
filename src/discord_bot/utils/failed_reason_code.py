from enum import Enum


class FailedReasonCode(Enum):
    """失敗理由コード"""

    # 不明な理由
    UNKNOWN = "E0000"

    # Discord API トークンが無効
    INVALID_TOKEN = "E0001"

    # Discord API レートリミット
    RATE_LIMIT = "E0002"

    # サーバーが見つからない
    GUILD_NOT_FOUND = "E0003"

    # サーバーにアクセスできない
    GUILD_ACCESS_DENIED = "E0004"

    # チャンネルが見つからない
    CHANNEL_NOT_FOUND = "E0005"

    # チャンネルにアクセスできない
    CHANNEL_ACCESS_DENIED = "E0006"

    # 権限的にメッセージを削除できない
    MESSAGE_DELETE_PERMISSION_DENIED = "E0007"

    def __str__(self):
        return self.name