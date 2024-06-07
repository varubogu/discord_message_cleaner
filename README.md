# discord_message_cleaner

English | [日本語](README.ja.md)

A Discord bot that periodically deletes messages in designated channels.

## Features

- Set target channels for deletion
- Remove channels from deletion targets
- Set exclusions
- Remove exclusions
- View settings within the server
- Delete all messages in a channel

## Feature Details

### Set Target Channels for Deletion

Set the channels to be monitored. To prevent accidental deletions, a confirmation message is displayed after executing the command. Press the Ok button within 10 seconds to complete the setting. If the same channel is set again, the setting will be overwritten. Note that messages with exclusion settings will not be deleted.

```discord
/enable [#target_channel] [lifetime]
```

#### Parameter ① Target Channel for Deletion

Enter the target channel in the form of a channel mention (#channel_name). This is a required parameter.

#### Parameter ② Lifetime

Specify how long the messages should remain. Enter the lifetime as a string as shown below.

| Input | Abbreviation | Duration |
| --- | --- | --- |

| 1week | 1w| 1 week |
| 1day | 1d | 1 day (24 hours)  |
| 1hour | 1h | 1 hour |
| 1min | 1mi | 1 minute |
| 1sec | 1s | 1 second |

You can also specify multiple combinations.

| Input | Duration |
| --- | --- |
| "1week 2day 3hour 4min 5sec" | 9 days 3 hours 4 minutes 5 seconds |
| "1 day 1 hour 30 min" | 1 day 1 hour 30 minutes |
| "90mi" | 1 hour 30 minutes |
| "1h90mi" | 2 hour 30 minutes |

This is an optional parameter. If omitted, "1day" will be set by default.

### Remove Target Channels for Deletion

Remove channels from the monitoring targets.

```discord
/disable {#target_channel}
```

#### Parameter ① Target Channel for Removal

Enter the target channel in the form of a channel mention (#channel_name). This is a required parameter.

### Set Message Exclusions

Exclude messages from deletion by specifying the message URL or message ID.

```discord
/exclude add [msgurl]
```

#### Parameter ① Message URL

Specify the message URL. This is a required parameter.

### Remove Message Exclusions

Remove the set exclusion settings. To prevent accidental deletions, a confirmation message is displayed after executing the command. Press the Ok button within 10 seconds to complete the setting.

```discord
/exclude remove [msgurl]
```

#### Parameter ① Message URL

Specify the message URL. This is a required parameter.

### View Settings Within the Server

Display the current settings in the server.

```discord
/setttings [channel]
```

#### Parameter ① Channel to Display

This is an optional parameter. If omitted, all settings within the server will be displayed.

### Delete All Messages in a Channel

Delete all messages in the specified channel. To prevent accidental deletions, a confirmation message is displayed after executing the command. Press the Ok button within 10 seconds to complete the setting.

```discord
/clear {#target_channel}
```

#### Parameter ① Target Channel for Deletion

Enter the target channel in the form of a channel mention (#channel_name). This is a required parameter.

## Required Permissions for the Bot

The bot requires the following permissions.

| Permission Name | Description |
| --- | --- |
| Manage Messages | Required to access and delete messages in the target channels |
| Read Message History | Required to delete messages sent before the bot was added |

## Data Collection

The bot collects the following data for its operation.

| Data Target | Data Content |
| --- | --- |
| Server ID, Channel ID | Used to identify the target server and channels |
| Message ID | Used to set exclusions (the content of the messages or reactions are not read) |

## Permissions to Operate the Bot

For security reasons, only users with the role "message_cleaner_admin" are allowed to operate this function. The Bot does not automatically generate this role, so an administrator must create it and assign it only to trusted individuals.

Note: Users with this role will effectively have the authority to delete messages from channels that the Bot can access.

## Development Environment

### Environment Setup

The basic settings are summarized in the devcontainer.

1. Clone the source using git
2. Prepare environment files in the config folder
   1. .env (copy .env.example)
   2. .env.db.production (copy .env.db.production and edit the DB connection information)
   3. .env.testing (copy .env.example)
3. Open with VSCode and select "Reopen in Container"

### Debug Execution

You can run the debugger directly from the "Run" tab as .vscode/launch.json is provided.

### Test Execution

We use pytest and pytest-asyncio. These environments are automatically created during the devcontainer setup, so you can run tests from the "Test" tab.

## Production Environment

### Execution

We use docker-compose. Note: We plan to migrate to k8s in the future.

Start production environment

```bash
sh prod.up.sh
```

To clear cache and start

```bash
sh prod.up.nocache.sh
```

To stop

```bash
sh prod.down.sh
```

## Libraries and Licenses

### discord.py

- A Python wrapper for the Discord API.
- GitHub: [https://github.com/Rapptz/discord.py](https://github.com/Rapptz/discord.py)
- License: [MIT License](https://github.com/Rapptz/discord.py/blob/master/LICENSE)

### python-dotenv

- A library to read key-value pairs from a .env file and set them as environment variables.
- GitHub: [https://github.com/theskumar/python-dotenv](https://github.com/theskumar/python-dotenv)
- License: [MIT License](https://github.com/theskumar/python-dotenv/blob/main/LICENSE)

### asyncpg

- A fast PostgreSQL database client library for Python/asyncio.
- GitHub: [https://github.com/MagicStack/asyncpg](https://github.com/MagicStack/asyncpg)
- License: [Apache License 2.0](https://github.com/MagicStack/asyncpg/blob/master/LICENSE)

### sqlalchemy

- A SQL toolkit and Object Relational Mapper for Python.
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
