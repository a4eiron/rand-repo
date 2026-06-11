import { createClient } from "redis";
import { createNodeRedisClient } from "bullmq";

const redisClient = createClient({
  socket: {
    port: 6380,
    host: "localhost",
  },
});

redisClient.on("connect", () => {
  console.log("redis up");
});

redisClient.on("error", (err) => {
  console.log("failed to connect to redis", err);
  throw Error(err);
});

await redisClient.connect();

// @ts-ignore it works
const connection = createNodeRedisClient(redisClient);

export { connection };
