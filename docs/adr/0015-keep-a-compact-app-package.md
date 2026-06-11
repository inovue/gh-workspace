# Keep a compact app package

`gh-workspace` keeps command routing, use-case orchestration, GitHub CLI integration, and focused selection UI behavior together in `internal/app`, while repository-reference parsing and validation live in `internal/domain`. Interfaces remain at the app boundary only where tests need substitutes.

This replaces the planned multi-package ports-lite layout. The current command surface is small enough that additional adapter and UI packages would add navigation and interface overhead without creating useful ownership boundaries.
