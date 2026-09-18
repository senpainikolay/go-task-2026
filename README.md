# recoTask

Periodically pulls users and projects from the Asana API and writes each
record as its own JSON file under `out/users/<gid>.json` and
`out/projects/<gid>.json`.

## Run

```sh
cp .env.template .env   # fill in the values, see below
go run ./cmd
```

Stop it with Ctrl+C (SIGINT) or SIGTERM — it shuts down gracefully.

## Test

```sh
go test ./...
```

## Config (env vars)

Set in `.env` (see `.env.template`):

| Var | Required | Default | Meaning |
|---|---|---|---|
| `ASANA_API_TOKEN` | yes | - | Asana personal access token |
| `ASANA_API_DEFAULT_WORKSPACE` | yes | - | Workspace GID to query |
| `ASANA_API_LIMITER_MAX_RETRIES` | no | `3` | Max attempts per request before giving up |
| `ASANA_API_MAXIMUM_REQUESTS_PER_MINUTE` | no | `120` | Client-side rate limit |
| `ASANA_API_PAGINATION_LIMIT` | no | `1` | Page size used when listing users/projects | 


## Think of scale—imagine your company has thousands of projects and employees. How do you handle it?

- Multiple Tokens ( Rate-Limited);
- Pagination;
- Add cache and handle objects;

