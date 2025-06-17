from typing import Optional
import discord
from discord import Interaction
from discord import app_commands
from discord.ext import commands
from result import Err, Ok
from sqlalchemy.ext.asyncio import AsyncSession
from discord_bot.models.exclusion_message import ExclusionMessage
from discord_bot.models.monitoring_channels import MonitoringChannels
from discord_bot.models.session import AsyncSessionLocal
from discord_bot.utils.discord_helper import DiscordHelper
from discord_bot.utils.permission import Permission


class SettingsShowCog(commands.Cog):
    def __init__(self, bot: commands.Bot):
        self.__bot = bot

    @property
    def bot(self) -> commands.Bot:
        return self.__bot

    @app_commands.checks.has_role("message_cleaner_admin")
    @app_commands.command(
        name="settings",
        description="現在のサーバーで設定中の設定内容を表示します。必須：[message_cleaner_admin]ロール",
    )
    @app_commands.describe(
        channel="入力任意のパラメータです。 省略した場合、サーバー内の全ての設定が表示されます。"
    )
    async def execute(
        self, interaction: Interaction, channel: Optional[discord.TextChannel]
    ):
        try:
            await interaction.response.defer()
            async with self.bot.db_lock:
                async with AsyncSessionLocal() as session:
                    if channel is None:
                        if interaction.guild is None:
                            await interaction.followup.send(
                                "チャンネルを指定してください。", ephemeral=True
                            )
                            return
                        embed = await self.server_show(
                            session, interaction, interaction.guild
                        )
                    else:
                        permission_result = await Permission.is_message_read_permission(
                            interaction.user, channel
                        )
                        match permission_result:
                            case Err(err_value):
                                msg = "\n".join(err_value)
                                await interaction.followup.send(msg, ephemeral=True)
                                return
                        embed = await self.channel_show(session, interaction, channel)

            await interaction.followup.send("", embed=embed, ephemeral=True)
        except Exception as e:
            print(e)
            await interaction.followup.send("err", ephemeral=True)

    async def server_show(
        self, session: AsyncSession, interaction: Interaction, guild: discord.Guild
    ) -> discord.Embed:
        monitoring_channels = await MonitoringChannels.select_with_guild(
            session, guild.id
        )

        exclusions = await ExclusionMessage.select_with_guild(session, guild.id)

        embed = discord.Embed(title=f"{guild.name}の設定一覧", description="")

        # 定期削除対象チャンネルを収集
        channel_infos: list[str] = []
        ch_messages: dict[str, list[str]] = {}
        for monitoring_channel in monitoring_channels:
            channel_result = await DiscordHelper.get_or_fetch_channel_from_guild(
                guild, monitoring_channel.channel_id
            )
            match channel_result:
                case Ok(ok_value):
                    channel = ok_value
                case Err(err_value):
                    print(err_value)
                    continue

            channel_infos.append(f"{channel.mention}: {monitoring_channel.interval}")
        if len(channel_infos) > 0:
            channels_str = "\n".join([x for x in channel_infos])
        else:
            channels_str = "なし"

        embed.add_field(
            name="定期削除有効チャンネル: ライフタイム",
            value=channels_str,
            inline=False,
        )

        # 除外メッセージを収集
        for exclusion in exclusions:
            channel_result = await DiscordHelper.get_or_fetch_channel_from_guild(
                guild, exclusion.channel_id
            )
            match channel_result:
                case Ok(ok_value):
                    channel = ok_value
                case Err(err_value):
                    print(err_value)
                    continue

            try:
                m = await channel.fetch_message(exclusion.message_id)
            except discord.Forbidden:
                continue
            except discord.NotFound:
                continue

            if ch_messages.get(channel.mention) is None:
                ch_messages[channel.mention] = []
            ch_messages[channel.mention].append(m.jump_url)

        msg = ""
        for ch, urls in ch_messages.items():
            # チャンネル名:
            msg += f"- {ch}:\n"
            for url in urls:
                # メッセージURL
                msg += f"  - {url}\n"

        if msg == "":
            msg = "なし"

        embed.add_field(name="除外したメッセージ", value=msg, inline=False)
        return embed

    async def channel_show(
        self,
        session: AsyncSession,
        interaction: Interaction,
        channel: discord.TextChannel,
    ) -> discord.Embed:

        monitoring_channels = await MonitoringChannels.select(
            session, channel.guild.id, channel.id
        )
        exclusions = await ExclusionMessage.select_with_guild_channel(
            session, channel.guild.id, channel.id
        )

        embed = discord.Embed(title=f"{channel.mention}の設定一覧", description="")
        embed.add_field(
            name="定期削除",
            value=f'{"無効" if monitoring_channels is None else "有効"}',
            inline=False,
        )
        embed.add_field(
            name="ライフタイム",
            value=f'{"無効" if monitoring_channels is None else monitoring_channels.interval}',
            inline=False,
        )

        messages: list[discord.Message] = []
        for exclusion in exclusions:
            try:
                m = await channel.fetch_message(exclusion.message_id)
            except discord.Forbidden:
                continue
            except discord.NotFound:
                continue

            messages.append(m)
        if len(messages) > 0:
            url_str = "\n".join([f"{m.jump_url}: {m.id}" for m in messages])
        else:
            url_str = "なし"
        embed.add_field(name="除外したメッセージ", value=url_str, inline=False)
        return embed


async def setup(bot: commands.Bot):
    await bot.add_cog(SettingsShowCog(bot))
