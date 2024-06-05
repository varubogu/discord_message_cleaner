import pytest


async def is_magic_method(name: str) -> bool:
    return name.startswith("__") and name.endswith("__")


async def test_get_properties(cls) -> set[str]:
    obj = cls()
    return set(prop for prop in dir(obj)
               if not callable(getattr(obj, prop))
               and not prop.startswith("__")
               and not prop.endswith("__"))


async def test_class_properties(actual_cls, expected_properties) -> None:

    obj = actual_cls()
    actual_properties = await test_get_properties(obj)

    assert actual_properties == expected_properties, "プロパティの一覧が期待と異なります"
