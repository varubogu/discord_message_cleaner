import discord
from result import Result, Ok, Err

from discord_bot.utils.failed_reason_code import FailedReasonCode
from discord import TextChannel, VoiceChannel, Thread

class Permission:

    @classmethod
    async def is_message_read_permission(
        cls,
        user: discord.User | discord.Member,
        channel: TextChannel | VoiceChannel | Thread
    ) -> Result[None, list[FailedReasonCode]]:
        message_list = []
        if isinstance(user, discord.User):
            return Ok(None)
        a = channel.permissions_for(user)
        if a.administrator is False:
            if a.read_messages is False:
                message_list.append(FailedReasonCode.MESSAGE_READ_PERMISSION_DENIED)
            elif a.read_message_history is False:
                message_list.append(FailedReasonCode.MESSAGE_READ_HISTORY_PERMISSION_DENIED)
        if len(message_list) == 0:
            return Ok(None)
        else:
            return Err(message_list)

    @classmethod
    async def is_message_delete_permission(
        cls,
        user: discord.User | discord.Member,
        channel: TextChannel | VoiceChannel | Thread
    ) -> Result[None, list[FailedReasonCode]]:
        message_list = []
        if isinstance(user, discord.User):
            return Ok(None)
        a = channel.permissions_for(user)
        if a.administrator is False:
            if a.read_messages is False:
                message_list.append(FailedReasonCode.MESSAGE_DELETE_PERMISSION_DENIED)
            elif a.read_message_history is False:
                message_list.append(FailedReasonCode.MESSAGE_READ_HISTORY_PERMISSION_DENIED)

            if a.manage_messages is False:
                message_list.append(FailedReasonCode.MESSAGE_DELETE_PERMISSION_DENIED)
        if len(message_list) == 0:
            return Ok(None)
        else:
            return Err(message_list)

