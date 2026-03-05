# Web Dashboard — `apps/web/`

## Stack
React 19, TypeScript ~5.9, Vite 7, Tailwind CSS v4, TanStack Query v5, Zustand 5, React Router v7

## Directory Structure
```
src/
  pages/             # Route-level components (Dashboard, Posts, Trending, Generate, etc.)
    onboarding/      # Multi-step onboarding wizard
  components/
    layout/          # RootLayout, PlatformLayout, Sidebar, TopBar, PlatformSwitcher
    shared/          # Reusable UI: PostCard, ContentCard, EmptyState, LoadingSpinner, etc.
    generate/        # Content generation feature components
    posts/           # My Posts feature components
    trending/        # Trending discovery feature components
    publish/         # Publishing & scheduling components
    history/         # Content history components
    autopilot/       # Daemon automation components
    repo/            # GitHub repository components
  hooks/             # Custom hooks (TanStack Query wrappers + SSE + extension bridge)
  lib/
    api.ts           # API client with fetch wrapper + SSE support
    types.ts         # TypeScript interfaces for all data models
    platforms.ts     # Platform definitions
    format.ts        # Formatting utilities (relative time, counts)
  stores/
    platform-store.ts  # Active platform + CSS variable setter
    ui-store.ts        # Sidebar state, workflow step
  styles/
    globals.css        # Tailwind + theme imports
    themes/x.css       # X dark theme CSS variables
    themes/linkedin.css # LinkedIn light theme CSS variables
```

## State Management
- **TanStack Query** — all server state (posts, trending, persona, config, history). Query keys: `['posts', platform]`, `['trending', filters]`, etc. Default `staleTime: 30_000`, `retry: 1`.
- **Zustand** — UI-only state: `usePlatformStore` (active platform), `useUIStore` (sidebar, workflow step)
- **Local state** (`useState`) — form inputs, modals, temporary UI

## API Client (`lib/api.ts`)
```typescript
apiClient.get<T>(path, params?)   // GET with query params
apiClient.post<T>(path, body?)    // POST with JSON body
apiClient.patch<T>(path, body?)   // PATCH
apiClient.delete(path)            // DELETE
apiClient.sse(path, body, onEvent) // SSE for long-running ops, returns { cancel }
```
- `getUserID()` generates/retrieves UUID from localStorage (`goviral_user_id`)
- All requests include `X-User-ID` header and `Content-Type: application/json`
- Errors wrapped in `ApiError` class with status and body

## SSE Hook (`hooks/useSSE.ts`)
```typescript
const { mutate, cancel, progress, isRunning, result, error } = useSSEMutation<T>(endpoint)
```
- **Event types**: `"progress"` / `"warning"` update progress state, `"complete"` sets result, `"error"` sets error
- Uses `ReadableStream` with text decoder to parse `data: {...}\n` JSON events
- Used by: `useGenerate`, `useFetchPosts`, `useTrending` (discover), `usePersona` (build)

## Platform Switching
- **URL-based routes**: `/:platform/page` (e.g., `/x/dashboard`, `/linkedin/trending`)
- **`usePlatformParam()`** — extracts platform from URL params
- **`PlatformLayout`** — validates platform, redirects if invalid
- **`PlatformSwitcher`** — tab UI for switching platforms while preserving page
- Platform change sets `document.documentElement.dataset.platform` for CSS variable switching

## Styling: CSS Variables + Tailwind
All colors use CSS variables for platform theming:
```css
/* Usage in components */
bg-[var(--color-bg)]  text-[var(--color-text)]  border-[var(--color-border)]
rounded-[var(--radius-card)]  rounded-[var(--radius-button)]
```
- **X theme** (dark): bg `#000000`, cards `#16181c`, accent `#1d9bf0`
- **LinkedIn theme** (light): bg `#f3f2ef`, cards `#ffffff`, accent `#0a66c2`
- Theme applied via `[data-platform="x"]` / `[data-platform="linkedin"]` CSS selectors

## Extension Bridge
- **`useExtensionCookies`** — postMessage to Chrome extension for X/LinkedIn cookie extraction (10s timeout)
- **`useExtensionLinkedIn`** — fetches LinkedIn posts/feed/trending via extension bridge (120s timeout), ingests via API

## Image Fetching
Raw `fetch()` calls for images MUST include `X-User-ID` header:
```typescript
fetch(`${BASE_URL}/content/${id}/image`, {
  headers: { 'X-User-ID': getUserID() }
}).then(res => res.blob()).then(blob => URL.createObjectURL(blob))
```
Cleanup: `URL.revokeObjectURL(blobUrl)` on unmount.

## Component Conventions
- **Feature folders**: components organized by feature within `src/components/`
- **PascalCase** for component files and exports
- **Shared** reusable components in `components/shared/`
- **Layout** wrappers in `components/layout/`
- **Hooks** wrap TanStack Query/Zustand with domain-specific API; mutations invalidate query keys on success

## Build Config
- Vite proxies `/api` to `http://localhost:8080` in dev
- Path alias: `@/*` resolves to `./src/*`
- TypeScript strict mode enabled
