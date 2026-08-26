/* fuwari 动态文章查看器（服务端注入，不改任何前端源码）
 *
 * 解决 SSG 前端下「运行时创建/编辑的文章不在构建产物」的展示问题：
 *   1) /posts/<slug>/ 命中运行时文章（dist 无该页）→ 拉取 API，用 Cherry
 *      Markdown 渲染进主内容区，并准备 #post-container 供评论挂件挂载；
 *   2) 首页/归档等列表页 → 把构建列表之外的运行时文章卡片追加到列表末尾。
 *
 * 与评论挂件共用 Cherry（本脚本先加载则自备，否则复用 comments.js 已加载的实例）。
 */
(function () {
  'use strict';
  var fwBase = window.FUWARI_BASE || '/';
  // 兜底防护：后台/编辑器页面不执行（服务端已不注入，双保险防止底部渲染文章列表）
  if (window.location.pathname.indexOf('/admin') !== -1 || window.location.pathname.indexOf('/editor') !== -1) {
    return;
  }
  var postRe = /\/posts\/(.+?)\/?$/;
  var m = window.location.pathname.match(postRe);

  function escapeHtml(s) {
    return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;')
      .replace(/>/g, '&gt;').replace(/"/g, '&quot;');
  }

  // 确保 Cherry 可用（已加载则立即回调；加载中则轮询等待）
  function loadCherry(cb) {
    if (window.Cherry) { cb(); return; }
    if (document.querySelector('script[src*="cherry-markdown.core.js"]')) {
      var t = setInterval(function () {
        if (window.Cherry) { clearInterval(t); cb(); }
      }, 120);
      setTimeout(function () { clearInterval(t); if (!window.Cherry) cb(); }, 10000);
      return;
    }
    var css = document.createElement('link');
    css.rel = 'stylesheet';
    css.href = fwBase + 'assets/cherry/cherry-markdown.min.css';
    document.head.appendChild(css);
    var s = document.createElement('script');
    s.src = fwBase + 'assets/cherry/cherry-markdown.core.js';
    s.onload = function () { cb(); };
    s.onerror = function () { cb(); };
    document.head.appendChild(s);
  }

  // Markdown → HTML（Cherry setValue 异步 + 初始 afterChange 空触发：事件驱动 + 非空判断）
  var renderEngine = null;
  function renderMarkdown(md, cb) {
    loadCherry(function () {
      if (!window.Cherry) { cb(escapeHtml(md)); return; }
      if (!renderEngine) {
        var host = document.createElement('div');
        host.id = 'fuwari-post-renderer-host';
        host.style.cssText = 'position:fixed;top:0;left:0;width:600px;height:100px;z-index:-1;visibility:hidden';
        document.body.appendChild(host);
        renderEngine = new window.Cherry({
          id: 'fuwari-post-renderer-host',
          value: '',
          editor: { defaultModel: 'previewOnly' },
        });
      }
      var done = false;
      var fin = function () {
        if (done) return;
        try {
          var h = renderEngine.getHtml(false);
          if (h && String(h).length > 0) { done = true; cb(String(h)); }
        } catch (e) { done = true; cb(escapeHtml(md)); }
      };
      renderEngine.on('afterChange', fin);
      try { renderEngine.setValue(md); } catch (e) { done = true; cb(escapeHtml(md)); return; }
      setTimeout(fin, 1200);
    });
  }

  // ---------- 1) 运行时文章详情 ----------
  if (m) {
    var slug = decodeURIComponent(m[1]);
    var existing = document.getElementById('post-container');
    // dist 已有该文章页（构建时存在）→ 不动
    if (existing && existing.textContent.trim().length > 300) { return; }
    fetch(fwBase + 'api/posts/' + encodeURIComponent(slug))
      .then(function (r) { return r.json(); })
      .then(function (json) {
        if (!json || json.code !== 0) return; // 文章不存在：保持 fallback 页面
        var p = json.data;
        var wrap = document.getElementById('content-wrapper') || document.body;

        var art = document.createElement('article');
        art.className = 'card-base rounded-[var(--radius-large)] overflow-hidden';
        var meta = '';
        if (p.published) meta += '<time>' + escapeHtml(p.published) + '</time>';
        if (p.category) meta += (meta ? ' · ' : '') + '<span>' + escapeHtml(p.category) + '</span>';
        if (p.tags && p.tags.length) {
          meta += (meta ? ' · ' : '') + p.tags.map(function (t) {
            return '<a href="' + fwBase + 'archive/?tag=' + encodeURIComponent(t) + '" class="mr-1">#' + escapeHtml(t) + '</a>';
          }).join('');
        }
        art.innerHTML =
          '<header class="pt-8 px-6 md:px-8">' +
            '<h1 class="font-bold text-3xl md:text-4xl leading-tight">' + escapeHtml(p.title) + '</h1>' +
            (p.description ? '<p class="mt-4 opacity-80">' + escapeHtml(p.description) + '</p>' : '') +
            (meta ? '<div class="mt-3 text-sm opacity-70">' + meta + '</div>' : '') +
          '</header>' +
          '<div id="post-container">' +
            '<div class="markdown-content prose dark:prose-invert prose-base !max-w-none custom-md px-6 md:px-8 py-6"></div>' +
          '</div>';

        // 替换正文区但保留 footer（避免运行时文章页丢失版权标识）
        var footer = wrap.querySelector('.footer, footer');
        Array.prototype.forEach.call(wrap.children, function (el) {
          if (el !== footer) el.remove();
        });
        if (footer) wrap.insertBefore(art, footer); else wrap.appendChild(art);

        renderMarkdown(p.body || '', function (html) {
          var body = art.querySelector('.markdown-content');
          if (body) body.innerHTML = html;
          // 通知评论挂件挂载（它监听 #post-container 出现）
          document.dispatchEvent(new CustomEvent('fuwari:post-ready'));
        });
      })
      .catch(function () {});
    return;
  }

  // ---------- 2) 列表页补全运行时文章 ----------
  fetch(fwBase + 'api/posts')
    .then(function (r) { return r.json(); })
    .then(function (json) {
      if (!json || json.code !== 0) return;
      var list = json.data.list || [];
      if (!list.length) return;
      var wrap = document.getElementById('content-wrapper') || document.body;
      var seen = {};
      // 仅在内容区内统计已见 slug（避免 sidebar「最新文章」等区域链接干扰定位）
      wrap.querySelectorAll('a[href*="/posts/"]').forEach(function (a) {
        var mm = a.getAttribute('href').match(postRe);
        if (mm) seen[decodeURIComponent(mm[1])] = true;
      });
      // 文章列表容器：内容区内第一个包含 ≥2 篇文章链接的祖先（内部追加 = footer 之前）
      var listEl = null;
      var firstLink = wrap.querySelector('a[href*="/posts/"]');
      if (firstLink) {
        var anc = firstLink;
        while (anc && anc !== wrap && anc !== document.body && anc !== document.documentElement) {
          if (anc.querySelectorAll('a[href*="/posts/"]').length >= 2) { listEl = anc; break; }
          anc = anc.parentElement;
        }
      }
      var footer = wrap.querySelector('.footer, footer');
      var added = 0;
      list.forEach(function (p) {
        if (p.draft || seen[p.slug]) return;
        var card = document.createElement('div');
        card.className = 'card-base rounded-[var(--radius-large)] overflow-hidden p-6 mb-6';
        card.innerHTML =
          '<a href="' + fwBase + 'posts/' + encodeURIComponent(p.slug) + '/" class="transition font-bold text-2xl hover:text-[var(--primary)]">' + escapeHtml(p.title) + '</a>' +
          '<div class="mt-2 text-sm opacity-70">' + escapeHtml(p.published || '') + (p.category ? ' · ' + escapeHtml(p.category) : '') + '</div>' +
          (p.description ? '<p class="mt-3 opacity-80">' + escapeHtml(p.description) + '</p>' : '');
        // 插入顺序：列表容器内 > footer 之前（绝不落到版权下方）
        if (listEl) listEl.appendChild(card);
        else if (footer) wrap.insertBefore(card, footer);
        else wrap.appendChild(card);
        added++;
      });
      if (added) console.log('[fuwari] 已追加 ' + added + ' 篇运行时文章到列表');
    })
    .catch(function () {});
})();
