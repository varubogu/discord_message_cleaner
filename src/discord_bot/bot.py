import asyncio
import os

import discord
from discord.ext import commands
from dotenv import load_dotenv

# 環境変数読み込み
dotenv_filepath = os.path.join(os.environ['CONFIG_FOLDER'], '.env')
load_dotenv(override=True, dotenv_path=dotenv_filepath)

# 自分の Bot のアクセストークン
DISCORD_TOKEN = os.environ['DISCORD_TOKEN']

intents = discord.Intents.none()
intents.guilds = True


class DiscordBot(commands.Bot):

    def __init__(self):
        super().__init__(
            command_prefix='/',
            intents=intents
        )

        self.INITIAL_EXTENSIONS = [
            'cogs.clear.channel_clear',
            'cogs.exclusion.add',
            'cogs.exclusion.remove',
            'cogs.monitoring.enable',
            'cogs.monitoring.disable',
            'cogs.schedule.minute_schedule',
            'cogs.settings.show',
        ]
        self.db_lock = asyncio.Lock()

    async def on_ready(self):
        print('bot is online')

    async def load_cogs(self, extensions):
        for extension in extensions:
            try:
                await self.load_extension(extension)
            except Exception as e:
                print(f'{extension}の読み込みでエラー発生 {e}')

    async def setup_hook(self):
        await self.load_cogs(self.INITIAL_EXTENSIONS)
        await self.tree.sync()

    async def close(self):
        await super().close()


async def main():
    from discord_bot import models
    from discord_bot.models.session import engine

    async with engine.begin() as conn:
        await models.init_db(conn)
    bot = DiscordBot()
    await bot.start(DISCORD_TOKEN)


if __name__ == "__main__":
    asyncio.run(main())
