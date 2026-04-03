
insert into admin.users ("id", "name", "email", "password", "status") values ($1, $2, $3, crypt($4, gen_salt('bf')), $5);

