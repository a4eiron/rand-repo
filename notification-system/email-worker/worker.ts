import { Worker } from "bullmq";
import { connection } from "./redisConn.ts";

const worker = new Worker(
  "email_queue",
  (job) => {
    console.log(job.id);
    // console.log(job.name);
    // console.log(job.data.id);

    throw new Error(`test retry ${job.id}`);
  },
  { connection },
);

await worker.waitUntilReady();
console.log("READY");

worker.on("ready", () => {
  console.log("worker ready");
});

worker.on("completed", (job) => {
  console.log(job.id, "completed");
});

worker.on("failed", (job) => {
  console.log(job?.id, "failed");
});
