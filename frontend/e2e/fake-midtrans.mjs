import { createServer } from "node:http";

const port = Number(process.env.E2E_MIDTRANS_PORT ?? 8125);
const grossAmountByOrder = new Map();

async function readJson(request) {
  let raw = "";
  for await (const chunk of request) raw += chunk;
  return JSON.parse(raw || "{}");
}

function send(response, status, body) {
  response.writeHead(status, { "Content-Type": "application/json" });
  response.end(JSON.stringify(body));
}

createServer(async (request, response) => {
  const { pathname } = new URL(request.url ?? "/", "http://localhost");

  if (request.method === "POST" && pathname === "/snap/v1/transactions") {
    const { transaction_details: details } = await readJson(request);
    grossAmountByOrder.set(details.order_id, details.gross_amount);
    return send(response, 201, {
      token: `e2e-${details.order_id}`,
      redirect_url: `http://127.0.0.1:${port}/redirect/${details.order_id}`,
    });
  }

  const [, orderId, action] = pathname.match(/^\/v2\/([^/]+)\/(status|expire)$/) ?? [];
  if (orderId && grossAmountByOrder.has(orderId)) {
    if (action === "expire") return send(response, 200, { status_code: "407", transaction_status: "expire" });
    return send(response, 200, {
      order_id: orderId,
      status_code: "200",
      transaction_status: "settlement",
      fraud_status: "accept",
      gross_amount: `${grossAmountByOrder.get(orderId)}.00`,
    });
  }

  send(response, 404, { status_code: "404", status_message: "Transaction doesn't exist." });
}).listen(port, () => console.log(`fake Midtrans listening on ${port}`));
