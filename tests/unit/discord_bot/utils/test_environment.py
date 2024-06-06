import pytest
from discord_bot.utils.environment import get_os_environ_safety

@pytest.mark.asyncio
async def test_get_os_environ_safety_there():

    # 環境変数あり
    actual = get_os_environ_safety('ENVTEST_VALUE_INT', int, -1)
    assert actual == 1

    actual = get_os_environ_safety('ENVTEST_VALUE_FLOAT', float, 1.5)
    assert actual == 2.1

    actual = get_os_environ_safety('ENVTEST_VALUE_STR', str, "dd")
    assert actual == "abc"


@pytest.mark.asyncio
@pytest.mark.parametrize("capture", [True, False])
async def test_get_os_environ_safety_none(capsys, capture):

    # 環境変数定義なし
    actual = get_os_environ_safety('__ENVTEST_VALUE_INT', int, -1)
    assert actual == -1
    captured = capsys.readouterr()
    assert captured.out == "warning: '__ENVTEST_VALUE_INT' is None. use default value: -1\n"

    actual = get_os_environ_safety('__ENVTEST_VALUE_FLOAT', float, 1.5)
    assert actual == 1.5
    captured = capsys.readouterr()
    assert captured.out == "warning: '__ENVTEST_VALUE_FLOAT' is None. use default value: 1.5\n"

    actual = get_os_environ_safety('__ENVTEST_VALUE_STR', str, "dd")
    assert actual == "dd"
    captured = capsys.readouterr()
    assert captured.out == "warning: '__ENVTEST_VALUE_STR' is None. use default value: dd\n"

@pytest.mark.asyncio
@pytest.mark.parametrize("capture", [True, False])
async def test_get_os_environ_safety_different_type(capsys, capture):

    # 型違い
    actual = get_os_environ_safety('ENVTEST_VALUE_STR', int, 9)
    assert actual == 9
    captured = capsys.readouterr()
    assert captured.out == "warning: 'ENVTEST_VALUE_STR' is not int value. use default value: 9\n"

    actual = get_os_environ_safety('ENVTEST_VALUE_STR', float, 51.2)
    assert actual == 51.2
    captured = capsys.readouterr()
    assert captured.out == "warning: 'ENVTEST_VALUE_STR' is not float value. use default value: 51.2\n"

    actual = get_os_environ_safety('ENVTEST_VALUE_FLOAT', int, 8)
    assert actual == 8
    captured = capsys.readouterr()
    assert captured.out == "warning: 'ENVTEST_VALUE_FLOAT' is not int value. use default value: 8\n"
