(function () {
  const BACKEND = "http://localhost:8080";

  const RESERVED = new Set([
    "settings", "notifications", "orgs", "marketplace", "explore", "topics",
    "collections", "sponsors", "new", "login", "logout", "about", "pricing",
    "features", "dashboard", "pulls", "issues", "codespaces", "search",
  ]);

  function repoFromPath() {
    const parts = location.pathname.split("/").filter(Boolean);
    if (parts.length < 2) return null;
    const [owner, name] = parts;
    if (RESERVED.has(owner.toLowerCase())) return null;
    return { owner, name };
  }

  function send(msg) {
    return new Promise((resolve) => chrome.runtime.sendMessage(msg, resolve));
  }

  function styleButton(btn) {
    btn.style.cssText =
      "position:fixed;bottom:20px;right:20px;z-index:9999;border:none;cursor:pointer;" +
      "color:#fff;padding:10px 14px;border-radius:8px;font-family:system-ui,sans-serif;" +
      "font-size:14px;box-shadow:0 2px 8px rgba(0,0,0,.2);";
  }

  // Reflect the API result on the button (label, color, and what a click does).
  function setState(btn, res) {
    if (!res || res.error) {
      btn.textContent = "⚠️ error";
      btn.style.background = "#d1242f";
      btn.dataset.state = "error";
      return;
    }
    if (!res.loggedIn) {
      btn.textContent = "🔔 Log in to subscribe";
      btn.style.background = "#57606a";
      btn.dataset.state = "login";
      return;
    }
    if (res.subscribed) {
      btn.textContent = "✓ Subscribed - click to unsubscribe";
      btn.style.background = "#8250df";
      btn.dataset.state = "subscribed";
    } else {
      btn.textContent = "🔔 Subscribe to good first issues";
      btn.style.background = "#2da44e";
      btn.dataset.state = "unsubscribed";
    }
  }

  async function onClick() {
    const btn = document.getElementById("gfi-subscribe-btn");
    const { owner, name, state } = btn.dataset;

    if (state === "login") {
      window.open(BACKEND + "/auth/login", "_blank");
      return;
    }
    btn.disabled = true;
    const type = state === "subscribed" ? "unsubscribe" : "subscribe";
    const res = await send({ type, owner, name });
    btn.disabled = false;
    setState(btn, res);
  }

  async function refresh() {
    const repo = repoFromPath();
    let btn = document.getElementById("gfi-subscribe-btn");

    if (!repo) {
      if (btn) btn.remove();
      return;
    }
    if (!btn) {
      btn = document.createElement("button");
      btn.id = "gfi-subscribe-btn";
      styleButton(btn);
      btn.addEventListener("click", onClick);
      document.body.appendChild(btn);
    }
    btn.dataset.owner = repo.owner;
    btn.dataset.name = repo.name;
    btn.textContent = "…";

    const res = await send({ type: "status", owner: repo.owner, name: repo.name });
    setState(btn, res);
  }

  refresh();

  let last = location.pathname;
  setInterval(function () {
    if (location.pathname !== last) {
      last = location.pathname;
      refresh();
    }
  }, 1000);
})();
