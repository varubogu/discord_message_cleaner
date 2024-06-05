from datetime import datetime
from decimal import Decimal
import pytest
from result import is_ok, is_err

from discord_bot.utils.parse import ParseUtil


@pytest.mark.asyncio
async def test_try_parse_from_int():
    actual = await ParseUtil.try_parse(11, str)
    assert is_ok(actual)
    assert actual.ok_value == "11"

    actual = await ParseUtil.try_parse(1, bool)
    assert is_ok(actual)
    assert actual.ok_value is True

    actual = await ParseUtil.try_parse(0, bool)
    assert is_ok(actual)
    assert actual.ok_value is False

@pytest.mark.asyncio
async def test_try_parse_from_float():
    actual = await ParseUtil.try_parse(89.99, int)
    assert is_ok(actual)
    assert actual.ok_value == 89

    actual = await ParseUtil.try_parse(50.2, str)
    assert is_ok(actual)
    assert actual.ok_value == "50.2"

    actual = await ParseUtil.try_parse(0.1, bool)
    assert is_ok(actual)
    assert actual.ok_value is True

    # フォーマットすべき値なので除外
    # actual = await ParseUtil.try_parse(9544.454, Decimal)
    # assert is_ok(actual)
    # assert actual.ok_value == Decimal("9544.454")

@pytest.mark.asyncio
async def test_try_parse_from_decimal():
    actual = await ParseUtil.try_parse(Decimal("52.4412"), int)
    assert is_ok(actual)
    assert actual.ok_value == 52

    actual = await ParseUtil.try_parse(Decimal("52.4567"), float)
    assert is_ok(actual)
    assert actual.ok_value == 52.4567

    # フォーマッタを使ってフォーマットすべきなので除外
    actual = await ParseUtil.try_parse(Decimal("985.4551"), str)
    assert is_ok(actual)
    assert actual.ok_value == "985.4551"

@pytest.mark.asyncio
async def test_try_parse_from_str():
    actual = await ParseUtil.try_parse("1", int)
    assert is_ok(actual)
    assert actual.ok_value == 1

    actual = await ParseUtil.try_parse(" 20.2 ", int)
    assert is_err(actual)
    assert actual.err_value == "変換に失敗しました。値:[ 20.2 ], 変換しようとした型:[int]"

    actual = await ParseUtil.try_parse("aa", int)
    assert is_err(actual)
    assert actual.err_value == "変換に失敗しました。値:[aa], 変換しようとした型:[int]"

    actual = await ParseUtil.try_parse("20.4599", float)
    assert is_ok(actual)
    assert actual.ok_value == 20.4599

    actual = await ParseUtil.try_parse("50.413", Decimal)
    assert is_ok(actual)
    assert actual.ok_value == Decimal("50.413")

    # datetimeへの変換は不可
    # actual = await ParseUtil.try_parse("2023-12-31 23:59:58", datetime)
    # assert is_ok(actual)
    # assert actual.ok_value == datetime(2023, 12, 31, 23, 59, 58)

@pytest.mark.asyncio
async def test_try_parse_from_bool():
    actual = await ParseUtil.try_parse(True, str)
    assert is_ok(actual)
    assert actual.ok_value == "True"

    actual = await ParseUtil.try_parse(False, str)
    assert is_ok(actual)
    assert actual.ok_value == "False"

@pytest.mark.asyncio
async def test_try_parse_from_datetime():
    actual = await ParseUtil.try_parse(datetime(2023, 12, 31, 23, 59, 58), str)
    assert is_ok(actual)
    assert actual.ok_value == "2023-12-31 23:59:58"
