import pytest
from result import is_err, is_ok

from discord_bot.utils.lifetime import LifeTimeUtil

empty_dict = {
    "weeks": 0,
    "days": 0,
    "hours": 0,
    "minutes": 0,
    "seconds": 0
}

@pytest.mark.asyncio
async def test_lifetime_calc_single():

    actual = await LifeTimeUtil.calc('5 w')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "weeks": 5,
    }

    actual = await LifeTimeUtil.calc('6week')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "weeks": 6,
    }

    actual = await LifeTimeUtil.calc(' 7d ')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "days": 7,
    }

    actual = await LifeTimeUtil.calc(' 8 day ')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "days": 8,
    }

    actual = await LifeTimeUtil.calc('9h')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "hours": 9,
    }

    actual = await LifeTimeUtil.calc('10hour')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "hours": 10,
    }

    actual = await LifeTimeUtil.calc(' 11mi')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "minutes": 11,
    }

    actual = await LifeTimeUtil.calc('12min')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "minutes": 12,
    }

    actual = await LifeTimeUtil.calc('13minute')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "minutes": 13,
    }

    actual = await LifeTimeUtil.calc('14s')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "seconds": 14
    }

    actual = await LifeTimeUtil.calc(' 15sec ')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "seconds": 15
    }

    actual = await LifeTimeUtil.calc(' 16 seconds ')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "seconds": 16
    }

@pytest.mark.asyncio
async def test_lifetime_calc_upper():

    actual = await LifeTimeUtil.calc('5 W')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "weeks": 5
    }

    actual = await LifeTimeUtil.calc('6WEEK')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "weeks": 6,
    }

    actual = await LifeTimeUtil.calc(' 7D ')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "days": 7,
    }

    actual = await LifeTimeUtil.calc(' 8 DAY ')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "days": 8,
    }

    actual = await LifeTimeUtil.calc('9H')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "hours": 9,
    }

    actual = await LifeTimeUtil.calc('10HOUR')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "hours": 10,
    }

    actual = await LifeTimeUtil.calc(' 11MI')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "minutes": 11,
    }

    actual = await LifeTimeUtil.calc('12MIN')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "minutes": 12,
    }

    actual = await LifeTimeUtil.calc('13MINUTE')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "minutes": 13,
    }

    actual = await LifeTimeUtil.calc('14S')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "seconds": 14
    }

    actual = await LifeTimeUtil.calc(' 15SEC ')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "seconds": 15
    }

    actual = await LifeTimeUtil.calc(' 16 SECONDS ')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "seconds": 16
    }

@pytest.mark.asyncio
async def test_lifetime_calc_all():

    actual = await LifeTimeUtil.calc('3week4day5hour6minute7second')
    assert is_ok(actual)
    assert actual.ok_value == {
        "weeks": 3,
        "days": 4,
        "hours": 5,
        "minutes": 6,
        "seconds": 7
    }

    actual = await LifeTimeUtil.calc(' 3 week 6 day 7 hour 8 minute 9 second ')
    assert is_ok(actual)
    assert actual.ok_value == {
        "weeks": 3,
        "days": 6,
        "hours": 7,
        "minutes": 8,
        "seconds": 9
    }

@pytest.mark.asyncio
async def test_lifetime_calc_multi():

    actual = await LifeTimeUtil.calc('5hour6minute7second')
    assert is_ok(actual)
    assert actual.ok_value == {
        "weeks": 0,
        "days": 0,
        "hours": 5,
        "minutes": 6,
        "seconds": 7
    }

    actual = await LifeTimeUtil.calc(' 3 week 6 day 7 hour 8 minute 9 second')
    assert is_ok(actual)
    assert actual.ok_value == {
        "weeks": 3,
        "days": 6,
        "hours": 7,
        "minutes": 8,
        "seconds": 9
    }

@pytest.mark.asyncio
async def test_lifetime_calc_random_order():

    actual = await LifeTimeUtil.calc('4day 3week6 minute7second 5hour')
    assert is_ok(actual)
    assert actual.ok_value == {
        "weeks": 3,
        "days": 4,
        "hours": 5,
        "minutes": 6,
        "seconds": 7
    }

    actual = await LifeTimeUtil.calc(' 3 week 6 day 7 hour 8 minute 9 second')
    assert is_ok(actual)
    assert actual.ok_value == {
        "weeks": 3,
        "days": 6,
        "hours": 7,
        "minutes": 8,
        "seconds": 9
    }

