SELECT 
	u.id,
	u.name,
	(u.password = crypt($1, u.password)) AS password_match,
	STRING_AGG(DISTINCT ur.roles, ', ') AS roles,
	STRING_AGG(DISTINCT rp.id_privilege, ', ') AS privileges
FROM 
	admin.users u
JOIN 
	admin.user_roles ur ON u.id = ur.user_id
LEFT JOIN 
	admin.roles_privileges rp ON ur.roles = rp.id_role
WHERE 
	LOWER(u.email) = LOWER($2)
	AND u.status = 'enabled'
GROUP BY 
	u.id
