-- +migrate Up
CREATE TABLE IF NOT EXISTS message (
    message_id serial PRIMARY KEY,
    author_id integer,
    to_user integer,
    created timestamp with time zone DEFAULT now() NOT NULL,
    is_edited boolean DEFAULT FALSE NOT NULL,
    message_text text NOT NULL
);

-- +migrate Down
DROP TABLE IF EXISTS message;
