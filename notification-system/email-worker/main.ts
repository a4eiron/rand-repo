import { load } from "@std/dotenv";
import { emailQueue } from "./queue.ts";

await load({
  envPath: "../.env",
  export: true,
});

await import("./redisConn.ts");
await import("./queue.ts");
await import("./worker.ts");
await import("./consumer.ts");
