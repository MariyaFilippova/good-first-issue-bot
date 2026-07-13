const BACKEND = "http://localhost:8080";

chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
  handle(msg)
    .then(sendResponse)
    .catch((err) => sendResponse({ error: String(err) }));
  return true;
});

async function handle(msg) {
  if (msg.type === "status") {
    const url =
      BACKEND +
      "/api/status?owner=" +
      encodeURIComponent(msg.owner) +
      "&name=" +
      encodeURIComponent(msg.name);
    const r = await fetch(url, { credentials: "include" });
    if (r.status === 401) return { loggedIn: false };
    if (!r.ok) throw new Error("status " + r.status);
    const j = await r.json();
    return { loggedIn: true, subscribed: j.subscribed };
  }

  if (msg.type === "subscriptions") {
    const r = await fetch(BACKEND + "/api/subscriptions", { credentials: "include" });
    if (r.status === 401) return { loggedIn: false };
    if (!r.ok) throw new Error("status " + r.status);
    const repos = await r.json();
    return { loggedIn: true, repos: repos || [] };
  }

  if (msg.type === "subscribe" || msg.type === "unsubscribe") {
    const r = await fetch(BACKEND + "/api/" + msg.type, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ owner: msg.owner, name: msg.name }),
    });
    if (r.status === 401) return { loggedIn: false };
    if (!r.ok) throw new Error("status " + r.status);
    const j = await r.json();
    return { loggedIn: true, subscribed: j.subscribed };
  }

  throw new Error("unknown message type: " + msg.type);
}
