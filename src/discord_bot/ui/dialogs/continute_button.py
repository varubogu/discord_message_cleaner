from typing import Any, Callable, Coroutine, Generic, Union

import discord
from discord import Interaction
from discord_bot.ui.dialogs.abstruct_dialog_button import AbstructDialogButton
from discord_bot.ui.dialogs.confirm_dialog import TParameter


class ContinueButton(AbstructDialogButton, Generic[TParameter]):
    def __init__(
            self,
            user: Union[discord.User, discord.Member],
            callback_function: Callable[[Interaction, Any], Coroutine[Any, Any, None]],
            callback_parameter: TParameter,
            style: discord.ButtonStyle = discord.ButtonStyle.primary,
            label: str = "Ok",
            **kwargs
    ):
        self.callback_function = callback_function
        self.callback_parameter = callback_parameter
        super().__init__(user, style=style, label=label, **kwargs)

    async def execute(self, interaction: Interaction):
        try:
            await self.callback_function(interaction, self.callback_parameter)
        except Exception as e:
            print(e)
            await interaction.followup.send("err")
