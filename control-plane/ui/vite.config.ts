/// <reference types="vitest/config" />
import { defineConfig } from 'vite'

export default defineConfig({
  build: {
    // Builds straight into the Go module so cmd/gateway can `//go:embed`
    // it directly -- one shipped binary, no separate static-file sync step.
    outDir: '../cmd/gateway/uidist',
    // Never inside `root`, so Vite already skips the pre-build wipe by
    // default -- kept explicit so a stray flag can't delete the checked-in
    // `.gitkeep` placeholder cmd/gateway needs to compile before the UI's
    // ever been built. Stale chunks from old builds just sit there, unused
    // and gitignored, until the next real deploy overwrites them.
    emptyOutDir: false,
  },
  test: {
    environment: 'jsdom',
  },
})
