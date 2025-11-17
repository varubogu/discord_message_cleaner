from typing import Optional, Tuple, Any
import discord
from result import Result, Ok, Err
from discord.guild import Guild
from discord.message import Message
from discord.ext import commands
from discord import TextChannel, VoiceChannel, Thread
from discord_bot.utils.failed_reason_code import FailedReasonCode

class DiscordHelper:

    @classmethod
    async def is_message_delete_permission(
        cls,
        user: discord.User | discord.Member,
        channel: TextChannel | VoiceChannel | Thread
    ) -> Result[None, list[FailedReasonCode]]:
        msg = []
        a = channel.permissions_for(user)
        if a.administrator is False:
            if a.read_message_history is False:
                msg.append(FailedReasonCode.MESSAGE_READ_PERMISSION_DENIED)
            if a.manage_messages is False:
                msg.append(FailedReasonCode.MESSAGE_DELETE_PERMISSION_DENIED)
        if msg == "":
            return Ok(None)
        else:
            return Err(msg)

    @classmethod
    async def get_or_fetch_guild(
        cls,
        bot: commands.Bot,
        guild_id: int,
    ) -> Result[Guild, FailedReasonCode]:
        guild = bot.get_guild(guild_id)
        if guild is None:
            try:
                guild = await bot.fetch_guild(guild_id)
                if guild is None:
                    return Err(FailedReasonCode.GUILD_NOT_FOUND)
            except discord.Forbidden:
                return Err(FailedReasonCode.GUILD_ACCESS_DENIED)
            except discord.HTTPException:
                return Err(FailedReasonCode.GUILD_NOT_FOUND)
        return Ok(guild)

    @classmethod
    async def get_or_fetch_channel_from_bot(
        cls,
        bot: commands.Bot,
        channel_id: int
    ) -> Result[Any, FailedReasonCode]:
        channel = bot.get_channel(channel_id)
        if channel is None:
            try:
                channel = await bot.fetch_channel(channel_id)
                if channel is None:
                    return Err(FailedReasonCode.CHANNEL_NOT_FOUND)
            except discord.Forbidden:
                return Err(FailedReasonCode.CHANNEL_ACCESS_DENIED)
            except discord.HTTPException:
                return Err(FailedReasonCode.CHANNEL_NOT_FOUND)
        return Ok(channel)

    @classmethod
    async def get_or_fetch_channel_from_guild(
        cls,
        guild: Guild,
        channel_id: int
    ) -> Result[Any, FailedReasonCode]:
        channel = guild.get_channel(channel_id)
        if channel is None:
            try:
                channel = await guild.fetch_channel(channel_id)
                if channel is None:
                    return Err(FailedReasonCode.CHANNEL_NOT_FOUND)
            except discord.InvalidData:
                return Err(FailedReasonCode.CHANNEL_NOT_FOUND)
            except discord.NotFound:
                return Err(FailedReasonCode.CHANNEL_NOT_FOUND)
            except discord.Forbidden:
                return Err(FailedReasonCode.CHANNEL_ACCESS_DENIED)
            except discord.HTTPException:
                return Err(FailedReasonCode.CHANNEL_NOT_FOUND)
        return Ok(channel)


    @classmethod
    async def fetch_guild_channel_message(
        cls,
        bot: commands.Bot,
        guild_id: Optional[int],
        channel_id: Optional[int],
        message_id: Optional[int]
    ) -> Result[
        Tuple[Optional[Guild], Optional[TextChannel | VoiceChannel | Thread], Optional[Message]],
        FailedReasonCode
    ]:
        guild = None
        channel = None
        message = None

        if guild_id is None:
            guild = None
        else:
            try:
                guild_result = await cls.get_or_fetch_guild(bot, guild_id)
                match guild_result:
                    case Ok(ok_value):
                        guild = ok_value
                    case Err(err_value):
                        return Err(err_value)
            except discord.Forbidden:
                return Err(FailedReasonCode.GUILD_ACCESS_DENIED)
            except discord.HTTPException:
                return Err(FailedReasonCode.GUILD_NOT_FOUND)

        if channel_id is None:
            channel = None
        else:

            try:
                if guild is None:
                    channel_result = await cls.get_or_fetch_channel_from_bot(bot, channel_id)
                else:
                    channel_result = await cls.get_or_fetch_channel_from_guild(guild, channel_id)

                match channel_result:
                    case Ok(ok_value):
                        channel = ok_value
                    case Err(err_value):
                        return Err(err_value)
            except discord.NotFound:
                return Err(FailedReasonCode.CHANNEL_NOT_FOUND)
            except discord.Forbidden:
                return Err(FailedReasonCode.CHANNEL_ACCESS_DENIED)
            except discord.HTTPException:
                return Err(FailedReasonCode.CHANNEL_NOT_FOUND)

        if message_id is None:
            message = None
        else:
            try:
                if channel is not None:
                    message = await channel.fetch_message(message_id)
                else:
                    message = None
            except discord.NotFound:
                return Err(FailedReasonCode.MESSAGE_NOT_FOUND)
            except discord.Forbidden:
                return Err(FailedReasonCode.MESSAGE_ACCESS_DENIED)
            except discord.HTTPException:
                return Err(FailedReasonCode.MESSAGE_NOT_FOUND)
        result = (guild, channel, message)
        return Ok(result)