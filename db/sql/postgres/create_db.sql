
----------------------------------------------------------
-- WARNING: This script should be executed by superuser --
----------------------------------------------------------

CREATE DATABASE ekhoesdb;
CREATE USER ekhoesadmin WITH PASSWORD '{{DB_PASSWORD}}';
GRANT ALL PRIVILEGES ON DATABASE ekhoesdb TO ekhoesadmin;

-- Connect to database
\c ekhoesdb

-- Extension is created inside the current database
CREATE EXTENSION IF NOT EXISTS pgcrypto;
