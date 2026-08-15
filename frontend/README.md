# Chartmux frontend

React gallery for the real Chartmux package output. Every preview in `public/demos` is emitted by the terminal renderer, so the website stays visually faithful to the CLI instead of redrawing charts in the browser.

```bash
npm install
npm run dev
```

Regenerate the checked-in ANSI previews after changing the renderer or built-in demo data:

```bash
npm run generate:demos
```

Vite prints the local URL, normally `http://localhost:5173`.

## Production checks

```bash
npm run check
npm run build
npm run preview
npm run deploy:dry
```

## Deploy to Cloudflare Workers

Authenticate once, then deploy the static application and its SPA routing configuration:

```bash
npx wrangler login
npm run deploy
```

The official Cloudflare Vite plugin creates the final assets configuration during the Vite build. `wrangler.jsonc` only owns the project name, compatibility settings, and SPA fallback behavior.
