CREATE USER bot_user WITH PASSWORD 'bot_password';
CREATE DATABASE bot_database WITH OWNER bot_user;
ALTER USER bot_user WITH PASSWORD 'bot_password';
