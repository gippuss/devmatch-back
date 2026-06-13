-- +goose Up
INSERT INTO skills (name)
VALUES
    -- Languages
    ('Go'),
    ('Python'),
    ('JavaScript'),
    ('TypeScript'),
    ('Java'),
    ('Kotlin'),
    ('Swift'),
    ('Rust'),
    ('C++'),
    ('C#'),
    ('PHP'),
    ('Ruby'),
    ('Scala'),
    -- Backend
    ('Node.js'),
    ('Django'),
    ('FastAPI'),
    ('Spring Boot'),
    ('GraphQL'),
    ('REST API'),
    ('gRPC'),
    ('PostgreSQL'),
    ('Redis'),
    ('MongoDB'),
    ('Elasticsearch'),
    ('Kafka'),
    ('RabbitMQ'),
    -- Frontend
    ('React'),
    ('Vue.js'),
    ('Angular'),
    ('Next.js'),
    ('Svelte'),
    ('Tailwind CSS'),
    ('WebGL'),
    ('Three.js'),
    -- Mobile
    ('React Native'),
    ('Flutter'),
    ('iOS'),
    ('Android'),
    -- DevOps & Infra
    ('Docker'),
    ('Kubernetes'),
    ('Terraform'),
    ('AWS'),
    ('GCP'),
    ('Azure'),
    ('Linux'),
    ('Nginx'),
    ('CI/CD'),
    -- Data & AI
    ('Machine Learning'),
    ('Data Analysis'),
    ('PyTorch'),
    ('TensorFlow'),
    ('LLM'),
    ('Computer Vision'),
    -- Design & Product
    ('UI/UX'),
    ('Figma'),
    ('Product Management'),
    -- Other
    ('Testing'),
    ('Blockchain'),
    ('WebAssembly'),
    ('Security'),
    ('Technical Writing')
ON CONFLICT (name) DO NOTHING;

INSERT INTO tags (name, is_system)
VALUES
    -- Domain
    ('fintech',      TRUE),
    ('edtech',       TRUE),
    ('ai',           TRUE),
    ('saas',         TRUE),
    ('marketplace',  TRUE),
    ('social',       TRUE),
    ('gaming',       TRUE),
    ('healthcare',   TRUE),
    ('e-commerce',   TRUE),
    ('devtools',     TRUE),
    ('security',     TRUE),
    ('blockchain',   TRUE),
    ('iot',          TRUE),
    ('ar/vr',        TRUE),
    -- Stage
    ('startup',      TRUE),
    ('mvp',          TRUE),
    ('open-source',  TRUE),
    ('non-profit',   TRUE),
    -- Tech focus
    ('web',          TRUE),
    ('mobile',       TRUE),
    ('backend',      TRUE),
    ('frontend',     TRUE),
    ('fullstack',    TRUE),
    ('data',         TRUE),
    ('ml/ai',        TRUE),
    ('infrastructure', TRUE),
    ('api',          TRUE),
    ('real-time',    TRUE)
ON CONFLICT (name) DO UPDATE SET is_system = TRUE;

-- +goose Down
DELETE FROM tags  WHERE is_system = TRUE;
DELETE FROM skills;
