-- 1. Remove the simple role column from users
ALTER TABLE users DROP COLUMN IF EXISTS role;

-- 2. Create Permissions Table
CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(100) UNIQUE NOT NULL,
    description VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 3. Create Roles Table
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description VARCHAR(255),
    is_system BOOLEAN DEFAULT false, -- Protects core roles like 'admin' from being deleted
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 4. Create Role_Permissions Pivot Table
CREATE TABLE role_permissions (
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (role_id, permission_id)
);

-- 5. Create User_Roles Pivot Table
CREATE TABLE user_roles (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, role_id)
);

-- =========================================================================
-- SEED DATA: Default roles and permissions for every new organization
-- =========================================================================

-- Seed basic system permissions
INSERT INTO permissions (code, description) VALUES
('users:read', 'Ver lista de usuarios'),
('users:write', 'Invitar y editar usuarios'),
('roles:read', 'Ver roles y permisos'),
('roles:write', 'Crear y editar roles personalizados'),
('settings:write', 'Modificar configuración de la organización');

-- Seed the system default 'admin' role
INSERT INTO roles (name, description, is_system) VALUES
('admin', 'Administrador del sistema con acceso total. No puede ser eliminado.', true);

-- Assign ALL permissions to the admin role automatically
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'admin';
