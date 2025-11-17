from typing import Optional


class MessageDeleteResult:
    def __init__(
            self,
            result: bool,
            is_complete: bool,
            is_max_deleted: bool,
            is_deleted: bool,
            delete_count: int,
            exception: Optional[Exception] = None
    ):
        self.result = result
        self.is_complete = is_complete
        self.is_max_deleted = is_max_deleted
        self.is_deleted = is_deleted
        self.delete_count = delete_count
        self.exception = exception
