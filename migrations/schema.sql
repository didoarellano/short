CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  name TEXT,
  email TEXT UNIQUE NOT NULL,
  oauth_provider TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE subscriptions (
  id SERIAL PRIMARY KEY,
  name TEXT UNIQUE NOT NULL,
  max_links_per_month INT NOT NULL,
  can_customise_path BOOLEAN NOT NULL DEFAULT FALSE,
  can_create_duplicates BOOLEAN NOT NULL DEFAULT FALSE,
  can_view_analytics BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE user_subscriptions (
  user_id INT NOT NULL REFERENCES users(id),
  subscription_id INT NOT NULL REFERENCES subscriptions(id),
  status TEXT NOT NULL CHECK(status IN ('active', 'expired')) DEFAULT 'active',
  start_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
  end_date TIMESTAMP NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE links (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  short_code TEXT UNIQUE NOT NULL,
  destination_url TEXT NOT NULL,
  title  TEXT,
  notes  TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE user_monthly_usage (
  id SERIAL PRIMARY KEY,
  user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  links_created INT NOT NULL DEFAULT 0,
  cycle_start_date DATE NOT NULL,
  cycle_end_date DATE NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX idx_user_monthly_usage_user_id ON user_monthly_usage (user_id);

CREATE TABLE analytics (
  id SERIAL PRIMARY KEY,
  short_code TEXT NOT NULL REFERENCES links(short_code) ON DELETE CASCADE,
  geo_data JSONB,
  user_agent_data JSONB,
  referrer_url TEXT,
  recorded_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_analytics_short_code ON analytics (short_code);
