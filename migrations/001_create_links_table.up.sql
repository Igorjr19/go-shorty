CREATE TABLE IF NOT EXISTS links (
    code VARCHAR(50) PRIMARY KEY,
    original_url TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_links_code ON links(code);
CREATE INDEX idx_links_created_at ON links(created_at);
