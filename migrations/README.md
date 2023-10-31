# Database migrations
DB migrations are managed by the [migrate][] tool.

To apply a migration, run (in the repository root directory):

```
$ migrate -database "postgresql://$user:$pass@localhost:5432/$dbname?sslmode=disable" -path migrations up
```

To create a new migration, use (likewise in the repository root):

```
$ migrate create -ext sql -dir migrations -seq $name
```

Tips:

- Use `_` to separate words in the migration name (for consistency).
- After writing a new migration, run `up`, `down`, and `up` again to test it.
- Wrap everything in `BEGIN` and `COMMIT` to make migrations transactional.
  (That it's not done by default I consider an oversight.)

[migrate]: https://github.com/golang-migrate/migrate/blob/master/GETTING_STARTED.md "golang-migrate/migrate: Getting started"
