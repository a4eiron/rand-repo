if (Deno.args.length < 3) {
  throw new Error("want X-Room-Id, X-User-Id, server Port");
}

const ws = new WebSocket(`ws://localhost:${Deno.args[2]}`, {
  headers: {
    "X-Room-Id": Deno.args[0],
    "X-User-Id": Deno.args[1],
  },
});

ws.onopen = async (_e) => {
  console.log("Connected to the server");

  const decoder = new TextDecoder();
  const buf = new Uint8Array(256);

  while (true) {
    const n = await Deno.stdin.read(buf);
    if (n === null) {
      break;
    }
    const input = decoder.decode(buf.subarray(0, n)).trim();
    if (input == "quit") {
      ws.close();
      break;
    }
    ws.send(input);
  }
};

ws.onmessage = (e) => {
  const msg = JSON.parse(e.data);
  const binString = atob(msg.data);

  const bytes = new Uint8Array(binString.length);
  for (let i = 0; i < binString.length; i++) {
    bytes[i] = binString.charCodeAt(i);
  }

  const text = new TextDecoder().decode(bytes);
  msg.data = text;
  console.log(msg);
};

ws.onerror = (e) => {
  console.error("WebSocket error observed:", e);
};

ws.onclose = (e) => {
  console.log(`WebSocket closed: Code=${e.code}, Reason=${e.reason}`);
};
