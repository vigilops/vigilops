import { createEnv } from "@t3-oss/env-core"
import { z } from "zod"

export const env = createEnv({
  server: {},

  clientPrefix: "VITE_",

  client: {
    VITE_VIGIL_API_URL: z.url(),
    VITE_VIGIL_API_KEY: z.string().optional(),
  },

  runtimeEnv: import.meta.env,

  emptyStringAsUndefined: true,
})
