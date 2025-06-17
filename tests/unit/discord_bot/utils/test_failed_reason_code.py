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
    }
    # 未テストのメンバーを検出
    untested_members = all_members - tested_members
    assert len(untested_members) == 0, f"未テストのEnumメンバーがあります: {untested_members}"
