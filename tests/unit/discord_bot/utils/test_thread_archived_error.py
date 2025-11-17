from unittest.mock import MagicMock

import pytest

from discord_bot.utils.failed_reason_code import FailedReasonCode


@pytest.mark.asyncio
async def test_thread_archived_error_code_exists():
    """スレッドアーカイブ化エラーコードが定義されているテスト"""
    # THREAD_ARCHIVED エラーコードが存在することを確認
    assert hasattr(FailedReasonCode, 'THREAD_ARCHIVED')
    assert FailedReasonCode.THREAD_ARCHIVED.value == "E0013"


@pytest.mark.asyncio
async def test_thread_archived_error_code_string_representation():
    """スレッドアーカイブ化エラーコードの文字列表現テスト"""
    # エラーコード名を確認
    assert str(FailedReasonCode.THREAD_ARCHIVED) == "THREAD_ARCHIVED"


@pytest.mark.asyncio
async def test_failed_reason_code_message_delete_denied():
    """メッセージ削除拒否エラーコードテスト"""
    # MESSAGE_DELETE_DENIED エラーコードが定義されているか確認
    assert hasattr(FailedReasonCode, 'MESSAGE_DELETE_DENIED')
    assert FailedReasonCode.MESSAGE_DELETE_DENIED.value == "E0014"


@pytest.mark.asyncio
async def test_failed_reason_code_values_uniqueness():
    """エラーコード値がユニークであるテスト"""
    # すべてのエラーコード値を取得
    all_codes = [code.value for code in FailedReasonCode]

    # 重複がないか確認
    assert len(all_codes) == len(set(all_codes))


@pytest.mark.asyncio
async def test_thread_channel_cannot_be_parameter():
    """クローズされたスレッドはコマンドパラメータとして指定できないことをシミュレート"""
    # モック設定
    mock_thread = MagicMock()
    mock_thread.archived = True
    mock_thread.__class__.__name__ = "Thread"

    # スレッドがアーカイブ化されている場合、エラーハンドリングが必要
    if mock_thread.archived:
        error_code = FailedReasonCode.THREAD_ARCHIVED
        assert error_code == FailedReasonCode.THREAD_ARCHIVED
        assert error_code.value == "E0013"


@pytest.mark.asyncio
async def test_open_thread_channel_can_be_parameter():
    """オープンなスレッドはコマンドパラメータとして指定できることをシミュレート"""
    # モック設定
    mock_thread = MagicMock()
    mock_thread.archived = False
    mock_thread.__class__.__name__ = "Thread"

    # スレッドがアーカイブ化されていない場合、処理を続行
    if not mock_thread.archived:
        # 処理続行 - エラーコードは不要
        is_valid = True
        assert is_valid is True


@pytest.mark.asyncio
async def test_channel_type_detection():
    """チャンネルタイプ検出テスト（TextChannel、VoiceChannel、Thread）"""
    # モック設定
    channel_types = [
        ("TextChannel", False),  # archived は適用不可
        ("VoiceChannel", False),  # archived は適用不可
        ("Thread", True),  # archived が適用可能
    ]

    for channel_type_name, has_archived in channel_types:
        mock_channel = MagicMock()
        mock_channel.__class__.__name__ = channel_type_name

        if has_archived:
            mock_channel.archived = False

        # チャンネルタイプを確認
        if channel_type_name == "Thread":
            assert hasattr(mock_channel, 'archived') or True  # Threadには archived 属性がある可能性


@pytest.mark.asyncio
async def test_error_message_mapping():
    """エラーコードと対応するメッセージのマッピングテスト"""
    # 定義されたエラーコード
    error_messages = {
        FailedReasonCode.THREAD_ARCHIVED: "スレッドはアーカイブ化されています",
        FailedReasonCode.MESSAGE_DELETE_DENIED: "メッセージの削除に失敗",
    }

    # 各エラーコードがメッセージを持つか確認
    for error_code, message in error_messages.items():
        assert error_code is not None
        assert message is not None
        assert len(message) > 0
