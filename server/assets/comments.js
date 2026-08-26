/* fuwari 评论挂件
 * 由服务端注入到文章页 </body> 前，不修改任何前端源码。
 * 依赖：/assets/cherry/cherry-markdown.core.js + cherry-markdown.min.css
 * 角色：Cherry 同时承担「评论内容渲染器」与「评论编辑器」。
 * 挂载策略：MutationObserver 监听 DOM，检测到 #post-container（swup 导航后同样适用）
 * 即自动挂载；已挂载（存在 #fuwari-comments）则跳过，保证幂等。
 */
(function () {
	'use strict';

	// 反代挂载前缀（服务端注入 window.FUWARI_BASE，如 "/name/"；无前缀回退 "/"）。
	// 所有 API/资源路径一律经此拼接，严禁硬编码绝对路径。
	var fwBase = window.FUWARI_BASE || '/';

	// 从 URL 推导文章 slug：/posts/guide/ -> guide ；/posts/a/b/ -> a/b
	// 兼容反代前缀（/name/posts/guide/）与普通路径。
	function detectSlug() {
		var m = window.location.pathname.match(/\/posts\/(.+?)\/?$/);
		return m ? decodeURIComponent(m[1]) : null;
	}

	var SLUG = null;
	var cherryReady = false;
	var cherryCtor = null;
	var editorInstance = null; // 评论编辑器的 Cherry 实例
	var renderEngine = null; // 评论渲染的 Cherry 实例（previewOnly）
	var renderHolder = null;

	function loadCherry(cb) {
		if (cherryReady) {
			cb();
			return;
		}
		var css = document.createElement('link');
		css.rel = 'stylesheet';
		css.href = fwBase + 'assets/cherry/cherry-markdown.min.css';
		document.head.appendChild(css);

		var s = document.createElement('script');
		s.src = fwBase + 'assets/cherry/cherry-markdown.core.js';
		s.onload = function () {
			cherryReady = true;
			cherryCtor = window.Cherry;
			cb();
		};
		s.onerror = function () {
			cb();
		};
		document.head.appendChild(s);
	}

	// 渲染队列：Cherry 渲染引擎共享，setValue 异步且互斥，必须串行渲染
	var renderQueue = [];
	var rendering = false;

	function renderMarkdown(md, cb) {
		renderQueue.push({ md: md, cb: cb });
		if (!rendering) {
			processRenderQueue();
		}
	}

	function processRenderQueue() {
		if (rendering || !renderQueue.length) { return; }
		var job = renderQueue.shift();
		rendering = true;
		renderOne(job.md, function (html) {
			rendering = false;
			job.cb(html);
			processRenderQueue();
		});
	}

	// 渲染单条评论 Markdown -> HTML（Cherry setValue 后监听 afterChange 事件，可靠获取渲染结果）
	function renderOne(md, cb) {
		loadCherry(function () {
			if (!cherryCtor) {
				cb(escapeHtml(md));
				return;
			}
			if (!renderEngine) {
				renderHolder = document.createElement('div');
				renderHolder.id = 'fuwari-renderer-host';
				renderHolder.style.cssText = 'position:fixed;top:0;left:0;width:600px;height:100px;z-index:-1;visibility:hidden';
				document.body.appendChild(renderHolder);
				try {
					renderEngine = new cherryCtor({
						id: 'fuwari-renderer-host',
						value: '',
						editor: { defaultModel: 'previewOnly' },
					});
				} catch (e) {
					renderEngine = null;
				}
			}
			if (!renderEngine) {
				cb(escapeHtml(md));
				return;
			}
			var done = false;
			var finalize = function () {
				if (done) { return; }
				try {
					var html = renderEngine.getHtml(false);
					// Cherry 初始化（value:''）也会触发一次 afterChange，此时结果为空串。
					// 仅当拿到真实渲染结果时才完成，避免空结果被 done 锁定。
					if (html && String(html).length > 0) {
						done = true;
						cb(String(html));
					}
				} catch (e) {
					done = true;
					cb(escapeHtml(md));
				}
			};
			// 事件驱动：渲染完成即取结果；超时兜底
			renderEngine.on('afterChange', finalize);
			try {
				renderEngine.setValue(md);
			} catch (e) {
				done = true;
				cb(escapeHtml(md));
				return;
			}
			setTimeout(finalize, 800);
		});
	}

	function escapeHtml(s) {
		var d = document.createElement('div');
		d.textContent = s;
		return d.innerHTML;
	}

	function fmtTime(iso) {
		try {
			var d = new Date(iso);
			if (isNaN(d.getTime())) {
				return iso;
			}
			return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
		} catch (e) {
			return iso;
		}
	}

	function fetchComments(cb) {
		fetch(fwBase + 'api/comments?slug=' + encodeURIComponent(SLUG) + '&page=1&page_size=100')
			.then(function (r) { return r.json(); })
			.then(function (json) {
				cb(json && json.code === 0 ? json.data : null);
			})
			.catch(function () { cb(null); });
	}

	function buildList(items, container) {
		container.innerHTML = '';
		if (!items || !items.length) {
			var empty = document.createElement('div');
			empty.className = 'fw-empty';
			empty.textContent = '暂无评论，来抢沙发吧～';
			container.appendChild(empty);
			return;
		}
		items.forEach(function (c) {
			var item = document.createElement('div');
			item.className = 'fw-item';

			var meta = document.createElement('div');
			meta.className = 'fw-meta';
			var av = document.createElement('span');
			av.className = 'fw-avatar';
			av.textContent = (c.nickname || '匿')[0].toUpperCase();
			var nick = document.createElement('span');
			nick.className = 'fw-nickname';
			nick.textContent = c.nickname || '匿名';
			var time = document.createElement('span');
			time.className = 'fw-time';
			time.textContent = fmtTime(c.created_at);
			meta.appendChild(av);
			meta.appendChild(nick);
			meta.appendChild(time);

			var content = document.createElement('div');
			content.className = 'fw-content';
			renderMarkdown(c.content || '', function (html) {
				content.innerHTML = html;
			});

			item.appendChild(meta);
			item.appendChild(content);
			container.appendChild(item);
		});
	}

	function submitComment(nickname, content, cb) {
		fetch(fwBase + 'api/comments', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ slug: SLUG, nickname: nickname, content: content }),
		})
			.then(function (r) { return r.json(); })
			.then(cb)
			.catch(function () { cb({ code: -1, message: '网络错误' }); });
	}

	function mount() {
		SLUG = detectSlug();
		if (!SLUG) {
			return;
		}
		if (document.getElementById('fuwari-comments')) {
			return;
		}
		var host = document.getElementById('post-container');
		if (!host) {
			return;
		}

		var wrap = document.createElement('section');
		wrap.id = 'fuwari-comments';

		var title = document.createElement('div');
		title.className = 'fw-title';
		var count = document.createElement('span');
		count.className = 'fw-count';
		count.textContent = '…';
		title.appendChild(document.createTextNode('💬 评论'));
		title.appendChild(count);
		wrap.appendChild(title);

		var list = document.createElement('div');
		list.className = 'fw-list';
		wrap.appendChild(list);

		var form = document.createElement('form');
		form.className = 'fw-form';
		form.innerHTML =
			'<div class="fw-form-row">' +
			'<input type="text" name="nickname" placeholder="昵称" maxlength="32" required>' +
			'</div>' +
			'<div class="fw-editor-wrap"><div id="fuwari-comment-editor"></div></div>' +
			'<div class="fw-actions">' +
			'<button type="submit" class="fw-submit">发表评论</button>' +
			'<span class="fw-hint">支持 Markdown 语法</span>' +
			'</div>';
		wrap.appendChild(form);

		host.insertAdjacentElement('afterend', wrap);

		// 加载评论列表
		fetchComments(function (data) {
			if (!data) {
				count.textContent = '0';
				return;
			}
			count.textContent = String(data.total || (data.list && data.list.length) || 0);
			buildList(data.list, list);
		});

		// Cherry 编辑器（懒加载）
		loadCherry(function () {
			var target = document.getElementById('fuwari-comment-editor');
			if (!target || !cherryCtor) {
				return;
			}
			try {
				editorInstance = new cherryCtor({
					id: 'fuwari-comment-editor',
					value: '',
					editor: { defaultModel: 'editOnly', height: '100%' },
					toolbars: {
						showToolbar: true,
						toolbar: ['bold', 'italic', 'strikethrough', 'quote', 'ul', 'ol', 'link', 'hr'],
						bubble: ['bold', 'italic', 'quote', 'link'],
						floatMenu: [],
					},
				});
			} catch (e) {
				editorInstance = null;
			}
		});

		form.addEventListener('submit', function (e) {
			e.preventDefault();
			var nickname = form.querySelector('input[name=nickname]').value.trim();
			var content = editorInstance ? editorInstance.getMarkdown() : '';
			if (!nickname || !content.trim()) {
				return;
			}
			var btn = form.querySelector('.fw-submit');
			btn.disabled = true;
			submitComment(nickname, content.trim(), function (res) {
				btn.disabled = false;
				if (res && res.code === 0) {
					if (editorInstance) {
						editorInstance.setValue('');
					}
					fetchComments(function (data) {
						count.textContent = data ? String(data.total || (data.list && data.list.length) || 0) : '0';
						buildList(data.list, list);
					});
				} else {
					var hint = form.querySelector('.fw-hint');
					hint.textContent = (res && res.message) || '发表失败';
					hint.className = 'fw-error';
				}
			});
		});
	}

	// 幂等挂载：MutationObserver 兜底（swup 替换 main 容器后同样触发）
	function scheduleMount() {
		var existing = document.getElementById('fuwari-comments');
		if (existing) {
			existing.remove();
		}
		mount();
	}

	if (document.readyState === 'loading') {
		document.addEventListener('DOMContentLoaded', function () {
			mount();
			startObserver();
		});
	} else {
		mount();
		startObserver();
	}

	function startObserver() {
		if (window.__fuwariObserver) {
			return;
		}
		var lastSlug = window.location.pathname;
		window.__fuwariObserver = new MutationObserver(function () {
			var p = window.location.pathname;
			if (p !== lastSlug) {
				lastSlug = p;
				scheduleMount();
				return;
			}
			var host = document.getElementById('post-container');
			if (host && !document.getElementById('fuwari-comments') && detectSlug()) {
				mount();
			}
		});
		window.__fuwariObserver.observe(document.body, { childList: true, subtree: true });
	}

	// swup 事件（v4 命名兼容）
	document.addEventListener('swup:content-replaced', scheduleMount);
	document.addEventListener('swup:content:replace', scheduleMount);
	document.addEventListener('swup:page:view', scheduleMount);
})();
