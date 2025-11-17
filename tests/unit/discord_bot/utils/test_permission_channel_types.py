from unittest.mock import MagicMock

import pytest
from result import Err, Ok

from discord_bot.utils.permission import Permission


@pytest.mark.asyncio
async def test_permission_is_message_delete_permission_text_channel():
    """Permission - テキストチャンネルでの削除権限確認テスト"""
    # モック設定
    mock_user = MagicMock()
    mock_user.__class__.__name__ = "Member"

    mock_channel = MagicMock()
    mock_channel.__class__.__name__ = "TextChannel"

    # 権限あり
    mock_perms = MagicMock()
    mock_perms.administrator = False
    mock_perms.read_messages = True
    mock_perms.read_message_history = True
    mock_perms.manage_messages = True

    mock_channel.permissions_for.return_value = mock_perms

    # テスト実行
    result = await Permission.is_message_delete_permission(mock_user, mock_channel)

    # 検証
    assert isinstance(result, type(Ok(None)))


@pytest.mark.asyncio
async def test_permission_is_message_delete_permission_voice_channel():
    """Permission - ボイスチャンネルでの削除権限確認テスト"""
    # モック設定
    mock_user = MagicMock()
    mock_user.__class__.__name__ = "Member"

    mock_channel = MagicMock()
    mock_channel.__class__.__name__ = "VoiceChannel"

    # 権限あり
    mock_perms = MagicMock()
    mock_perms.administrator = False
    mock_perms.read_messages = True
    mock_perms.read_message_history = True
    mock_perms.manage_messages = True

    mock_channel.permissions_for.return_value = mock_perms

    # テスト実行
    result = await Permission.is_message_delete_permission(mock_user, mock_channel)

    # 検証
    assert isinstance(result, type(Ok(None)))


@pytest.mark.asyncio
async def test_permission_is_message_delete_permission_thread():
    """Permission - スレッドでの削除権限確認テスト"""
    # モック設定
    mock_user = MagicMock()
    mock_user.__class__.__name__ = "Member"

    mock_channel = MagicMock()
    mock_channel.__class__.__name__ = "Thread"

    # 権限あり
    mock_perms = MagicMock()
    mock_perms.administrator = False
    mock_perms.read_messages = True
    mock_perms.read_message_history = True
    mock_perms.manage_messages = True

    mock_channel.permissions_for.return_value = mock_perms

    # テスト実行
    result = await Permission.is_message_delete_permission(mock_user, mock_channel)

    # 検証
    assert isinstance(result, type(Ok(None)))


@pytest.mark.asyncio
async def test_permission_admin_user_all_channels():
    """Permission - 管理者ユーザーは全チャンネルタイプで権限ありテスト"""
    # モック設定
    mock_user = MagicMock()
    mock_user.__class__.__name__ = "Member"

    # 3種類のチャンネルをテスト
    channel_types = ["TextChannel", "VoiceChannel", "Thread"]

    for channel_type in channel_types:
        mock_channel = MagicMock()
        mock_channel.__class__.__name__ = channel_type

        # 管理者権限
        mock_perms = MagicMock()
        mock_perms.administrator = True

        mock_channel.permissions_for.return_value = mock_perms

        # テスト実行
        result = await Permission.is_message_delete_permission(mock_user, mock_channel)

        # 検証
        assert isinstance(result, type(Ok(None)))


@pytest.mark.asyncio
async def test_permission_no_manage_messages_permission():
    """Permission - メッセージ管理権限なしのテスト"""
    # モック設定
    mock_user = MagicMock()
    mock_user.__class__.__name__ = "Member"

    mock_channel = MagicMock()
    mock_channel.__class__.__name__ = "TextChannel"

    # 権限なし（manage_messagesがFalse）
    mock_perms = MagicMock()
    mock_perms.administrator = False
    mock_perms.read_messages = True
    mock_perms.read_message_history = True
    mock_perms.manage_messages = False

    mock_channel.permissions_for.return_value = mock_perms

    # テスト実行
    result = await Permission.is_message_delete_permission(mock_user, mock_channel)

    # 検証
    assert isinstance(result, type(Err([])))


@pytest.mark.asyncio
async def test_permission_no_read_messages_permission():
    """Permission - メッセージ読み取り権限なしのテスト"""
    # モック設定
    mock_user = MagicMock()
    mock_user.__class__.__name__ = "Member"

    mock_channel = MagicMock()
    mock_channel.__class__.__name__ = "TextChannel"

    # 権限なし（read_messagesがFalse）
    mock_perms = MagicMock()
    mock_perms.administrator = False
    mock_perms.read_messages = False
    mock_perms.read_message_history = True
    mock_perms.manage_messages = True

    mock_channel.permissions_for.return_value = mock_perms

    # テスト実行
    result = await Permission.is_message_delete_permission(mock_user, mock_channel)

    # 検証
    assert isinstance(result, type(Err([])))


@pytest.mark.asyncio
async def test_permission_user_object_always_ok():
    """Permission - discord.Userオブジェクトは常にOKテスト"""
    import discord as discord_module

    # discord.Userのモックを作成
    mock_user = MagicMock(spec=discord_module.User)

    mock_channel = MagicMock()
    mock_channel.__class__.__name__ = "TextChannel"

    # テスト実行
    result = await Permission.is_message_delete_permission(mock_user, mock_channel)

    # 検証 - Userオブジェクトはpermissions_forが呼ばれない
    assert isinstance(result, type(Ok(None)))
    mock_channel.permissions_for.assert_not_called()


@pytest.mark.asyncio
async def test_permission_is_message_read_permission():
    """Permission - メッセージ読み取り権限確認テスト"""
    # モック設定
    mock_user = MagicMock()
    mock_user.__class__.__name__ = "Member"

    mock_channel = MagicMock()
    mock_channel.__class__.__name__ = "TextChannel"

    # 読み取り権限あり
    mock_perms = MagicMock()
    mock_perms.administrator = False
    mock_perms.read_messages = True
    mock_perms.read_message_history = True

    mock_channel.permissions_for.return_value = mock_perms

    # テスト実行
    result = await Permission.is_message_read_permission(mock_user, mock_channel)

    # 検証
    assert isinstance(result, type(Ok(None)))


@pytest.mark.asyncio
async def test_permission_is_message_read_permission_no_history():
    """Permission - メッセージ履歴読み取り権限なしのテスト"""
    # モック設定
    mock_user = MagicMock()
    mock_user.__class__.__name__ = "Member"

    mock_channel = MagicMock()
    mock_channel.__class__.__name__ = "TextChannel"

    # メッセージ履歴読み取り権限なし
    mock_perms = MagicMock()
    mock_perms.administrator = False
    mock_perms.read_messages = True
    mock_perms.read_message_history = False

    mock_channel.permissions_for.return_value = mock_perms

    # テスト実行
    result = await Permission.is_message_read_permission(mock_user, mock_channel)

    # 検証 - エラーを返す
    assert isinstance(result, type(Err([])))
