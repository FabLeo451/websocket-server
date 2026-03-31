
-- WARNING: This script should be executed by superuser

CREATE DATABASE ekhoesdb;
CREATE USER ekhoesadmin WITH PASSWORD '{{DB_PASSWORD}}';

GRANT ALL PRIVILEGES ON DATABASE ekhoesdb TO ekhoesadmin;
\c ekhoesdb

CREATE EXTENSION IF NOT EXISTS pgcrypto;
