from typing import Union

import discord
from discord import Interaction

from discord_bot.ui.dialogs.abstruct_dialog_button import AbstructDialogButton


class CancelButton(AbstructDialogButton):
    def __init__(
            self,
            user: Union[discord.User, discord.Member],
            style: discord.ButtonStyle = discord.ButtonStyle.secondary,
            label: str = "Cancel",
            **kwargs
    ):
        super().__init__(user, style=style, label=label, **kwargs)

    async def execute(self, interaction: Interaction):
        await interaction.response.send_message("キャンセルしました", ephemeral=True)