@pytest.mark.asyncio
async def test_lifetime_calc_error_empty():

    actual = await LifeTimeUtil.calc('')
    assert is_err(actual)
    assert actual.err_value == "文字を入力してください。"

    actual = await LifeTimeUtil.calc('      \t')
    assert is_err(actual)
    assert actual.err_value == "文字を入力してください。"

@pytest.mark.asyncio
async def test_lifetime_calc_error_duplicate():
    actual = await LifeTimeUtil.calc('1 week 2 week')
    assert is_err(actual)
    assert actual.err_value == "重複している項目があります。:'weeks'"

    actual = await LifeTimeUtil.calc('1 hour 2 months 3hours')
    assert is_err(actual)
    assert actual.err_value == "重複している項目があります。:'hours'"

    actual = await LifeTimeUtil.calc('1 hour 2 minute 3 hour, 4 minute')
    assert is_err(actual)
    assert actual.err_value == "重複している項目があります。:'hours', 'minutes'"


@pytest.mark.asyncio
async def test_lifetime_calc_error_nomatch():
    actual = await LifeTimeUtil.calc('aaa')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict

    actual = await LifeTimeUtil.calc('1hour bbb')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "hours": 1
    }

    actual = await LifeTimeUtil.calc('ccc 1minute')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict | {
        "minutes": 1
    }

    actual = await LifeTimeUtil.calc('day')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict

    actual = await LifeTimeUtil.calc('55')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict

    actual = await LifeTimeUtil.calc('1m')
    assert is_ok(actual)
    assert actual.ok_value == empty_dict

# @pytest.mark.asyncio
# async def test_lifetime_calc_error_failure():
#     actual = await LifeTimeUtil.calc('aaa')
#     assert is_err(actual)
#     assert actual.err_value == "不正な文字が入力されました: 'aaa'"

#     actual = await LifeTimeUtil.calc('1year bbb')
#     assert is_err(actual)
#     assert actual.err_value == "不正な文字が入力されました: 'bbb'"

#     actual = await LifeTimeUtil.calc('ccc 1month')
#     assert is_err(actual)
#     assert actual.err_value == "不正な文字が入力されました: 'ccc'"

#     actual = await LifeTimeUtil.calc('day')
#     assert is_err(actual)
#     assert actual.err_value == "数値を入力してください: 'day'"

#     actual = await LifeTimeUtil.calc('55')
#     assert is_err(actual)
#     assert actual.err_value == "単位を入力してください: '55'"

#     actual = await LifeTimeUtil.calc('1m')
#     assert is_err(actual)
#     assert actual.err_value == "不正な文字が入力されました: 'm'"

@pytest.mark.asyncio
async def test_interval_dict_to_string_empty():
    actual = await LifeTimeUtil.interval_dict_to_string({
    })
    assert actual == ""

    actual = await LifeTimeUtil.interval_dict_to_string({
        "years": 0
    })
    assert actual == ""

@pytest.mark.asyncio
async def test_interval_dict_to_string_single():

    actual = await LifeTimeUtil.interval_dict_to_string({
        "weeks": 3
    })
    assert actual == "3 weeks"

    actual = await LifeTimeUtil.interval_dict_to_string({
        "days": 4
    })
    assert actual == "4 days"

    actual = await LifeTimeUtil.interval_dict_to_string({
        "hours": 5,
        "minutes": 0
    })
    assert actual == "5 hours"

    actual = await LifeTimeUtil.interval_dict_to_string({
        "minutes": 6
    })
    assert actual == "6 minutes"

    actual = await LifeTimeUtil.interval_dict_to_string({
        "seconds": 7
    })
    assert actual == "7 seconds"

@pytest.mark.asyncio
async def test_interval_dict_to_string_multi():
    # 一部
    actual = await LifeTimeUtil.interval_dict_to_string({
        "days": 4,
        "hours": 5,
        "minutes": 6,
    })
    assert actual == "4 days 5 hours 6 minutes"

@pytest.mark.asyncio
async def test_interval_dict_to_string_all():
    # 全て
    actual = await LifeTimeUtil.interval_dict_to_string({
        "weeks": 3,
        "days": 4,
        "hours": 5,
        "minutes": 6,
        "seconds": 7
    })
    assert actual == "3 weeks 4 days 5 hours 6 minutes 7 seconds"

    # 順番バラバラ
    actual = await LifeTimeUtil.interval_dict_to_string({
        "minutes": 6,
        "hours": 5,
        "weeks": 3,
        "seconds": 7,
        "days": 4,
    })
    assert actual == "3 weeks 4 days 5 hours 6 minutes 7 seconds"

