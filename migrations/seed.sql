INSERT INTO subscriptions
  (name, max_links_per_month, can_customise_slug, can_create_duplicates, can_view_analytics)
VALUES
  ('basic',  12, FALSE, FALSE, FALSE),
  ('pro1',  120, FALSE, TRUE, TRUE),
  ('pro2', 1200, TRUE,  TRUE, TRUE);
