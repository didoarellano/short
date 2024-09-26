# 🩳 Short

## Commands

### Run DB migrations

Make sure `DEV_HOST_DB_URL` and/or `PROD_HOST_DB_URL` vars are in the `.env` file.

#### Development

```
make migrate
```

#### Production

```
make migrate ENV=prod
```
