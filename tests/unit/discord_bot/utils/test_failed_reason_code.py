import pytest
from discord_bot.utils.failed_reason_code import FailedReasonCode

@pytest.mark.asyncio
async def test_failed_reason_code_values():
    # 各種コードの名称とコードを検証
    assert FailedReasonCode.UNKNOWN.value == "E0000"
    assert FailedReasonCode.INVALID_TOKEN.value == "E0001"
    assert FailedReasonCode.RATE_LIMIT.value == "E0002"
    assert FailedReasonCode.GUILD_NOT_FOUND.value == "E0003"
    assert FailedReasonCode.GUILD_ACCESS_DENIED.value == "E0004"
    assert FailedReasonCode.CHANNEL_NOT_FOUND.value == "E0005"
    assert FailedReasonCode.CHANNEL_ACCESS_DENIED.value == "E0006"
    assert FailedReasonCode.MESSAGE_DELETE_PERMISSION_DENIED.value == "E0007"
    assert FailedReasonCode.MESSAGE_READ_PERMISSION_DENIED.value == "E0008"
    assert FailedReasonCode.MESSAGE_NOT_FOUND.value == "E0009"
    assert FailedReasonCode.MESSAGE_ACCESS_DENIED.value == "E0010"
    assert FailedReasonCode.MESSAGE_READ_HISTORY_PERMISSION_DENIED.value == "E0011"
    assert FailedReasonCode.NO_BOT_USAGE_PERMISSION.value == "E0012"
    assert FailedReasonCode.THREAD_ARCHIVED.value == "E0013"
    assert FailedReasonCode.MESSAGE_DELETE_DENIED.value == "E0014"

@pytest.mark.asyncio
async def test_failed_reason_code_names():
    # 各種コードの名称を検証
    assert str(FailedReasonCode.UNKNOWN) == "UNKNOWN"
    assert str(FailedReasonCode.INVALID_TOKEN) == "INVALID_TOKEN"
    assert str(FailedReasonCode.RATE_LIMIT) == "RATE_LIMIT"
    assert str(FailedReasonCode.GUILD_NOT_FOUND) == "GUILD_NOT_FOUND"
    assert str(FailedReasonCode.GUILD_ACCESS_DENIED) == "GUILD_ACCESS_DENIED"
    assert str(FailedReasonCode.CHANNEL_NOT_FOUND) == "CHANNEL_NOT_FOUND"
    assert str(FailedReasonCode.CHANNEL_ACCESS_DENIED) == "CHANNEL_ACCESS_DENIED"
    assert str(FailedReasonCode.MESSAGE_DELETE_PERMISSION_DENIED) == "MESSAGE_DELETE_PERMISSION_DENIED"
    assert str(FailedReasonCode.MESSAGE_READ_PERMISSION_DENIED) == "MESSAGE_READ_PERMISSION_DENIED"
    assert str(FailedReasonCode.MESSAGE_NOT_FOUND) == "MESSAGE_NOT_FOUND"
    assert str(FailedReasonCode.MESSAGE_ACCESS_DENIED) == "MESSAGE_ACCESS_DENIED"
    assert str(FailedReasonCode.MESSAGE_READ_HISTORY_PERMISSION_DENIED) == "MESSAGE_READ_HISTORY_PERMISSION_DENIED"
    assert str(FailedReasonCode.NO_BOT_USAGE_PERMISSION) == "NO_BOT_USAGE_PERMISSION"
    assert str(FailedReasonCode.THREAD_ARCHIVED) == "THREAD_ARCHIVED"
    assert str(FailedReasonCode.MESSAGE_DELETE_DENIED) == "MESSAGE_DELETE_DENIED"

@pytest.mark.asyncio
async def test_failed_reason_code_enum():
    # Enumの全てのメンバーを取得
    all_members = set(FailedReasonCode)
    # テストで使用されているメンバーを取得
    tested_members = {
        FailedReasonCode.UNKNOWN,
        FailedReasonCode.INVALID_TOKEN,
        FailedReasonCode.RATE_LIMIT,
        FailedReasonCode.GUILD_NOT_FOUND,
        FailedReasonCode.GUILD_ACCESS_DENIED,
        FailedReasonCode.CHANNEL_NOT_FOUND,
        FailedReasonCode.CHANNEL_ACCESS_DENIED,
        FailedReasonCode.MESSAGE_DELETE_PERMISSION_DENIED,
        FailedReasonCode.MESSAGE_READ_PERMISSION_DENIED,
        FailedReasonCode.MESSAGE_NOT_FOUND,
        FailedReasonCode.MESSAGE_ACCESS_DENIED,
        FailedReasonCode.MESSAGE_READ_HISTORY_PERMISSION_DENIED,
        FailedReasonCode.NO_BOT_USAGE_PERMISSION,
        FailedReasonCode.THREAD_ARCHIVED,
        FailedReasonCode.MESSAGE_DELETE_DENIED,
    }
    # 未テストのメンバーを検出
    untested_members = all_members - tested_members
    assert len(untested_members) == 0, f"未テストのEnumメンバーがあります: {untested_members}"
