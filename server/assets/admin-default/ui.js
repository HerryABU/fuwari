/* fuwari 管理后台 —— 完全复刻前台交互语言。
 * 依赖：前台构建样式（服务端注入）+ 主题变量 + Cherry Markdown。
 * 交互：与前台共享 localStorage（theme / hue），默认中文，移动端 float-panel 导航。
 * 热加载：直接编辑本文件，刷新页面即生效。
 */
(function () {
  'use strict';

  var BASE = (window.FUWARI_BASE || '/');
  var API = BASE + 'api';
  var TOKEN_KEY = 'fuwari_admin_token';
  var currentSlug = null;
  var cherry = null;
  var cherryInit = false;

  /* ---------- 深浅色同步（与前台 localStorage 'theme' 完全一致，html.dark 驱动） ---------- */
  function syncTheme() {
    var t = localStorage.getItem('theme') || 'auto';
    var dark = t === 'dark' || (t === 'auto' && window.matchMedia('(prefers-color-scheme: dark)').matches);
    document.documentElement.classList.toggle('dark', dark);
  }
  function toggleTheme() {
    var cur = document.documentElement.classList.contains('dark');
    localStorage.setItem('theme', cur ? 'light' : 'dark');
    syncTheme();
  }
  if (window.matchMedia) {
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', syncTheme);
  }

  /* ---------- 站点主色 hue（与前台一致：localStorage hue → --hue） ---------- */
  function applyHue() {
    var cfg = parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--configHue')) || 250;
    var hue = localStorage.getItem('hue');
    if (hue !== null && !isNaN(parseFloat(hue))) cfg = parseFloat(hue);
    document.documentElement.style.setProperty('--hue', cfg);
  }

  /* ---------- i18n（首次部署默认中文；双语表，lang 存 localStorage） ---------- */
  var I18N = {
    zh: {
      tabHome: '🏠 面板', tabPosts: '📝 文章', tabComments: '💬 评论', tabThemes: '🎨 主题', tabPassword: '🔑 密码', tabSystem: 'ℹ️ 系统',
      siteRole: '管理控制台', openSite: '🌐 打开前台',
      quickTitle: '⚡ 快捷操作', quickNew: '✍️ 写新文章', quickComments: '💬 评论管理', quickThemes: '🎨 主题设置', quickPwd: '🔑 修改密码', quickSys: 'ℹ️ 系统信息',
      sysTitle: '🖥️ 系统状态', sysVersion: '版本', sysUptime: '运行时长', sysDatabase: '数据库', sysPort: '端口',
      welcomeTitle: '👋 欢迎回来', welcomeTip: '这里是站点管理面板，左侧可快速开始写作与维护，下方是站点的实时概览。',
      statPosts: '文章', statDrafts: '草稿', statComments: '评论', statThemes: '主题',
      recentTitle: '🕒 最近文章', emptyRecent: '暂无文章', emptyPosts: '暂无文章，点击「＋ 新建」开始', emptyComments: '暂无评论', emptyThemes: '暂无主题',
      editorTitle: '✍️ 编辑器', phSearch: '搜索标题或 slug…', btnNew: '＋ 新建', btnSave: '保存', btnDelete: '删除', draftLabel: '草稿（不对外发布）',
      phTitle: '标题 *', phCategory: '分类', phTags: '标签（逗号分隔）', phDescription: '摘要描述', phCommentFilter: '按文章 slug 过滤（留空 = 全部）', btnLoad: '加载',
      pwdTitle: '🔑 修改管理员密码', pwdOld: '当前密码', pwdNew: '新密码（至少 6 个字符）', pwdNew2: '确认新密码', pwdSubmit: '确认修改', pwdTip: '忘记密码？停止服务后运行 fuwari-server -re pwd 命令行重置。',
      themeTip: '点击卡片即可切换主题；主题文件位于 themes/ 目录，修改后刷新即生效。',
      systemTitle: 'ℹ️ 系统信息', aboutTitle: '🍥 关于', aboutTip: 'Go 后端 + 内嵌 Astro 前端单二进制博客系统。后台 UI 与前台共用同一套样式与主题变量。',
      draftTag: '草稿', saved: '已保存', deleted: '已删除', loadFail: '加载失败', uploadFail: '上传失败', networkErr: '网络错误，请重试',
      newMode: '新文章模式：保存后自动生成 slug', needTitle: '请填写标题', needToken: '请先填写管理员密码', noSelection: '未选择文章',
      pwdEnter: '请输入当前密码', pwdShort: '新密码至少 6 个字符', pwdMismatch: '两次输入的新密码不一致', pwdOk: '✅ 密码已修改，请使用新密码', pwdFail: '修改失败',
      themeSwitch: '切换主题', tokenSaved: '✅ 管理员密码已保存', authOk: '🔓 已认证', authNo: '🔒 未认证', confirmDelete: '确定删除',
      langLabel: '语言'
    },
    en: {
      tabHome: '🏠 Home', tabPosts: '📝 Posts', tabComments: '💬 Comments', tabThemes: '🎨 Themes', tabPassword: '🔑 Password', tabSystem: 'ℹ️ System',
      siteRole: 'Admin Console', openSite: '🌐 Open Site',
      quickTitle: '⚡ Quick Actions', quickNew: '✍️ New Post', quickComments: '💬 Comments', quickThemes: '🎨 Themes', quickPwd: '🔑 Password', quickSys: 'ℹ️ System Info',
      sysTitle: '🖥️ System Status', sysVersion: 'Version', sysUptime: 'Uptime', sysDatabase: 'Database', sysPort: 'Port',
      welcomeTitle: '👋 Welcome back', welcomeTip: 'This is the site admin panel. Use the left column to start writing and maintaining, below is a live overview of the site.',
      statPosts: 'Posts', statDrafts: 'Drafts', statComments: 'Comments', statThemes: 'Themes',
      recentTitle: '🕒 Recent Posts', emptyRecent: 'No posts yet', emptyPosts: 'No posts — click "＋ New"', emptyComments: 'No comments', emptyThemes: 'No themes',
      editorTitle: '✍️ Editor', phSearch: 'Search title or slug…', btnNew: '＋ New', btnSave: 'Save', btnDelete: 'Delete', draftLabel: 'Draft (not published)',
      phTitle: 'Title *', phCategory: 'Category', phTags: 'Tags (comma separated)', phDescription: 'Description', phCommentFilter: 'Filter by post slug (empty = all)', btnLoad: 'Load',
      pwdTitle: '🔑 Change Admin Password', pwdOld: 'Current password', pwdNew: 'New password (min 6 chars)', pwdNew2: 'Confirm new password', pwdSubmit: 'Change', pwdTip: 'Forgot it? Stop the service and run fuwari-server -re pwd to reset from the command line.',
      themeTip: 'Click a card to switch theme; theme files live in themes/, edit and refresh.',
      systemTitle: 'ℹ️ System Info', aboutTitle: '🍥 About', aboutTip: 'A Go-backend blog system with an embedded Astro frontend. The admin UI shares the exact same styles and theme variables as the frontend.',
      draftTag: 'Draft', saved: 'Saved', deleted: 'Deleted', loadFail: 'Load failed', uploadFail: 'Upload failed', networkErr: 'Network error, please retry',
      newMode: 'New post mode: slug is auto-generated on save', needTitle: 'Please fill in the title', needToken: 'Please enter the admin password first', noSelection: 'No post selected',
      pwdEnter: 'Enter the current password', pwdShort: 'New password must be at least 6 characters', pwdMismatch: 'The two passwords do not match', pwdOk: '✅ Password updated, use the new one', pwdFail: 'Change failed',
      themeSwitch: 'Switch theme', tokenSaved: '✅ Admin password saved', authOk: '🔓 Authed', authNo: '🔒 Not authed', confirmDelete: 'Delete',
      langLabel: 'Language'
    }
  };
  var lang = localStorage.getItem('fuwari_lang') || 'zh'; // 首次部署默认中文
  function T(k) { return (I18N[lang] && I18N[lang][k]) || k; }
  function applyI18n() {
    document.querySelectorAll('[data-i18n]').forEach(function (el) { el.textContent = T(el.dataset.i18n); });
    document.querySelectorAll('[data-i18n-ph]').forEach(function (el) { el.placeholder = T(el.dataset.i18nPh); });
    document.title = lang === 'zh' ? 'Fuwari 管理后台' : 'Fuwari Admin';
    var ls = document.querySelectorAll('[id="lang-switch"],[id="lang-switch-m"]');
    ls.forEach(function (s) { if (s.value !== lang) s.value = lang; });
  }

  function el(id) { return document.getElementById(id); }

  /* ---------- 轻提示 ---------- */
  function toast(msg, isErr) {
    var t = document.createElement('div');
    t.className = 'toast' + (isErr ? ' toast-err' : '');
    t.textContent = msg;
    document.body.appendChild(t);
    setTimeout(function () { t.classList.add('show'); }, 10);
    setTimeout(function () {
      t.classList.remove('show');
      setTimeout(function () { t.remove(); }, 320);
    }, 2600);
  }

  /* ---------- 认证 ---------- */
  function getToken() {
    return localStorage.getItem(TOKEN_KEY) || '';
  }
  function setToken(v) {
    var s = String(v || '').trim();
    if (s) localStorage.setItem(TOKEN_KEY, s); else localStorage.removeItem(TOKEN_KEY);
    var a = el('token'); if (a && a.value !== s) a.value = s;
    var m = el('token-m'); if (m && m.value !== s) m.value = s;
    updateAuthBadge();
  }
  function updateAuthBadge() {
    var b = el('auth-badge');
    if (!b) return;
    var ok = !!getToken();
    b.textContent = ok ? '🔓' : '🔒';
    b.classList.toggle('badge-ok', ok);
    b.title = ok ? T('authOk') : T('authNo');
  }
  function authHeaders(extra) {
    var h = { 'Content-Type': 'application/json', 'X-Admin-Token': getToken() };
    if (extra) Object.keys(extra).forEach(function (k) { h[k] = extra[k]; });
    return h;
  }
  function handleRes(res) {
    return res.json().then(function (json) {
      if (json && json.code === 0) return json.data;
      var err = new Error((json && json.message) || T('loadFail'));
      err.needToken = (json && (json.code === 2 || json.code === 3));
      throw err;
    });
  }

  /* ---------- 视图解析（真实子页面 URL：/admin → home，/admin/posts → posts …） ---------- */
  function resolveView() {
    var p = window.location.pathname.replace(/\/+$/, '');
    var m = p.match(/\/(posts|comments|themes|password|system)$/);
    if (m) return m[1];
    return 'home';
  }

  function activateView(name) {
    document.querySelectorAll('#tabs a, #nav-menu-panel a').forEach(function (b) {
      b.classList.toggle('active', b.dataset.tab === name);
    });
    document.querySelectorAll('.fw-view').forEach(function (v) { v.classList.remove('active'); });
    var view = el('view-' + name);
    if (view) view.classList.add('active');
    if (name === 'posts') {
      if (!cherryInit && window.Cherry) initCherry();
      var qs = new URLSearchParams(location.search);
      if (qs.get('new') === '1') newPost();
      else if (qs.get('slug')) openPost(qs.get('slug'));
      else loadList();
    }
    if (name === 'comments') loadComments();
    if (name === 'themes') loadThemeGrid();
    if (name === 'system') loadSystemInfo();
    if (name === 'home') loadDashboard();
  }

  /* ---------- 面板首页（dashboard） ---------- */
  function loadDashboard() {
    fetch(API + '/posts?include_draft=1').then(function (r) { return r.json(); }).then(function (j) {
      if (j.code !== 0) return;
      var list = j.data.list || [];
      var posts = list.filter(function (p) { return !p.draft; }).length;
      el('stat-posts').textContent = posts;
      el('stat-drafts').textContent = list.length - posts;
      var box = el('recent-posts');
      box.innerHTML = '';
      if (!list.length) { box.innerHTML = '<div class="empty-tip py-4">' + T('emptyRecent') + '</div>'; return; }
      list.slice(0, 5).forEach(function (p) {
        var a = document.createElement('a');
        a.href = BASE + 'admin/posts?slug=' + encodeURIComponent(p.slug);
        a.className = 'flex items-center gap-2 px-3 py-2.5 rounded-lg hover:bg-[var(--btn-plain-bg-hover)] active:bg-[var(--btn-plain-bg-active)] transition-all text-sm';
        a.innerHTML =
          '<span class="font-medium truncate min-w-0">' + escapeHtml(p.title) + '</span>' +
          '<span class="ml-auto text-xs shrink-0 ' + (p.draft ? 'badge badge-draft' : 'text-[var(--meta-divider)]') + '">' +
          (p.draft ? T('draftTag') : escapeHtml(p.published || '')) + '</span>';
        box.appendChild(a);
      });
    }).catch(function () {});
    fetch(API + '/comments?page=1&page_size=1').then(function (r) { return r.json(); }).then(function (j) {
      if (j.code === 0) el('stat-comments').textContent = j.data.total || 0;
    }).catch(function () {});
    fetch(API + '/themes').then(function (r) { return r.json(); }).then(function (j) {
      if (j.code === 0) el('stat-themes').textContent = (j.data.list || []).length;
    }).catch(function () {});
    fetch(API + '/health').then(function (r) { return r.json(); }).then(function (j) {
      var d = j.data || {};
      var box = el('sys-mini');
      box.innerHTML = '';
      [
        [T('sysVersion'), d.version || ''],
        [T('sysUptime'), d.uptime ? (d.uptime + 's') : ''],
        [T('sysDatabase'), d.database || ''],
        [T('sysPort'), d.port || ''],
      ].forEach(function (r2) {
        var row = document.createElement('div');
        row.className = 'sys-row';
        row.innerHTML = '<span>' + escapeHtml(r2[0]) + '</span><span>' + escapeHtml(String(r2[1])) + '</span>';
        box.appendChild(row);
      });
    }).catch(function () {});
  }

  /* ---------- 主题切换 ---------- */
  function loadThemes() {
    fetch(API + '/themes')
      .then(function (r) { return r.json(); })
      .then(function (json) {
        if (json.code !== 0) return;
        var list = json.data.list || [];
        document.querySelectorAll('[id="theme-switch"],[id="theme-switch-m"]').forEach(function (sel) {
          var prev = sel.value;
          sel.innerHTML = '';
          list.forEach(function (t) {
            var opt = document.createElement('option');
            opt.value = t.name;
            opt.textContent = t.name + (t.description ? ' · ' + t.description : '');
            if (t.active) opt.selected = true;
            sel.appendChild(opt);
          });
          if (prev && list.some(function (t) { return t.name === prev; })) sel.value = prev;
        });
      })
      .catch(function () {});
  }
  function applyTheme(name) {
    fetch(API + '/theme', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ theme: name }),
    }).then(function () { location.reload(); }).catch(function () { location.reload(); });
  }
  function loadThemeGrid() {
    fetch(API + '/themes')
      .then(function (r) { return r.json(); })
      .then(function (json) {
        if (json.code !== 0) return;
        var box = el('theme-grid');
        box.innerHTML = '';
        var list = json.data.list || [];
        if (!list.length) { box.innerHTML = '<div class="empty-tip">' + T('emptyThemes') + '</div>'; return; }
        list.forEach(function (t) {
          var card = document.createElement('div');
          card.className = 'theme-card' + (t.active ? ' active' : '');
          card.innerHTML =
            '<div class="nm">' + escapeHtml(t.name) + (t.active ? ' <span class="badge badge-ok">✓</span>' : '') + '</div>' +
            (t.description ? '<div class="ds">' + escapeHtml(t.description) + '</div>' : '') +
            (t.author ? '<div class="ds">by ' + escapeHtml(t.author) + '</div>' : '');
          card.onclick = function () {
            if (t.active) return;
            if (t.name === 'default' || confirm(T('themeSwitch') + ': ' + t.name + '?')) applyTheme(t.name);
          };
          box.appendChild(card);
        });
      })
      .catch(function () {});
  }

  /* ---------- 文章 ---------- */
  var postCache = [];
  function loadList() {
    fetch(API + '/posts?include_draft=1')
      .then(function (r) { return r.json(); })
      .then(function (json) {
        if (json.code !== 0) return;
        postCache = json.data.list || [];
        renderList();
      })
      .catch(function () { toast(T('loadFail'), true); });
  }
  function renderList() {
    var kw = (el('post-search').value || '').trim().toLowerCase();
    var list = postCache.filter(function (p) {
      if (!kw) return true;
      return (p.title || '').toLowerCase().indexOf(kw) >= 0 || (p.slug || '').toLowerCase().indexOf(kw) >= 0;
    });
    var box = el('post-list');
    box.innerHTML = '';
    if (!list.length) { box.innerHTML = '<div class="empty-tip">' + T('emptyPosts') + '</div>'; return; }
    list.forEach(function (p) {
      var item = document.createElement('div');
      item.className = 'post-item' + (p.slug === currentSlug ? ' active' : '');
      item.innerHTML =
        '<span class="t">' + escapeHtml(p.title) + '</span>' +
        (p.draft ? '<span class="badge badge-draft">' + T('draftTag') + '</span>' : '') +
        '<span class="s">' + escapeHtml(p.slug) + '</span>';
      item.onclick = function () { openPost(p.slug); };
      box.appendChild(item);
    });
  }
  function openPost(slug) {
    fetch(API + '/posts/' + encodeURIComponent(slug))
      .then(function (r) { return r.json(); })
      .then(function (json) {
        if (json.code !== 0) return;
        var p = json.data;
        currentSlug = p.slug;
        el('f-title').value = p.title || '';
        el('f-category').value = p.category || '';
        el('f-tags').value = (p.tags || []).join(', ');
        el('f-description').value = p.description || '';
        el('f-draft').checked = !!p.draft;
        if (cherry) cherry.setValue(p.body || '');
        renderList();
      })
      .catch(function () { toast(T('loadFail'), true); });
  }
  function resetForm() {
    el('f-title').value = ''; el('f-category').value = ''; el('f-tags').value = '';
    el('f-description').value = ''; el('f-draft').checked = false;
    if (cherry) cherry.setValue('');
  }
  function newPost() {
    currentSlug = null;
    resetForm();
    toast(T('newMode'));
    renderList();
  }
  function savePost() {
    var title = el('f-title').value.trim();
    if (!title) { toast(T('needTitle'), true); return; }
    var body = cherry ? cherry.getMarkdown() : '';
    var payload = {
      title: title,
      description: el('f-description').value.trim(),
      category: el('f-category').value.trim(),
      tags: el('f-tags').value.split(/[,，]/).map(function (s) { return s.trim(); }).filter(Boolean),
      draft: el('f-draft').checked,
      body: body,
    };
    var url = API + '/posts';
    var method = 'POST';
    if (currentSlug) { url = API + '/posts/' + encodeURIComponent(currentSlug); method = 'PUT'; }
    fetch(url, { method: method, headers: authHeaders(), body: JSON.stringify(payload) })
      .then(handleRes)
      .then(function (data) {
        currentSlug = data.slug || currentSlug;
        toast(T('saved') + ': ' + currentSlug);
        loadList();
      })
      .catch(function (e) {
        if (e.needToken) { toast(T('needToken'), true); openAdminPanel(); }
        else toast(e.message, true);
      });
  }
  function deletePost() {
    if (!currentSlug) { toast(T('noSelection'), true); return; }
    if (!confirm(T('confirmDelete') + ' "' + currentSlug + '"?')) return;
    fetch(API + '/posts/' + encodeURIComponent(currentSlug), { method: 'DELETE', headers: authHeaders() })
      .then(handleRes)
      .then(function () {
        currentSlug = null;
        resetForm();
        toast(T('deleted'));
        loadList();
      })
      .catch(function (e) {
        if (e.needToken) { toast(T('needToken'), true); openAdminPanel(); }
        else toast(e.message, true);
      });
  }
  function initCherry() {
    if (cherry) return;
    cherryInit = true;
    // 动态加载 Cherry 样式（与评论挂件一致，FUWARI_BASE 拼接，严禁硬编码绝对路径）
    if (!document.querySelector('link[data-cherry-css]')) {
      var link = document.createElement('link');
      link.rel = 'stylesheet';
      link.href = BASE + 'assets/cherry/cherry-markdown.min.css';
      link.setAttribute('data-cherry-css', '1');
      document.head.appendChild(link);
    }
    cherry = new Cherry({
      id: 'editor',
      value: '',
      editor: { defaultModel: 'edit&preview' },
      image: {
        accept: 'jpg|jpeg|png|gif|webp|avif',
        uploadHandler: function (files, callback) {
          if (!files || !files.length) { callback({}); return; }
          var fd = new FormData();
          fd.append('file', files[0]);
          fetch(API + '/admin/upload', { method: 'POST', headers: { 'X-Admin-Token': getToken() }, body: fd })
            .then(function (r) { return r.json(); })
            .then(function (json) {
              if (json.code === 0) { callback({ url: json.data.url }); }
              else { toast(T('uploadFail') + ': ' + json.message, true); callback({}); }
            })
            .catch(function () { toast(T('networkErr'), true); callback({}); });
        },
      },
      toolbars: { showToolbar: true },
    });
  }

  /* ---------- 评论管理 ---------- */
  function loadComments() {
    var slug = el('cmt-slug').value.trim();
    var q = slug ? '?slug=' + encodeURIComponent(slug) : '';
    fetch(API + '/comments' + q)
      .then(function (r) { return r.json(); })
      .then(function (json) {
        if (json.code !== 0) return;
        var list = json.data.list || [];
        var box = el('cmt-list');
        box.innerHTML = '';
        if (!list.length) { box.innerHTML = '<div class="empty-tip">' + T('emptyComments') + '</div>'; return; }
        list.forEach(function (cm) {
          var item = document.createElement('div');
          item.className = 'list-item';
          item.innerHTML =
            '<div class="head"><span class="nm">' + escapeHtml(cm.nickname || '') + '</span>' +
            '<span class="tm">' + escapeHtml(cm.target_slug || '') + ' · ' + escapeHtml(cm.created_at || '') + '</span></div>' +
            '<div class="ct">' + escapeHtml(cm.content || '') + '</div>' +
            '<div class="act"><button class="fw-btn-danger rounded-lg h-9 px-4 text-sm" data-id="' + cm.id + '">' + T('btnDelete') + '</button></div>';
          item.querySelector('button').onclick = function () {
            if (!confirm(T('confirmDelete') + ' comment #' + cm.id + '?')) return;
            fetch(API + '/comments/' + cm.id, { method: 'DELETE', headers: authHeaders() })
              .then(handleRes)
              .then(function () { toast(T('deleted')); loadComments(); })
              .catch(function (e) {
                if (e.needToken) { toast(T('needToken'), true); openAdminPanel(); }
                else toast(e.message, true);
              });
          };
          box.appendChild(item);
        });
      })
      .catch(function () { toast(T('loadFail'), true); });
  }

  /* ---------- 密码 ---------- */
  function submitPwdChange() {
    var oldP = el('pwd-old').value.trim();
    var newP = el('pwd-new').value;
    var new2 = el('pwd-new2').value;
    var msg = el('pwd-msg');
    msg.classList.remove('ok');
    if (!oldP) { msg.textContent = T('pwdEnter'); return; }
    if (newP.length < 6) { msg.textContent = T('pwdShort'); return; }
    if (newP !== new2) { msg.textContent = T('pwdMismatch'); return; }
    fetch(API + '/admin/password', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Admin-Token': oldP },
      body: JSON.stringify({ old_password: oldP, new_password: newP }),
    })
      .then(handleRes)
      .then(function () {
        setToken(newP);
        msg.classList.add('ok');
        msg.textContent = T('pwdOk');
        toast(T('pwdOk'));
      })
      .catch(function (e) { msg.textContent = (e.message || T('pwdFail')); });
  }

  /* ---------- 系统 ---------- */
  function loadSystemInfo() {
    var box = el('sys-info');
    box.innerHTML = '<div class="empty-tip">' + T('systemTitle') + '…</div>';
    fetch(API + '/health')
      .then(function (r) { return r.json(); })
      .then(function (json) {
        var d = json.data || {};
        box.innerHTML = '';
        [
          ['Version', d.version || ''],
          ['Host', d.hostname || ''],
          ['Uptime', d.uptime ? (d.uptime + 's') : ''],
          ['Posts Dir', d.posts_dir || ''],
          ['Themes', (d.themes && d.themes.join(', ')) || ''],
          ['Extensions', (d.extensions && d.extensions.join(', ')) || ''],
          ['Database', d.database || ''],
        ].forEach(function (r) {
          var row = document.createElement('div');
          row.className = 'sys-row';
          row.innerHTML = '<span>' + escapeHtml(r[0]) + '</span><span>' + escapeHtml(String(r[1])) + '</span>';
          box.appendChild(row);
        });
      })
      .catch(function () { box.innerHTML = '<div class="empty-tip">' + T('loadFail') + '</div>'; });
  }

  /* ---------- 浮层面板（前台 float-panel 语言） ---------- */
  function togglePanel(id) {
    var p = el(id);
    if (p) p.classList.toggle('float-panel-closed');
  }
  function openAdminPanel() {
    el('admin-panel').classList.remove('float-panel-closed');
    var m = el('token-m'); if (m) m.focus();
  }
  function bindClickOutside(panelId, ignoreIds) {
    document.addEventListener('click', function (e) {
      var panel = el(panelId);
      if (!panel || panel.classList.contains('float-panel-closed')) return;
      var t = e.target;
      if (!(t instanceof Node)) return;
      for (var i = 0; i < ignoreIds.length; i++) {
        var ig = el(ignoreIds[i]);
        if (ig && (ig === t || ig.contains(t))) return;
      }
      panel.classList.add('float-panel-closed');
    });
  }

  function escapeHtml(s) {
    return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
  }

  /* ---------- init ---------- */
  function init() {
    // 认证
    setToken(getToken());
    el('token').addEventListener('change', function () { setToken(this.value); });
    el('token-m').addEventListener('keydown', function (e) { if (e.key === 'Enter') el('token-ok').click(); });
    el('token-ok').addEventListener('click', function () {
      setToken(el('token-m').value);
      toast(T('tokenSaved'));
      el('admin-panel').classList.add('float-panel-closed');
    });

    // 浮层面板
    el('admin-switch').addEventListener('click', function (e) { e.stopPropagation(); togglePanel('admin-panel'); });
    el('nav-menu-switch').addEventListener('click', function (e) { e.stopPropagation(); togglePanel('nav-menu-panel'); });
    bindClickOutside('admin-panel', ['admin-panel', 'admin-switch']);
    bindClickOutside('nav-menu-panel', ['nav-menu-panel', 'nav-menu-switch']);

    // 深浅色（桌面 + 移动）
    el('scheme-switch').addEventListener('click', toggleTheme);
    el('scheme-switch-m').addEventListener('click', toggleTheme);

    // 主题 / 语言（桌面 + 移动同步）
    document.querySelectorAll('[id="theme-switch"],[id="theme-switch-m"]').forEach(function (sel) {
      sel.addEventListener('change', function (e) {
        document.querySelectorAll('[id="theme-switch"],[id="theme-switch-m"]').forEach(function (o) { if (o !== e.target) o.value = e.target.value; });
        applyTheme(e.target.value);
      });
    });
    document.querySelectorAll('[id="lang-switch"],[id="lang-switch-m"]').forEach(function (sel) {
      sel.addEventListener('change', function (e) {
        lang = e.target.value;
        localStorage.setItem('fuwari_lang', lang);
        applyI18n();
        activateView(resolveView());
      });
    });

    // 文章
    el('btn-save').addEventListener('click', savePost);
    el('btn-new').addEventListener('click', function (e) { e.preventDefault(); newPost(); });
    el('btn-delete').addEventListener('click', deletePost);
    el('post-search').addEventListener('input', renderList);
    el('pwd-ok').addEventListener('click', submitPwdChange);
    el('cmt-load').addEventListener('click', loadComments);
    el('cmt-slug').addEventListener('keydown', function (e) { if (e.key === 'Enter') loadComments(); });
    el('f-title').addEventListener('keydown', function (e) { if (e.key === 'Enter') { e.preventDefault(); el('f-category').focus(); } });

    applyHue();
    syncTheme();
    applyI18n();
    updateAuthBadge();
    loadThemes();
    activateView(resolveView());
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
