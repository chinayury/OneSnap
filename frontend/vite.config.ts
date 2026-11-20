import {defineConfig} from 'vite'
import react from '@vitejs/plugin-react'
import {resolve} from 'path'
import {readFileSync} from 'fs'

// Read version from root package.json (import.meta.dirname: vite 8 native config loader)
const rootPackageJson = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../package.json'), 'utf-8')
)

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  define: {
    __APP_VERSION__: JSON.stringify(rootPackageJson.version)
  }
})
