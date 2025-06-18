import os
import discord
from discord import Interaction
from discord import app_commands
from discord.ext import commands
from result import Err
from discord_bot.models.monitoring_channels import MonitoringChannels
from discord_bot.models.session import AsyncSessionLocal
from discord_bot.utils.messages import SingletonMessages
from discord_bot.utils.permission import Permission


class DisableCog(commands.Cog):

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
        name="disable",
        description="チャンネルを監視対象から解除します。必須ロール:message_cleaner_admin"
    )
    @app_commands.describe(
        channel="監視解除対象チャンネル",
    )
    async def execute(
            self,
            interaction: Interaction,
            channel: discord.TextChannel
    ):
        try:
            await interaction.response.defer()

            messages = await SingletonMessages.get_instance()

            permission_result = await Permission.is_message_delete_permission(interaction.user, channel)
            match permission_result:
                case Err(err_value):
                    log_message, display_message = await messages.get_log_and_display_message(err_value[0], os.environ.get("MESSAGE_LANGUAGE", "en"))
                    print(f"DisableCog.execute error: {log_message}")
                    await interaction.followup.send(display_message, ephemeral=True)
                    return

            async with self.bot.db_lock:
                async with AsyncSessionLocal() as session:
                    mc = await MonitoringChannels.select(
                        session,
                        channel.guild.id,
                        channel.id
                    )
                    await mc.delete(session)
                    await session.commit()

            msg = f"{channel.mention}の設定を解除しました"
            await interaction.followup.send(msg, ephemeral=True)
        except Exception as e:
            print(e)
            await interaction.followup.send("err", ephemeral=True)


async def setup(bot: commands.Bot):
    await bot.add_cog(DisableCog(bot))
