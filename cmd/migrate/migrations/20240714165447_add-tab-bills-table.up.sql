CREATE TABLE IF NOT EXISTS tab_bills (
  shop_id INT NOT NULL,
  tab_id INT NOT NULL,
  id SERIAL NOT NULL,
  start_date DATE NOT NULL,
  end_date DATE NOT NULL,
  is_paid BOOLEAN NOT NULL DEFAULT FALSE,

  PRIMARY KEY(shop_id, tab_id, id),
  FOREIGN KEY(shop_id, tab_id) REFERENCES tabs(shop_id, id) ON DELETE CASCADE
);
