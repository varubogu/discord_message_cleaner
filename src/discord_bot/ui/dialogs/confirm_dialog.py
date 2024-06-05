from typing import Generic, TypeVar
from typing import Callable, Coroutine, Optional, Union, Any

import discord
from discord import Interaction

TParameter = TypeVar('TParameter')

class ConfirmDialog(discord.ui.View, Generic[TParameter]):
    def __init__(
            self,
            user: Union[discord.User, discord.Member],
            callback_function: Callable[[Interaction, TParameter], Coroutine[Any, Any, None]],
            callback_parameter: TParameter
    ):
        # TParameterの循環参照防止で遅延import
        from discord_bot.ui.dialogs.cancel_button import CancelButton
        from discord_bot.ui.dialogs.continute_button import ContinueButton
        super().__init__()
        self.parent_message: Optional[discord.Message] = None
        self.continue_button = ContinueButton(user, callback_function, callback_parameter)
        self.add_item(self.continue_button)
        self.cancel_button = CancelButton(user)
        self.add_item(self.cancel_button)

    async def on_timeout(self) -> None:
        await self.message_release()
        return await super().on_timeout()

    async def after_interaction(self, interaction: Interaction):
        await self.message_release()

    async def message_release(self):
        if self.parent_message is not None:
            try:
                await self.parent_message.delete()
                self.parent_message = None
            except Exception:
                # 1回削除すればいいのであとは全無視
                pass
