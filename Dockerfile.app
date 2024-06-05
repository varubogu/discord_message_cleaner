# base image
FROM python:3.11-slim AS base-image


ENV WORK_FOLDER=/app/bot
ENV CONFIG_FOLDER=${WORK_FOLDER}/config
ENV SOURCE_FOLDER=${WORK_FOLDER}/src
ENV SOURCE_BOT_FOLDER=${SOURCE_FOLDER}/discord_bot

WORKDIR $WORK_FOLDER

COPY requirements.txt ./
COPY src/ ./src
COPY config/ ./config

ENV PYTHONPATH=${SOURCE_FOLDER}:${SOURCE_BOT_FOLDER}

RUN pip install --no-cache-dir -r requirements.txt

RUN echo $PYTHONPATH



# debug image
FROM base-image AS debug-image

# デバッグ用ポート解放
# EXPOSE 5678

# デバッグ待機状態にする
# 未完成
# CMD [ "python", "-m", "debugpy", "--wait-for-client", "--listen", "5678", "-m" "bot" ]



# production image
FROM base-image AS production-image
CMD [ "python", "-u", "-m", "src.discord_bot.bot"]
