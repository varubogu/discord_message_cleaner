import discord
from result import Result, Ok, Err

class Permission:

    @classmethod
    async def is_message_read_permission(
        cls,
        user: discord.User | discord.Member,
        channel: discord.TextChannel
    ) -> Result[None, list[str]]:
        message_list = []
        if isinstance(user, discord.User):
            return Ok(None)
        a = channel.permissions_for(user)
        if a.administrator is False:
            if a.read_messages is False:
                message_list.append("チャンネルの閲覧権限がありません。")
            elif a.read_message_history is False:
                message_list.append("メッセージ履歴を見る権限がありません。")
        if len(message_list) == 0:
            return Ok(None)
        else:
            return Err(message_list)

    @classmethod
    async def is_message_delete_permission(
        cls,
        user: discord.User | discord.Member,
        channel: discord.TextChannel
    ) -> Result[None, list[str]]:
        message_list = []
        if isinstance(user, discord.User):
            return Ok(None)
        a = channel.permissions_for(user)
        if a.administrator is False:
            if a.read_messages is False:
                message_list.append("あなたにはこのチャンネルの閲覧権限がありません。")
            elif a.read_message_history is False:
                message_list.append("あなたにはこのメッセージ履歴を見る権限がありません。")

            if a.manage_messages is False:
                message_list.append("あなたにはこのチャンネルのメッセージを削除する権限がありません。")
        if len(message_list) == 0:
            return Ok(None)
        else:
            return Err(message_list)

