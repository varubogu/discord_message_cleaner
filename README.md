# discord_message_cleaner

English | [日本語](README.ja.md)

This is a Discord bot that periodically deletes messages in designated channels.

## Features

- Set target channels for deletion
- Remove target channels from deletion list
- Exclusion settings
- Remove exclusion settings
- View current server settings
- Delete all messages in a channel

## Feature Details

### Set Target Channels for Deletion

Set the channels to be monitored for deletion. A confirmation message is displayed after the command is executed, and the setting is completed by pressing the approval button within one minute to prevent accidental deletions. If the same channel is set again, the setting will be overwritten. Excluded messages will not be deleted.

```discord
/enable [#target-channel] [lifetime]
```

#### Parameter ① Target Channel

The target channel should be entered in the form of a channel mention (#channel-name). This parameter is required.

#### Parameter ② Lifetime

Specify how long you want to keep messages. Enter the lifetime as follows:

| Input | Duration |
| --- | --- |
| 1month | 1 month |
| 1week | 1 week |
| 1day | 1 day (24 hours) |
| 1hour | 1 hour |
| 1min | 1 minute |
| 1sec | 1 second |

Multiple combinations can also be specified.

| Input | Duration |
| --- | --- |
| "1hour30min" | 1 hour 30 minutes |
| "1day 1hour 30min" | 1 day 1 hour 30 minutes |

This parameter is optional. If omitted, the default is "1day".

### Remove Target Channels from Deletion List

Remove channels from the monitoring list. A confirmation message is displayed after the command is executed, and the setting is completed by pressing the approval button within one minute to prevent accidental deletions.

```discord
/disable {#target-channel}
```

#### Parameter ① Target Channel

The target channel should be entered in the form of a channel mention (#channel-name). This parameter is required.

### Message Exclusion Settings

Specify the message URL or message ID to exclude it from deletion settings.

```discord
/exclude add [msgurl]
```

#### Parameter ① Message URL

Specify the message URL. This parameter is required.

### Remove Message Exclusion

Remove the exclusion settings.

```discord
/exclude remove [msgurl]
```

#### Parameter ① Message URL

Specify the message URL. This parameter is required.

### View Current Server Settings

Display the current settings in the server.

```discord
/settings [channel]
```

#### Parameter ① Display Target Channel

This parameter is optional. If omitted, all settings in the server will be displayed.

### Delete All Messages in a Channel

Delete all messages in the target channel. A confirmation message is displayed after the command is executed, and the deletion is carried out by pressing the approval button within one minute to prevent accidental deletions.

```discord
/clear {#target-channel}
```

#### Parameter ① Target Channel

The target channel should be entered in the form of a channel mention (#channel-name). This parameter is required.

## Required Permissions for the Bot

The bot uses the following permissions:

| Permission Name | Description |
|---|---|
| Manage Messages | Needed to access and delete messages in the target channels |
| Read Message History | Needed if you want to delete messages from before the bot was added |

## Collected Data

The bot collects the following data to operate:

| Collected Item | Content |
|---|---|
| Server ID, Channel ID | Used to identify the target servers and channels |
| Message ID | Used to set exclusions (does not read the content or reactions of the messages) |

## Permissions for Bot Operations

For security reasons, only server administrators can operate the bot.

## Development Environment

### Setting Up the Environment

Using devcontainer, basic settings are included.

1. Clone the source code
2. Prepare environment files in the config folder
   1. .env (copy .env.example)
   2. .env.db.production (copy .env.db.production and edit DB connection information)
   3. .env.testing (copy .env.example)
3. Open in VSCode and select "Reopen in Container"

### Debug Execution

.vscode/launch.json is prepared, so you can run it directly from the "Run" tab.

### Test Execution

Using pytest and pytest-asyncio. These environments are automatically created when the devcontainer is built, so tests can be run from the "Test" tab.

## Production Environment

### Execution

Using docker-compose.
*Planning to migrate to k8s in the future.*

Start production environment

```bash
sh prod.up.sh
```

To run with cache cleared

```bash
sh prod.up.nocache.sh
```

To stop

```bash
sh prod.down.sh
```

## Libraries and Licenses Used

### discord.py

- Python wrapper for the Discord API.
- GitHub: [https://github.com/Rapptz/discord.py](https://github.com/Rapptz/discord.py)
- License: [MIT License](https://github.com/Rapptz/discord.py/blob/master/LICENSE)

### python-dotenv

- Reads key-value pairs from a .env file and sets them as environment variables.
- GitHub: [https://github.com/theskumar/python-dotenv](https://github.com/theskumar/python-dotenv)
- License: [MIT License](https://github.com/theskumar/python-dotenv/blob/main/LICENSE)

### asyncpg

- A fast PostgreSQL database client library for Python/asyncio.
- GitHub: [https://github.com/MagicStack/asyncpg](https://github.com/MagicStack/asyncpg)
- License: [Apache License 2.0](https://github.com/MagicStack/asyncpg/blob/master/LICENSE)

### sqlalchemy

- SQL toolkit and Object-Relational Mapping library for Python.
- GitHub: [https://github.com/sqlalchemy/sqlalchemy](https://github.com/sqlalchemy/sqlalchemy)
- License: [MIT License](https://github.com/sqlalchemy/sqlalchemy/blob/main/LICENSE)

### result

- A simple Rust-like Result type for Python.
- GitHub: [https://github.com/dbrgn/result](https://github.com/dbrgn/result)
- License: [MIT License](https://github.com/dbrgn/result/blob/master/LICENSE)

### pytest

- A mature full-featured Python testing tool.
- GitHub: [https://github.com/pytest-dev/pytest](https://github.com/pytest-dev/pytest)
- License: [MIT License](https://github.com/pytest-dev/pytest/blob/main/LICENSE)

### pytest-asyncio

- A Pytest plugin to support asyncio.
- GitHub: [https://github.com/pytest-dev/pytest-asyncio](https://github.com/pytest-dev/pytest-asyncio)
- License: [Apache License 2.0](https://github.com/pytest-dev/pytest-asyncio/blob/main/LICENSE)

### pytest-mock

- A thin wrapper around the mock package for easier use with Pytest.
- GitHub: [https://github.com/pytest-dev/pytest-mock](https://github.com/pytest-dev/pytest-mock)
- License: [MIT License](https://github.com/pytest-dev/pytest-mock/blob/main/LICENSE)
