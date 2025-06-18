import pytest
import json
from unittest.mock import AsyncMock, MagicMock
from discord_bot.utils.messages import SingletonMessages, Messages, Message

@pytest.mark.asyncio
async def test_singleton_messages():
    # シングルトンインスタンスの取得
    instance1 = await SingletonMessages.get_instance()
    instance2 = await SingletonMessages.get_instance()

    # 同じインスタンスであることを確認
    assert instance1 is instance2

@pytest.mark.asyncio
async def test_load_messages(mocker):
    # メッセージの読み込みをモック
    mock_data = {
        "greeting": {"ja": "こんにちは", "en": "Hello"},
        "farewell": {"ja": "さようなら", "en": "Goodbye"}
    }

    # 非同期コンテキストマネージャをサポートするモックを作成
    mock_file = AsyncMock()
    mock_file.__aenter__.return_value.read = AsyncMock(return_value=json.dumps(mock_data))

    # aiofiles.openをモック
    mocker.patch("aiofiles.open", return_value=mock_file)

    messages = await SingletonMessages.load()

    # メッセージが正しく読み込まれたことを確認
    assert messages["greeting"].ja == "こんにちは"
    assert messages["greeting"].en == "Hello"
    assert messages["farewell"].ja == "さようなら"
    assert messages["farewell"].en == "Goodbye"

@pytest.mark.asyncio
async def test_message_model():
    # Messageモデルのテスト
    message = Message(ja="テスト", en="Test")

    assert message.ja == "テスト"
    assert message.en == "Test"

@pytest.mark.asyncio
async def test_messages_model():
    # Messagesモデルのテスト
    message_dict = {
        "greeting": Message(ja="こんにちは", en="Hello"),
        "farewell": Message(ja="さようなら", en="Goodbye")
    }
    messages = Messages.model_validate(message_dict)

    assert messages["greeting"].ja == "こんにちは"
    assert messages["farewell"].en == "Goodbye"

    # __getitem__メソッドのテスト
    assert messages["greeting"] == message_dict["greeting"]
