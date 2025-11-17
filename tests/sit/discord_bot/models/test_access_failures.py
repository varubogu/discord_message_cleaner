from datetime import datetime

import pytest

from discord_bot.models.access_failures import AccessFailures
from discord_bot.utils.failed_reason_code import FailedReasonCode


@pytest.mark.asyncio
async def test_access_failures_insert(async_db_session):
    """AccessFailuresレコードの挿入テスト"""
    # テストデータの作成
    failure = AccessFailures()
    failure.guild_id = 123456789
    failure.channel_id = None
    failure.failed_at = datetime.now()
    failure.failed_reason_code = FailedReasonCode.GUILD_NOT_FOUND.value
    failure.failed_reason = "ギルドが見つかりません"

    # 挿入
    await failure.insert(async_db_session)
    await async_db_session.commit()

    # 検証
    count = await AccessFailures.count_guild(async_db_session, 123456789)
    assert count == 1


@pytest.mark.asyncio
async def test_access_failures_count_guild(async_db_session):
    """AccessFailures.count_guild - サーバーレベルのカウント機能テスト"""
    guild_id = 111111111

    # 複数のレコードを挿入
    for i in range(3):
        failure = AccessFailures()
        failure.guild_id = guild_id
        failure.channel_id = None
        failure.failed_at = datetime.now()
        failure.failed_reason_code = FailedReasonCode.GUILD_ACCESS_DENIED.value
        failure.failed_reason = f"ギルドアクセス拒否 {i}"
        await failure.insert(async_db_session)

    await async_db_session.commit()

    # カウント検証
    count = await AccessFailures.count_guild(async_db_session, guild_id)
    assert count == 3


@pytest.mark.asyncio
async def test_access_failures_count_channel(async_db_session):
    """AccessFailures.count_channel - チャンネルレベルのカウント機能テスト"""
    guild_id = 222222222
    channel_id = 333333333

    # 複数のレコードを挿入（同じギルド、同じチャンネル）
    for i in range(4):
        failure = AccessFailures()
        failure.guild_id = guild_id
        failure.channel_id = channel_id
        failure.failed_at = datetime.now()
        failure.failed_reason_code = FailedReasonCode.CHANNEL_NOT_FOUND.value
        failure.failed_reason = f"チャンネルが見つかりません {i}"
        await failure.insert(async_db_session)

    await async_db_session.commit()

    # カウント検証
    count = await AccessFailures.count_channel(async_db_session, guild_id, channel_id)
    assert count == 4


@pytest.mark.asyncio
async def test_access_failures_count_channel_not_found(async_db_session):
    """AccessFailures.count_channel - 存在しないチャンネルのカウント"""
    guild_id = 444444444
    channel_id = 555555555

    # レコードが存在しない場合、0を返す（異常時は-1だが通常時あり得ない）
    count = await AccessFailures.count_channel(async_db_session, guild_id, channel_id)
    assert count == 0


@pytest.mark.asyncio
async def test_access_failures_reset_guild(async_db_session):
    """AccessFailures.reset_guild - サーバーレベルのリセット機能テスト"""
    guild_id = 666666666

    # 複数のレコードを挿入
    for i in range(3):
        failure = AccessFailures()
        failure.guild_id = guild_id
        failure.channel_id = None
        failure.failed_at = datetime.now()
        failure.failed_reason_code = FailedReasonCode.RATE_LIMIT.value
        failure.failed_reason = f"レート制限 {i}"
        await failure.insert(async_db_session)

    await async_db_session.commit()

    # リセット前の確認
    count_before = await AccessFailures.count_guild(async_db_session, guild_id)
    assert count_before == 3

    # リセット実行
    await AccessFailures.reset_guild(async_db_session, guild_id)
    await async_db_session.commit()

    # リセット後の確認
    count_after = await AccessFailures.count_guild(async_db_session, guild_id)
    assert count_after == 0


@pytest.mark.asyncio
async def test_access_failures_reset_channel(async_db_session):
    """AccessFailures.reset_channel - チャンネルレベルのリセット機能テスト"""
    guild_id = 777777777
    channel_id = 888888888

    # 複数のレコードを挿入
    for i in range(5):
        failure = AccessFailures()
        failure.guild_id = guild_id
        failure.channel_id = channel_id
        failure.failed_at = datetime.now()
        failure.failed_reason_code = FailedReasonCode.MESSAGE_DELETE_PERMISSION_DENIED.value
        failure.failed_reason = f"削除権限なし {i}"
        await failure.insert(async_db_session)

    await async_db_session.commit()

    # リセット前の確認
    count_before = await AccessFailures.count_channel(async_db_session, guild_id, channel_id)
    assert count_before == 5

    # リセット実行
    await AccessFailures.reset_channel(async_db_session, guild_id, channel_id)
    await async_db_session.commit()

    # リセット後の確認
    count_after = await AccessFailures.count_channel(async_db_session, guild_id, channel_id)
    assert count_after == 0


@pytest.mark.asyncio
async def test_access_failures_guild_and_channel_isolation(async_db_session):
    """AccessFailures - ギルド・チャンネルレコードの独立性テスト"""
    guild_id = 999999999
    channel_id_1 = 111111111
    channel_id_2 = 222222222

    # ギルドレベルのレコードを挿入
    guild_failure = AccessFailures()
    guild_failure.guild_id = guild_id
    guild_failure.channel_id = None
    guild_failure.failed_at = datetime.now()
    guild_failure.failed_reason_code = FailedReasonCode.GUILD_NOT_FOUND.value
    guild_failure.failed_reason = "ギルドが見つかりません"
    await guild_failure.insert(async_db_session)

    # チャンネルレベルのレコードを複数挿入
    for i in range(2):
        channel_failure = AccessFailures()
        channel_failure.guild_id = guild_id
        channel_failure.channel_id = channel_id_1 if i == 0 else channel_id_2
        channel_failure.failed_at = datetime.now()
        channel_failure.failed_reason_code = FailedReasonCode.CHANNEL_ACCESS_DENIED.value
        channel_failure.failed_reason = f"チャンネルアクセス拒否 {i}"
        await channel_failure.insert(async_db_session)

    await async_db_session.commit()

    # 各種カウント検証
    guild_count = await AccessFailures.count_guild(async_db_session, guild_id)
    channel_1_count = await AccessFailures.count_channel(async_db_session, guild_id, channel_id_1)
    channel_2_count = await AccessFailures.count_channel(async_db_session, guild_id, channel_id_2)

    assert guild_count == 1
    assert channel_1_count == 1
    assert channel_2_count == 1

    # リセット時の独立性確認
    await AccessFailures.reset_channel(async_db_session, guild_id, channel_id_1)
    await async_db_session.commit()

    guild_count_after = await AccessFailures.count_guild(async_db_session, guild_id)
    channel_1_count_after = await AccessFailures.count_channel(async_db_session, guild_id, channel_id_1)
    channel_2_count_after = await AccessFailures.count_channel(async_db_session, guild_id, channel_id_2)

    assert guild_count_after == 1  # ギルドレベルは影響を受けない
    assert channel_1_count_after == 0  # リセット対象のチャンネル
    assert channel_2_count_after == 1  # 他のチャンネルは影響を受けない
