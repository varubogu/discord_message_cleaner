import asyncio
from result import Ok, Err
import discord
from discord import Interaction, TextChannel
from discord import app_commands
from discord.ext import commands
from discord_bot.models.exclusion_message import ExclusionMessage
from discord_bot.models.monitoring_channels import MonitoringChannels
from discord_bot.models.session import AsyncSessionLocal
from discord_bot.ui.dialogs.confirm_dialog import ConfirmDialog
from discord_bot.utils.environment import get_os_environ_safety

LOOP_DELETE_SIZE = get_os_environ_safety('LOOP_DELETE_SIZE', int, 10)


class ChannelClearCog(commands.Cog):

    def __init__(
            self,
            bot: commands.Bot
    ):
        self.__bot = bot
        self.MESSAGE_DELETE_INTERVAL = get_os_environ_safety('MESSAGE_DELETE_INTERVAL', int, 1)
        self.MAX_SIZE = get_os_environ_safety('MAX_SIZE', int, 100)


    @property
    def bot(self) -> commands.Bot:
        return self.__bot

    @app_commands.checks.has_role("message_cleaner_admin")
    @app_commands.command(
        name="clear",
        description="対象したチャンネルのメッセージをすべて削除します。必須ロール:message_cleaner_admin"
    )
    @app_commands.describe(
        channel="削除対象チャンネル"
    )
    async def execute(
            self,
            interaction: Interaction,
            channel: discord.TextChannel
    ):
        try:
            await interaction.response.defer()
            async with self.bot.db_lock:
                async with AsyncSessionLocal() as session:
                    monitorings = await MonitoringChannels.select(session, channel.guild.id, channel.id)

            msg = ""
            if monitorings is None:
                msg = f"{channel.mention}は削除監視対象外のテーブルです。\n" \
                    "続行する場合、除外したメッセージを除く全てのメッセージが削除されます。\n"
            msg += f"{channel.mention}の削除を実行しますか？\n(続行する場合、10秒以内に押してください)"
            view = ConfirmDialog[self.TaskParameter](
                interaction.user,
                self.continue_task,
                self.TaskParameter(monitorings, channel)
            )
            view.timeout = 10
            result_message = await interaction.followup.send(msg, view=view, ephemeral=True)
            view.parent_message = result_message

        except Exception as e:
            print(e)
            await interaction.followup.send("err", ephemeral=True)

    class TaskParameter():
        def __init__(self, monitorings: MonitoringChannels, channel: TextChannel) -> None:
            self.monitorings = monitorings
            self.channel = channel

    async def continue_task(self, interaction: Interaction, parameter: TaskParameter):
        await interaction.response.defer()
        await interaction.followup.send("削除中...", ephemeral=True)

        channel = parameter.channel

        (is_complete, is_deleted) = await self.message_delete(channel, self.MAX_SIZE)

        if not is_deleted:
            msg = "削除対象がありませんでした。"
        else:
            msg = "削除が完了しました。"
            if not is_complete:
                msg += f"\n1度に消せるのは{self.MAX_SIZE}件までです。\n続けて削除する場合はもう一度コマンドを実行してください。"
        await interaction.followup.send(msg, ephemeral=True)

    async def message_delete(self, channel: TextChannel, limit: int = LOOP_DELETE_SIZE):
        async with self.bot.db_lock:
            async with AsyncSessionLocal() as session:
                remove_range_result = await MonitoringChannels.select_remove_range(session, channel.guild.id, channel.id)
                exclusions = await ExclusionMessage.select_with_guild_channel(session, channel.guild.id, channel.id)

        exclusion_ids = [e.message_id for e in exclusions]
        match remove_range_result:
            case Ok(ok_value):
                remove_range = ok_value
            case Err():
                remove_range = None

        is_complete = False # ライフタイムと除外メッセージ以外の全てを削除しおえたか
        is_deleted = False # 1件以上削除したか
        async for m in channel.history(limit=limit, oldest_first=True):
            if m.id in exclusion_ids:
                continue
            if remove_range is not None and m.created_at >= remove_range:
                is_complete = True
                break
            await m.delete()
            await asyncio.sleep(self.MESSAGE_DELETE_INTERVAL)
            is_deleted = True
        return (is_complete, is_deleted)

    async def test(self):
        print("hello")

async def setup(bot: commands.Bot):
    await bot.add_cog(ChannelClearCog(bot))
