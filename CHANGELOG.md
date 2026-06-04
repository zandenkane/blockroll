# Changelog

All notable changes to blockroll are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.1.0] - 2026-05-28

### Added
- Privacy engine with four trust tiers (Public, Neighbor, Trusted, Inner Circle).
- Visible() predicate with full tier matrix unit tests.
- CanModify() and EffectiveTier() helper functions in the privacy package.
- TierNames() function for iterating valid tier labels.
- SQLite storage via modernc.org/sqlite (pure Go, no CGO).
- Session cookie authentication with bcrypt password hashing.
- Neighborhood creation and join by code system.
- Skill/resource listings with per listing trust tier visibility.
- Trust tier management (promote/demote neighbors).
- CLI harness for exercising the privacy engine without HTTP.
- CLI `remove-listing` command for deleting listings by title.
- CLI `stats` command for viewing neighborhood member and listing counts.
- GetNeighborhoodByName store method for name based lookups.
- MemberCount and ListingCount store methods for neighborhood stats.
- CleanExpiredSessions store method for session garbage collection.
- Ping store method for health checks.
- Health check endpoint at /health (returns JSON status).
- HTTP handler unit tests (session auth middleware, context extraction).
- Additional store tests (member count, listing count, duplicate join, user listings).
- Additional privacy tests (CanModify, EffectiveTier, TierNames, edge cases).
- Dark mode CSS (follows prefers-color-scheme).
- Focus styles on form inputs for better accessibility.
- Card hover shadow effect.
- Makefile with build, test, vet, fmt, cover, clean, and run targets.
- Server rendered HTML frontend with Go templates embedded in the binary.
- GitHub Actions CI pipeline.
- MIT license.
