import { createEnv } from "@t3-oss/env-core"
import { z } from "zod"

export const env = createEnv({
  server: {},

  clientPrefix: "VITE_",

  client: {
    VITE_KEELWAVE_API_URL: z.url(),
    VITE_KEELWAVE_API_KEY: z.string().optional(),
  },

  runtimeEnv: import.meta.env,

  emptyStringAsUndefined: true,
})
