## Summary
<!-- Briefly describe the purpose and content of this Pull Request -->

## Type of Change
- [ ] 🐛 Bug fix (`fix:`)
- [ ] ✨ New feature (`feat:`)
- [ ] ⚡ Performance optimization (`perf:`)
- [ ] ♻️ Code refactoring (`refactor:`)
- [ ] 📝 Documentation (`docs:`)
- [ ] 🔧 Build / CI / Tooling (`chore:`)

## Related Issues
<!-- e.g., Closes #12 -->

## Verification & Testing
- [ ] `go vet ./...` passes
- [ ] `go test -race -count=1 ./internal/meta ./internal/db ./internal/organizer ./internal/source ./internal/sync ./internal/version` passes
- [ ] `go test -count=1 ./internal/ui ./cmd/lumina-photo-sync` passes
- [ ] Tested on Linux / macOS (specify OS & source mode: )

## Standards Checklist
- [ ] Bumped version in `internal/version/version.go`, `Makefile`, and `dist/Info.plist` (if releasing a functional change)
- [ ] Commit messages follow Conventional Commits format in English
- [ ] No private local paths, emails, or credentials committed
