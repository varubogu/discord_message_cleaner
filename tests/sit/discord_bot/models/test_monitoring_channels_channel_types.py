from datetime import timedelta

import pytest

from discord_bot.models.monitoring_channels import MonitoringChannels


@pytest.mark.asyncio
async def test_monitoring_channels_insert_text_channel(async_db_session):
    """MonitoringChannels - テキストチャンネルの挿入テスト"""
    guild_id = 123456789
    channel_id = 111111111
    interval = timedelta(days=1)

    # テキストチャンネル監視対象を作成
    mc = MonitoringChannels()
    mc.guild_id = guild_id
    mc.channel_id = channel_id
    mc.interval = interval

    # 挿入
    await mc.merge(async_db_session)
    await async_db_session.commit()

    # 検証
    retrieved = await MonitoringChannels.select(async_db_session, guild_id, channel_id)
    assert retrieved is not None
    assert retrieved.guild_id == guild_id
    assert retrieved.channel_id == channel_id
    assert retrieved.interval == interval


@pytest.mark.asyncio
async def test_monitoring_channels_insert_voice_channel(async_db_session):
    """MonitoringChannels - ボイスチャンネルの挿入テスト"""
    guild_id = 222222222
    channel_id = 333333333
    interval = timedelta(days=7)

    # ボイスチャンネル監視対象を作成（チャンネルIDが異なるだけで同じモデル）
    mc = MonitoringChannels()
    mc.guild_id = guild_id
    mc.channel_id = channel_id
    mc.interval = interval

    # 挿入
    await mc.merge(async_db_session)
    await async_db_session.commit()

    # 検証
    retrieved = await MonitoringChannels.select(async_db_session, guild_id, channel_id)
    assert retrieved is not None
    assert retrieved.guild_id == guild_id
    assert retrieved.channel_id == channel_id
    assert retrieved.interval == interval


@pytest.mark.asyncio
async def test_monitoring_channels_insert_thread(async_db_session):
    """MonitoringChannels - スレッドの挿入テスト"""
    guild_id = 444444444
    thread_id = 555555555
    interval = timedelta(hours=12)

    # スレッド監視対象を作成（スレッドもチャンネルIDで識別）
    mc = MonitoringChannels()
    mc.guild_id = guild_id
    mc.channel_id = thread_id
    mc.interval = interval

    # 挿入
    await mc.merge(async_db_session)
    await async_db_session.commit()

    # 検証
    retrieved = await MonitoringChannels.select(async_db_session, guild_id, thread_id)
    assert retrieved is not None
    assert retrieved.guild_id == guild_id
    assert retrieved.channel_id == thread_id
    assert retrieved.interval == interval


@pytest.mark.asyncio
async def test_monitoring_channels_multiple_channel_types(async_db_session):
    """MonitoringChannels - 複数のチャンネルタイプ混合テスト"""
    guild_id = 666666666
    text_channel_id = 111111111
    voice_channel_id = 222222222
    thread_id = 333333333

    # 各チャンネルタイプの監視対象を作成
    channels_data = [
        (text_channel_id, timedelta(days=1)),
        (voice_channel_id, timedelta(days=7)),
        (thread_id, timedelta(hours=6)),
    ]

    for channel_id, interval in channels_data:
        mc = MonitoringChannels()
        mc.guild_id = guild_id
        mc.channel_id = channel_id
        mc.interval = interval
        await mc.merge(async_db_session)

    await async_db_session.commit()

    # 各チャンネルが正しく保存されたか検証
    for channel_id, expected_interval in channels_data:
        retrieved = await MonitoringChannels.select(async_db_session, guild_id, channel_id)
        assert retrieved is not None
        assert retrieved.guild_id == guild_id
        assert retrieved.channel_id == channel_id
        assert retrieved.interval == expected_interval


@pytest.mark.asyncio
async def test_monitoring_channels_select_all(async_db_session):
    """MonitoringChannels - 全監視対象チャンネル取得テスト"""
    guild_id = 777777777
    channels = [
        (111111111, timedelta(days=1)),
        (222222222, timedelta(days=7)),
        (333333333, timedelta(hours=12)),
    ]

    # 複数の監視対象を挿入
    for channel_id, interval in channels:
        mc = MonitoringChannels()
        mc.guild_id = guild_id
        mc.channel_id = channel_id
        mc.interval = interval
        await mc.merge(async_db_session)

    await async_db_session.commit()

    # 全監視対象を取得
    all_monitoring = await MonitoringChannels.select_all(async_db_session)
    assert all_monitoring is not None
    assert len(all_monitoring) >= 3


@pytest.mark.asyncio
async def test_monitoring_channels_delete(async_db_session):
    """MonitoringChannels - 監視対象チャンネルの削除テスト"""
    guild_id = 888888888
    channel_id = 999999999
    interval = timedelta(days=1)

    # 監視対象を挿入
    mc = MonitoringChannels()
    mc.guild_id = guild_id
    mc.channel_id = channel_id
    mc.interval = interval
    await mc.merge(async_db_session)
    await async_db_session.commit()

    # 削除前に存在確認
    retrieved_before = await MonitoringChannels.select(async_db_session, guild_id, channel_id)
    assert retrieved_before is not None

    # 削除
    await retrieved_before.delete(async_db_session)
    await async_db_session.commit()

    # 削除後に存在確認
    retrieved_after = await MonitoringChannels.select(async_db_session, guild_id, channel_id)
    assert retrieved_after is None


@pytest.mark.asyncio
async def test_monitoring_channels_interval_variations(async_db_session):
    """MonitoringChannels - 様々なインターバル値テスト"""
    guild_id = 1111111111
    channel_id = 2222222222

    # 様々なインターバル値
    intervals = [
        timedelta(seconds=30),
        timedelta(minutes=5),
        timedelta(hours=1),
        timedelta(days=1),
        timedelta(weeks=1),
    ]

    for i, interval in enumerate(intervals):
        mc = MonitoringChannels()
        mc.guild_id = guild_id
        mc.channel_id = channel_id + i
        mc.interval = interval
        await mc.merge(async_db_session)

    await async_db_session.commit()

    # 各インターバル値が正しく保存されたか検証
    for i, expected_interval in enumerate(intervals):
        retrieved = await MonitoringChannels.select(async_db_session, guild_id, channel_id + i)
        assert retrieved is not None
        assert retrieved.interval == expected_interval
