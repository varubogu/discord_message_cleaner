import os

import discord
from discord import Interaction, app_commands
from discord.ext import commands

from discord_bot.utils.messages import SingletonMessages


class HelpCog(commands.Cog):
    def __init__(self, bot: commands.Bot):
        self.__bot = bot

    @property
    def bot(self) -> commands.Bot:
        return self.__bot

    @app_commands.command(
        name="help",
        description="Display bot usage and command list."
    )
    async def execute(self, interaction: Interaction):
        try:
            await interaction.response.defer(ephemeral=True)
            
            messages = await SingletonMessages.get_instance()
            locale = os.environ.get("MESSAGE_LANGUAGE", "en")
            
            # ヘルプメッセージの取得
            title = await messages.get_message("HELP_TITLE", locale)
            description = await messages.get_message("HELP_DESCRIPTION", locale)
            commands_title = await messages.get_message("HELP_COMMANDS_TITLE", locale)
            permission_note = await messages.get_message("HELP_PERMISSION_NOTE", locale)
            lifetime_examples = await messages.get_message("HELP_LIFETIME_EXAMPLES", locale)
            description_label = await messages.get_message("HELP_DESCRIPTION_LABEL", locale)
            usage_label = await messages.get_message("HELP_USAGE_LABEL", locale)
            or_text = await messages.get_message("HELP_OR", locale)
            footer_text = await messages.get_message("HELP_FOOTER", locale)
            error_message = await messages.get_message("HELP_ERROR", locale)
            
            # 各コマンドの説明を取得
            enable_desc = await messages.get_message("HELP_ENABLE_DESC", locale)
            disable_desc = await messages.get_message("HELP_DISABLE_DESC", locale)
            exclude_add_desc = await messages.get_message("HELP_EXCLUDE_ADD_DESC", locale)
            exclude_remove_desc = await messages.get_message("HELP_EXCLUDE_REMOVE_DESC", locale)
            settings_desc = await messages.get_message("HELP_SETTINGS_DESC", locale)
            clear_desc = await messages.get_message("HELP_CLEAR_DESC", locale)
            schedule_desc = await messages.get_message("HELP_SCHEDULE_DESC", locale)
            
            # Embedの作成
            embed = discord.Embed(
                title=title,
                description=description,
                color=0x00ff00
            )
            
            # コマンド一覧を追加
            embed.add_field(
                name=f"`/enable` - {commands_title}",
                value=f"{description_label} {enable_desc}\n{usage_label} `/enable #general 1day`",
                inline=False
            )
            
            embed.add_field(
                name="`/disable`",
                value=f"{description_label} {disable_desc}\n{usage_label} `/disable #general`",
                inline=False
            )
            
            embed.add_field(
                name="`/exclude add`",
                value=f"{description_label} {exclude_add_desc}\n{usage_label} `/exclude add https://discord.com/channels/...`",
                inline=False
            )
            
            embed.add_field(
                name="`/exclude remove`",
                value=f"{description_label} {exclude_remove_desc}\n{usage_label} `/exclude remove https://discord.com/channels/...`",
                inline=False
            )
            
            embed.add_field(
                name="`/settings`",
                value=f"{description_label} {settings_desc}\n{usage_label} `/settings` {or_text} `/settings #general`",
                inline=False
            )
            
            embed.add_field(
                name="`/clear`",
                value=f"{description_label} {clear_desc}\n{usage_label} `/clear #general`",
                inline=False
            )
            
            embed.add_field(
                name="`/schedule`",
                value=f"{description_label} {schedule_desc}\n{usage_label} `/schedule 5min`",
                inline=False
            )
            
            # 注意事項を追加
            embed.add_field(
                name="",
                value=f"{permission_note}\n\n{lifetime_examples}",
                inline=False
            )
            
            # フッターを追加
            embed.set_footer(text=footer_text)
            
            await interaction.followup.send(embed=embed, ephemeral=True)
            
        except Exception as e:
            print(f"HelpCog.execute error: {e}")
            await interaction.followup.send(error_message, ephemeral=True)


async def setup(bot: commands.Bot):
    await bot.add_cog(HelpCog(bot))
