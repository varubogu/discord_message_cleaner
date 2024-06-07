from datetime import timedelta
from result import Ok, Err
import discord
from discord import Interaction, TextChannel
from discord import app_commands
from discord.ext import commands

from discord_bot.models.monitoring_channels import MonitoringChannels
from discord_bot.ui.dialogs.confirm_dialog import ConfirmDialog
from discord_bot.utils.lifetime import LifeTimeUtil
from discord_bot.models.session import AsyncSessionLocal
from discord_bot.utils.permission import Permission


class EnableCog(commands.Cog):

    def __init__(
            self,
            bot: commands.Bot
    ):
        self.__bot = bot

    @property
    def bot(self) -> commands.Bot:
        return self.__bot

    @app_commands.checks.has_role("message_cleaner_admin")
    @app_commands.command(
        name="enable",
        description="監視対象のチャンネルを設定します。必須ロール:message_cleaner_admin"
    )
    @app_commands.describe(
        channel="削除対象チャンネル",
        lifetime="どれだけの期間メッセージを残すか（省略時: 1day）"
    )
    async def execute(
            self,
            interaction: Interaction,
            channel: discord.TextChannel,
            lifetime: str = '1day'
    ):
        try:
            await interaction.response.defer()

            interval_parse_result = await LifeTimeUtil.calc(lifetime)

            match interval_parse_result:
                case Ok(value):
                    interval_object = value
                case Err(err):
                    await interaction.followup.send(err, ephemeral=True)
                    return
                
            interval = await LifeTimeUtil.convert_dict_to_timedelta(interval_object)

            if interval.total_seconds() <= 0:
                msg = "ライフタイムが指定されていません。"
                await interaction.followup.send(msg, ephemeral=True)
                return

            permission_result = await Permission.is_message_delete_permission(interaction.user, channel)
            match permission_result:
                case Err(err_value):
                    msg = "\n".join(err_value)
                    await interaction.followup.send(msg, ephemeral=True)
                    return

            msg = f"{channel.mention}を監視対象にしますか？\n" \
                f"監視対象のチャンネルのメッセージは[{interval}]が経過すると削除されます。" \
                "\n(続行する場合、10秒以内に押してください)"
            view = ConfirmDialog[self.TaskParameter](
                interaction.user,
                self.continue_task,
                self.TaskParameter(channel, interval)
            )
            view.timeout = 10
            result_message = await interaction.followup.send(msg, view=view, ephemeral=True)
            view.parent_message = result_message
        except Exception as e:
            print(e)
            await interaction.followup.send("err", ephemeral=True)

    class TaskParameter():
        def __init__(self, channel: TextChannel, interval: timedelta) -> None:
            self.channel = channel
            self.interval = interval

    async def continue_task(self, interaction: Interaction, parameter: TaskParameter):
        try:
            await interaction.response.defer()
            channel = parameter.channel
            interval = parameter.interval

            mc = MonitoringChannels()
            mc.guild_id = channel.guild.id
            mc.channel_id = channel.id
            mc.interval = interval

            async with self.bot.db_lock:
                async with AsyncSessionLocal() as session:
                    await mc.merge(session)
                    await session.commit()

            msg = f"[{channel.mention}]に定期削除を設定しました。\nライフタイム[{interval}]"
            await interaction.followup.send(msg, ephemeral=True)
        except Exception as e:
            print(e)
            await interaction.followup.send("err", ephemeral=True)


async def setup(bot: commands.Bot):
    await bot.add_cog(EnableCog(bot))
