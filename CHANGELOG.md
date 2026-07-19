# Changelog

All notable changes to Seicho are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.12] - 2026-07-19

### Changed

- Unified the header and sidebar backgrounds with the same soft neutral color.
- Highlighted the Markdown pane heading with a near-white blue-gray background.

## [0.2.11] - 2026-07-18

### Fixed

- Widened the unsaved-changes dialog so its Japanese heading does not wrap awkwardly.

## [0.2.10] - 2026-07-17

### Added

- Added a Markdown-focused Zen Mode that hides navigation, metadata, preview, and destructive actions while retaining Save and Escape-to-exit controls.

### Changed

- Reordered editor actions to Zen Mode, Save, and Delete, with extra separation before Delete to reduce accidental clicks.
- Matched the Zen Mode button dimensions to the Save button.

## [0.2.9] - 2026-07-17

### Changed

- Simplified the site selection dialog by relying on automatic browser-language detection and the main header language control.
- Changed API errors to use stable, language-independent error codes while preserving localized messages and trace diagnostics.
- Reduced the horizontal width of the Save button in the editor toolbar.
- Updated UI and editor font stacks to use native system fonts consistently across Windows, macOS, and Linux.
- Changed post titles in the Contents list to use regular font weight.
- Removed the redundant × icon from the editor toolbar Delete button.

## [0.2.8] - 2026-07-17

### Added

- Added Japanese and English language switching for the interface and API error messages, with the selection remembered in the browser.

### Changed

- Standardized the English new-post action label as "New Post".

## [0.2.7] - 2026-07-17

### Added

- Added a maintained project changelog and repository release workflow guidance.

### Changed

- Unified the header and sidebar with a light paper-inspired color palette.
- Lightened the sidebar background to improve visual hierarchy.

## [0.2.6] - 2026-07-17

### Changed

- Translated the README and command documentation into English.

## [0.2.5] - 2026-07-17

### Changed

- Unified primary and secondary action buttons with a blue-gray palette.
- Made destructive navigation and delete actions visually consistent.
- Improved the unsaved-changes dialog labels and action hierarchy.
- Replaced the always-visible sort selector with an icon-triggered pull-down.
- Reordered the sidebar controls to sort, content heading, and new post.
- Simplified the editor toolbar text.

## [0.2.4] - 2026-07-16

### Added

- Added a custom unsaved-changes confirmation dialog for in-app navigation.

### Changed

- Replaced browser-native confirmation prompts used when switching posts or creating a post.

## [0.2.3] - 2026-07-16

### Added

- Added the `-trace` command-line option.
- Added startup diagnostics for the Seicho version, OS, architecture, Go runtime, PID, listen address, and selected site.
- Added access logs containing a request ID, HTTP method, relative URL, response status, and processing time.
- Added API error details and request IDs to diagnostic logs and responses.
- Added panic recovery with request-scoped stack traces.
- Added detailed logging for file operations and `hugo new` failures.
- Added a custom delete confirmation dialog.

## [0.2.2] - 2026-07-16

### Added

- Added a custom new-post dialog with validation, error display, cancel controls, and Escape-key support.

### Changed

- Replaced the browser-native new-post prompt.

## [0.2.1] - 2026-07-16

### Changed

- Changed the default listen port from `1314` to `1221`.

## [0.2.0] - 2026-07-15

### Added

- Added the Apache License 2.0.

## [0.1.3] - 2026-07-15

### Changed

- Increased the file path input size for better readability.

## [0.1.2] - 2026-07-15

### Changed

- Moved save and delete actions into a sticky editor toolbar.
- Stacked the file path and front matter fields vertically.
- Added a configurable `-port` command-line option.

## [0.1.1] - 2026-07-15

### Added

- Added a close button and Escape-key support to the site selection dialog.

## [0.1.0] - 2026-07-15

### Added

- Added a local browser-based editor for Hugo content files.
- Added Hugo site selection and validation.
- Added post listing, search, and sorting by modification or publication date.
- Added post creation through `hugo new content` with archetype support.
- Added editing for Markdown content and YAML or TOML front matter.
- Added post saving, renaming, and deletion.
- Added conflict detection for files modified outside Seicho.
- Added path validation to prevent access outside the site's `content` directory.
- Added a live Markdown preview using Goldmark, the same parser used by Hugo.
- Added command-line version and usage output.
- Added a Windows executable build workflow.

### Fixed

- Fixed false save conflicts caused by JavaScript integer precision loss.
- Fixed Windows file saving verification and repeated saves of existing posts.
- Fixed preservation and removal of leading blank lines in Markdown content.
- Fixed path edits creating duplicate files instead of renaming the original post.
- Preserved YAML and TOML front matter delimiters when saving.
