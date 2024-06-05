# Discordメッセージクリーナー

[English](README.md) | 日本語

設定したチャンネルのメッセージを定期的に削除するDiscord Botです。

## 機能一覧

- 削除対象チャンネルの設定
- 削除対象チャンネルの解除
- 除外設定
- 除外解除
- サーバー内設定一覧を確認
- チャンネル内全メッセージ削除

## 機能詳細

### 削除対象チャンネルの設定

監視対象のチャンネルを設定します。
誤って削除しないようにコマンド実行後に確認メッセージを表示し、1分以内に承認ボタンを押すことで設定が完了します。
同じチャンネルに再度設定した場合、設定が上書きされます。
なお、除外設定のあるメッセージは削除されません。

```discord
/enable [#削除対象チャンネル] [ライフタイム]
```

#### パラメータ① 削除対象チャンネル

削除対象チャンネルはチャンネルメンション（#チャンネル名）の形式で入力してください。
入力必須のパラメータです。

#### パラメータ② ライフタイム

メッセージをどれだけの期間残したいかです。
以下のような文字列で入力します。

| 入力文字 | 期間 |
| --- | --- |
| 1month | 1ヶ月 |
| 1week | 1週間 |
| 1day | 1日 (24時間) |
| 1hour | 1時間 |
| 1min | 1分 |
| 1sec | 1秒 |

複数の組み合わせで指定することもできます。

| 入力文字 | 期間 |
| --- | --- |
| "1hour30min" |1時間30分 |
| "1day 1hour 30min" | 1日1時間30分 |

入力任意のパラメータです。
省略した場合、"1day"がデフォルトで設定されます。

### 削除対象チャンネルの解除

チャンネルを監視対象から解除します。
誤って削除しないようにコマンド実行後に確認メッセージを表示し、1分以内に承認ボタンを押すことで設定が完了します。

```discord
/disable {#解除対象チャンネル}
```

#### パラメータ① 解除対象チャンネル

解除対象チャンネルはチャンネルメンション（#チャンネル名）の形式で入力してください。
入力必須のパラメータです。

### メッセージ除外設定

メッセージURL、またはメッセージIDを指定することで削除設定から除外することができます。

```discord
/exclude add [msgurl]
```

#### パラメータ① メッセージURL

メッセージURLを指定します。
入力必須のパラメータです。

### メッセージ除外解除

設定済みの除外設定を解除します。

```discord
/exclude remove [msgurl]
```

#### パラメータ① メッセージURL

メッセージURLを指定します。
入力必須のパラメータです。

### サーバー内設定一覧を確認

現在のサーバーで設定中の設定内容を表示します。

```discord
/setttings [channel]
```

#### パラメータ① 表示対象のチャンネル

入力任意のパラメータです。
省略した場合、サーバー内の全ての設定が表示されます。

### チャンネル内全メッセージ削除

対象したチャンネルのメッセージをすべて削除します。
誤って削除しないようにコマンド実行後に確認メッセージを表示し、1分以内に承認ボタンを押すことで削除が実行されます。

```discord
/clear {#削除対象チャンネル}
```

#### パラメータ① 削除対象チャンネル

削除対象チャンネルはチャンネルメンション（#チャンネル名）の形式で入力してください。
入力必須のパラメータです。

## Botに必要な権限について

このBotは以下の権限を利用します。

| 権限名 | 権限内容 |
|---|---|
| メッセージを管理 | 削除対象のチャンネルにアクセスし、削除するために必要です |
| メッセージ履歴を読む | Bot導入以前のメッセージを削除したい場合に必要です。 |

## 収集するデータについて

このBotの動作のために以下のデータを収集します。

| 収集対象 | 収集内容 |
|---|---|
| サーバーID、チャンネルID | 削除対象のサーバー、チャンネルを識別するために使用します |
| メッセージID | 除外対象を設定するために使用します（メッセージの内容やリアクションの読み取りはしません） |

## Botの操作が可能な権限について

セキュリティの都合上、サーバーの管理者のみ操作ができるようにしています。

## 開発環境

### 環境構築

devcontainerを使用しており、基本的な設定はそこにまとめています。

1. ソースをgit cloneする
2. configフォルダに環境ファイルを用意する
   1. .env（.env.exampleをコピーしてください）
   2. .env.db.production（.env.db.productionをコピーし、DB接続情報を編集してください）
   3. .env.testing（.env.exampleをコピーしてください）
3. VSCodeで開き「コンテナで再度開く」を選択

### デバッグ実行

.vscode/launch.json が作ってあるため、そのまま「実行」タブから実行できます。

### テスト実行

pytest、pytest-asyncioを使用しています。
devcontainer作成の時点でこれらの環境は自動的に作られるため、「テスト」タブから実行可能です。

## 本番環境

### 実行

docker-composeを使用しています。
※将来的にはk8sへ移行する予定

本番環境開始

```bash
sh prod.up.sh
```

キャッシュをクリアして実行したい場合

```bash
sh prod.up.nocache.sh
```

停止する場合

```bash
sh prod.down.sh
```

## 使用ライブラリとライセンスについて

### discord.py

- Discord APIのためのPythonラッパー。
- GitHub: [https://github.com/Rapptz/discord.py](https://github.com/Rapptz/discord.py)
- ライセンス: [MIT License](https://github.com/Rapptz/discord.py/blob/master/LICENSE)

### python-dotenv

- .envファイルからキーと値のペアを読み取り、それらを環境変数として設定するライブラリ。
- GitHub: [https://github.com/theskumar/python-dotenv](https://github.com/theskumar/python-dotenv)
- ライセンス: [MIT License](https://github.com/theskumar/python-dotenv/blob/main/LICENSE)

### asyncpg

- Python/asyncio用の高速なPostgreSQLデータベースクライアントライブラリ。
- GitHub: [https://github.com/MagicStack/asyncpg](https://github.com/MagicStack/asyncpg)
- ライセンス: [Apache License 2.0](https://github.com/MagicStack/asyncpg/blob/master/LICENSE)

### sqlalchemy

- Python用のSQLツールキットおよびオブジェクトリレーショナルマッパー。
- GitHub: [https://github.com/sqlalchemy/sqlalchemy](https://github.com/sqlalchemy/sqlalchemy)
- ライセンス: [MIT License](https://github.com/sqlalchemy/sqlalchemy/blob/main/LICENSE)

### result

- Python用のシンプルなRust風のResult型。
- GitHub: [https://github.com/dbrgn/result](https://github.com/dbrgn/result)
- ライセンス: [MIT License](https://github.com/dbrgn/result/blob/master/LICENSE)

### pytest

- 成熟したフル機能のPythonテストツール。
- GitHub: [https://github.com/pytest-dev/pytest](https://github.com/pytest-dev/pytest)
- ライセンス: [MIT License](https://github.com/pytest-dev/pytest/blob/main/LICENSE)

### pytest-asyncio

- asyncioをサポートするPytestプラグイン。
- GitHub: [https://github.com/pytest-dev/pytest-asyncio](https://github.com/pytest-dev/pytest-asyncio)
- ライセンス: [Apache License 2.0](https://github.com/pytest-dev/pytest-asyncio/blob/main/LICENSE)

### pytest-mock

- Pytestでの使用を容易にするmockパッケージの薄いラッパー。
- GitHub: [https://github.com/pytest-dev/pytest-mock](https://github.com/pytest-dev/pytest-mock)
- ライセンス: [MIT License](https://github.com/pytest-dev/pytest-mock/blob/main/LICENSE)
