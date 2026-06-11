import amqp from "amqplib";
import { emailQueue } from "./queue.ts";

async function main() {
  const connection = await amqp.connect(Deno.env.get("RMQ_URI")!);
  const channel = await connection.createChannel();

  const exchange = await channel.assertExchange("notifications", "topic", {
    durable: true,
  });

  const queue = await channel.assertQueue("", {
    exclusive: true,
  });

  await channel.bindQueue(
    queue.queue,
    exchange.exchange,
    "notifications.email",
  );

  await channel.consume(
    queue.queue,
    async (msg) => {
      if (!msg) return;

      const payload = JSON.parse(msg.content.toString());

      await emailQueue.add(
        "send-email",
        {
          id: payload.id,
          to: payload.to,
          body: payload.body,
          priority: payload.priority,
        },
        {
          priority: payload.priority === "transactional" ? 1 : 100,
          removeOnComplete: true,
          removeOnFail: false,
          attempts: 3,
          backoff: {
            type: "exponential",
            delay: 1000,
          },
        },
      );

      channel.ack(msg);
    },
    { noAck: false },
  );
}

main();
