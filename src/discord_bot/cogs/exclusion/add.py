from result import Ok, Err
from discord import Interaction
from discord import app_commands
from discord.ext import commands
from discord_bot.models.exclusion_message import ExclusionMessage
from discord_bot.utils.discord_helper import DiscordHelper
from discord_bot.models.session import AsyncSessionLocal
from discord_bot.utils.url import UrlUtil


class ExclusionAddCog(commands.Cog):

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
        name="exclusion_add",
        description="メッセージURLを指定することで削除設定から除外することができます。必須ロール:message_cleaner_admin"
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
                msg = "違うサーバーの情報は登録できません。\nそのサーバーからコマンドを実行してください。"
                await interaction.followup.send(msg, ephemeral=True)
                return

            fetch_result = await DiscordHelper.fetch_guild_channel_message(self.bot, **ok_value)
            match fetch_result:
                case Ok(ok_value):
                    (guild, channel, message) = ok_value
                case Err(err_value):
                    msg = err_value
                    await interaction.followup.send(msg, ephemeral=True)
                    return
                case _:
                    msg = "メッセージの取得に失敗しました。"
                    await interaction.followup.send(msg, ephemeral=True)
                    return

            mc = ExclusionMessage()
            mc.guild_id = guild.id
            mc.channel_id = channel.id
            mc.message_id = message.id

            async with self.bot.db_lock:
                async with AsyncSessionLocal() as session:
                    await mc.merge(session)
                    await session.commit()

            msg = f"{message.jump_url}に除外設定を追加しました。"
            await interaction.followup.send(msg, ephemeral=True)
        except Exception as e:
            print(e)
            await interaction.followup.send("err", ephemeral=True)


async def setup(bot: commands.Bot):
    await bot.add_cog(ExclusionAddCog(bot))
