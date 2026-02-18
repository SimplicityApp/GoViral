# GoViral Web Dashboard

React-based web UI for GoViral. Provides a visual interface for all CLI operations.

## Development

```sh
npm install
npm run dev
```

The Vite dev server starts on `http://localhost:5173` and proxies `/api` requests to `http://localhost:8080` (the Go server).

Make sure the server is running in another terminal:

```sh
go run ./apps/server/
```

## Build

```sh
npm run build
```

Output goes to `dist/`. To embed in the server binary, copy `dist/` to `apps/server/static/` and build with `-tags embedweb`.

## Scripts

| Script | Description |
|--------|-------------|
| `npm run dev` | Start Vite dev server with HMR |
| `npm run build` | Type-check and build for production |
| `npm run lint` | Run ESLint |
| `npm run preview` | Preview production build locally |

## Tech Stack

- **Vite** 7 — bundler and dev server
- **React** 19 — UI framework
- **TypeScript** 5.9 — type safety
- **Tailwind CSS** v4 — styling
- **Zustand** — client state management
- **TanStack Query** — server state and caching
- **React Router** v7 — client-side routing
- **Lucide React** — icons
- **Sonner** — toast notifications

## Pages

| Page | Description |
|------|-------------|
| Dashboard | Overview with key metrics |
| Posts | Browse fetched posts from X and LinkedIn |
| Trending | Discover and filter trending content; repost button on each card |
| Generate | Generate AI content from trending posts; supports Quote Tweet mode for reposts |
| Publish | Publish or schedule generated content; single-tweet preview for quote tweets |
| History | View past generated content with status; REPOST badge on quote tweets |
| Settings | Manage API keys, niches, and server config |

## Theming

The app uses a `data-platform` attribute for platform-aware theming:
- **X**: dark theme
- **LinkedIn**: light theme

## Path Aliases

`@` is aliased to `/src` in both Vite and TypeScript configs, so imports look like:

```ts
import { api } from '@/lib/api'
```
