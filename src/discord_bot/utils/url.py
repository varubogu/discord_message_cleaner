import re
from result import Result, Ok, Err

class UrlUtil:

    @classmethod
    async def is_url(cls, url: str) -> bool:
        pattern = r"https://" # 先頭部分で妥協する
        m = re.search(pattern, url)
        if m:
            return True
        else:
            return False

    @classmethod
    async def try_parse_discord_url(cls, url: str) -> Result[dict[str, int], str]:
        pattern = r"https://discord\.com/channels/(\d+)/(\d+)/(\d+)"
        m = re.search(pattern, url)
        if m:
            return Ok({
                "guild_id": int(m.group(1)),
                "channel_id": int(m.group(2)),
                "message_id": int(m.group(3))
            })
        return Err("URLの解析に失敗しました。")
    
    @classmethod
    async def try_parse_guild_id(cls, url) -> Result[int, str]:
        pattern = r"https://discord\.com/channels/(\d+)/\d+/\d+"
        m = re.search(pattern, url)
        if m:
            return Ok(int(m.group(1)))
        return Err("URLからguild_idの取得に失敗しました。")

    @classmethod
    async def try_parse_channel_id(cls, url) -> Result[int, str]:
        pattern = r"https://discord\.com/channels/\d+/(\d+)/\d+"
        m = re.search(pattern, url)
        if m:
            return Ok(int(m.group(1)))
        return Err("URLからchannel_idの取得に失敗しました。")

    @classmethod
    async def try_parse_message_id(cls, url) -> Result[int, str]:
        pattern = r"https://discord\.com/channels/\d+/\d+/(\d+)"
        m = re.search(pattern, url)
        if m:
            return Ok(int(m.group(1)))
        return Err("URLからmessage_idの取得に失敗しました。")
