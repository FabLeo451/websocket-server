		
SELECT
    u.id,
    u.name,
    (u.password = ?) AS password_match,
    GROUP_CONCAT(DISTINCT ur.roles) AS roles,
    GROUP_CONCAT(DISTINCT rp.id_privilege) AS privileges
FROM 
    users u
JOIN 
    user_roles ur ON u.id = ur.user_id
LEFT JOIN 
    roles_privileges rp ON ur.roles = rp.id_role
WHERE 
    LOWER(u.email) = LOWER(?)
    AND u.status = 'enabled'
GROUP BY 
    u.id, u.name;