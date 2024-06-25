CREATE TABLE IF NOT EXISTS shops (
  id UUID NOT NULL,
  owner_id UUID NOT NULL,
  name VARCHAR(255) NOT NULL,

  PRIMARY KEY(id),
  FOREIGN KEY(owner_id) REFERENCES users(id) ON DELETE CASCADE,
  UNIQUE(owner_id, name)
);
