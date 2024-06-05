from datetime import timedelta
import enum
import re
from typing import Any
from result import Result, Ok, Err

class LifeTimeUtil:

    @classmethod
    async def calc(cls, lifetime_str: str) -> Result[Any, str]:

        if lifetime_str.strip() == "":
            return Err("文字を入力してください。")

        err_messages: list[str] = []

        # TODO:誤りのロジックを実装して適切な文言でないと動かないようにしたい
        # # 誤りの部分を検出
        # success_str_list = [
        #     r"\s",
        #     r"\d",
        #     *[u for unit_list in unit_map.values() for u in unit_list]
        # ]

        # err_regex = "".join([f'(?!{s})' for s in success_str_list]) + r"\S+"
        # # err_regex = rf"(?!\s+|\d+|{unit_reg})"
        # err_matches = re.finditer(err_regex, lifetime_str, re.IGNORECASE)
        # err_string: list[str] = []
        # for err_match in err_matches:
        #     err_string.append(err_match.string)
        
        # if len(err_string) > 0:
        #     split_str = "', '"
        #     err_messages.append(f"不正な文字が検出されました。:'{split_str.join([e_str for e_str in err_string])}'")

        # 正しい部分の解析
        unit_reg = '|'.join([u for unit_list in unit_map.values() for u in unit_list])
        regex = rf"(?P<quantity>\d+)\s*(?P<unit>{unit_reg})"
        matches = re.finditer(regex, lifetime_str, re.IGNORECASE)

        time_units = {
            unit.value: 0
            for unit in DateTimeUnit
        }
        time_units_count = {
            unit.value: 0
            for unit in DateTimeUnit
        }

        for match in matches:
            quantity = int(match.group("quantity"))
            unit = match.group("unit")

            key = await cls.unit_to_mapkey(unit.lower())
            if key is not None:
                time_units[key] = quantity
                time_units_count[key] += 1

        duplicate_keys = []
        for key, count in time_units_count.items():
            if count >= 2:
                duplicate_keys.append(key)
        
        if len(duplicate_keys) > 0:
            dkeys = "', '".join([d_key for d_key in duplicate_keys])
            err_messages.append(f"重複している項目があります。:'{dkeys}'")

        if len(err_messages) > 0:
            return Err("\n".join(err_messages))

        return Ok(time_units)
    
    @classmethod
    async def convert_dict_to_timedelta(cls, obj: dict[str, int]):
        return timedelta(**obj)
    
    @classmethod
    async def interval_dict_to_string(cls, obj: dict[str, int]):
        return " ".join([
            f"{obj[u.value]} {u.value}"
            for u in DateTimeUnit
            if u.value in obj and obj[u.value] > 0
        ])

    @classmethod
    async def unit_to_mapkey(cls, unit):
        for u_key, list in unit_map.items():
            if unit in list:
                return u_key
        return None

class DateTimeUnit(enum.Enum):
    Weeks = "weeks"
    Days = "days"
    Hours = "hours"
    Minutes = "minutes"
    Seconds = "seconds"


unit_map: dict[str, list[str]] = dict()
unit_map[DateTimeUnit.Weeks.value] = ["w", "week"]
unit_map[DateTimeUnit.Days.value] = ["d", "day"]
unit_map[DateTimeUnit.Hours.value] = ["h", "hour"]
unit_map[DateTimeUnit.Minutes.value] = ["mi", "min", "minute"]
unit_map[DateTimeUnit.Seconds.value] = ["s", "sec", "second"]
