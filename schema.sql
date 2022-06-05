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

INSERT INTO users (id,email,username) VALUES
      (1,'youssef@youssef.com' , 'youssef'),
      (2 , 'mamdouh@mamdouh.com' , 'Ahmed');