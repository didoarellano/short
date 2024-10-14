INSERT INTO subscriptions
  (name, max_links_per_month, can_customise_path, can_create_duplicates)
VALUES
  ('basic',  12, FALSE, FALSE),
  ('pro1',  120, FALSE, TRUE),
  ('pro2', 1200, TRUE,  TRUE);
