CREATE TABLE IF NOT EXISTS roles (
    name VARCHAR(50) PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS permissions (
    name VARCHAR(100) PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_name VARCHAR(50) REFERENCES roles(name) ON DELETE CASCADE,
    permission_name VARCHAR(100) REFERENCES permissions(name) ON DELETE CASCADE,
    PRIMARY KEY (role_name, permission_name)
);

ALTER TABLE students ADD COLUMN owner_id INT REFERENCES users(id) ON DELETE SET NULL;

INSERT INTO roles (name) VALUES ('admin'), ('staff'), ('user') ON CONFLICT DO NOTHING;

INSERT INTO permissions (name) VALUES 
('student:list'), ('student:read:any'), ('student:create'), 
('student:update:any'), ('student:delete') 
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name) VALUES 
('admin', 'student:list'), ('admin', 'student:read:any'), ('admin', 'student:create'), 
('admin', 'student:update:any'), ('admin', 'student:delete'),
('staff', 'student:list'), ('staff', 'student:read:any'), ('staff', 'student:create') 
ON CONFLICT DO NOTHING;