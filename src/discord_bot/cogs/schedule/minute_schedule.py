import asyncio
from datetime import datetime
import os
import traceback
from typing import Optional, Sequence, Tuple
from discord.ext import commands, tasks
from result import Err, Ok
from sqlalchemy.ext.asyncio import AsyncSession
from discord_bot.utils.environment import get_os_environ_safety
from discord_bot.utils.discord_helper import DiscordHelper
from discord_bot.models.monitoring_channels import MonitoringChannels
from discord_bot.models.session import AsyncSessionLocal
from discord_bot.cogs.clear.channel_clear import ChannelClearCog
from discord_bot.utils.failed_reason_code import FailedReasonCode
from discord_bot.utils.messages import SingletonMessages

LOOP_SEC = get_os_environ_safety('LOOP_SEC', int, 10)


class MinuteSchedule(commands.Cog):

    def __init__(self, bot: commands.Bot):
        self.bot = bot
        self.cleaner: Optional[ChannelClearCog] = None
        self.CHANNEL_DELETE_INTERVAL = get_os_environ_safety('CHANNEL_DELETE_INTERVAL', int, 10)

    async def cog_load(self):
        self.loop.start()

    async def cog_unload(self):
        self.loop.cancel()

    @tasks.loop(seconds=LOOP_SEC)
    async def loop(self):
        now = datetime.now()

        if not self.bot.is_ready():
            return

        if self.cleaner is None:
            self.cleaner = self.bot.cogs.get("ChannelClearCog")

        try:
            async with self.bot.db_lock:
                async with AsyncSessionLocal() as session:
                    monitoring_channels, = await self.inner_db_loop(session, now)
            await self.inner_loop(session, now, monitoring_channels)
        except Exception as e:
            print(f'schedule.loopで例外発生:{e}\n{traceback.format_exc()}')

    async def inner_db_loop(
            self,
            session: AsyncSession,
            now: datetime
    ) -> Tuple[Sequence[MonitoringChannels]]:
        return (await MonitoringChannels.select_all(session),)


    async def inner_loop(
            self,
            session: AsyncSession,
            now: datetime,
            monitoring_channels: Sequence[MonitoringChannels]
    ):
        messages = await SingletonMessages.get_instance()
        guild = None
        channel = None
        for monitoring in monitoring_channels:
            guild_result = await DiscordHelper.get_or_fetch_guild(self.bot, monitoring.guild_id)
            match guild_result:
                case Ok(ok_value):
                    guild = ok_value
                case Err(err_value):
                    print(f"サーバー取得で例外発生:guild_id={monitoring.guild_id}, {err_value}")
                    log_message, _ = await messages.get_log_and_display_message(err_value, os.environ.get("MESSAGE_LANGUAGE", "en"))
                    print(log_message)

                    if err_value == FailedReasonCode.GUILD_NOT_FOUND:
                        print(f"サーバーが見つからないので監視対象から削除:guild_id={monitoring.guild_id}")
                        await monitoring.delete(session)
                        await session.commit()
                    else:
                        continue

            channel_result = await DiscordHelper.get_or_fetch_channel_from_guild(guild, monitoring.channel_id)
            match channel_result:
                case Ok(ok_value):
                    channel = ok_value
                case Err(err_value):
                    print(f"チャンネル取得で例外発生:guild_id={monitoring.guild_id}, channel_id={monitoring.channel_id}, {err_value}")
                    log_message, _ = await messages.get_log_and_display_message(err_value, os.environ.get("MESSAGE_LANGUAGE", "en"))
                    print(log_message)

                    if err_value == FailedReasonCode.CHANNEL_NOT_FOUND:
                        print(f"チャンネルが見つからないので監視対象から削除:guild_id={monitoring.guild_id}, channel_id={monitoring.channel_id}")
                        await monitoring.delete(session)
                        await session.commit()
                    else:
                        continue

            await self.cleaner.message_delete(channel)
            await asyncio.sleep(self.CHANNEL_DELETE_INTERVAL)


async def setup(bot: commands.Bot):
    await bot.add_cog(MinuteSchedule(bot))
