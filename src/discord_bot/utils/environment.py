from typing import TypeVar, Callable
import os

T = TypeVar('T')

def get_os_environ_safety(key: str, t: Callable[[str], T], default: T) -> T:
    try:
        s = os.environ.get(key)
        if s is None:
            print(f"warning: '{key}' is None. use default value: {default}")
            return default
        v = t(s)
        return v
    except TypeError:
        print(f"warning: '{key}' is not {t.__name__}. use default value: {default}")
        return default
    except ValueError:
        print(f"warning: '{key}' is not {(t.__name__)} value. use default value: {default}")
        return default
