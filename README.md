# 🩳 Short

## Commands

### Generate DB code

Prerequisites:

- [Install `sqlc`](https://sqlc.dev/)

```
make db
```

### Run DB migrations

Prerequisites:

- [Install `atlas`](https://atlasgo.io/)
- `DEV_HOST_DB_URL` and/or `PROD_HOST_DB_URL` vars in the [`.env`](.env.example) file

#### Development

```
make migrate
```

#### Production

```
make migrate ENV=prod
```
