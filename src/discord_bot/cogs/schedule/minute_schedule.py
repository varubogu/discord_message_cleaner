from datetime import datetime
from discord.ext import commands, tasks
from sqlalchemy.ext.asyncio import AsyncSession
from discord_bot.models.session import AsyncSessionLocal


class MinuteSchedule(commands.Cog):

    def __init__(self, bot: commands.Bot):
        self.bot = bot
        self.channel = None

    async def cog_load(self):
        self.loop.start()

    async def cog_unload(self):
        self.loop.cancel()

    @tasks.loop(seconds=30)
    async def loop(self):
        now = datetime.now()

        if not self.bot.is_ready():
            return

        try:
            async with self.bot.db_lock:
                async with AsyncSessionLocal() as session:
                    await self.inner_loop(session, now)
        except Exception as e:
            print(f'schedule.loopで例外発生:{e}')

    async def inner_loop(
            self,
            session: AsyncSession,
            now: datetime
    ):
        pass




async def setup(bot: commands.Bot):
    await bot.add_cog(MinuteSchedule(bot))
