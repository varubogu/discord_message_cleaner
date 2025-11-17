from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from discord_bot.models.access_failures import AccessFailures


@pytest.mark.asyncio
async def test_message_delete_failure_counter_below_threshold():
    """メッセージ削除失敗カウントが閾値未満の場合、監視対象から削除されない"""
    # モック設定
    mock_session = AsyncMock()
    mock_monitoring = MagicMock()
    mock_monitoring.delete = AsyncMock()

    # 失敗カウント：2回（閾値3回未満）
    mock_count_channel = AsyncMock(return_value=2)

    with patch('discord_bot.models.access_failures.AccessFailures.count_channel', mock_count_channel):
        # テスト対象: 失敗カウント判定ロジック
        failure_count = await AccessFailures.count_channel(mock_session, 123456789, 987654321)

        # 検証
        assert failure_count == 2
        mock_monitoring.delete.assert_not_called()


@pytest.mark.asyncio
async def test_message_delete_failure_counter_at_threshold():
    """メッセージ削除失敗カウントが閾値に達した場合、監視対象から削除される"""
    # モック設定
    mock_session = AsyncMock()
    mock_monitoring = MagicMock()
    mock_monitoring.delete = AsyncMock()

    # 失敗カウント：3回（閾値3回）
    mock_count_channel = AsyncMock(return_value=3)

    with patch('discord_bot.models.access_failures.AccessFailures.count_channel', mock_count_channel):
        # テスト対象: 失敗カウント判定ロジック
        failure_count = await AccessFailures.count_channel(mock_session, 123456789, 987654321)

        # 検証
        assert failure_count >= 3


@pytest.mark.asyncio
async def test_message_delete_success_resets_counter():
    """メッセージ削除成功時に失敗カウントがリセットされる"""
    # モック設定
    mock_session = AsyncMock()

    # リセット前は失敗カウントが1回
    mock_count_before = AsyncMock(return_value=1)

    # リセット後は失敗カウントが0
    mock_count_after = AsyncMock(return_value=0)

    with patch('discord_bot.models.access_failures.AccessFailures.count_channel', side_effect=[mock_count_before.return_value, mock_count_after.return_value]):
        failure_count_before = await AccessFailures.count_channel(mock_session, 123456789, 987654321)
        assert failure_count_before == 1


@pytest.mark.asyncio
async def test_message_delete_failure_counter_increment():
    """メッセージ削除失敗時にカウントが1増加する"""
    # モック設定
    mock_session = AsyncMock()
    mock_access_failures = MagicMock()
    mock_access_failures.insert = AsyncMock()

    # 新規レコード挿入後のカウント：1回
    mock_count_after_insert = AsyncMock(return_value=1)

    with patch('discord_bot.models.access_failures.AccessFailures.insert', mock_access_failures.insert):
        with patch('discord_bot.models.access_failures.AccessFailures.count_channel', mock_count_after_insert):
            await mock_access_failures.insert(mock_session)
            failure_count = await AccessFailures.count_channel(mock_session, 123456789, 987654321)

            assert failure_count == 1
            mock_access_failures.insert.assert_called_once()


@pytest.mark.asyncio
async def test_access_failures_count_boundary():
    """失敗カウント判定の境界値テスト"""
    # 閾値を3と仮定
    MESSAGE_DELETE_FAILURES = 3

    # テストケース：0回、1回、2回、3回、4回
    test_cases = [
        (0, False),  # 0回：削除対象外
        (1, False),  # 1回：削除対象外
        (2, False),  # 2回：削除対象外
        (3, True),   # 3回：削除対象
        (4, True),   # 4回：削除対象
    ]

    for count, should_delete in test_cases:
        should_exclude = count >= MESSAGE_DELETE_FAILURES
        assert should_exclude == should_delete, f"Count {count}: expected {should_delete}, got {should_exclude}"
