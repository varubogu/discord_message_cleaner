from typing import Any, TypeVar, Callable
from result import Result, Ok, Err

T = TypeVar('T')

class ParseUtil:

    @classmethod
    async def try_parse(cls, from_obj: Any, factory: Callable[[Any], T]) -> Result[T, str]:
        try:
            result_value = factory(from_obj)
            return Ok(result_value)
        except ValueError:
            return Err(f"変換に失敗しました。値:[{from_obj}], 変換しようとした型:[{factory.__name__}]")
    
