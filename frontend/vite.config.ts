import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
  build: {
    outDir: 'dist',
    rolldownOptions: {
      output: {
        codeSplitting: {
          groups: [
            {
              name: 'zrender',
              test: /node_modules[\\/]zrender[\\/]/,
              priority: 20,
            },
            {
              name: 'echarts',
              test: /node_modules[\\/]echarts[\\/]/,
              priority: 10,
            },
          ],
        },
      },
    },
  },
})
