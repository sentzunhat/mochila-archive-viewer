import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
  plugins: [svelte()],
  server: {
    middlewares: [
      {
        apply: 'serve',
        enforce: 'pre',
        handle(req, res, next) {
          // Pass /media/* requests directly to backend (don't return SPA index)
          if (req.url?.startsWith('/media/')) {
            return next()
          }
          next()
        }
      }
    ]
  }
})
