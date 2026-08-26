
(function () {
  'use strict';

  // ---------- 深浅色同步（与前台 localStorage 'theme' 完全一致，html.dark 驱动） ----------
  function syncTheme() {
    var t = localStorage.getItem('theme') || 'auto';
    var dark = t === 'dark' || (t === 'auto' && window.matchMedia('(prefers-color-scheme: dark)').matches);
    document.documentElement.classList.toggle('dark', dark);
    var btn = document.getElementById('scheme-switch');
    if (btn) btn.textContent = dark ? '☀️' : '🌙';
  }
  function toggleTheme() {
    var cur = document.documentElement.classList.contains('dark');
    localStorage.setItem('theme', cur ? 'light' : 'dark');
    syncTheme();
  }
  if (window.matchMedia) {
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', syncTheme);
  }

  // ---------- 站点主色 hue（与前台 configHue 一致：html style --configHue / localStorage hue） ----------
  function applyHue() {
    var cfg = parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--configHue')) || 250;
    var hue = localStorage.getItem('hue');
    if (hue !== null && !isNaN(parseFloat(hue))) cfg = parseFloat(hue);
    document.documentElement.style.setProperty('--hue', cfg);
  }

  var API = (window.FUWARI_BASE || '/') + 'api';
  var currentSlug = null;
  var cherry = null;
  var cherryInit = false;
  var TOKEN_KEY = 'fuwari_admin_token';

  // ---------- i18n ----------
  var I18N = {
    zh: {
      tabPosts: '📝 文章', tabComments: '💬 评论', tabThemes: '🎨 主题', tabPassword: '🔑 密码', tabSystem: 'ℹ️ 系统',
      btnNew: '＋ 新建', btnSave: '保存', btnDelete: '删除', draftLabel: '草稿（不对外发布）',
      phTitle: '标题 *', phCategory: '分类', phTags: '标签（逗号分隔）', phDescription: '摘要描述', phCommentFilter: '按文章 slug 过滤（留空=全部）',
      btnLoad: '加载', pwdTitle: '🔑 修改管理员密码', pwdOld: '当前密码', pwdNew: '新密码（至少 6 个字符）', pwdNew2: '确认新密码', pwdSubmit: '确认修改',
      emptyPosts: '暂无文章，点击「＋ 新建」开始', emptyComments: '暂无评论', emptyThemes: '暂无主题', loadFail: '加载失败', saved: '已保存', deleted: '已删除',
      newMode: '新文章模式（保存后自动生成 slug）', needTitle: '请填写标题', needToken: '请先填写管理员密码', noSelection: '未选择文章',
      pwdEnter: '请输入当前密码', pwdShort: '新密码至少 6 个字符', pwdMismatch: '两次输入的新密码不一致', pwdOk: '✅ 密码已修改，请使用新密码', pwdFail: '修改失败', networkErr: '网络错误，请重试',
      uploadFail: '上传失败', ready: '就绪', systemTitle: '系统信息', themeSwitch: '切换主题', themeActive: '当前', langLabel: '语言',
      opsTitle: '⚙️ 操作', infoTitle: '📄 文章信息', filterTitle: '🔍 过滤',
      themeTipTitle: '💡 说明', themeTip: '点击卡片即可切换主题；主题文件位于 themes/ 目录，修改后刷新即生效。',
      pwdTipTitle: '🔐 忘记密码？', pwdTip: '停止服务后运行 fuwari-server -re pwd 命令行重置。',
      aboutTitle: '🍥 Fuwari', aboutTip: 'Go 后端 + 内嵌 Astro 前端单二进制博客系统。后台 UI 与前台共用同一套样式与主题变量。'
    },
    en: {
      tabPosts: '📝 Posts', tabComments: '💬 Comments', tabThemes: '🎨 Themes', tabPassword: '🔑 Password', tabSystem: 'ℹ️ System',
      btnNew: '＋ New', btnSave: 'Save', btnDelete: 'Delete', draftLabel: 'Draft (not published)',
      phTitle: 'Title *', phCategory: 'Category', phTags: 'Tags (comma separated)', phDescription: 'Description', phCommentFilter: 'Filter by post slug (empty = all)',
      btnLoad: 'Load', pwdTitle: '🔑 Change Admin Password', pwdOld: 'Current password', pwdNew: 'New password (min 6 chars)', pwdNew2: 'Confirm new password', pwdSubmit: 'Change',
      emptyPosts: 'No posts yet — click "＋ New"', emptyComments: 'No comments', emptyThemes: 'No themes', loadFail: 'Load failed', saved: 'Saved', deleted: 'Deleted',
      newMode: 'New post mode (slug auto-generated on save)', needTitle: 'Please fill in the title', needToken: 'Please enter the admin password first', noSelection: 'No post selected',
      pwdEnter: 'Enter the current password', pwdShort: 'New password must be at least 6 characters', pwdMismatch: 'The two passwords do not match', pwdOk: '✅ Password updated, use the new one', pwdFail: 'Change failed', networkErr: 'Network error, please retry',
      uploadFail: 'Upload failed', ready: 'Ready', systemTitle: 'System Info', themeSwitch: 'Switch theme', themeActive: 'Active', langLabel: 'Language',
      opsTitle: '⚙️ Actions', infoTitle: '📄 Post Info', filterTitle: '🔍 Filter',
      themeTipTitle: '💡 Tips', themeTip: 'Click a card to switch theme; theme files live in themes/, edit and refresh.',
      pwdTipTitle: '🔐 Forgot it?', pwdTip: 'Stop the service and run fuwari-server -re pwd to reset from the command line.',
      aboutTitle: '🍥 Fuwari', aboutTip: 'A Go-backend blog system with an embedded Astro frontend. The admin UI shares the exact same styles and theme variables as the frontend.'
    }
  };
  var lang = localStorage.getItem('fuwari_lang') || ((navigator.language || '').toLowerCase().startsWith('zh') ? 'zh' : 'en');
  function T(k) { return (I18N[lang] && I18N[lang][k]) || k; }
  function applyI18n() {
    document.querySelectorAll('[data-i18n]').forEach(function (el) { el.textContent = T(el.dataset.i18n); });
    document.querySelectorAll('[data-i18n-ph]').forEach(function (el) { el.placeholder = T(el.dataset.i18nPh); });
    document.title = lang === 'zh' ? 'Fuwari 管理后台' : 'Fuwari Admin';
  }
  document.addEventListener('DOMContentLoaded', function () {
    el('lang-switch').value = lang;
    applyI18n();
  });

  function el(id) { return document.getElementById(id); }
  function getToken() {
    var t = el('token').value.trim();
    if (!t) t = localStorage.getItem(TOKEN_KEY) || '';
    return t;
  }
  function setStatus(msg, isErr) {
    var s = el('status');
    s.textContent = msg;
    s.style.color = isErr ? 'var(--fw-err,#ef4444)' : 'var(--meta-divider,#6b7280)';
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

  // ---------- Tabs ----------
  function switchTab(name) {
    document.querySelectorAll('#tabs button').forEach(function (b) { b.classList.toggle('active', b.dataset.tab === name); });
    document.querySelectorAll('.fw-view').forEach(function (v) { v.classList.remove('active'); });
    el('view-' + name).classList.add('active');
    if (name === 'posts' && !cherryInit && window.Cherry) initCherry();
    if (name === 'comments') loadComments();
    if (name === 'themes') loadThemeGrid();
    if (name === 'system') loadSystemInfo();
  }

  // ---------- 主题切换 ----------
  function loadThemes() {
    var sel = el('theme-switch');
    fetch(API + '/themes')
      .then(function (r) { return r.json(); })
      .then(function (json) {
        if (json.code !== 0) return;
        var list = json.data.list || [];
        sel.innerHTML = '';
        list.forEach(function (t) {
          var opt = document.createElement('option');
          opt.value = t.name;
          opt.textContent = t.name + (t.description ? ' - ' + t.description : '');
          if (t.active) opt.selected = true;
          sel.appendChild(opt);
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
            '<div class="nm">' + t.name + (t.active ? ' ✓' : '') + '</div>' +
            (t.description ? '<div class="ds">' + t.description + '</div>' : '') +
            (t.author ? '<div class="ds">by ' + t.author + '</div>' : '');
          card.onclick = function () {
            if (t.name === 'default' || confirm(T('themeSwitch') + ': ' + t.name + '?')) applyTheme(t.name);
          };
          box.appendChild(card);
        });
      })
      .catch(function () {});
  }

  // ---------- 文章 ----------
  function loadList() {
    fetch(API + '/posts?include_draft=1')
      .then(function (r) { return r.json(); })
      .then(function (json) {
        if (json.code !== 0) return;
        var list = json.data.list || [];
        var box = el('post-list');
        box.innerHTML = '';
        if (!list.length) { box.innerHTML = '<div class="empty">' + T('emptyPosts') + '</div>'; return; }
        list.forEach(function (p) {
          var item = document.createElement('div');
          item.className = 'post-item' + (p.slug === currentSlug ? ' active' : '');
          item.innerHTML = '<div class="t">' + escapeHtml(p.title) + '</div>' +
            '<div class="s">' + escapeHtml(p.slug) + (p.draft ? ' · draft' : '') + '</div>';
          item.onclick = function () { openPost(p.slug); };
          box.appendChild(item);
        });
      })
      .catch(function () { setStatus(T('loadFail'), true); });
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
        loadList();
        setStatus(T('saved') ? '' : '');
      })
      .catch(function () { setStatus(T('loadFail'), true); });
  }
  function resetForm() {
    el('f-title').value = ''; el('f-category').value = ''; el('f-tags').value = '';
    el('f-description').value = ''; el('f-draft').checked = false;
    if (cherry) cherry.setValue('');
  }
  function newPost() {
    currentSlug = null;
    resetForm();
    setStatus(T('newMode'));
    loadList();
  }
  function savePost() {
    var title = el('f-title').value.trim();
    if (!title) { setStatus(T('needTitle'), true); return; }
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
        setStatus(T('saved') + ': ' + currentSlug);
        loadList();
      })
      .catch(function (e) {
        if (e.needToken) setStatus(T('needToken'), true);
        else setStatus(e.message, true);
      });
  }
  function deletePost() {
    if (!currentSlug) { setStatus(T('noSelection'), true); return; }
    if (!confirm('Delete "' + currentSlug + '"?')) return;
    fetch(API + '/posts/' + encodeURIComponent(currentSlug), { method: 'DELETE', headers: authHeaders() })
      .then(handleRes)
      .then(function () {
        currentSlug = null;
        resetForm();
        setStatus(T('deleted'));
        loadList();
      })
      .catch(function (e) {
        if (e.needToken) setStatus(T('needToken'), true);
        else setStatus(e.message, true);
      });
  }
  function initCherry() {
    if (cherry) return;
    cherryInit = true;
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
              else { setStatus(T('uploadFail') + ': ' + json.message, true); callback({}); }
            })
            .catch(function () { setStatus(T('networkErr'), true); callback({}); });
        },
      },
      toolbars: { showToolbar: true },
    });
  }

  // ---------- 评论管理 ----------
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
            '<div class="act"><button class="btn-mini btn-danger" data-id="' + cm.id + '">' + (lang === 'zh' ? '删除' : 'Delete') + '</button></div>';
          item.querySelector('button').onclick = function () {
            if (!confirm('Delete comment #' + cm.id + '?')) return;
            fetch(API + '/comments/' + cm.id, { method: 'DELETE', headers: authHeaders() })
              .then(handleRes)
              .then(function () { loadComments(); })
              .catch(function (e) { setStatus(e.needToken ? T('needToken') : e.message, true); });
          };
          box.appendChild(item);
        });
      })
      .catch(function () { setStatus(T('loadFail'), true); });
  }

  // ---------- 密码 ----------
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
        localStorage.setItem(TOKEN_KEY, newP);
        el('token').value = newP;
        msg.classList.add('ok');
        msg.textContent = T('pwdOk');
      })
      .catch(function (e) { msg.textContent = (e.message || T('pwdFail')) + (e.needToken ? '' : ''); });
  }

  // ---------- 系统 ----------
  function loadSystemInfo() {
    var box = el('sys-info');
    box.innerHTML = '<div class="empty-tip">' + T('systemTitle') + '…</div>';
    fetch(API + '/health')
      .then(function (r) { return r.json(); })
      .then(function (json) {
        var d = json.data || {};
        box.innerHTML = '';
        var rows = [
          ['Version', d.version || ''],
          ['Host', d.hostname || ''],
          ['Uptime', d.uptime ? (d.uptime + 's') : ''],
          ['Posts Dir', d.posts_dir || ''],
          ['Themes', (d.themes && d.themes.join(', ')) || ''],
          ['Extensions', (d.extensions && d.extensions.join(', ')) || ''],
          ['Database', d.database || ''],
        ];
        rows.forEach(function (r) {
          var row = document.createElement('div');
          row.className = 'sys-row';
          row.innerHTML = '<span>' + escapeHtml(r[0]) + '</span><span>' + escapeHtml(String(r[1])) + '</span>';
          box.appendChild(row);
        });
      })
      .catch(function () { box.innerHTML = '<div class="empty-tip">' + T('loadFail') + '</div>'; });
  }

  function escapeHtml(s) {
    return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
  }

  // ---------- init ----------
  function init() {
    document.querySelectorAll('#tabs button').forEach(function (b) {
      b.addEventListener('click', function () { switchTab(b.dataset.tab); });
    });
    el('btn-save').addEventListener('click', savePost);
    el('btn-new').addEventListener('click', newPost);
    el('btn-delete').addEventListener('click', deletePost);
    el('pwd-ok').addEventListener('click', submitPwdChange);
    el('cmt-load').addEventListener('click', loadComments);
    el('theme-switch').addEventListener('change', function (e) { applyTheme(e.target.value); });
    el('lang-switch').addEventListener('change', function (e) {
      lang = e.target.value;
      localStorage.setItem('fuwari_lang', lang);
      applyI18n();
      switchTab(document.querySelector('#tabs button.active').dataset.tab);
    });
    el('scheme-switch').addEventListener('click', toggleTheme);
    applyHue();
    syncTheme();
    el('token').addEventListener('keydown', function (e) { if (e.key === 'Enter') savePost(); });
    var saved = localStorage.getItem(TOKEN_KEY);
    if (saved) el('token').value = saved;
    loadThemes();
    loadList();
    setStatus(T('ready'));
    // 默认进入文章页签；URL ?tab=xxx 可直达
    var want = new URLSearchParams(location.search).get('tab');
    if (want && document.querySelector('#tabs button[data-tab="' + want + '"]')) switchTab(want);
    else if (window.Cherry) initCherry();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
