-- Migration 007: permission moderasi ulasan
INSERT INTO permissions (name) VALUES
    ('review:moderate')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name) VALUES
    ('admin', 'review:moderate')
ON CONFLICT DO NOTHING;
