# Contributing to Juli

Thanks for your interest in contributing to Juli!

## Development setup

You need Go 1.25+ and Node.js 20+ installed, plus `tmux` on your system.

```bash
# Backend
go run ./cmd/juli --token dev --port 8080

# Frontend (separate terminal)
cd web && npm install && npm run dev
```

The Nuxt dev server proxies API requests to the Go backend automatically.

## Running tests

```bash
go test ./...           # backend
cd web && npm test      # frontend
```

## Submitting changes

1. Fork the repo and create a branch from `main`.
2. Make your changes. Add tests if applicable.
3. Run the full test suite and make sure it passes.
4. Open a pull request with a clear description of what you changed and why.

## Reporting bugs

Open an issue with:
- What you expected to happen
- What actually happened
- Steps to reproduce
- Your OS, Go version, tmux version, and browser

## Code style

- Go: follow standard `gofmt` formatting.
- TypeScript/Vue: follow the existing patterns in `web/`.
- Keep files under 500 lines when possible.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
