import { resolve } from 'node:path'

export default {
  plugins: {
    // Absolute config path so Tailwind resolves its config regardless of the
    // process working directory (e.g. when Vite is launched from a parent dir).
    tailwindcss: { config: resolve(import.meta.dirname, 'tailwind.config.js') },
    autoprefixer: {},
  },
}
