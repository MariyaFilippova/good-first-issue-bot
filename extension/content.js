(function () {
  const BACKEND = "https://gfibot.com";

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

  // ---- one-time styles + widget shell ----

  const CSS = `
    #gfi-widget { position:fixed; bottom:20px; right:20px; z-index:99999;
      font-family:-apple-system,BlinkMacSystemFont,system-ui,sans-serif;
      display:flex; flex-direction:column; align-items:flex-end; gap:8px; }
    .gfi-btn { border:none; cursor:pointer; border-radius:999px; padding:11px 18px;
      font-size:14px; font-weight:600; color:#fff; display:inline-flex; align-items:center; gap:8px;
      box-shadow:0 2px 10px rgba(0,0,0,.18); transition:transform .08s ease, box-shadow .12s ease, filter .12s ease; }
    .gfi-btn:hover { transform:translateY(-1px); box-shadow:0 5px 16px rgba(0,0,0,.24); filter:brightness(1.05); }
    .gfi-btn:active { transform:translateY(0); }
    .gfi-btn:disabled { opacity:.6; cursor:default; }
    .gfi-btn--subscribe  { background:#1f883d; }
    .gfi-btn--subscribed { background:#8250df; }
    .gfi-btn--login      { background:#57606a; }
    .gfi-btn--error      { background:#cf222e; }
    .gfi-link { background:#fff; border:1px solid #d0d7de; color:#0969da; cursor:pointer;
      font-size:13px; font-weight:600; padding:7px 12px; border-radius:999px; box-shadow:0 1px 4px rgba(0,0,0,.1);
      font-family:inherit; transition:background .12s ease; }
    .gfi-link:hover { background:#f6f8fa; }
    .gfi-panel { width:320px; max-height:360px; overflow:hidden; background:#fff; color:#1f2328;
      border:1px solid #d0d7de; border-radius:14px; box-shadow:0 10px 30px rgba(0,0,0,.2); font-size:13px; }
    .gfi-panel__head { padding:14px 16px; border-bottom:1px solid #d0d7de; }
    .gfi-panel__title { font-weight:700; font-size:14px; display:flex; align-items:center; gap:8px; }
    .gfi-panel__sub { color:#57606a; font-size:12px; margin-top:2px; }
    .gfi-panel__body { padding:6px; max-height:290px; overflow:auto; }
    .gfi-row { display:flex; justify-content:space-between; align-items:center; gap:10px;
      padding:8px 10px; border-radius:9px; }
    .gfi-row:hover { background:#f6f8fa; }
    .gfi-row a { color:#0969da; text-decoration:none; font-weight:600; }
    .gfi-row a:hover { text-decoration:underline; }
    .gfi-x { border:none; background:none; cursor:pointer; color:#cf222e; font-size:14px;
      border-radius:6px; padding:2px 7px; line-height:1; }
    .gfi-x:hover { background:#ffebe9; }
    .gfi-empty { padding:20px 16px; color:#57606a; text-align:center; line-height:1.5; }
  `;

  function ensureUI() {
    if (document.getElementById("gfi-widget")) return;

    const style = document.createElement("style");
    style.id = "gfi-styles";
    style.textContent = CSS;
    document.head.appendChild(style);

    const widget = document.createElement("div");
    widget.id = "gfi-widget";

    const panel = document.createElement("div");
    panel.id = "gfi-panel";
    panel.className = "gfi-panel";
    panel.style.display = "none";

    const listBtn = document.createElement("button");
    listBtn.id = "gfi-list-btn";
    listBtn.className = "gfi-link";
    listBtn.textContent = "📋 My subscriptions";
    listBtn.addEventListener("click", togglePanel);

    // Order: panel (top), subscribe button (added by refresh), list link (bottom).
    widget.append(panel, listBtn);
    document.body.appendChild(widget);
  }

  // ---- per-repo subscribe/unsubscribe button ----

  function setState(btn, res) {
    btn.className = "gfi-btn";
    if (!res || res.error) {
      btn.textContent = "⚠️ Error — is the server running?";
      btn.classList.add("gfi-btn--error");
      btn.dataset.state = "error";
      btn.title = "";
      return;
    }
    if (!res.loggedIn) {
      btn.textContent = "🔔 Log in to get good-first-issue alerts";
      btn.classList.add("gfi-btn--login");
      btn.dataset.state = "login";
      btn.title = "Sign in with GitHub";
      return;
    }
    if (res.subscribed) {
      btn.textContent = "✓ Watching for good first issues";
      btn.classList.add("gfi-btn--subscribed");
      btn.dataset.state = "subscribed";
      btn.title = "Click to stop watching this repo";
    } else {
      btn.textContent = "🔔 Subscribe to good first issues";
      btn.classList.add("gfi-btn--subscribe");
      btn.dataset.state = "unsubscribed";
      btn.title = "Get an email when this repo gets a new good first issue";
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
      btn.className = "gfi-btn gfi-btn--subscribe";
      btn.addEventListener("click", onClick);
      // Insert above the "My subscriptions" link.
      document.getElementById("gfi-widget").insertBefore(btn, document.getElementById("gfi-list-btn"));
    }
    btn.dataset.owner = repo.owner;
    btn.dataset.name = repo.name;
    btn.textContent = "…";

    const res = await send({ type: "status", owner: repo.owner, name: repo.name });
    setState(btn, res);
  }

  // ---- subscriptions panel ----

  async function togglePanel() {
    const panel = document.getElementById("gfi-panel");
    if (panel.style.display === "block") {
      panel.style.display = "none";
      return;
    }
    panel.style.display = "block";
    panel.replaceChildren(makeEmpty("Loading…"));
    renderPanel(await send({ type: "subscriptions" }));
  }

  function makeEmpty(text) {
    const d = document.createElement("div");
    d.className = "gfi-empty";
    d.textContent = text;
    return d;
  }

  function renderPanel(res) {
    const panel = document.getElementById("gfi-panel");
    panel.replaceChildren();

    const head = document.createElement("div");
    head.className = "gfi-panel__head";
    const title = document.createElement("div");
    title.className = "gfi-panel__title";
    title.textContent = "🔔 Good first issue alerts";
    const sub = document.createElement("div");
    sub.className = "gfi-panel__sub";
    sub.textContent = "Repos you're watching for new good first issues";
    head.append(title, sub);
    panel.append(head);

    if (!res || res.error) { panel.append(makeEmpty("Couldn't load — is the server running?")); return; }
    if (!res.loggedIn) { panel.append(makeEmpty("Log in first — click the subscribe button on any repo.")); return; }

    const repos = res.repos || [];
    if (repos.length === 0) {
      panel.append(makeEmpty("No subscriptions yet.\nOpen a repo and click “Subscribe to good first issues.”"));
      return;
    }

    const body = document.createElement("div");
    body.className = "gfi-panel__body";
    for (const r of repos) {
      const row = document.createElement("div");
      row.className = "gfi-row";

      const link = document.createElement("a");
      link.href = "https://github.com/" + r.owner + "/" + r.name;
      link.target = "_blank";
      link.textContent = r.owner + "/" + r.name;

      const x = document.createElement("button");
      x.className = "gfi-x";
      x.textContent = "✕";
      x.title = "Unsubscribe";
      x.addEventListener("click", async () => {
        x.disabled = true;
        await send({ type: "unsubscribe", owner: r.owner, name: r.name });
        row.remove();
        refresh(); // keep the per-repo button in sync if it's this repo
      });

      row.append(link, x);
      body.append(row);
    }
    panel.append(body);
  }

  // ---- init ----
  ensureUI();
  refresh();

  let last = location.pathname;
  setInterval(function () {
    if (location.pathname !== last) {
      last = location.pathname;
      refresh();
    }
  }, 1000);
})();
