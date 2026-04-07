SELECT 
	u.id,
	u.name,
	(u.password = crypt($1, u.password)) AS password_match
FROM 
	hn.users u
WHERE 
	LOWER(u.email) = LOWER($2)
	AND u.status = 'enabled'

