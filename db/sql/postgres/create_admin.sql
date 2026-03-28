
insert into ekhoes.users ("id", "name", "email", "password", "status", "reserved") values ('1000', 'Administrator', $1, crypt('admin', gen_salt('bf')), 'enabled', true);

