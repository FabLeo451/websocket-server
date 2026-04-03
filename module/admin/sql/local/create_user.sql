
insert into ekhoes.users ("id", "name", "email", "password", "status") values (?, ?, ?, crypt(?, gen_salt('bf')), ?);

