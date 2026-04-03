-- SQLite non supporta gli schemi, tutto va nel DB principale.

-- DROP TABLE (ordine inverso delle foreign key)
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS confirmations;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS news;
DROP TABLE IF EXISTS roles_privileges;

-- Users
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY NOT NULL,
    email TEXT UNIQUE,
    password TEXT,
    name TEXT,
    status TEXT DEFAULT 'pending',
    last_access TEXT,
    reserved INTEGER DEFAULT 0,
    created TEXT DEFAULT CURRENT_TIMESTAMP,
    updated TEXT DEFAULT CURRENT_TIMESTAMP
);

-- Roles
CREATE TABLE IF NOT EXISTS roles (
    id TEXT,
    label TEXT
);

INSERT INTO roles(id, label) VALUES ('ADMIN', 'Administrator');
INSERT INTO roles(id, label) VALUES ('POWER_USER', 'Power user');
INSERT INTO roles(id, label) VALUES ('USER', 'User');

-- Roles/Privileges
DROP TABLE IF EXISTS roles_privileges;
CREATE TABLE IF NOT EXISTS roles_privileges (
    id_role TEXT,
    id_privilege TEXT
);

INSERT INTO roles_privileges(id_role, id_privilege) VALUES ('ADMIN', 'ek_admin');
INSERT INTO roles_privileges(id_role, id_privilege) VALUES ('POWER_USER', 'ek_access');
INSERT INTO roles_privileges(id_role, id_privilege) VALUES ('POWER_USER', 'ek_read_user');
INSERT INTO roles_privileges(id_role, id_privilege) VALUES ('POWER_USER', 'ek_read_session');
INSERT INTO roles_privileges(id_role, id_privilege) VALUES ('POWER_USER', 'ek_read_metrics');
INSERT INTO roles_privileges(id_role, id_privilege) VALUES ('USER', 'ek_access');

-- User/Roles
CREATE TABLE IF NOT EXISTS user_roles (
    user_id TEXT,
    roles TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Confirmations
CREATE TABLE IF NOT EXISTS confirmations (
    user_id TEXT,
    request TEXT,
    token TEXT,
    created TEXT DEFAULT CURRENT_TIMESTAMP
);

-- Messages
CREATE TABLE IF NOT EXISTS messages (
    id TEXT,
    name TEXT,
    message TEXT,
    created TEXT DEFAULT CURRENT_TIMESTAMP
);

-- News
CREATE TABLE IF NOT EXISTS news (
    id TEXT,
    text TEXT,
    created TEXT DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO news(id, text) VALUES ('1', 'Database initialized');