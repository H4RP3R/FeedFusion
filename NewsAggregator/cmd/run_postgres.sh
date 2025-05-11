#!/bin/bash
mkdir -p db_init
cp schema.sql db_init/init.sql
docker run -d --rm \
    -p 5432:5432 \
    --name news_postgres_db \
    -e POSTGRES_PASSWORD=${POSTGRES_PASSWORD} \
    -e POSTGRES_HOST=${POSTGRES_HOST} \
    -e POSTGRES_PORT=${POSTGRES_PORT} \
    -v $(pwd)/db_init:/docker-entrypoint-initdb.d \
    postgres