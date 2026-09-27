import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: { port: 5173 },
  build: {
    rolldownOptions: {
      output: {
        codeSplitting: {
          groups: [
            { name: 'react-vendor', test: /node_modules[\\/](react|react-dom|scheduler)[\\/]/, priority: 30 },
            { name: 'ant-design', test: /node_modules[\\/](antd|@ant-design)[\\/]/, maxSize: 280_000, priority: 20 },
            { name: 'vendor', test: /node_modules/, maxSize: 280_000, priority: 10 },
          ],
        },
      },
    },
  },
})
