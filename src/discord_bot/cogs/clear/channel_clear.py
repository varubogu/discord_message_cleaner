import asyncio
import os
from result import Ok, Err
import discord
from discord import Interaction, TextChannel, VoiceChannel, Thread
from discord import app_commands
from discord.ext import commands
from discord_bot.models.exclusion_message import ExclusionMessage
from discord_bot.models.monitoring_channels import MonitoringChannels
from discord_bot.models.session import AsyncSessionLocal
from discord_bot.ui.dialogs.confirm_dialog import ConfirmDialog
from discord_bot.utils.failed_reason_code import FailedReasonCode
from discord_bot.utils.messages import SingletonMessages
from discord_bot.utils.permission import Permission
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
            channel: TextChannel | VoiceChannel | Thread
    ):
        try:
            await interaction.response.defer()

            messages = await SingletonMessages.get_instance()

            async with self.bot.db_lock:
                async with AsyncSessionLocal() as session:
                    monitorings = await MonitoringChannels.select(session, channel.guild.id, channel.id)
            permission_result = await Permission.is_message_delete_permission(interaction.user, channel)
            match permission_result:
                case Err(err_value):
                    log_message, display_message = await messages.get_log_and_display_message(err_value[0], os.environ.get("MESSAGE_LANGUAGE", "en"))
                    print(f"ChannelClearCog.execute error: {log_message}")
                    await interaction.followup.send(display_message, ephemeral=True)
                    return

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
        def __init__(
            self,
            monitorings: MonitoringChannels,
            channel: TextChannel | VoiceChannel | Thread
        ) -> None:
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

    async def message_delete(
        self,
        channel: TextChannel | VoiceChannel | Thread,
        limit: int = LOOP_DELETE_SIZE
    ):
        if channel is None:
            print("channel_clear.message_deleteでチャンネルが見つかりませんでした。")
            return (False, False)

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
        try:
            async for m in channel.history(limit=limit, oldest_first=True):
                if m.id in exclusion_ids:
                    continue
                if remove_range is not None and m.created_at >= remove_range:
                    is_complete = True
                    break
                try:
                    await m.delete()
                except Exception as e:
                    print(f"メッセージ削除で例外発生:{e}")
                await asyncio.sleep(self.MESSAGE_DELETE_INTERVAL)
                is_deleted = True
        except discord.Forbidden:
            print(f"channel_clear.message_deleteでチャンネルのメッセージ削除に失敗しました。:Forbidden guild_id={channel.guild.id}, channel_id={channel.id}")
            return (False, False)
        except discord.HTTPException:
            print(f"channel_clear.message_deleteでチャンネルのメッセージ削除に失敗しました。:HTTPException guild_id={channel.guild.id}, channel_id={channel.id}")
            return (False, False)
        except Exception as e:
            print(f"channel_clear.message_deleteで例外発生:{e}")
            return (False, False)

        return (is_complete, is_deleted)

    @execute.error
    async def execute_error(
        self,
        interaction: discord.Interaction,
        error: app_commands.AppCommandError
    ):
        messages = await SingletonMessages.get_instance()

        # 「message_cleaner_admin」ロールがコマンド使用者に無い場合
        if isinstance(error, discord.app_commands.MissingRole):
            _, display_message = await messages.get_log_and_display_message(
                FailedReasonCode.NO_BOT_USAGE_PERMISSION,
                os.environ.get("MESSAGE_LANGUAGE", "en")
            )
            await interaction.response.send_message(
                display_message,
                ephemeral=True
            )
        elif isinstance(error, discord.app_commands.TransformerError):
            e: discord.app_commands.TransformerError = error
            channel_id = e.value
            _, display_message = await messages.get_log_and_display_message(
                FailedReasonCode.CHANNEL_ACCESS_DENIED,
                os.environ.get("MESSAGE_LANGUAGE", "en")
            )
            display_message += f": {channel_id}"
            await interaction.response.send_message(
                display_message,
                ephemeral=True
            )
        else:
            log, display_message = await messages.get_log_and_display_message(
                FailedReasonCode.UNKNOWN,
                os.environ.get("MESSAGE_LANGUAGE", "en")
            )

            print(log + f":{error}")
            await interaction.response.send_message(
                display_message,
                ephemeral=True
            )


async def setup(bot: commands.Bot):
    await bot.add_cog(ChannelClearCog(bot))
