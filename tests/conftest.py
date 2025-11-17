import asyncio
import os

import pytest
import pytest_asyncio
from dotenv import load_dotenv
from sqlalchemy.ext.asyncio import AsyncEngine, create_async_engine
from sqlalchemy.ext.asyncio.session import AsyncSession

from discord_bot import models


@pytest.fixture(scope="session", autouse=True)
def auto001_load_env():
    env_path = os.path.join(os.environ['CONFIG_FOLDER'], '.env.testing')
    load_dotenv(override=True, dotenv_path=env_path)


@pytest_asyncio.fixture
async def engine():
    """テストDBエンジンを生成する

    Returns:
        _type_: _description_
    """
    # db init
    DBUSER = os.environ['TEST_DBUSER']
    DBPASSWORD = os.environ['TEST_DBPASSWORD']
    DBHOST = os.environ['TEST_DBHOST']
    DBDATABASE = os.environ['TEST_DBDATABASE']

    URL = f'postgresql+asyncpg://{DBUSER}:{DBPASSWORD}@{DBHOST}/{DBDATABASE}'
    engine = create_async_engine(URL, echo=True)
    try:
        yield engine
    finally:
        # AsyncEngine は非同期で dispose する
        await engine.dispose()


@pytest_asyncio.fixture
async def create_temp_table(engine: AsyncEngine):
    """テストDBにテーブルをセッション単位で作成する。

    個別テストでの DDL（create/drop）が並行して実行されると
    `asyncpg` の "another operation is in progress" エラーを招くため、
    テーブル作成/削除はセッション単位で行い、各テストの後処理は
    データ削除（TRUNCATE）で代替します。
    """
    await models.init_db_from_engine(engine)
    try:
        yield
    finally:
        await models.drop_db_from_engine(engine)


@pytest_asyncio.fixture
async def clear_tables(engine: AsyncEngine):
    """各テスト後にテーブルのデータを削除して状態をリセットするフィクスチャ。"""
    from sqlalchemy import text

    yield

    # モデル定義から全テーブル名を動的に取得し、TRUNCATE
    async with engine.begin() as conn:
        metadata = models.get_metadata()
        table_names = [table.name for table in metadata.sorted_tables]
        if table_names:
            tables_csv = ', '.join(table_names)
            sql = f"TRUNCATE TABLE {tables_csv} RESTART IDENTITY CASCADE;"
            await conn.execute(text(sql))


@pytest_asyncio.fixture
async def async_db_session(engine, create_temp_table, clear_tables):
    """DBセッションを生成する

    Args:
        engine (Engine): テストDBエンジン

    Returns:
        AsyncSession: DBセッション
    """
    async_session = AsyncSession(bind=engine)
    try:
        yield async_session
    finally:
        # 正しいクリーンアップ: AsyncSession を閉じる
        # AsyncSession.close() は非同期コンテキスト内で await 可能な実装のため await する
        await async_session.close()


@pytest.fixture
def event_loop():
    """pytest-asyncioで使用するasyncio.event_loop
        pytest-asyncioの標準のevent_loopを上書きすることで
        関数スコープ→テストセッションスコープへと変更
        これにより全テストでDBセッションが複数回発生しない

    Yields:
        _type_: _description_
    """

    loop = asyncio.new_event_loop()
    try:
        yield loop
    finally:
        loop.close()
