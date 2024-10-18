#!/usr/bin/env bash

sql="
UPDATE user_monthly_usage
SET usage_count = 0,
    cycle_start_date = CURRENT_DATE,
    cycle_end_date = CURRENT_DATE + INTERVAL '1 month',
    updated_at = NOW()
WHERE cycle_end_date <= CURRENT_DATE;
"

psql $DATABASE_URL -c "$sql"
echo "$(date '+%Y-%m-%d %H:%M:%S') - User monthly usage limits reset"
