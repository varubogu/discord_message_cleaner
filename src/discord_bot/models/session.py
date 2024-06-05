from sqlalchemy.ext.asyncio import create_async_engine
from sqlalchemy.orm import scoped_session
from sqlalchemy.ext.asyncio.session import AsyncSession

import os
from sqlalchemy.orm import sessionmaker


DBUSER = os.environ['DBUSER']
DBPASSWORD = os.environ['DBPASSWORD']
DBHOST = os.environ['DBHOST']
DBDATABASE = os.environ['DBDATABASE']

DB_URL = f'postgresql+asyncpg://{DBUSER}:{DBPASSWORD}@{DBHOST}/{DBDATABASE}'
engine = create_async_engine(DB_URL, echo=False)
AsyncSessionLocal = scoped_session(
    sessionmaker(
        bind=engine,
        autocommit=False,
        autoflush=False,
        expire_on_commit=False,
        class_=AsyncSession
    )
)
