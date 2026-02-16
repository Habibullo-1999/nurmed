INSERT INTO permissions (code, module, resource, action)
VALUES ('users.create', 'users', 'user', 'create')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'users.create'
WHERE r.code = 'owner'
ON CONFLICT (role_id, permission_id) DO NOTHING;
