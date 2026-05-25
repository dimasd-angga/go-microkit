# Contributing

Thanks for taking the time to look — patches and ideas are welcome.

## Ground rules

- **One thing per PR.** Small, focused diffs get merged fast.
- **Tests required for behaviour changes.** Bug fix? Add a regression test.
- **Follow the existing style.** `gofmt`, `goimports`, and `golangci-lint run` should all be clean before you open a PR.
- **Be kind in code review.** This is a side project; we're all here to learn.

## Local setup

```bash
git clone https://github.com/dimasd-angga/go-microkit.git
cd go-microkit
cp .env.example .env
make up
make test
```

If `make up` doesn't work on your machine, that's a bug — please open an issue.

## Project layout

See the README for the full tour. The short version:

- `services/<name>/` — each service has its own `entity/`, `repository/`, `usecase/`, `handler/`.
- `internal/` — shared packages used by every service. **Do not** put service-specific code here.
- `gateway/` — HTTP entrypoint, fans out to services over gRPC.
- `proto/` — gRPC contracts. Regenerate with `make proto` if you edit a `.proto` file.

## Pull request flow

1. Fork and branch from `main`.
2. Make your change. Add a test.
3. Run `make test && make lint`.
4. Open the PR. Describe **what** changed and **why**.
5. Reviews usually happen within a few days.

## Reporting bugs

Include:

- What you ran (the exact `make` target / `curl`).
- What you expected.
- What actually happened (logs!).
- Your OS, Go version, Docker version.

## Asking for features

Open an issue with the `enhancement` label. If you're not sure whether it fits the scope of the kit, ask first — saves you writing code that won't merge.

## License

By contributing you agree your work will be released under the [MIT License](LICENSE).
