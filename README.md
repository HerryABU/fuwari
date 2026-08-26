# 🍥 Fuwari Server

![Node.js >= 20](https://img.shields.io/badge/node.js-%3E%3D20-brightgreen)
![pnpm >= 9](https://img.shields.io/badge/pnpm-%3E%3D9-blue)
![Go >= 1.25](https://img.shields.io/badge/go-%3E%3D1.25-blue)
[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-red.svg)](LICENSE)

A self-contained blog system: **Go backend + embedded Astro frontend in a
single binary**. Refactored from the
[Fuwari](https://github.com/saicaca/fuwari) static Astro theme (MIT), keeping
the original UI pixel-identical while adding a runtime backend.

```text
fuwari-server.exe   ← one file: frontend + API + SQLite + themes + editor
```

---

## ✨ Features

**Frontend (original Fuwari theme, untouched)**
- Astro + Tailwind CSS, light/dark mode, customizable hue & banner
- Smooth page transitions, responsive design
- Search with Pagefind, RSS feed, Markdown extended syntax, TOC

**Backend (Go)**
- 🗄️ **Comments in SQLite** — readers post freely (per-IP rate limit + XSS
  sanitization); deletion requires the admin password; manageable in `/admin`
- 📁 **Posts on the file system** — Markdown + YAML frontmatter, seeded from
  `src/content/posts` on first boot; full CRUD API
- ✍️ **Admin console at `/admin`** — unified backend (posts editor, comment
  management, theme switcher, password change, system info), Cherry Markdown
  editor with image upload
- 🌐 **Dynamic runtime posts** — posts created in `/admin` are rendered on the
  frontend via Cherry even though the SSG frontend was built earlier; home &
  list pages auto-append them
- 🎨 **Runtime theme system** (alist-style) — hot reload, front **and**
  `/admin` backend share the exact same CSS variables
- 🧩 **Hot-loadable extensions** — live2d/kanban-musume, analytics, any
  `index.js`/`index.css`, injected into every page
- 🔗 **Reverse-proxy sub-path mounting** — works under any `/{name}/` prefix,
  no hardcoding, no recompiling
- 🔐 **Admin password** — bcrypt in SQLite; change in `/admin` or reset from
  the command line (`-re pwd`)
- 🌍 **i18n** — admin console & comment widget support Chinese/English
  (`fuwari_lang` in localStorage, `?lang=en|zh`, or browser language)
- 🌐 **IPv6** dual-stack listening (`ENABLE_IPV6=true`)

---

## 🚀 Quick Start

### Build (Windows)

```bat
build.bat
```

This runs `pnpm install → pnpm build → sync dist → go build`, producing
`fuwari-server.exe` (frontend embedded via `go:embed`).

Or manually:

```sh
pnpm install
pnpm build                          # -> dist/
robocopy dist server/dist /E        # sync embedded frontend (or xcopy)
cd server && go build -o fuwari-server.exe .
```

### Run

```sh
fuwari-server.exe
```

- Site: <http://localhost:9000>
- Admin console: <http://localhost:9000/admin> (posts / comments / themes /
  password / system; `/editor` still works as an alias)
- Health: <http://localhost:9000/api/health>

On first boot the server generates a `.env` template, seeds the content
directory, and creates the initial admin password (see below).

---

## 🔐 Admin Password

The admin password (bcrypt-hashed in the SQLite `settings` table) protects
all write operations: creating/saving/deleting posts, deleting comments and
changing the password itself.

**First boot** — the initial password is chosen once:

- `ADMIN_TOKEN` set in `.env` → used as the initial password (existing
  deployments migrate seamlessly);
- otherwise a random password is generated and printed to the startup log
  (shown once — save it).

**Change it** (when you know the password):

- `/admin` → "🔑 密码" tab (current + new password), or
- `POST /api/admin/password` with `X-Admin-Token: <current>`.

**Reset it** (forgot the password) — no server needed:

```bat
:: 场景：服务在运行，但用户忘记了密码

:: 1. 停止服务 (Ctrl+C)
:: 2. 带重置参数启动
fuwari-server -re pwd

:: 输出：
:: > 请输入新密码: ******
:: > 请再次输入: ******
:: > ✅ 管理员密码已重置，请重新启动服务
:: > 按任意键继续...

:: 3. 正常启动
fuwari-server
```

New passwords must be at least 6 characters. The reset command also works
piped (`echo -e "pass\npass" | fuwari-server -re pwd`) for automation.

Auth headers: `X-Admin-Token: <password>` or `Authorization: Bearer <password>`.

---

## 🎨 Theme System (hot-reload, no recompiling)

The UI — frontend **and** the `/admin` backend — is fully themeable at
runtime, alist-style:

```
themes/
  ocean/
    theme.css        # CSS variable overrides (front & back UI share the same vars)
    background.jpg   # optional background image (referenced in theme.css)
    custom.js        # optional custom script (mascot/live2d, analytics, ...)
    manifest.json    # optional metadata (name/description/author/version)
```

- **Switch**: URL `?theme=ocean`, Cookie `fuwari_theme`, or the switcher in
  the admin console — persisted in the cookie.
- **Hot reload**: edit any file under `themes/<name>/` and refresh — no
  recompiling, no restart.
- **Front/back consistency**: the backend editor consumes the exact same CSS
  variables as the frontend (`--page-bg`, `--card-bg`, `--deep-text`,
  `--primary`, `--hue`, ...). Note: fuwari sets `--hue` inline in the original
  Layout, so theme CSS must use `!important` on overridden variables.
- **Defaults**: the `default` theme is the original untouched look
  (embedded). Template themes are seeded into the runtime `themes/` dir on
  first run.

---

## 🧩 Extensions (hot-loadable, live2d-style)

Drop a folder under `extensions/<name>/` with `index.js` / `index.css`; they
are injected into **every page** (frontend + backend) at serve time:

```
extensions/
  live2d/
    index.js         # mascot / kanban-musume entry
    model/           # model assets (served at /extensions/live2d/model/...)
```

Edit and refresh — no recompiling. See `extensions/README.md` for details.

---

## 💬 Comments

- **Posting is open to readers**: `POST /api/comments` with
  `{slug, nickname, content}` — per-IP rate limited (10/min) and sanitized
  (HTML stripped, Markdown rendered safely).
- The comment widget is injected into article pages at serve time (frontend
  sources untouched); Cherry Markdown renders each comment and provides the
  composer editor.
- **Deletion requires the admin password**: `DELETE /api/comments/:id`, or
  use the Comments tab in `/admin` (list all / filter by slug / delete).

---

## ✍️ Admin Console (/admin)

The unified management console (no frontend changes), fully themed like the
frontend and i18n (中文/English):

- **📝 Posts** — list/create/save/delete posts against the file-system store;
  full Cherry Markdown editor (`edit&preview`) with **image upload**
  (`POST /api/admin/upload` → stored under `content/posts/uploads/`)
- **💬 Comments** — view all comments (or filter by slug), delete
- **🎨 Themes** — theme cards + instant switching
- **🔑 Password** — change the admin password
- **ℹ️ System** — version / host / uptime / paths / themes / extensions

Access: <http://localhost:9000/admin> — enter the admin password (kept in
`localStorage` between sessions). `/editor` remains as a compatible alias.

**Layout mirrors the frontend architecture** (centered `--page-width`
container, two-column grid, card-base panels) and **UI is dynamically
injected & hot-reloadable**: the console's styles and logic live in the
runtime `admin/ui.css` + `admin/ui.js` (seeded from the
embedded defaults on first boot; `ADMIN_DIR`). Edit them and refresh — no
recompiling, no restart — just like themes and extensions.

---

## 🌐 Runtime Posts on the Frontend

The Astro frontend is built statically, so posts created at runtime are not
in the embedded `dist`. A serve-time injected viewer script solves this
without touching frontend sources:

- Visiting `/posts/<new-slug>/` renders the post with Cherry Markdown
  (title / date / tags / body / images), then the comment widget attaches.
- Home/list pages append cards for runtime posts that are missing from the
  built-in list.

## 🌍 i18n (Admin & Comment Widget)

The site defaults to Chinese (config lang `zh_CN`) on first deploy. The
admin console and the comment widget ship with Chinese and English.
Language is picked from, in order: `localStorage.fuwari_lang`, URL
`?lang=en|zh`, then the browser language. The admin header has a language
switcher that persists the choice. Post `lang` frontmatter is also exposed
via the API for content-level language marking.

The whole site — pages, **js/css assets**, API, comments, `/admin`, themes,
extensions and pagefind search — works under **any** reverse-proxy sub-path
such as `https://host:8088/{name}/`. The proxy must keep the prefix
(no path rewriting); fuwari auto-detects it per request:

- HTML absolute references (`/_astro/*.js|css`, `/assets/*`, `/themes/*`,
  `/extensions/*`, `/pagefind/*`, favicon, nav links) are rewritten to
  `/{name}/...` at serve time — nothing is hardcoded, no recompiling.
- API calls from the comment widget and the `/admin` console use
  `window.FUWARI_BASE` (injected into `<head>`) to stay prefix-agnostic.
- A mount-aware handler strips the prefix internally and re-routes
  (`/{name}/api/...` → `/api/...`), so direct access keeps working unchanged.

Example nginx (**keep the prefix** — do not use a trailing slash in
`proxy_pass`, otherwise page js/css resolves outside `/{name}/` and 404s):

```nginx
location /name/ {
    proxy_pass http://127.0.0.1:9000;   # no trailing slash: /name/ is preserved
    proxy_set_header Host $host;
}
```

No configuration needed on the fuwari side.

---

## ⚙️ Configuration (.env)

Auto-generated on first boot; edit and restart to apply:

| Variable | Default | Description |
|:---|:---|:---|
| `SERVER_PORT` | `9000` | HTTP port |
| `DB_PATH` | `./data/fuwari.db` | SQLite file (comments + settings) |
| `POSTS_DIR` | `./content/posts` | Runtime post store (seeded from `SRC_POSTS_DIR`) |
| `SRC_POSTS_DIR` | `./src/content/posts` | Seed source (read-only) |
| `ADMIN_TOKEN` | *(empty)* | First-boot initial admin password; if empty a random one is printed once |
| `THEMES_DIR` | `./themes` | Runtime themes (seeded from repo `themes/`) |
| `DEFAULT_THEME` | `default` | Fallback theme name |
| `EXTENSIONS_DIR` | `./extensions` | Runtime extensions (seeded from repo `extensions/`) |
| `BIND_IPV4` | `0.0.0.0` | IPv4 bind address |
| `ENABLE_IPV6` | `false` | `true` → listen on `[::]` (dual-stack) |
| `BIND_IPV6` | `::` | IPv6 bind address |
| `COMMENT_PAGE_SIZE` | `20` | Comments per page |
| `COMMENT_MAX_LENGTH` | `4000` | Max comment length (chars) |

---

## 📡 API Overview

Unified response envelope: `{code, message, data}` (`0` = success).

| Method | Path | Auth | Description |
|:---|:---|:---|:---|
| GET | `/api/health` | — | Health check |
| GET | `/api/posts` | — | List posts (`?include_draft=1` for drafts) |
| GET | `/api/posts/:slug` | — | Post detail (frontmatter + body) |
| GET | `/api/posts/:slug/raw` | — | Raw markdown source |
| POST | `/api/posts` | ✅ | Create post |
| PUT | `/api/posts/:slug` | ✅ | Update post |
| DELETE | `/api/posts/:slug` | ✅ | Delete post |
| GET | `/api/comments` | — | List comments (empty `slug` = all, for admin) |
| POST | `/api/comments` | — | Post comment (rate-limited, sanitized) |
| DELETE | `/api/comments/:id` | ✅ | Delete comment |
| GET | `/api/themes` | — | List themes |
| POST | `/api/theme` | — | Switch theme (cookie) |
| POST | `/api/admin/password` | ✅ | Change admin password |
| POST | `/api/admin/upload` | ✅ | Upload image (multipart `file`, ≤5MB) → `/uploads/*` |
| GET | `/themes/:name/*` | — | Theme assets (runtime-first, embedded fallback) |
| GET | `/extensions/:name/*` | — | Extension assets |

---

## 🗂 Project Layout

```text
├── build.bat              # one-shot build: frontend → embed → Go binary
├── src/                   # Astro frontend (original Fuwari theme, untouched)
│   └── content/posts/     # seed posts (read-only source)
├── themes/                # theme templates (seeded to runtime on first boot)
├── extensions/            # extension templates
└── server/                # Go backend (mirrors the NVS architecture)
    ├── dist/              # Astro build output — embedded via go:embed
    ├── assets/            # Cherry Markdown, admin console, widgets (comments/post-viewer), default theme
    ├── config/            # .env loading & runtime config
    ├── models/            # GORM models (Comment, Setting) + SQLite
    ├── handlers/          # posts (fs), comments, themes, extensions, admin
    ├── security/          # password auth, rate limit, sanitization
    ├── utils/             # unified response
    └── main.go / mount.go / embed.go / admin_cli.go
```

---

## 🧑‍💻 Frontend Development (optional)

The original Fuwari workflow still works for the Astro frontend:

| Command | Action |
|:---|:---|
| `pnpm install` | Install dependencies |
| `pnpm dev` | Local dev server at `localhost:4321` |
| `pnpm build` | Build the site to `./dist/` |
| `pnpm check` | Run checks |
| `pnpm new-post <filename>` | Create a new post in `src/content/posts/` |

Post frontmatter (aligned with the Fuwari collection schema):

```yaml
---
title: My First Blog Post
published: 2023-09-09
description: This is the first post of my new Astro blog.
image: ./cover.jpg
tags: [Foo, Bar]
category: Front-end
draft: false
---
```

---

## 📄 License

**GNU Affero General Public License v3.0 (AGPL-3.0)**

- This repository is a derivative work of the
  [Fuwari](https://github.com/saicaca/fuwari) blog theme by saicaca
  (MIT License). The original copyright notice is preserved in
  [THIRD-PARTY-NOTICES.md](./THIRD-PARTY-NOTICES.md).
- If you deploy this program as a network service, you MUST provide your
  users with access to the corresponding source code (AGPL v3.0 section 13,
  see [NOTICE](./NOTICE)).
- See [LICENSE](./LICENSE) for the full license text and
  [THIRD-PARTY-NOTICES.md](./THIRD-PARTY-NOTICES.md) for bundled third-party
  components (Cherry Markdown, Go modules, frontend packages).
