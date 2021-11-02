# Snapr Changelog
All notable changes to Snapr will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.1] - 2021-11-02
- The flag `--filesystem` is now `--file-system`.
- Failed HTTP requests are now retried for 10 minutes.
- Added a CHANGELOG.md to track changes per release.
- Fixed a `Makefile` issue where the linux build would fail if there was no pre-existing build directory.

## [1.0.0] - 2021-11-01
- The first production release of snapr.

[Unreleased]: https://github.com/ckornie/snapr/compare/v1.0.0...HEAD
[1.0.1]: https://github.com/ckornie/snapr/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/ckornie/snapr/releases/tag/v1.0.0
