-- Dev seed data. Safe to re-run.

INSERT INTO users (id, email, name, password_hash, role)
VALUES (
    '11111111-1111-1111-1111-111111111111',
    'demo@example.com',
    'Demo User',
    -- bcrypt for "demopass1" (cost 12)
    '$2a$12$Q9wQ3oF1m7n.7uIuD3qPgu5KX5jYj6m6Y1Yy6F4Iu1m8FZ4U1S0EC',
    'user'
)
ON CONFLICT (email) DO NOTHING;
