import os
from result import Ok, Err
from discord import Guild, Interaction, Message, TextChannel
from discord import app_commands
from discord.ext import commands
from discord_bot.models.exclusion_message import ExclusionMessage
from discord_bot.ui.dialogs.confirm_dialog import ConfirmDialog
from discord_bot.utils.discord_helper import DiscordHelper
from discord_bot.models.session import AsyncSessionLocal
from discord_bot.utils.messages import SingletonMessages
from discord_bot.utils.permission import Permission
from discord_bot.utils.url import UrlUtil


class ExclusionRemoveCog(commands.Cog):

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
        name="exclusion_remove",
        description="設定済みの除外設定を解除します。必須ロール:message_cleaner_admin"
    )
    @app_commands.describe(
        message_url="メッセージURLを指定します。"
    )
    async def execute(
            self,
            interaction: Interaction,
            message_url: str
    ):
        try:
            await interaction.response.defer()

            messages = await SingletonMessages.get_instance()

            if not await UrlUtil.is_url(message_url):
                msg = "メッセージURLがURL形式ではありません"
                await interaction.followup.send(msg, ephemeral=True)
                return

            url_parse_result = await UrlUtil.try_parse_discord_url(message_url)
            match url_parse_result:
                case Ok(ok_value):
                    pass
                case Err(err_value):
                    msg = err_value
                    await interaction.followup.send(msg, ephemeral=True)
                    return
                case _:
                    msg = "メッセージの取得に失敗しました。\n"
                    await interaction.followup.send(msg, ephemeral=True)
                    return

            if interaction.guild is not None and ok_value["guild_id"] != interaction.guild.id:
                msg = "違うサーバーの情報は削除できません。\nそのサーバーからコマンドを実行してください。"
                await interaction.followup.send(msg, ephemeral=True)
                return

            fetch_result = await DiscordHelper.fetch_guild_channel_message(self.bot, **ok_value)
            match fetch_result:
                case Ok(ok_value):
                    (guild, channel, message) = ok_value
                case Err(err_value):
                    log_message, display_message = await messages.get_log_and_display_message(err_value, os.environ.get("MESSAGE_LANGUAGE", "en"))
                    print(f"ExclusionRemoveCog.execute error: {log_message}")
                    await interaction.followup.send(display_message, ephemeral=True)
                    return
                case _:
                    msg = "メッセージの取得に失敗しました。"
                    await interaction.followup.send(msg, ephemeral=True)
                    return

            permission_result = await Permission.is_message_read_permission(interaction.user, channel)
            match permission_result:
                case Err(err_value):
                    msg = "メッセージの取得に失敗しました。"
                    await interaction.followup.send(msg, ephemeral=True)
                    return

            msg = f"{message.jump_url}の除外設定を解除しますか？\n" \
                "除外設定解除によりメッセージが削除される場合があります。\n" \
                "(続行する場合、10秒以内に押してください)"

            view = ConfirmDialog[self.TaskParameter](
                interaction.user,
                self.continue_task,
                self.TaskParameter(guild, channel, message)
            )
            view.timeout = 10
            result_message = await interaction.followup.send(msg, view=view, ephemeral=True)
            view.parent_message = result_message
        except Exception as e:
            print(e)
            await interaction.followup.send("err", ephemeral=True)

    class TaskParameter():
        def __init__(self, guild: Guild, channel: TextChannel, message: Message) -> None:
            self.guild = guild
            self.channel = channel
            self.message = message

    async def continue_task(self, interaction: Interaction, parameter: TaskParameter):
        try:
            await interaction.response.defer()

            guild = parameter.guild
            channel = parameter.channel
            message = parameter.message

            async with self.bot.db_lock:
                async with AsyncSessionLocal() as session:
                    mc = await ExclusionMessage.select(
                        session,
                        guild.id,
                        channel.id,
                        message.id
                    )
                    if mc is None:
                        msg = f"{message.jump_url}の除外設定はありません。"
                        await interaction.followup.send(msg, ephemeral=True)
                        return
                    await mc.delete(session)
                    await session.commit()

            msg = f"{message.jump_url}の除外設定を解除しました。"
            await interaction.followup.send(msg, ephemeral=True)
        except Exception as e:
            print(e)
            await interaction.followup.send("err", ephemeral=True)


async def setup(bot: commands.Bot):
    await bot.add_cog(ExclusionRemoveCog(bot))
