#!/bin/bash

# 環境変数ファイルの読み込み
source ./config/.env.db.production

# コンテナが存在するか確認
if [ ! "$(docker ps -q -f name=dev-db)" ]; then
    if [ "$(docker ps -aq -f status=exited -f name=dev-db)" ]; then
        # 停止中のコンテナがある場合は削除
        docker rm dev-db
    fi
    echo "🚀 開発用データベースを起動します..."
    # Dockerイメージのビルド
    docker build -t dev-db-image -f Dockerfile.db .
    # コンテナの起動
    docker run -d \
        --name dev-db \
        -v pgdata:/var/lib/postgresql/data \
        -e POSTGRES_USER=${DBUSER} \
        -e POSTGRES_PASSWORD=${DBPASSWORD} \
        -e POSTGRES_DB=${DBDATABASE} \
        -p 5432:5432 \
        dev-db-image
else
    echo "✨ データベースは既に起動しています"
fi

# データベースの接続確認
echo "🔍 データベースの接続を確認しています..."
until docker exec dev-db pg_isready -U ${DBUSER}
do
    echo "🕐 データベースの準備を待っています..."
    sleep 2
done

echo "✅ データベースの準備が完了しました！"
echo "接続情報:"
echo "Host: localhost"
echo "Port: 5432"
echo "User: ${DBUSER}"
echo "Database: ${DBDATABASE}"
