import { bindings, defineConfig, defineWorker } from "cf/config";
// import { createWorkersResponseStoreServiceBindingConfig } from "@vinext/cloudflare/cache/config";

// const responseStore = await createWorkersResponseStoreServiceBindingConfig({
//   worker: {
//     name: "somo-tracker-frontend-response-store",
//     compatibilityDate: "2026-11-16",
//   },
//   bucket: "somo-tracker-frontend-response-store-cache-bodies",
// });

// export const responseStoreServiceBinding = responseStore.serviceBindingWorker;

export default defineConfig({
  worker: defineWorker({
    // ...responseStore.applicationWorker,
    name: "somo-tracker-frontend",
    entrypoint: "vinext/server/fetch-handler",
    compatibilityDate: "2026-11-16",
    assets: {
      htmlHandling: "none",
      notFoundHandling: "none",
      runWorkerFirst: ["/*", "!/_next/static/*", "!/*/_next/static/*"],
    },
    env: {
      // ...responseStore.applicationWorker.env,
      ASSETS: bindings.assets(),
      IMAGES: bindings.images(),
    },
  }),
});
