# Development instructions

## Changelog

- Record every user-visible change in `CHANGELOG.md` under `Unreleased`.
- Update the changelog automatically without waiting for an explicit user request.
- Use the Keep a Changelog categories: Added, Changed, Fixed, Removed, Deprecated, and Security.
- Describe changes from the user's perspective in concise English.
- Do not include generated files, formatting-only changes, tests alone, or internal refactoring unless they affect users.

## Version updates

When updating the application version:

1. Review all user-visible changes since the previous version.
2. Move the relevant entries from `Unreleased` into a new version section.
3. Use the heading format `## [x.y.z] - YYYY-MM-DD`.
4. Leave an empty `## [Unreleased]` section at the top.
5. Update documentation that identifies the current release when necessary.
   Do not update the version number shown in README command-output examples,
   including the `-trace` output example, solely because the application version changed.
6. Run the JavaScript syntax check and all Go tests.
7. Rebuild the executable and verify its reported version.
8. Include the version change and `CHANGELOG.md` in the same commit when a commit is requested.

## Verification

- Whenever source or embedded web files that affect the executable are changed,
  run the applicable syntax checks and tests, then rebuild the executable.
- Do this automatically without waiting for an explicit build request.
- Check JavaScript with `node --check web/app.js`.
- Run Go tests with `go test ./...`.
- Build with `go build -buildvcs=false -o seicho .`.
