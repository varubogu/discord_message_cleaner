from abc import abstractmethod
from typing import Union


import discord
from discord import Interaction



class AbstructDialogButton(discord.ui.Button):
    def __init__(
            self,
            user: Union[discord.User, discord.Member],
            style: discord.ButtonStyle,
            label: str,
            **kwargs
    ):
        self.user_id = user.id
        super().__init__(style=style, label=label, **kwargs)

    async def callback(self, interaction: Interaction):
        if interaction.user.id != self.user_id:
            return
        await self.execute(interaction)
        await self.after(interaction)

    
    @abstractmethod
    async def execute(self, interaction: Interaction):
        pass

    async def after(self, interaction: Interaction):
        from discord_bot.ui.dialogs.confirm_dialog import ConfirmDialog

        if self.view is ConfirmDialog:
            # 元メッセージ削除
            await self.view.after_interaction()


