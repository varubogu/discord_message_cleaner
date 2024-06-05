import pytest
from result import is_err, is_ok
from discord_bot.utils.url import UrlUtil


@pytest.mark.asyncio
async def test_is_url():
    actual = await UrlUtil.is_url("not_url")
    assert actual is False

    actual = await UrlUtil.is_url("https://discord.com/channels/1111/2222/3333")
    assert actual is True


@pytest.mark.asyncio
async def test_try_parse_discord_url():

    actual = await UrlUtil.try_parse_discord_url("https://discord.com/channels/aaaa/2222/3333")
    assert is_err(actual)
    assert actual.err_value == "URLの解析に失敗しました。"

    actual = await UrlUtil.try_parse_discord_url("https://discord.com/channels/1111/bbbb/3333")
    assert is_err(actual)
    assert actual.err_value == "URLの解析に失敗しました。"


    actual = await UrlUtil.try_parse_discord_url("https://discord.com/channels/1111/2222/cccc")
    assert is_err(actual)
    assert actual.err_value == "URLの解析に失敗しました。"


    actual = await UrlUtil.try_parse_discord_url("https://discord.com/channels/1111/2222/3333")
    assert is_ok(actual)
    assert actual.ok_value == {
        "guild_id": 1111,
        "channel_id": 2222,
        "message_id": 3333
    }


@pytest.mark.asyncio
async def test_try_parse_guild_id():
    actual = await UrlUtil.try_parse_guild_id("https://discord.com/channels/aaaa/2222/3333")
    assert is_err(actual)
    assert actual.err_value == "URLからguild_idの取得に失敗しました。"

    actual = await UrlUtil.try_parse_guild_id("https://discord.com/channels/1111/2222/3333")
    assert is_ok(actual)
    assert actual.ok_value == 1111

@pytest.mark.asyncio
async def test_try_parse_channel_id():
    actual = await UrlUtil.try_parse_channel_id("https://discord.com/channels/1111/bbbb/3333")
    assert is_err(actual)
    assert actual.err_value == "URLからchannel_idの取得に失敗しました。"

    actual = await UrlUtil.try_parse_channel_id("https://discord.com/channels/1111/2222/3333")
    assert is_ok(actual)
    assert actual.ok_value == 2222

@pytest.mark.asyncio
async def test_try_parse_message_id():
    actual = await UrlUtil.try_parse_message_id("https://discord.com/channels/1111/2222/cccc")
    assert is_err(actual)
    assert actual.err_value == "URLからmessage_idの取得に失敗しました。"

    actual = await UrlUtil.try_parse_message_id("https://discord.com/channels/1111/2222/3333")
    assert is_ok(actual)
    assert actual.ok_value == 3333
