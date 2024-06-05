from typing import Optional, Tuple
from discord.guild import Guild
from discord.message import Message
from result import Result, Ok, Err
import discord
from discord.ext import commands

class DiscordHelper:

    @classmethod
    async def get_or_fetch_guild(
        cls,
        bot: commands.Bot,
        guild_id: int,
    ) -> Guild:
        return bot.get_guild(guild_id) or await bot.fetch_guild(guild_id)
    
    @classmethod
    async def get_or_fetch_channel_from_bot(
        cls,
        bot: commands.Bot,
        channel_id: int
    ):
        return bot.get_channel(channel_id) or await bot.fetch_channel(channel_id)

    @classmethod
    async def get_or_fetch_channel_from_guild(
        cls,
        guild: Guild,
        channel_id: int
    ):
        return guild.get_channel(channel_id) or await guild.fetch_channel(channel_id)


    @classmethod
    async def fetch_guild_channel_message(
        cls,
        bot: commands.Bot,
        guild_id: Optional[int],
        channel_id: Optional[int],
        message_id: Optional[int]
    ) -> Result[Tuple[Optional[Guild], Optional[discord.TextChannel], Optional[Message]], str]:

        if guild_id is None:
            guild = None
        else:
            try:
                guild = await cls.get_or_fetch_guild(bot, guild_id)
            except discord.Forbidden:
                msg = "指定されたサーバーへのアクセス権限がありません。"
                return Err(msg)
            except discord.HTTPException:
                msg = "処理に失敗しました。再度お試しください。"
                return Err(msg)

        if channel_id is None:
            channel = None
        else:

            try:
                if guild is None:
                    channel = await cls.get_or_fetch_channel_from_bot(bot, channel_id)
                else:
                    channel = await cls.get_or_fetch_channel_from_guild(guild, channel_id)
            except discord.NotFound:
                msg = "指定されたチャンネルが存在しません。"
                return Err(msg)
            except discord.Forbidden:
                msg = "指定されたチャンネルへのアクセス権限がありません。"
                return Err(msg)
            except discord.HTTPException:
                msg = "処理に失敗しました。再度お試しください。"
                return Err(msg)

        if message_id is None:
            message = None
        else:
            try:
                if channel is not None:
                    message = await channel.fetch_message(message_id)
                else:
                    message = None
            except discord.NotFound:
                msg = "指定されたメッセージが存在しません。"
                return Err(msg)
            except discord.Forbidden:
                msg = "指定されたメッセージのアクセス権限がありません。"
                return Err(msg)
            except discord.HTTPException:
                msg = "処理に失敗しました。再度お試しください。"
                return Err(msg)
        result = (guild, channel, message)
        return Ok(result)