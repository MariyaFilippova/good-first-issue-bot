# Good First Issue Bot

Get emailed the moment your favorite repositories post a **good first issue**.

GFI Bot watches the GitHub repositories you care about and notifies you by email as
soon as a new `good first issue` is opened. Subscribe in one click with the Chrome extension.

🌐 **[gfibot.com](https://gfibot.com)**

---
<img width="1228" height="941" alt="Screenshot 2026-07-14 at 13 59 58" src="https://github.com/user-attachments/assets/a96b4c2b-c0b9-4a53-b724-9ff531b93169" />
<img width="1227" height="941" alt="Screenshot 2026-07-14 at 14 00 42" src="https://github.com/user-attachments/assets/5c77b301-322c-41fd-b60c-0d00c923c832" />

## Install

Add the extension from the Chrome Web Store:

**[➡️ Get GFI Bot for Chrome](https://chromewebstore.google.com/detail/good-first-issue-subscrib/lpaepaloljaebdgnhnimnbpgadedkmpl)**

Then click the extension button on any GitHub repository page to subscribe.

## Tech stack

- **Go** - HTTP server + background poller (single static binary)
- **PostgreSQL** - storage (see [`schema.sql`](schema.sql))
- **Resend** - transactional email
- **Chrome extension** (Manifest V3) - the `extension/` directory
- **Caddy** - reverse proxy with automatic HTTPS (production)

## License

See [LICENSE](LICENSE).
