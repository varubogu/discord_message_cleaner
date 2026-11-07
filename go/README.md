# Discordメッセージクリーナー（Go版）

設定したチャンネルのメッセージを、指定した期間（ライフタイム）が経過したら自動で削除する Discord Bot です。
Python 版（discord.py）を Go + [disgo](https://github.com/disgoorg/disgo) で作り直したもので、コマンドの使い方は Python 版と同じです。

## Python版からの主な変更点

| | Python版 | Go版 |
|---|---|---|
| 削除対象の検出 | 10秒ごとに各チャンネルの履歴を API で取得して判定 | **メッセージ投稿イベントを受けて ID・発言日時・削除予定日時を DB に記録** |
| 削除のタイミング | チャンネルを順番に巡回するため遅延が大きい | **DB を毎秒確認し、期限を過ぎたものをすぐ削除** |
| 削除方法 | 1件ずつ削除（1秒間隔） | **14日以内のものは一括削除 API で最大100件/リクエスト** |
| ライフタイム変更時 | （履歴を再取得して判定） | **メッセージを再取得せず、DB の削除予定日時のみ再計算** |
| 他者による削除 | 不明（次回巡回時に判明） | 削除イベントを受けて記録を消し、無駄なリクエストを送らない |
| 除外設定の表示 | 除外メッセージを1件ずつ API で取得 | URL を ID から組み立てるので API リクエスト不要 |

## 動作の仕組み

```text
 Gateway イベント                        DB (PostgreSQL)                    Discord REST API
┌──────────────────┐   まとめて INSERT   ┌──────────────────────┐
│ MESSAGE_CREATE   │ ─────────────────▶ │ tracked_messages     │
│ (監視チャンネル)   │   (250ms / 500件)  │  message_id          │
├──────────────────┤                    │  posted_at  発言日時   │
│ MESSAGE_DELETE   │ ─────────────────▶ │  delete_at  削除予定   │
│ (他者が削除)       │   まとめて DELETE  │  next_attempt_at     │
└──────────────────┘                    └─────────┬────────────┘
                                                  │ 毎秒: 期限切れを最大1000件リース取得
                                                  ▼
                                   ┌───────────────────────────────┐
                                   │ チャンネルごとのキュー/ワーカー     │
                                   │  ・14日以内: 一括削除(≤100件)     │ ──▶ POST /messages/bulk-delete
                                   │  ・14日超  : 1件ずつ削除          │ ──▶ DELETE /messages/{id}
                                   └───────────────────────────────┘
```

- メッセージ本文は読み取りません（特権インテント `MESSAGE_CONTENT` は不要）。必要なのは `GUILDS` と `GUILD_MESSAGES` のみです。
- 発言日時はメッセージ ID（Snowflake）から算出するため、メッセージを取得する必要はありません。
- 監視対象かどうかの判定はメモリ上で行うため、監視対象外チャンネルの投稿で DB にアクセスすることはありません。

### 大量削除とレートリミット

同じ時刻に大量のメッセージが削除予定になっても（例: ライフタイムを短く変更した直後、`/clear` 実行時、長時間停止後の再起動時）、
以下の仕組みでレートリミットに抵触せずに処理します。

1. **一括削除 API**: 投稿から14日以内のメッセージは100件を1リクエストで削除します。10,000件でも約100リクエストです。
2. **チャンネル単位の直列化**: Discord のレートリミットはチャンネル単位のバケットで管理されるため、1チャンネルにつき1ワーカーで順番に送ります。
3. **同時実行数の上限**: 同時にリクエストを送るチャンネル数を `DELETE_WORKERS` で制限します。
4. **グローバル上限の事前制御**: Bot 全体のリクエストをトークンバケット（既定 40 req/s）で平準化し、Discord のグローバル上限（50 req/s）を超えないようにします。
5. **バケット追従と再送**: disgo のレートリミッタがレスポンスヘッダに従って待機し、万一 429 を受けた場合も `Retry-After` 後に再送します。
6. **キュー上限とリース**: DB から取り出す量をチャンネルごとのキュー上限で抑えるため、メモリが溢れることはありません。処理しきれない分は DB に残り、順次処理されます。取り出したメッセージにはリース（既定2分）を設定し、二重処理を防ぎます。
7. **無効リクエストの抑制**: 権限不足（403）などチャンネル単位のエラーが出たら、そのチャンネルの削除を一定時間保留します（Discord には「10分間に1万回の無効リクエストで一時 BAN」という制限があるため）。規定回数連続したら監視を解除します。
8. **一時エラーの再試行**: 5xx やネットワークエラーは指数バックオフで再試行し、上限回数を超えたものは諦めます。

テスト（`internal/deleter/integration_test.go`）では、5チャンネル計 10,025 件が同時に期限切れになった状態から、126 リクエスト・約3秒で全件削除できることを確認しています。

### 停止中のメッセージについて

Bot が停止・切断している間の投稿はイベントで受け取れないため、接続時に各監視チャンネルの「同期済み位置」以降のメッセージを API で取得して補完します（新着がなければ1チャンネル1リクエスト）。
Gateway のセッション再開（RESUME）に成功した場合は Discord がイベントを再送するため、補完は行いません。

## 機能一覧

コマンドは全て「message_cleaner_admin」ロールを持つ人のみ実行できます（`/help` を除く）。

| コマンド | 説明 |
|---|---|
| `/enable channel [lifetime]` | 監視対象のチャンネルを設定します（確認あり）。既に設定済みの場合はライフタイムを変更します。 |
| `/disable channel` | チャンネルを監視対象から解除します。 |
| `/exclusion_add message_url` | メッセージを削除対象から除外します。 |
| `/exclusion_remove message_url` | 除外設定を解除します（確認あり）。 |
| `/settings [channel]` | サーバー内（またはチャンネル）の設定と削除予定件数を表示します。 |
| `/clear channel` | チャンネル内のメッセージ（除外したものを除く）を削除します（確認あり）。 |
| `/help` | 使い方を表示します。 |

### ライフタイムの書式

| 入力文字 | 短縮表記 | 期間 |
| --- | --- | --- |
| 1week | 1w | 1週間 |
| 1day | 1d | 1日 (24時間) |
| 1hour | 1h | 1時間 |
| 1min | 1mi | 1分 |
| 1sec | 1s | 1秒 |

`"1week 2day 3hour 4min 5sec"`、`"1 day 1 hour 30 min"`、`"90mi"`、`"1h90mi"` のように組み合わせられます。省略時は `1day` です。
Python 版と異なり、解釈できない文字が含まれている場合はエラーになります。

### `/enable` で監視を始めたとき

`BACKFILL_ON_ENABLE=true`（既定）の場合、チャンネルの既存メッセージを1度だけ取得して DB に登録し、期限切れのものから削除します。
`false` の場合は監視開始以降に投稿されたメッセージのみが対象になります。

### `/clear` について

チャンネルのメッセージを最大 `CLEAR_MAX_MESSAGES` 件（既定1000件）取得し、即時削除予定として登録します。実際の削除はスケジューラが一括削除でまとめて行います。

## Botに必要な権限

| 権限名 | 用途 |
|---|---|
| チャンネルを見る | 監視対象チャンネルのイベントを受け取るため |
| メッセージを管理 | メッセージを削除するため |
| メッセージ履歴を読む | 既存メッセージの取り込み、停止中の投稿の補完、`/clear` のため |

`/enable`・`/clear` 実行時に Bot の権限が不足している場合はエラーになります。

## 収集するデータ

| 収集対象 | 収集内容 |
|---|---|
| サーバーID、チャンネルID | 削除対象のサーバー、チャンネルを識別するため |
| メッセージID | 削除予定の管理・除外設定のため（発言日時は ID から算出。メッセージの内容やリアクションは読み取りません） |

削除したメッセージの記録は削除後に消去します。チャンネルが削除された場合や Bot がサーバーから退出した場合は、関連する設定を全て消去します。

## 設定（環境変数）

`config/.env.example` と `config/.env.db.production.example` をコピーして編集してください。主な項目は以下の通りです。

| 変数 | 既定値 | 説明 |
|---|---|---|
| `DISCORD_TOKEN` | （必須） | Bot トークン |
| `DATABASE_URL` / `DBHOST` 等 | （必須） | DB 接続情報（`DBHOST`,`DBPORT`,`DBUSER`,`DBPASSWORD`,`DBDATABASE` は Python 版と互換） |
| `MESSAGE_LANGUAGE` / `LOG_LANGUAGE` | `en` | `ja` / `en` |
| `SCHEDULER_INTERVAL` | `1s` | DB を確認する周期 |
| `API_RATE_PER_SECOND` | `40` | Bot 全体の REST リクエスト上限 |
| `DELETE_WORKERS` | `8` | 同時にリクエストを送るチャンネル数 |
| `CHANNEL_FAILURE_LIMIT` | `3` | チャンネル単位のエラーが何回連続したら監視を解除するか（Python 版の `MESSAGE_DELETE_FAILURES`） |
| `BACKFILL_ON_ENABLE` | `true` | 監視開始時に既存メッセージを取り込むか |
| `CLEAR_MAX_MESSAGES` | `1000` | `/clear` で1回に予約する最大件数 |
| `DEV_GUILD_ID` | | 開発用。指定したサーバーにのみコマンドを登録（即時反映） |

その他の項目は `config/.env.example` を参照してください。

## 開発

### 必要なもの

- Go 1.26 以上
- PostgreSQL（Docker で起動する場合は不要）

### 実行

```bash
cp config/.env.example config/.env
cp config/.env.db.production.example config/.env.db.production
# 編集後
set -a; source config/.env; source config/.env.db.production; set +a
go run ./cmd/bot
```

テーブルは起動時に自動作成されます（`internal/store/migrations`）。

### テスト

```bash
go test ./...

# DB を使うテストも実行する場合
TEST_DATABASE_URL="postgres://postgres:postgres@127.0.0.1:5432/dmc_test?sslmode=disable" go test -race -p 1 ./...
```

### ディレクトリ構成

```text
cmd/bot/              エントリポイント
internal/bot/         Gateway 接続・イベント受信・スラッシュコマンド・確認ダイアログ
internal/tracker/     投稿/削除イベントを DB へまとめて書き込む
internal/deleter/     毎秒のスケジューラとチャンネルごとの削除ワーカー
internal/syncer/      起動時・監視開始時の履歴補完
internal/discordapi/  レート制御付き REST クライアントとエラー分類
internal/store/       PostgreSQL アクセスとマイグレーション
internal/registry/    監視チャンネル設定のメモリキャッシュ
internal/lifetime/    ライフタイム文字列の解析・表示
internal/discordurl/  メッセージURLの解析
internal/i18n/        messages.json（Python 版と同一形式）
```

## 本番環境

```bash
docker compose --env-file config/.env.db.production up -d --build
```

### Python版からの移行

Go 版は Python 版と異なるテーブル名を使うため、同じ DB を指定しても衝突しません。設定を引き継ぐ場合は以下の手順です。

1. Python 版を停止する
2. Go 版を Python 版と同じ DB で1度起動する（テーブルが作成される）
3. `psql "$DATABASE_URL" -f scripts/import_from_python.sql` を実行する
4. Go 版を再起動する（取り込んだチャンネルは履歴を先頭から同期し、既存メッセージも削除対象として登録する）

## 注意事項

- 単一プロセスでの稼働を前提にしています（監視チャンネル設定をメモリに保持するため）。
- シャーディングには対応していません（Python 版と同様）。

## 使用ライブラリとライセンス

| ライブラリ | 用途 | ライセンス |
|---|---|---|
| [disgo](https://github.com/disgoorg/disgo) | Discord API ラッパー | Apache License 2.0 |
| [pgx](https://github.com/jackc/pgx) | PostgreSQL ドライバ | MIT License |
| [golang.org/x/time](https://pkg.go.dev/golang.org/x/time/rate) | トークンバケット | BSD-3-Clause |
