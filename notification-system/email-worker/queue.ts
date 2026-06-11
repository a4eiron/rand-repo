import { QueueEvents } from "bullmq";
import { Queue } from "bullmq";
import { connection } from "./redisConn.ts";

const emailQueue = new Queue("email_queue", { connection });
const emailQueueEvents = new QueueEvents("email_queue", { connection });

emailQueueEvents.on("completed", (jobId) => {
  console.log("Job:", jobId, "completed");
});

emailQueueEvents.on("failed", async () => {
  const failedJobs = await emailQueue.getFailed();
  console.log(
    "failed jobs:",
    failedJobs.map((j) => j.id),
  );
});

export { emailQueue };
