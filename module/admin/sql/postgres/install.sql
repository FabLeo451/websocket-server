
CREATE SCHEMA IF NOT EXISTS admin AUTHORIZATION ekhoesadmin;
--GRANT ALL PRIVILEGES ON SCHEMA admin TO ekhoesadmin;

DROP TABLE IF EXISTS admin.USER_ROLES;
DROP TABLE IF EXISTS admin.USERS;
DROP TABLE IF EXISTS admin.ROLES;
DROP TABLE IF EXISTS admin.CONFIRMATIONS;
DROP TABLE IF EXISTS admin.MESSAGES;
DROP TABLE IF EXISTS admin.NEWS;

-- Users
CREATE TABLE IF NOT EXISTS admin.users (
	id VARCHAR(100) PRIMARY KEY NOT NULL,
	email VARCHAR(100) UNIQUE,
	password VARCHAR(200),
	name VARCHAR(100),
	status VARCHAR(50) DEFAULT 'pending',
	last_access TIMESTAMP WITH TIME ZONE,
	reserved bool default false,
	created TIMESTAMP DEFAULT NOW(),
	updated TIMESTAMP DEFAULT NOW()
);

-- Roles
CREATE TABLE IF NOT EXISTS admin.ROLES (
	id VARCHAR(20),
	label VARCHAR(50)
);

insert into admin.ROLES("id", "label") values ('ADMIN', 'Administrator');
insert into admin.ROLES("id", "label") values ('POWER_USER', 'Power user');
insert into admin.ROLES("id", "label") values ('USER', 'User');

-- Roles/Privileges
DROP TABLE IF EXISTS admin.ROLES_PRIVILEGES;
CREATE TABLE IF NOT EXISTS admin.ROLES_PRIVILEGES (
	id_role VARCHAR(20),
	id_privilege VARCHAR(20)
);

insert into admin.ROLES_PRIVILEGES("id_role", "id_privilege") values ('ADMIN', 'ek_admin');
insert into admin.ROLES_PRIVILEGES("id_role", "id_privilege") values ('POWER_USER', 'ek_access');
insert into admin.ROLES_PRIVILEGES("id_role", "id_privilege") values ('POWER_USER', 'ek_read_user');
insert into admin.ROLES_PRIVILEGES("id_role", "id_privilege") values ('POWER_USER', 'ek_read_session');
insert into admin.ROLES_PRIVILEGES("id_role", "id_privilege") values ('POWER_USER', 'ek_read_metrics');
insert into admin.ROLES_PRIVILEGES("id_role", "id_privilege") values ('USER', 'ek_access');

-- User/Roles
CREATE TABLE IF NOT EXISTS admin.USER_ROLES (
	user_id VARCHAR(100),
	roles VARCHAR(100),
	
	CONSTRAINT fk_user
		FOREIGN KEY (user_id)
		REFERENCES admin.users(id)
		ON DELETE CASCADE
);

-- Confirmation tokens
CREATE TABLE IF NOT EXISTS admin.CONFIRMATIONS (
	user_id VARCHAR(50),
	request VARCHAR(20),
	token VARCHAR(500),
	created timestamp default now()
);

-- Messages
CREATE TABLE IF NOT EXISTS admin.MESSAGES (
	id VARCHAR(50),
	name VARCHAR(20),
	message VARCHAR(500),
	created timestamp default now()
);

-- News
CREATE TABLE IF NOT EXISTS admin.NEWS (
	id VARCHAR(50),
	text VARCHAR(500),
	created timestamp default now()
);

insert into admin.NEWS("id", "text") values ('1', 'Database initialized');

-- Grants
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA admin TO ekhoesadmin;

