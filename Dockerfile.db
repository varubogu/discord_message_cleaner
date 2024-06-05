# base image
FROM postgres:latest

ENV ENTRY_FOLDER=docker-entrypoint-initdb.d

COPY ./db/init.sql ${ENTRY_FOLDER}/init.sql

