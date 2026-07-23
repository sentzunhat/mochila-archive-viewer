import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
  plugins: [
    svelte(),
    {
      name: 'media-passthrough',
      apply: 'serve',
      enforce: 'pre',
      configResolved(config) {
        // Store original middlewares
      },
      configureServer(server) {
        return () => {
          // Insert middleware at the very beginning, before SPA fallback
          server.middlewares.use((req, res, next) => {
            // Pass /media/* requests to the backend (Wails ServeHTTP handler)
            if (req.url?.startsWith('/media/')) {
              next()
              return
            }
            next()
          })
        }
      }
    }
  ]
})
