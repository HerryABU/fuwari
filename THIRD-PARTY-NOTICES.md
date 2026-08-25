# Third-Party Notices

This file lists third-party components bundled with or used by this project,
together with their licenses. The project as a whole is distributed under
the GNU AGPL v3.0 (see LICENSE), except for the separately-licensed
components listed below whose original notices are preserved.

## Upstream Project

### Fuwari (blog theme)

- Copyright (c) 2024 saicaca
- https://github.com/saicaca/fuwari
- License: MIT

Original license text:

> MIT License
>
> Copyright (c) 2024 saicaca
>
> Permission is hereby granted, free of charge, to any person obtaining a copy
> of this software and associated documentation files (the "Software"), to deal
> in the Software without restriction, including without limitation the rights
> to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
> copies of the Software, and to permit persons to whom the Software is
> furnished to do so, subject to the following conditions:
>
> The above copyright notice and this permission notice shall be included in all
> copies or substantial portions of the Software.
>
> THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
> IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
> FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
> AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
> LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
> OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
> SOFTWARE.

## Frontend Dependencies (npm, see package.json / pnpm-lock.yaml)

The following notable dependencies are used by the frontend build; all are
open-source licenses compatible with the project. Full dependency tree and
exact versions are recorded in `pnpm-lock.yaml`.

| Component | License |
|:---|:---|
| Astro | MIT |
| Svelte | MIT |
| Tailwind CSS | MIT |
| Pagefind | MIT |
| swup | MIT |
| expressive-code | MIT |
| markdown-it, KaTeX, PhotoSwipe | MIT |
| sanitize-html | MIT |
| Fontsource fonts (Roboto, JetBrains Mono) | OFL-1.1 / Apache-2.0 (see package metadata) |
| Iconify icons (Font Awesome, Material Symbols) | CC-BY-4.0 / Apache-2.0 (see icon package metadata) |

## Go Backend Dependencies (see server/go.mod / server/go.sum)

| Component | License |
|:---|:---|
| gin-gonic/gin | MIT |
| gin-contrib/gzip | MIT |
| gorm.io/gorm | MIT |
| glebarez/sqlite (pure-Go SQLite driver, modernc.org) | BSD-3-Clause |
| gopkg.in/yaml.v3 | MIT |

## Bundled Assets

### Cherry Markdown

- Copyright (C) 2021 THL A29 Limited, a Tencent company. All rights reserved.
- https://github.com/Tencent/cherry-markdown
- License: Apache-2.0

Cherry Markdown is vendored into the Go server binary as static assets
(editor and comment composer). Its Apache-2.0 license text and copyright
notice are preserved alongside the vendored files under
`server/vendor-assets/cherry-markdown/`.
