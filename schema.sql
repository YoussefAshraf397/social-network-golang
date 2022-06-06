DROP DATABASE IF EXISTS socialnetwork CASCADE;
CREATE DATABASE IF NOT EXISTS socialnetwork;
SET DATABASE = socialnetwork;

CREATE TABLE IF NOT EXISTS users (
    id SERIAL NOT NULL PRIMARY KEY ,
    email VARCHAR NOT NULL UNIQUE ,
    username VARCHAR NOT NULL UNIQUE,
    avatar VARCHAR,
    followers_count INT NOT NULL DEFAULT 0 CHECK (followers_count >= 0),
    followees_count INT NOT NULL DEFAULT 0 CHECK (followees_count >= 0)
);

CREATE TABLE IF NOT EXISTS follows (
   follower_id INT NOT NULL REFERENCES users ON DELETE CASCADE,
   followee_id INT NOT NULL REFERENCES users ON DELETE CASCADE,
   PRIMARY KEY (follower_id, followee_id)
);


CREATE TABLE IF NOT EXISTS posts (
    id SERIAL NOT NULL PRIMARY KEY ,
    user_id INT NOT NULL REFERENCES users ,
    content VARCHAR NOT NULL ,
    spoiler_of VARCHAR ,
    nsfw BOOLEAN ,
    likes_count INT NOT NULL DEFAULT 0 CHECK (likes_count >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS sorted_posts ON posts (created_at DESC);

CREATE TABLE IF NOT EXISTS timeline (
    id SERIAL NOT NULL PRIMARY KEY ,
    user_id INT NOT NULL REFERENCES users,
    post_id INT NOT NULL REFERENCES posts
);

CREATE UNIQUE INDEX IF NOT EXISTS timeline_unique ON timeline (user_id, post_id) ;


CREATE TABLE IF NOT EXISTS post_likes (
    user_id INT NOT NULL REFERENCES users,
    post_id INT NOT NULL REFERENCES posts,
    PRIMARY KEY (user_id , post_id)
);

INSERT INTO users (id,email,username) VALUES
      (1,'youssef@youssef.com' , 'youssef'),
      (2 , 'mamdouh@mamdouh.com' , 'Ahmed');


INSERT INTO posts (id,user_id,content) VALUES
     (1,1,'This is firstpost of first user');

INSERT INTO timeline (id,user_id,post_id) VALUES
    (1,1,1);


