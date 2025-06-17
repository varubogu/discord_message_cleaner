
import pytest
import json
from src.discord_bot.utils.failed_reason_code import FailedReasonCode

@pytest.mark.asyncio
async def test_failed_reason_code():
    # messages.jsonの内容を読み込む
    with open('messages.json', 'r', encoding='utf-8') as f:
        messages = json.load(f)

    # FailedReasonCodeの内容を取得
    reason_codes = {code.value: code.name for code in FailedReasonCode}

    # messages.jsonとFailedReasonCodeの比較
    for code, message in messages.items():
        assert code in reason_codes, f"{code} is in messages.json but not in FailedReasonCode"

    # FailedReasonCodeに存在する全てのコードがmessages.jsonに存在することを確認
    for code in reason_codes.keys():
        assert code in messages, f"{code} is in FailedReasonCode but not in messages.json"
