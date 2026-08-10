# Chartmux frontend

Full-screen terminal-style chart gallery built with React, Vite, Recharts, and the Cloudflare Vite plugin. The interface mirrors the demos available through the Chartmux Go CLI while using responsive SVG previews in the browser.

```bash
npm install
npm run dev
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
