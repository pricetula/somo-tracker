import { bindings, defineConfig, defineWorker } from "cf/config";

export default defineConfig({
  worker: defineWorker({
    name: "somo-tracker-frontend",
    entrypoint: "vinext/server/fetch-handler",
    compatibilityDate: "2026-11-16",
    assets: {
      htmlHandling: "none",
      notFoundHandling: "none",
      runWorkerFirst: ["/*", "!/_next/static/*", "!/*/_next/static/*"],
    },
    observability: {
      enabled: true,
      logs: { enabled: true, invocationLogs: true },
    },
    env: {
      ASSETS: bindings.assets(),
      IMAGES: bindings.images(),
      NEXT_PUBLIC_TURNSTILE_SITE_KEY: bindings.text("0x4AAAAAAGHleNH8xjjhEUQL"),
      SESSION_COOKIE_NAME: bindings.text("__Host-session"),
      PUBLIC_API_URL: bindings.text("https://somo-tracker.onrender.com"),
      PUBLIC_APP_URL: bindings.text("https://app.somotracker.com"),
    },
  }),
});
