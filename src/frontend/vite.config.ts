import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
  plugins: [svelte()],
  server: {
    middlewares: [
      {
        // Skip Vite's SPA fallback for backend API routes
        apply: 'serve',
        enforce: 'pre',
        handle(req, res, next) {
          if (!req.url) {
            next()
            return
          }
          // Don't intercept backend routes - let them 404 through to Wails
          if (req.url.startsWith('/media/')) {
            // Return 404 so request doesn't get SPA fallback treatment
            next()
            return
          }
          next()
        }
      }
    ]
  }
})
