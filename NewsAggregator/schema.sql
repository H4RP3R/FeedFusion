DROP DATABASE IF EXISTS news;
CREATE DATABASE news;

\c news;

DROP TABLE IF EXISTS posts;

CREATE TABLE posts (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    published TIMESTAMP WITH TIME ZONE NOT NULL, -- All posts are converted to UTC before being saved.
    link TEXT NOT NULL
);