-- 1. Remove the simple role column from users
ALTER TABLE users DROP COLUMN IF EXISTS role;

-- 2. Create Permissions Table
CREATE TABLE permissions (
    id CHAR(26) PRIMARY KEY,
    code VARCHAR(100) UNIQUE NOT NULL,
    description VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 3. Create Roles Table
CREATE TABLE roles (
    id CHAR(26) PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description VARCHAR(255),
    is_system BOOLEAN DEFAULT false, -- Protects core roles like 'admin' from being deleted
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 4. Create Role_Permissions Pivot Table
CREATE TABLE role_permissions (
    role_id CHAR(26) REFERENCES roles(id) ON DELETE CASCADE,
    permission_id CHAR(26) REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (role_id, permission_id)
);

-- 5. Create User_Roles Pivot Table
CREATE TABLE user_roles (
    user_id CHAR(26) REFERENCES users(id) ON DELETE CASCADE,
    role_id CHAR(26) REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, role_id)
);

-- =========================================================================
-- SEED DATA: Default roles and permissions for every new organization
-- =========================================================================

-- Seed basic system permissions
INSERT INTO permissions (id, code, description) VALUES
('01JW58TAT9M0N4R8M1P3Q6R9Y0', 'users:read', 'Ver lista de usuarios'),
('01JW58TAT9M0N4R8M1P3Q6R9Y1', 'users:write', 'Invitar y editar usuarios'),
('01JW58TAT9M0N4R8M1P3Q6R9Y2', 'roles:read', 'Ver roles y permisos'),
('01JW58TAT9M0N4R8M1P3Q6R9Y3', 'roles:write', 'Crear y editar roles personalizados'),
('01JW58TAT9M0N4R8M1P3Q6R9Y4', 'settings:write', 'Modificar configuración de la organización');

-- Seed the system default 'admin' role
INSERT INTO roles (id, name, description, is_system) VALUES
('01JW58TAT9M0N4R8M1P3Q6R9YA', 'admin', 'Administrador del sistema con acceso total. No puede ser eliminado.', true);

-- Assign ALL permissions to the admin role automatically
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'admin';
