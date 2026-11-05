(() => {
  'use strict';

  const $ = (sel, root = document) => root.querySelector(sel);
  const $$ = (sel, root = document) => Array.from(root.querySelectorAll(sel));

  const iconPaths = {
    grid: '<rect x="3" y="3" width="7" height="9" rx="1.5"/><rect x="14" y="3" width="7" height="5" rx="1.5"/><rect x="14" y="12" width="7" height="9" rx="1.5"/><rect x="3" y="16" width="7" height="5" rx="1.5"/>',
    waypoints: '<circle cx="12" cy="4" r="2"/><circle cx="4" cy="20" r="2"/><circle cx="20" cy="20" r="2"/><path d="M12 6v3.5a2 2 0 0 1-2 2H6a2 2 0 0 0-2 2V18"/><path d="M12 6v3.5a2 2 0 0 0 2 2h4a2 2 0 0 1 2 2V18"/>',
    branch: '<line x1="6" x2="6" y1="3" y2="15"/><circle cx="18" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><path d="M18 9a9 9 0 0 1-9 9"/>',
    terminal: '<polyline points="4 17 10 11 4 5"/><line x1="12" x2="20" y1="19" y2="19"/>',
    braces: '<path d="M8 3H7a2 2 0 0 0-2 2v5a2 2 0 0 1-2 2 2 2 0 0 1 2 2v5a2 2 0 0 0 2 2h1"/><path d="M16 3h1a2 2 0 0 1 2 2v5a2 2 0 0 0 2 2 2 2 0 0 0-2 2v5a2 2 0 0 1-2 2h-1"/>',
    sliders: '<line x1="21" x2="14" y1="4" y2="4"/><line x1="10" x2="3" y1="4" y2="4"/><line x1="21" x2="12" y1="12" y2="12"/><line x1="8" x2="3" y1="12" y2="12"/><line x1="21" x2="16" y1="20" y2="20"/><line x1="12" x2="3" y1="20" y2="20"/><line x1="14" x2="14" y1="2" y2="6"/><line x1="8" x2="8" y1="10" y2="14"/><line x1="16" x2="16" y1="18" y2="22"/>',
    moon: '<path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8Z"/>',
    sun: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.93 4.93l1.42 1.42M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.42-1.42M17.66 6.34l1.41-1.41"/>',
    wifi: '<path d="M5 12.55a11 11 0 0 1 14.08 0"/><path d="M1.42 9a16 16 0 0 1 21.16 0"/><path d="M8.53 16.11a6 6 0 0 1 6.95 0"/><line x1="12" x2="12.01" y1="20" y2="20"/>',
    menu: '<line x1="4" x2="20" y1="6" y2="6"/><line x1="4" x2="20" y1="12" y2="12"/><line x1="4" x2="20" y1="18" y2="18"/>',
    refresh: '<path d="M21 12a9 9 0 1 1-2.64-6.36"/><polyline points="21 3 21 9 15 9"/>',
    'chevron-down': '<path d="m6 9 6 6 6-6"/>',
    'chevron-right': '<path d="m9 18 6-6-6-6"/>',
    'chevron-left': '<path d="m15 18-6-6 6-6"/>',
    plus: '<path d="M12 5v14M5 12h14"/>',
    minus: '<path d="M5 12h14"/>',
    'arrow-right': '<path d="M5 12h14"/><path d="m12 5 7 7-7 7"/>',
    'arrow-up-right': '<path d="M7 17 17 7"/><path d="M7 7h10v10"/>',
    copy: '<rect x="9" y="9" width="11" height="11" rx="2"/><path d="M5 15V5a2 2 0 0 1 2-2h10"/>',
    check: '<path d="m20 6-11 11-5-5"/>',
    x: '<path d="M18 6 6 18M6 6l12 12"/>',
    alert: '<path d="M10.3 3.7 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.7a2 2 0 0 0-3.4 0Z"/><line x1="12" x2="12" y1="9" y2="13"/><line x1="12" x2="12.01" y1="17" y2="17"/>',
    info: '<circle cx="12" cy="12" r="10"/><path d="M12 16v-4"/><path d="M12 8h.01"/>',
    trash: '<path d="M3 6h18"/><path d="M8 6V4a1 1 0 0 1 1-1h6a1 1 0 0 1 1 1v2"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/><line x1="10" x2="10" y1="11" y2="17"/><line x1="14" x2="14" y1="11" y2="17"/>',
    pencil: '<path d="M17 3a2.85 2.85 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z"/><path d="m15 5 4 4"/>',
    grip: '<circle cx="9" cy="5" r="1.2" fill="currentColor" stroke="none"/><circle cx="15" cy="5" r="1.2" fill="currentColor" stroke="none"/><circle cx="9" cy="12" r="1.2" fill="currentColor" stroke="none"/><circle cx="15" cy="12" r="1.2" fill="currentColor" stroke="none"/><circle cx="9" cy="19" r="1.2" fill="currentColor" stroke="none"/><circle cx="15" cy="19" r="1.2" fill="currentColor" stroke="none"/>',
    play: '<path d="M7 4v16l13-8Z"/>',
    pause: '<path d="M8 5v14M16 5v14"/>',
    download: '<path d="M12 3v12"/><path d="m7 10 5 5 5-5"/><path d="M5 21h14"/>',
    search: '<circle cx="11" cy="11" r="7"/><path d="m20 20-3.2-3.2"/>',
    cloud: '<path d="M17.5 19a4.5 4.5 0 0 0 .4-8.98A7 7 0 0 0 4.2 12.5 4 4 0 0 0 6 20h11.5Z"/>',
    server: '<rect x="3" y="4" width="18" height="7" rx="2"/><rect x="3" y="13" width="18" height="7" rx="2"/><path d="M7 7.5h.01M7 16.5h.01"/>',
    globe: '<circle cx="12" cy="12" r="9"/><path d="M3 12h18"/><path d="M12 3a15 15 0 0 1 0 18"/><path d="M12 3a15 15 0 0 0 0 18"/>',
    shield: '<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10Z"/><path d="m9 12 2 2 4-4"/>',
    external: '<path d="M15 3h6v6"/><path d="M10 14 21 3"/><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>',
    more: '<circle cx="5" cy="12" r="1" fill="currentColor" stroke="none"/><circle cx="12" cy="12" r="1" fill="currentColor" stroke="none"/><circle cx="19" cy="12" r="1" fill="currentColor" stroke="none"/>',
    activity: '<path d="M3 12h4l3-8 4 16 3-8h4"/>',
    zap: '<path d="M13 2 3 14h7l-1 8 11-13h-7l1-7Z"/>',
    clock: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
    key: '<circle cx="8" cy="15" r="4"/><path d="m11 12 9-9"/><path d="m17 3 3 3"/><path d="m14 6 3 3"/>',
    code: '<path d="m16 18 6-6-6-6"/><path d="m8 6-6 6 6 6"/>',
    eye: '<path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7-10-7-10-7Z"/><circle cx="12" cy="12" r="3"/>',
    'eye-off': '<path d="m3 3 18 18"/><path d="M10.6 10.6a2 2 0 0 0 2.8 2.8"/><path d="M9.9 5.2A10.6 10.6 0 0 1 12 5c6.5 0 10 7 10 7a17.8 17.8 0 0 1-3.2 4.2"/><path d="M6.6 6.6C3.6 8.5 2 12 2 12s3.5 7 10 7a10.6 10.6 0 0 0 4.1-.8"/>',
    link: '<path d="M10 13a5 5 0 0 0 7.5.5l2-2a5 5 0 0 0-7-7l-1.2 1.2"/><path d="M14 11a5 5 0 0 0-7.5-.5l-2 2a5 5 0 0 0 7 7l1.2-1.2"/>',
    file: '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8Z"/><path d="M14 2v6h6"/><path d="M8 13h8M8 17h5"/>',
    lock: '<rect x="4" y="11" width="16" height="10" rx="2"/><path d="M8 11V7a4 4 0 0 1 8 0v4"/>',
    plug: '<path d="M12 22v-5"/><path d="M9 8V2"/><path d="M15 8V2"/><path d="M18 8v3a6 6 0 0 1-12 0V8Z"/>',
    list: '<path d="M8 6h13M8 12h13M8 18h13"/><path d="M3 6h.01M3 12h.01M3 18h.01"/>',
    radio: '<circle cx="12" cy="12" r="2"/><path d="M16.2 7.8a6 6 0 0 1 0 8.4M7.8 16.2a6 6 0 0 1 0-8.4M19.1 4.9a10 10 0 0 1 0 14.2M4.9 19.1a10 10 0 0 1 0-14.2"/>'
  };

  const icon = (name, cls = '') =>
    `<svg class="icon ${cls}" viewBox="0 0 24 24" aria-hidden="true">${iconPaths[name] || iconPaths.info}</svg>`;

  const escapeHtml = (value) =>
    String(value ?? '').replace(/[&<>"']/g, (ch) => ({
      '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    }[ch]));

  const formatClock = (date) => {
    const d = date instanceof Date ? date : new Date(date);
    return [d.getHours(), d.getMinutes(), d.getSeconds()].map((n) => String(n).padStart(2, '0')).join(':');
  };

  const pad = (n) => String(n).padStart(2, '0');

  const formatUptimeHtml = (ms) => {
    const total = Math.max(0, Math.floor(ms / 1000));
    const days = Math.floor(total / 86400);
    const hours = Math.floor((total % 86400) / 3600);
    const minutes = Math.floor((total % 3600) / 60);
    const seconds = total % 60;
    return `${days}<span>天</span> ${pad(hours)}:${pad(minutes)}:${pad(seconds)}`;
  };

  const shortId = (id) => `${id.slice(0, 8)}…${id.slice(-4)}`;
  const today = () => '2026-11-05';

  const uid = () => {
    if (window.crypto && crypto.randomUUID) return crypto.randomUUID();
    return 'id-' + Math.random().toString(36).slice(2, 10) + Date.now().toString(36);
  };

  const randomToken = () => {
    if (window.crypto && crypto.randomUUID) return 'tun_' + crypto.randomUUID().replace(/-/g, '');
    return 'tun_' + Math.random().toString(36).slice(2) + Date.now().toString(36);
  };

  const PAGES = {
    overview: { title: '概览' },
    tunnels: { title: '隧道' },
    routes: { title: '路由' },
    logs: { title: '运行日志' },
    api: { title: '接入' },
    settings: { title: '设置' }
  };

  const TUNNEL_MAIN = '8f3c2a1e-4d8b-4f1a-9b2c-7e5d6a1b3c90';
  const TUNNEL_LAB = '2b7d9c44-6f0e-4c2a-8d1b-5a3e7f9c2d41';
  const TUNNEL_BACKUP = 'c91e5f70-8a2b-4d6c-b3e9-1f4a6c8d0e72';

  const state = {
    page: 'overview',
    themePref: localStorage.getItem('tunnel-theme') || 'light',
    theme: 'light',
    tunnels: [
      { id: TUNNEL_MAIN, name: 'fnos-main', status: 'healthy', created: '2026-09-18', lastError: '' },
      { id: TUNNEL_LAB, name: 'home-lab', status: 'inactive', created: '2026-10-02', lastError: '' },
      { id: TUNNEL_BACKUP, name: 'edge-backup', status: 'error', created: '2026-10-21', lastError: 'Origin refused connection' }
    ],
    activeId: TUNNEL_MAIN,
    connectingId: null,
    connectedAt: Date.now() - ((3 * 86400 + 17 * 3600 + 8 * 60 + 21) * 1000),
    routesByTunnel: {
      [TUNNEL_MAIN]: [
        { id: 'r-nas', host: 'nas.example.com', service: 'http://127.0.0.1:5666', dns: 'created', noTLSVerify: false, path: '', hostHeader: '' },
        { id: 'r-ha', host: 'ha.example.com', service: 'http://192.168.0.42:8123', dns: 'created', noTLSVerify: true, path: '', hostHeader: '' },
        { id: 'r-ai', host: 'ai.example.com', service: 'http://127.0.0.1:3000', dns: 'created', noTLSVerify: false, path: '', hostHeader: '' },
        { id: 'r-pa', host: 'pa.example.com', service: 'http://127.0.0.1:19091', dns: 'pending', noTLSVerify: false, path: '', hostHeader: '' }
      ],
      [TUNNEL_LAB]: [
        { id: 'r-home', host: 'home.example.com', service: 'http://192.168.0.42:8123', dns: 'created', noTLSVerify: false, path: '', hostHeader: '' },
        { id: 'r-files', host: 'files.example.com', service: 'http://192.168.0.42:8080', dns: 'created', noTLSVerify: true, path: '', hostHeader: '' }
      ],
      [TUNNEL_BACKUP]: [
        { id: 'r-backup', host: 'backup.example.com', service: 'http://127.0.0.1:9000', dns: 'failed', noTLSVerify: false, path: '', hostHeader: '' }
      ]
    },
    tunnelFilter: 'all',
    tunnelQuery: '',
    logs: [],
    logPaused: false,
    logLevel: 'all',
    logQuery: '',
    autoScroll: true,
    credentials: [
      { id: 'c1', name: 'Home Assistant', scopes: ['tunnels:read', 'routes:read'], last: '2 天前' },
      { id: 'c2', name: 'n8n 自动化', scopes: ['tunnels:read', 'routes:write'], last: '昨天' }
    ],
    settings: {
      accountId: 'a1b2c3d4e5f60718',
      token: 'cf_demo_9f3a2b7c4e1d8a6f',
      verified: true,
      protocol: 'quic',
      edgeIp: 'auto',
      retry: true,
      autoConnect: true,
      autoStart: true,
      retention: '30',
      maxLines: '2000',
      reduceMotion: false
    },
    settingsDirty: false,
    activity: [
      { id: 'a1', icon: 'link', tone: 'green', title: '隧道 fnos-main 已连接', time: '2 分钟前' },
      { id: 'a2', icon: 'branch', tone: 'blue', title: '路由 ai.example.com 已更新', time: '18 分钟前' },
      { id: 'a3', icon: 'cloud', tone: 'teal', title: 'CNAME pa.example.com 已创建', time: '1 小时前' },
      { id: 'a4', icon: 'sliders', tone: '', title: '协议切换为 QUIC', time: '3 小时前' }
    ],
    activitySeq: 4,
    logSeq: 0
  };

  let lastAutoScrollAt = 0;

  const activeTunnel = () => state.tunnels.find((t) => t.id === state.activeId) || null;
  const routesFor = (id) => state.routesByTunnel[id] || [];

  const statusMeta = (status) => ({
    healthy: { label: '健康', pill: 'healthy', dot: 'healthy', micro: '已连接' },
    connecting: { label: '连接中', pill: 'warning', dot: 'warning', micro: '正在建立连接' },
    inactive: { label: '未连接', pill: 'neutral', dot: '', micro: '等待连接' },
    error: { label: '异常', pill: 'danger', dot: 'danger', micro: '连接中断' }
  }[status] || { label: '未知', pill: 'neutral', dot: '', micro: '' });

  const dnsMeta = (dns) => ({
    created: { label: '已创建', pill: 'healthy' },
    pending: { label: '待创建', pill: 'warning' },
    failed: { label: '失败', pill: 'danger' }
  }[dns] || { label: '未知', pill: 'neutral' });

  /* ---------- boot ---------- */
  hydrateIcons();
  applyTheme();
  seedLogs();
  bindStatic();
  const initial = location.hash.slice(1);
  state.page = PAGES[initial] ? initial : 'overview';
  renderPage({ animate: true });
  setInterval(updateUptime, 1000);
  setInterval(() => {
    if (state.logPaused) return;
    addLog(randomLog());
  }, 1700);

  function hydrateIcons(root = document) {
    $$('[data-icon]', root).forEach((el) => {
      if (!el.firstElementChild) el.innerHTML = icon(el.dataset.icon);
    });
  }

  function bindStatic() {
    $('#theme-toggle').addEventListener('click', () => {
      state.themePref = state.theme === 'dark' ? 'light' : 'dark';
      localStorage.setItem('tunnel-theme', state.themePref);
      applyTheme();
      if (state.page === 'settings') updateSegmentedUI();
    });

    $('#mobile-menu').addEventListener('click', () => {
      const open = !$('#app').classList.contains('sidebar-open');
      $('#app').classList.toggle('sidebar-open', open);
      $('#sidebar-scrim').hidden = !open;
    });
    $('#sidebar-scrim').addEventListener('click', closeSidebar);

    $('#refresh-btn').addEventListener('click', () => {
      const btn = $('#refresh-btn');
      btn.classList.add('is-spinning');
      setTimeout(() => btn.classList.remove('is-spinning'), 700);
      updateUptime();
      if (state.page === 'logs') scrollLogsToBottom();
      toast('状态已刷新', 'success');
    });

    document.addEventListener('click', onDocumentClick);
    window.addEventListener('hashchange', () => {
      const page = location.hash.slice(1);
      state.page = PAGES[page] ? page : 'overview';
      closeSidebar();
      renderPage({ animate: true });
    });
    window.addEventListener('resize', () => {
      if (window.innerWidth > 920) closeSidebar();
    });
    window.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') closeSidebar();
    });
    const mq = window.matchMedia('(prefers-color-scheme: dark)');
    if (mq.addEventListener) {
      mq.addEventListener('change', () => {
        if (state.themePref === 'system') applyTheme();
      });
    }
    if (state.settings.reduceMotion) document.documentElement.classList.add('reduce-motion');
  }

  function closeSidebar() {
    $('#app').classList.remove('sidebar-open');
    $('#sidebar-scrim').hidden = true;
  }

  function onDocumentClick(e) {
    const pageTrigger = e.target.closest('[data-page]');
    if (pageTrigger) {
      e.preventDefault();
      const page = pageTrigger.dataset.page;
      closeSidebar();
      if (state.page === page) {
        renderPage({ animate: false });
        window.scrollTo({ top: 0, behavior: 'smooth' });
      } else {
        location.hash = page;
      }
      return;
    }

    const filterBtn = e.target.closest('[data-filter]');
    if (filterBtn) {
      state.tunnelFilter = filterBtn.dataset.filter;
      $$('[data-filter]').forEach((b) => b.classList.toggle('is-active', b === filterBtn));
      renderTunnelList();
      return;
    }

    const levelBtn = e.target.closest('[data-level]');
    if (levelBtn) {
      state.logLevel = levelBtn.dataset.level;
      $$('[data-level]').forEach((b) => b.classList.toggle('is-active', b === levelBtn));
      renderLogRows();
      return;
    }

    const actionBtn = e.target.closest('[data-action]');
    if (actionBtn) {
      e.preventDefault();
      handleAction(actionBtn);
    }
  }

  async function handleAction(btn) {
    const action = btn.dataset.action;
    const id = btn.dataset.id;

    switch (action) {
      case 'account':
        toast('账户切换是原型占位', 'success');
        break;
      case 'refresh':
        toast('状态已刷新 · 刚刚', 'success');
        updateUptime();
        break;
      case 'open-route-drawer':
      case 'add-route':
        openRouteDrawer('add');
        break;
      case 'disconnect':
        await doDisconnect();
        break;
      case 'reconnect':
        if (state.activeId) doConnect(state.activeId);
        break;
      case 'new-tunnel':
        openNewTunnel();
        break;
      case 'connect-tunnel':
        await connectTunnel(id);
        break;
      case 'disconnect-tunnel':
        await disconnectTunnel(id);
        break;
      case 'delete-tunnel':
        await deleteTunnel(id);
        break;
      case 'go-routes':
        if (id === state.activeId) {
          location.hash = 'routes';
        } else {
          toast('请先连接该隧道，再编辑路由', 'error');
        }
        break;
      case 'edit-route':
        openRouteDrawer('edit', id);
        break;
      case 'delete-route':
        await deleteRoute(id);
        break;
      case 'copy-yaml':
        copyText(rawYaml(), 'config.yml 已复制');
        break;
      case 'copy-endpoint':
        copyText(btn.dataset.copy || '', '地址已复制');
        break;
      case 'toggle-log-pause':
        toggleLogPause();
        break;
      case 'clear-logs':
        clearLogs();
        break;
      case 'download-logs':
        downloadLogs();
        break;
      case 'new-credential':
        openCredentialModal();
        break;
      case 'verify-account':
        verifyAccount(btn);
        break;
      case 'token-help':
        openTokenHelp();
        break;
      case 'reset-config':
        await resetConfig();
        break;
      case 'save-settings':
        saveSettings();
        break;
      case 'toggle-token':
        toggleTokenVisibility(btn);
        break;
      default:
        break;
    }
  }

  /* ---------- theme ---------- */
  function resolveTheme() {
    if (state.themePref === 'system') {
      return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }
    return state.themePref;
  }

  function applyTheme() {
    state.theme = resolveTheme();
    document.documentElement.dataset.theme = state.theme;
    const toggle = $('#theme-toggle');
    if (toggle) toggle.innerHTML = icon(state.theme === 'dark' ? 'sun' : 'moon');
  }

  /* ---------- render ---------- */
  function renderPage({ animate = true } = {}) {
    const page = PAGES[state.page] ? state.page : 'overview';
    state.page = page;
    $('#crumb').textContent = PAGES[page].title;
    document.title = `${PAGES[page].title} · Tunnel for fnOS`;
    $$('.nav-link').forEach((link) => {
      const isActive = link.dataset.page === page;
      link.classList.toggle('is-active', isActive);
      if (isActive) link.setAttribute('aria-current', 'page');
      else link.removeAttribute('aria-current');
    });

    const pageEl = $('#page');
    const templates = {
      overview: tplOverview,
      tunnels: tplTunnels,
      routes: tplRoutes,
      logs: tplLogs,
      api: tplApi,
      settings: tplSettings
    };
    pageEl.innerHTML = templates[page]();
    hydrateIcons(pageEl);
    if (animate) {
      pageEl.classList.remove('is-entering');
      void pageEl.offsetWidth;
      pageEl.classList.add('is-entering');
    } else {
      pageEl.classList.remove('is-entering');
    }
    bindPage(page);
    updateNavCounts();
    updateSaveUI();
    updateUptime();
    if (page === 'tunnels') renderTunnelList();
    if (page === 'routes') {
      renderRouteRows();
      renderYaml();
    }
    if (page === 'logs') {
      renderLogRows();
      updateLogToolbar();
      scrollLogsToBottom();
    }
  }

  function bindPage(page) {
    if (page === 'tunnels') {
      const search = $('#tunnel-search');
      if (search) {
        search.addEventListener('input', () => {
          state.tunnelQuery = search.value;
          renderTunnelList();
        });
      }
    }
    if (page === 'routes') {
      bindRouteDrag();
    }
    if (page === 'logs') {
      const search = $('#log-search');
      if (search) {
        search.addEventListener('input', () => {
          state.logQuery = search.value;
          renderLogRows();
        });
      }
      const auto = $('#auto-scroll');
      if (auto) {
        auto.checked = state.autoScroll;
        auto.addEventListener('change', () => {
          state.autoScroll = auto.checked;
          if (state.autoScroll) scrollLogsToBottom();
        });
      }
      const body = $('#log-body');
      if (body) {
        body.addEventListener('scroll', () => {
          if (Date.now() - lastAutoScrollAt < 250) return;
          const distance = body.scrollHeight - body.scrollTop - body.clientHeight;
          if (distance > 28 && state.autoScroll) {
            state.autoScroll = false;
            const auto = $('#auto-scroll');
            if (auto) auto.checked = false;
          }
        });
      }
      const rows = $('#log-rows');
      if (rows) {
        rows.addEventListener('click', (e) => {
          const line = e.target.closest('.log-line');
          if (!line) return;
          const entry = line.closest('.log-entry');
          const detail = entry.querySelector('.log-detail');
          const open = line.getAttribute('aria-expanded') === 'true';
          line.setAttribute('aria-expanded', String(!open));
          detail.hidden = open;
          if (!open) {
            state.autoScroll = false;
            const auto = $('#auto-scroll');
            if (auto) auto.checked = false;
          }
        });
      }
    }
    if (page === 'settings') bindSettings();
  }

  function updateNavCounts() {
    const tunnelCount = $('[data-page="tunnels"] .nav-count');
    if (tunnelCount) tunnelCount.textContent = String(state.tunnels.length);
    const routeCount = $('[data-page="routes"] .nav-count');
    if (routeCount) routeCount.textContent = String(activeTunnel() ? routesFor(state.activeId).length : 0);
  }

  function updateUptime() {
    const t = activeTunnel();
    const nodes = $$('[data-uptime]');
    if (!nodes.length) return;
    if (!t || t.status !== 'healthy' || !state.connectedAt) return;
    const html = formatUptimeHtml(Date.now() - state.connectedAt);
    nodes.forEach((node) => { node.innerHTML = html; });
  }

  function pushActivity(title, tone = 'blue', iconName = 'branch') {
    state.activitySeq += 1;
    state.activity.unshift({ id: `a${state.activitySeq}`, icon: iconName, tone, title, time: '刚刚' });
    state.activity = state.activity.slice(0, 6);
  }

  function safeRefresh() {
    updateNavCounts();
    if (['overview', 'tunnels', 'routes', 'api'].includes(state.page)) {
      renderPage({ animate: false });
    }
  }

  /* ---------- overview ---------- */
  function tplOverview() {
    const t = activeTunnel();
    const routes = t ? routesFor(t.id) : [];
    const status = t ? t.status : 'inactive';
    const meta = statusMeta(status);
    const isHealthy = status === 'healthy';
    const isConnecting = status === 'connecting';
    const isError = status === 'error';

    const heroPill = isHealthy
      ? `<span class="pill healthy"><span class="dot"></span>已连接</span>`
      : isConnecting
        ? `<span class="pill warning"><span class="spinner" style="width:9px;height:9px;border-width:1.5px"></span>连接中</span>`
        : isError
          ? `<span class="pill danger"><span class="dot"></span>连接异常</span>`
          : `<span class="pill neutral"><span class="dot"></span>未连接</span>`;

    const heroDesc = !t
      ? '还没有连接任何隧道。选择一条隧道开始，把 NAS 上的服务带到公网。'
      : isHealthy
        ? 'Cloudflare Edge 已建立 4 条连接，流量正在经过香港节点。'
        : isConnecting
          ? '正在与 Cloudflare Edge 建立连接，通常需要 1–3 秒。'
          : isError
            ? `上次错误：${escapeHtml(t.lastError || '连接中断')}。检查日志或重新连接。`
            : '当前隧道未连接。连接后，所有路由规则才会开始转发。';

    const heroActions = !t
      ? `<button class="btn primary" data-page="tunnels">${icon('waypoints')}选择隧道</button>`
      : isHealthy
        ? `<button class="btn ghost danger-text" data-action="disconnect">断开连接</button>`
        : isConnecting
          ? `<button class="btn primary" disabled><span class="spinner"></span>连接中</button>`
          : isError
            ? `<button class="btn primary" data-action="reconnect">${icon('refresh')}重新连接</button>`
            : `<button class="btn primary" data-action="reconnect">${icon('plug')}连接隧道</button>`;

    const uptime = isHealthy
      ? `<span class="uptime" data-uptime>${formatUptimeHtml(Date.now() - state.connectedAt)}</span>`
      : `<span class="uptime is-idle">—</span>`;

    const routeRows = routes.slice(0, 3).map((r) => {
      const dns = dnsMeta(r.dns);
      return `
        <tr>
          <td><span class="host">${escapeHtml(r.host)}</span></td>
          <td><span class="service">${escapeHtml(r.service)}</span></td>
          <td><span class="pill ${dns.pill}">${dns.label}</span></td>
          <td class="cell-actions"><button class="icon-btn" type="button" data-action="edit-route" data-id="${r.id}" aria-label="编辑路由">${icon('arrow-up-right')}</button></td>
        </tr>`;
    }).join('');

    const routeTable = routes.length
      ? `<div class="table-wrap"><table class="data-table">
          <thead><tr><th>域名</th><th>内网服务</th><th>DNS</th><th></th></tr></thead>
          <tbody>${routeRows}</tbody>
        </table></div>`
      : `<div class="table-empty">当前隧道还没有路由。添加一条域名 → 内网服务的映射。</div>`;

    const activity = state.activity.slice(0, 4).map((a) => `
      <li>
        <span class="t-icon ${a.tone}">${icon(a.icon)}</span>
        <span class="t-copy"><strong>${escapeHtml(a.title)}</strong><span>${escapeHtml(a.time)}</span></span>
      </li>`).join('');

    return `
      <section class="page-head reveal" style="--i:0">
        <div>
          <p class="eyebrow">控制台</p>
          <h1>概览</h1>
          <p class="sub">一条隧道，把 NAS 上的服务安全地带到公网。状态、路由与日志都在一个地方。</p>
        </div>
        <div class="page-actions">
          <button class="btn ghost" type="button" data-action="refresh">${icon('refresh')}刷新</button>
          <button class="btn primary" type="button" data-action="open-route-drawer">${icon('plus')}添加路由</button>
        </div>
      </section>

      <div class="overview-grid">
        <section class="card hero reveal" style="--i:1">
          <div class="hero-top">
            <div class="hero-status">
              ${heroPill}
              <span class="micro">${isHealthy ? 'QUIC · HKG · 38 ms' : isConnecting ? '正在握手' : '—'}</span>
            </div>
            <button class="icon-btn" type="button" data-page="tunnels" aria-label="管理隧道">${icon('more')}</button>
          </div>
          <div class="hero-main">
            <div class="hero-copy">
              <p class="eyebrow">当前隧道</p>
              <h2>${t ? escapeHtml(t.name) : '未选择隧道'}${t ? `<span class="mono">${shortId(t.id)}</span>` : ''}</h2>
              <p class="hero-desc">${heroDesc}</p>
              <span class="uptime-label">运行时长</span>
              ${uptime}
              <div class="hero-actions">
                ${heroActions}
                ${t ? `<button class="btn subtle" type="button" data-page="routes">${icon('branch')}路由 ${routes.length}</button>` : ''}
              </div>
            </div>
            <div class="spine-wrap">
              ${spineSvg(isHealthy)}
            </div>
          </div>
        </section>

        <aside class="card quick reveal" style="--i:2">
          <div class="card-head"><div><h3>快捷操作</h3><p>常用动作，少点两下</p></div></div>
          <button class="quick-item" type="button" data-page="tunnels">
            <span class="q-icon">${icon('waypoints')}</span>
            <span class="q-copy"><strong>切换隧道</strong><small>${state.tunnels.length} 条隧道可用</small></span>
            <span class="q-chevron">${icon('chevron-right')}</span>
          </button>
          <button class="quick-item" type="button" data-action="open-route-drawer">
            <span class="q-icon">${icon('branch')}</span>
            <span class="q-copy"><strong>新增路由</strong><small>域名 → 内网服务</small></span>
            <span class="q-chevron">${icon('chevron-right')}</span>
          </button>
          <button class="quick-item" type="button" data-page="logs">
            <span class="q-icon">${icon('terminal')}</span>
            <span class="q-copy"><strong>实时日志</strong><small>查看连接与转发事件</small></span>
            <span class="q-chevron">${icon('chevron-right')}</span>
          </button>
          <div class="quick-health">
            <div class="qh-top"><span>本周可用性</span><strong>99.98<small>%</small></strong></div>
            <div class="qh-bars" aria-hidden="true">
              ${[62, 78, 70, 88, 74, 92, 84, 96, 80, 90, 86, 97, 93, 100].map((h) => `<span style="--h:${h}%"></span>`).join('')}
            </div>
            <div class="qh-foot"><span>过去 7 天</span><span class="trend up">+0.02%</span></div>
          </div>
          <div class="mini-note"><span class="pulse-dot"></span>Cloudflare API 正常<span class="mono">12 ms</span></div>
        </aside>
      </div>

      <div class="metric-row">
        ${metricCard('活跃路由', String(routes.length), '<span class="trend flat">当前隧道</span>', 'branch', [3, 4, 4, 5, 4, 6, 6], 'sparkA')}
        ${metricCard('24h 请求', '12,840', '<span class="trend up">+8.2%</span>', 'activity', [4, 6, 5, 8, 7, 10, 12], 'sparkB')}
        ${metricCard('平均延迟', isHealthy ? '38<span>ms</span>' : '—', isHealthy ? '<span class="trend down">-4 ms</span>' : '<span class="trend flat">无连接</span>', 'zap', [6, 8, 7, 9, 8, 7, 6], 'sparkC', true)}
        ${metricCard('出站流量', '1.42<span>GB</span>', '<span class="trend flat">过去 24h</span>', 'cloud', [3, 4, 6, 5, 7, 8, 9], 'sparkD')}
      </div>

      <div class="lower-grid">
        <section class="card reveal" style="--i:7">
          <div class="card-head">
            <div><h3>路由</h3><p>${routes.length} 条规则 · 首条匹配生效</p></div>
            <button class="link" type="button" data-page="routes">全部路由 →</button>
          </div>
          ${routeTable}
        </section>
        <section class="card activity reveal" style="--i:8">
          <div class="card-head"><div><h3>最近活动</h3><p>本机事件，非云端审计</p></div></div>
          <ol class="timeline">${activity}</ol>
        </section>
      </div>`;
  }

  function metricCard(label, value, foot, iconName, values, sparkId, green = false) {
    return `
      <section class="card metric reveal">
        <div class="metric-label">${icon(iconName)}${label}</div>
        <div class="metric-value">${value}</div>
        <div class="metric-foot">${foot}</div>
        ${sparkline(values, sparkId, green)}
      </section>`;
  }

  function sparkline(values, id, green) {
    const w = 84;
    const h = 28;
    const min = Math.min(...values);
    const max = Math.max(...values);
    const span = max - min || 1;
    const pts = values.map((v, i) => {
      const x = (i / (values.length - 1)) * w;
      const y = h - 4 - ((v - min) / span) * (h - 8);
      return [x, y];
    });
    const line = pts.map((p, i) => `${i ? 'L' : 'M'}${p[0].toFixed(1)} ${p[1].toFixed(1)}`).join(' ');
    const area = `${line} L ${w} ${h} L 0 ${h} Z`;
    return `<svg class="metric-spark ${green ? 'green' : ''}" viewBox="0 0 ${w} ${h}" aria-hidden="true">
      <defs><linearGradient id="${id}" x1="0" y1="0" x2="0" y2="1">
        <stop stop-color="${green ? 'var(--teal)' : 'var(--accent)'}" stop-opacity=".25"/>
        <stop offset="1" stop-color="${green ? 'var(--teal)' : 'var(--accent)'}" stop-opacity="0"/>
      </linearGradient></defs>
      <path d="${area}" fill="url(#${id})"/>
      <path d="${line}" class="line"/>
    </svg>`;
  }

  function spineSvg(isHealthy) {
    return `
      <svg class="spine ${isHealthy ? '' : 'is-idle'}" viewBox="0 0 560 320" fill="none" role="img" aria-label="NAS 通过 Cloudflare Edge 连接到互联网">
        <defs>
          <pattern id="spineDots" width="18" height="18" patternUnits="userSpaceOnUse">
            <circle cx="1" cy="1" r="1" fill="var(--line-strong)"/>
          </pattern>
          <linearGradient id="spineLine" x1="120" y1="160" x2="440" y2="160" gradientUnits="userSpaceOnUse">
            <stop stop-color="var(--accent)" stop-opacity=".55"/>
            <stop offset=".5" stop-color="var(--teal)" stop-opacity=".8"/>
            <stop offset="1" stop-color="var(--accent)" stop-opacity=".55"/>
          </linearGradient>
          <filter id="nodeGlow" x="-80%" y="-80%" width="260%" height="260%">
            <feGaussianBlur stdDeviation="7" result="blur"/>
            <feMerge><feMergeNode in="blur"/><feMergeNode in="SourceGraphic"/></feMerge>
          </filter>
        </defs>
        <rect x="0" y="0" width="560" height="320" fill="url(#spineDots)" opacity=".55"/>
        <path d="M126 160 L 434 160" stroke="url(#spineLine)" stroke-width="1.5" stroke-dasharray="5 7" opacity=".8"/>
        <path d="M126 160 C 190 104, 210 104, 280 104 S 370 104, 434 160" stroke="var(--line-strong)" stroke-width="1" stroke-dasharray="2 8" opacity=".55"/>
        <path d="M126 160 C 190 216, 210 216, 280 216 S 370 216, 434 160" stroke="var(--line-strong)" stroke-width="1" stroke-dasharray="2 8" opacity=".55"/>

        <g class="node node-nas" transform="translate(92 160)">
          <circle r="48" class="node-ring"/>
          <circle r="34" class="node-core"/>
          <g transform="translate(-11 -11) scale(0.92)" class="node-icon">
            <rect x="3" y="4" width="18" height="7" rx="2"/><rect x="3" y="13" width="18" height="7" rx="2"/><path d="M7 7.5h.01M7 16.5h.01"/>
          </g>
        </g>
        <g class="node node-edge" transform="translate(280 160)">
          <circle r="58" class="node-ring" opacity=".45"/>
          <circle r="40" class="node-core" filter="url(#nodeGlow)"/>
          <g transform="translate(-11 -11) scale(0.94)" class="node-icon">
            <path d="M17.5 19a4.5 4.5 0 0 0 .4-8.98A7 7 0 0 0 4.2 12.5 4 4 0 0 0 6 20h11.5Z"/>
          </g>
          <circle cx="0" cy="0" r="47" fill="none" stroke="var(--accent)" stroke-width="1" stroke-dasharray="2 7" opacity=".3"/>
        </g>
        <g class="node node-cloud" transform="translate(468 160)">
          <circle r="48" class="node-ring"/>
          <circle r="34" class="node-core"/>
          <g transform="translate(-11 -11) scale(0.92)" class="node-icon">
            <circle cx="12" cy="12" r="9"/><path d="M3 12h18"/><path d="M12 3a15 15 0 0 1 0 18"/><path d="M12 3a15 15 0 0 0 0 18"/>
          </g>
        </g>

        <g class="latency-badge" transform="translate(244 62)">
          <rect width="72" height="24" rx="12"/>
          <text x="36" y="16">38 ms · HKG</text>
        </g>

        <text x="92" y="226" class="node-label">your-nas</text>
        <text x="92" y="240" class="node-sub">192.168.1.10</text>
        <text x="280" y="238" class="node-label">Cloudflare Edge</text>
        <text x="280" y="252" class="node-sub">HKG · 4 connections</text>
        <text x="468" y="226" class="node-label">Internet</text>
        <text x="468" y="240" class="node-sub">HTTPS · TLS 1.3</text>

        <circle class="packet" r="3.2">
          <animateMotion dur="3.6s" repeatCount="indefinite" path="M126 160 L 434 160"/>
        </circle>
        <circle class="packet" r="2.4" opacity=".7">
          <animateMotion dur="3.6s" begin="1.8s" repeatCount="indefinite" path="M126 160 L 434 160"/>
        </circle>
      </svg>`;
  }

  /* ---------- tunnels ---------- */
  function tplTunnels() {
    return `
      <section class="page-head reveal" style="--i:0">
        <div>
          <p class="eyebrow">网络</p>
          <h1>隧道</h1>
          <p class="sub">创建、连接和切换 Cloudflare Tunnel。同一时间只保留一条活动连接，切换时会先断开当前隧道。</p>
        </div>
        <div class="page-actions">
          <button class="btn primary" type="button" data-action="new-tunnel">${icon('plus')}新建隧道</button>
        </div>
      </section>

      <div class="toolbar reveal" style="--i:1">
        <div class="segmented" role="group" aria-label="筛选隧道">
          <button type="button" data-filter="all" class="${state.tunnelFilter === 'all' ? 'is-active' : ''}">全部</button>
          <button type="button" data-filter="connected" class="${state.tunnelFilter === 'connected' ? 'is-active' : ''}">已连接</button>
          <button type="button" data-filter="idle" class="${state.tunnelFilter === 'idle' ? 'is-active' : ''}">未连接</button>
          <button type="button" data-filter="error" class="${state.tunnelFilter === 'error' ? 'is-active' : ''}">异常</button>
        </div>
        <div class="search">
          ${icon('search')}
          <input id="tunnel-search" type="search" placeholder="搜索隧道名称或 ID" value="${escapeHtml(state.tunnelQuery)}" autocomplete="off">
        </div>
        <div class="spacer"></div>
        <span class="micro">${state.tunnels.length} 条隧道 · 1 条活动连接</span>
      </div>

      <section class="card tunnel-list reveal" style="--i:2">
        <div id="tunnel-list"></div>
      </section>`;
  }

  function renderTunnelList() {
    const root = $('#tunnel-list');
    if (!root) return;
    const q = state.tunnelQuery.trim().toLowerCase();
    const filtered = state.tunnels.filter((t) => {
      if (state.tunnelFilter === 'connected' && !['healthy', 'connecting'].includes(t.status)) return false;
      if (state.tunnelFilter === 'idle' && t.status !== 'inactive') return false;
      if (state.tunnelFilter === 'error' && t.status !== 'error') return false;
      if (!q) return true;
      return t.name.toLowerCase().includes(q) || t.id.toLowerCase().includes(q);
    });

    if (!filtered.length) {
      root.innerHTML = `
        <div class="table-empty">
          <div style="font-weight:700;color:var(--ink-2);margin-bottom:4px">没有匹配的隧道</div>
          换一个筛选条件，或创建一条新隧道。
        </div>`;
      return;
    }

    root.innerHTML = filtered.map((t) => {
      const meta = statusMeta(t.status);
      const isActive = t.id === state.activeId;
      const routes = routesFor(t.id).length;
      const actions = t.status === 'healthy'
        ? `<button class="btn small ghost" type="button" data-action="disconnect-tunnel" data-id="${t.id}">断开</button>`
        : t.status === 'connecting'
          ? `<button class="btn small ghost" type="button" disabled><span class="spinner"></span>连接中</button>`
          : `<button class="btn small primary" type="button" data-action="connect-tunnel" data-id="${t.id}">连接</button>`;
      return `
        <div class="tunnel-row ${isActive ? 'is-active' : ''}" data-id="${t.id}">
          <div class="tunnel-ident">
            <span class="status-dot ${meta.dot}" aria-hidden="true"></span>
            <div class="t-copy">
              <strong>${escapeHtml(t.name)}</strong>
              <span class="mono">${shortId(t.id)}</span>
            </div>
          </div>
          <div class="tunnel-meta"><strong>${routes}</strong>条路由</div>
          <div class="tunnel-meta"><strong>${t.created}</strong>创建</div>
          <div class="tunnel-status">
            <span class="pill ${meta.pill}">${meta.label}</span>
            <span class="micro">${t.status === 'healthy' ? '已连接 · 当前' : t.status === 'error' ? escapeHtml(t.lastError || '连接中断') : meta.micro}</span>
          </div>
          <div class="row-actions">
            ${actions}
            <button class="btn small ghost" type="button" data-action="go-routes" data-id="${t.id}">路由</button>
            <button class="icon-btn" type="button" data-action="delete-tunnel" data-id="${t.id}" aria-label="删除隧道">${icon('trash')}</button>
          </div>
        </div>`;
    }).join('');
  }

  async function connectTunnel(id) {
    const target = state.tunnels.find((t) => t.id === id);
    if (!target || state.connectingId) return;
    const current = activeTunnel();
    if (current && current.id !== id && current.status === 'healthy') {
      const result = await dialog({
        title: '切换隧道？',
        message: `当前隧道 ${current.name} 将立即断开，${target.name} 会接管连接。`,
        confirmText: '切换连接'
      });
      if (!result.ok) return;
    }
    doConnect(id);
  }

  function doConnect(id) {
    const target = state.tunnels.find((t) => t.id === id);
    if (!target || state.connectingId) return;
    const current = activeTunnel();
    if (current && current.id !== id) current.status = 'inactive';
    state.connectingId = id;
    state.activeId = id;
    target.status = 'connecting';
    safeRefresh();
    setTimeout(() => {
      target.status = 'healthy';
      target.lastError = '';
      state.connectingId = null;
      state.connectedAt = Date.now();
      pushActivity(`隧道 ${target.name} 已连接`, 'green', 'link');
      toast(`隧道 ${target.name} 已连接`, 'success');
      safeRefresh();
    }, 950);
  }

  async function doDisconnect() {
    const t = activeTunnel();
    if (!t || t.status === 'inactive') return;
    const result = await dialog({
      title: '断开隧道？',
      message: '断开后，所有域名转发规则将暂时失效；隧道本身和路由配置会保留。',
      confirmText: '断开连接',
      tone: 'danger'
    });
    if (!result.ok) return;
    t.status = 'inactive';
    state.activeId = null;
    state.connectedAt = null;
    pushActivity(`隧道 ${t.name} 已断开`, 'danger', 'plug');
    toast('已断开连接', 'success');
    safeRefresh();
  }

  async function disconnectTunnel(id) {
    if (id !== state.activeId) return;
    await doDisconnect();
  }

  async function deleteTunnel(id) {
    const t = state.tunnels.find((item) => item.id === id);
    if (!t) return;
    const isActive = state.activeId === id;
    const result = await dialog({
      title: '删除隧道？',
      message: isActive
        ? `隧道 ${t.name} 正在使用中。删除会先断开连接，并移除本地路由记录；Cloudflare 上的隧道与 DNS 记录不会自动删除。`
        : `删除 ${t.name} 的本地记录。Cloudflare 上的隧道与 DNS 记录不会自动删除。`,
      confirmText: '删除隧道',
      tone: 'danger'
    });
    if (!result.ok) return;
    if (isActive) {
      state.activeId = null;
      state.connectedAt = null;
    }
    state.tunnels = state.tunnels.filter((item) => item.id !== id);
    delete state.routesByTunnel[id];
    pushActivity(`隧道 ${t.name} 已删除`, 'danger', 'trash');
    toast('隧道已从本地移除', 'success');
    safeRefresh();
  }

  function openNewTunnel() {
    const html = `
      <div class="modal" role="dialog" aria-modal="true" aria-labelledby="new-tunnel-title">
        <div class="modal-head">
          <div><p class="eyebrow">新建</p><h2 id="new-tunnel-title">创建隧道</h2></div>
          <button class="icon-btn" type="button" data-close aria-label="关闭">${icon('x')}</button>
        </div>
        <form id="new-tunnel-form" class="modal-body">
          <div class="form-grid">
            <div class="field" data-field="name">
              <label for="tunnel-name">隧道名称</label>
              <input class="input mono" id="tunnel-name" name="name" placeholder="fnos-home" autocomplete="off" spellcheck="false">
              <span class="hint">只用于识别，不影响域名。支持小写字母、数字和短横线。</span>
              <span class="error">请输入 2–32 位小写名称，例如 fnos-home</span>
            </div>
            <label class="check-row">
              <span class="switch"><input id="tunnel-auto-connect" type="checkbox" checked><i></i></span>
              <span class="check-copy"><strong>创建后立即连接</strong><small>创建成功后会接管当前连接；已有隧道会先断开。</small></span>
            </label>
            <div class="modal-warning">${icon('info')} 原型不会真的请求 Cloudflare；正式实现会调用 Cloudflare API 创建隧道。</div>
          </div>
        </form>
        <div class="modal-foot">
          <button class="btn ghost" type="button" data-close>取消</button>
          <button class="btn primary" type="submit" form="new-tunnel-form">创建隧道</button>
        </div>
      </div>`;
    const { layer, close } = openLayer({ html });
    const form = layer.querySelector('#new-tunnel-form');
    const input = layer.querySelector('#tunnel-name');
    input.addEventListener('input', () => {
      input.value = input.value.toLowerCase().replace(/[^a-z0-9-]/g, '');
      input.closest('.field').classList.remove('has-error');
    });
    form.addEventListener('submit', (e) => {
      e.preventDefault();
      const name = input.value.trim();
      if (!/^[a-z0-9][a-z0-9-]{1,31}$/.test(name)) {
        input.closest('.field').classList.add('has-error');
        input.focus();
        return;
      }
      if (state.tunnels.some((t) => t.name === name)) {
        input.closest('.field').classList.add('has-error');
        input.focus();
        return;
      }
      const id = uid();
      state.tunnels.push({ id, name, status: 'inactive', created: today(), lastError: '' });
      state.routesByTunnel[id] = [];
      pushActivity(`隧道 ${name} 已创建`, 'blue', 'plus');
      toast(`隧道 ${name} 已创建`, 'success');
      close();
      safeRefresh();
      if (layer.querySelector('#tunnel-auto-connect')?.checked) {
        setTimeout(() => doConnect(id), 140);
      }
    });
    setTimeout(() => input.focus(), 80);
  }

  /* ---------- routes ---------- */
  function tplRoutes() {
    const t = activeTunnel();
    if (!t) {
      return `
        <section class="page-head reveal" style="--i:0">
          <div><p class="eyebrow">路由</p><h1>Ingress 规则</h1><p class="sub">把公网域名映射到内网服务。先连接一条隧道，才能编辑路由。</p></div>
        </section>
        <section class="card reveal" style="--i:1">
          <div class="table-empty" style="padding:64px 24px">
            <div style="font-weight:800;color:var(--ink);font-size:16px;margin-bottom:6px">还没有活动隧道</div>
            <p class="muted" style="margin:0 0 16px">选择一条隧道并连接，然后为它配置域名路由。</p>
            <button class="btn primary" type="button" data-page="tunnels">${icon('waypoints')}去选择隧道</button>
          </div>
        </section>`;
    }

    const routes = routesFor(t.id);
    return `
      <section class="page-head reveal" style="--i:0">
        <div>
          <p class="eyebrow">路由</p>
          <h1>Ingress 规则</h1>
          <p class="sub">一个域名对应一个内网服务。规则按顺序匹配，首条命中后不再继续；最后一条为兜底 404。</p>
        </div>
        <div class="page-actions">
          <button class="btn primary" type="button" data-action="add-route">${icon('plus')}添加路由</button>
        </div>
      </section>

      <div class="toolbar reveal" style="--i:1">
        <button class="account-chip" type="button" data-page="tunnels" style="height:36px">
          <span class="status-dot ${statusMeta(t.status).dot}"></span>
          <span class="account-copy"><strong>${escapeHtml(t.name)}</strong><small>${shortId(t.id)}</small></span>
          <span class="chevron">${icon('chevron-down')}</span>
        </button>
        <span class="pill info">${routes.length} 条规则</span>
        <span class="micro">自动 DNS 已开启</span>
        <div class="spacer"></div>
        <span class="micro">拖动手柄可调整匹配顺序</span>
      </div>

      <div class="routes-layout">
        <section class="card route-list reveal" style="--i:2">
          <div id="route-rows"></div>
          <div class="route-note">${icon('info')} 规则按顺序匹配，首条命中后不再继续；最后一条为兜底 404。</div>
          <div class="dns-summary" id="dns-summary"></div>
        </section>
        <aside class="card yaml-card reveal" style="--i:3">
          <div class="yaml-head">
            <div class="file">${icon('code')}config.yml</div>
            <button class="icon-btn" type="button" data-action="copy-yaml" aria-label="复制 YAML">${icon('copy')}</button>
          </div>
          <pre class="yaml-code" id="yaml-code"></pre>
          <div class="yaml-foot"><span>由路由表实时生成</span><span>只读预览</span></div>
        </aside>
      </div>`;
  }

  function renderRouteRows() {
    const root = $('#route-rows');
    if (!root) return;
    const t = activeTunnel();
    if (!t) return;
    const routes = routesFor(t.id);
    const rows = routes.map((r, index) => {
      const dns = dnsMeta(r.dns);
      const pathLabel = r.path ? `路径 ${escapeHtml(r.path)}` : '根路径';
      return `
        <div class="route-row" draggable="true" data-id="${r.id}">
          <span class="route-index">${String(index + 1).padStart(2, '0')}</span>
          <button class="drag-handle" type="button" aria-label="拖动排序">${icon('grip')}</button>
          <div class="route-host">
            <strong>${escapeHtml(r.host)}</strong>
            <small>${pathLabel}</small>
          </div>
          <span class="route-arrow">${icon('arrow-right')}</span>
          <div class="route-service">${escapeHtml(r.service)}</div>
          <span class="pill ${dns.pill}">${dns.label}</span>
          <div class="route-actions">
            <button class="icon-btn" type="button" data-action="edit-route" data-id="${r.id}" aria-label="编辑路由">${icon('pencil')}</button>
            <button class="icon-btn" type="button" data-action="delete-route" data-id="${r.id}" aria-label="删除路由">${icon('trash')}</button>
          </div>
        </div>`;
    }).join('');

    root.innerHTML = rows + `
      <div class="route-row is-catchall">
        <span class="route-index">—</span>
        <span class="drag-handle" style="opacity:.35">${icon('lock')}</span>
        <div class="route-host"><strong>*</strong><small>兜底规则，始终最后</small></div>
        <span class="route-arrow">${icon('arrow-right')}</span>
        <div class="route-service">http_status:404</div>
        <span class="pill neutral">固定</span>
        <div class="route-actions"></div>
      </div>`;

    renderDnsSummary();
    bindRouteDrag();
  }

  function renderDnsSummary() {
    const root = $('#dns-summary');
    if (!root) return;
    const t = activeTunnel();
    const routes = t ? routesFor(t.id) : [];
    const created = routes.filter((r) => r.dns === 'created').length;
    const pending = routes.filter((r) => r.dns === 'pending').length;
    const failed = routes.filter((r) => r.dns === 'failed').length;
    root.innerHTML = `
      <span class="pill healthy">${created} 已创建</span>
      ${pending ? `<span class="pill warning">${pending} 待创建</span>` : ''}
      ${failed ? `<span class="pill danger">${failed} 失败</span>` : ''}
      <span class="micro">CNAME 指向当前隧道的 cfargotunnel.com</span>`;
  }

  function bindRouteDrag() {
    let dragId = null;
    $$('.route-row[draggable="true"]').forEach((row) => {
      row.addEventListener('dragstart', (e) => {
        dragId = row.dataset.id;
        e.dataTransfer.effectAllowed = 'move';
        e.dataTransfer.setData('text/plain', dragId);
        row.classList.add('is-dragging');
      });
      row.addEventListener('dragend', () => {
        row.classList.remove('is-dragging');
        $$('.route-row').forEach((r) => r.classList.remove('is-drag-over'));
      });
      row.addEventListener('dragover', (e) => {
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';
        if (row.dataset.id !== dragId) row.classList.add('is-drag-over');
      });
      row.addEventListener('dragleave', () => row.classList.remove('is-drag-over'));
      row.addEventListener('drop', (e) => {
        e.preventDefault();
        row.classList.remove('is-drag-over');
        if (!dragId || dragId === row.dataset.id) return;
        const t = activeTunnel();
        if (!t) return;
        const list = routesFor(t.id);
        const from = list.findIndex((r) => r.id === dragId);
        const to = list.findIndex((r) => r.id === row.dataset.id);
        if (from < 0 || to < 0) return;
        const [moved] = list.splice(from, 1);
        list.splice(to, 0, moved);
        pushActivity('路由顺序已更新', 'blue', 'list');
        toast('路由顺序已更新', 'success');
        renderRouteRows();
        renderYaml();
      });
    });
  }

  function rawYaml() {
    const t = activeTunnel();
    if (!t) return '';
    const routes = routesFor(t.id);
    const lines = [
      '# 由 Tunnel 路由表生成',
      `tunnel: ${t.id}`,
      'credentials-file: /var/apps/cloudflare-tunnel/var/credentials.json',
      'ingress:'
    ];
    routes.forEach((r) => {
      lines.push(`  - hostname: ${r.host}`);
      if (r.path) lines.push(`    path: ${r.path}`);
      lines.push(`    service: ${r.service}`);
      if (r.noTLSVerify || r.hostHeader) {
        lines.push('    originRequest:');
        if (r.noTLSVerify) lines.push('      noTLSVerify: true');
        if (r.hostHeader) lines.push(`      httpHostHeader: ${r.hostHeader}`);
      }
    });
    lines.push('  - service: http_status:404');
    return lines.join('\n');
  }

  function renderYaml() {
    const el = $('#yaml-code');
    if (!el) return;
    const t = activeTunnel();
    if (!t) {
      el.textContent = '# 连接隧道后生成配置';
      return;
    }
    const routes = routesFor(t.id);
    const lines = [];
    lines.push(`<span class="y-cmt"># 由 Tunnel 路由表生成</span>`);
    lines.push(`<span class="y-key">tunnel</span>: <span class="y-val">${escapeHtml(t.id)}</span>`);
    lines.push(`<span class="y-key">credentials-file</span>: <span class="y-val">/var/apps/cloudflare-tunnel/var/credentials.json</span>`);
    lines.push(`<span class="y-key">ingress</span>:`);
    routes.forEach((r) => {
      lines.push(`  <span class="y-dash">-</span> <span class="y-key">hostname</span>: <span class="y-val">${escapeHtml(r.host)}</span>`);
      if (r.path) lines.push(`    <span class="y-key">path</span>: <span class="y-val">${escapeHtml(r.path)}</span>`);
      lines.push(`    <span class="y-key">service</span>: <span class="y-val">${escapeHtml(r.service)}</span>`);
      if (r.noTLSVerify || r.hostHeader) {
        lines.push(`    <span class="y-key">originRequest</span>:`);
        if (r.noTLSVerify) lines.push(`      <span class="y-key">noTLSVerify</span>: <span class="y-val">true</span>`);
        if (r.hostHeader) lines.push(`      <span class="y-key">httpHostHeader</span>: <span class="y-val">${escapeHtml(r.hostHeader)}</span>`);
      }
    });
    lines.push(`  <span class="y-dash">-</span> <span class="y-key">service</span>: <span class="y-val">http_status:404</span>`);
    el.innerHTML = lines.join('\n');
  }

  function openRouteDrawer(mode, id) {
    const t = activeTunnel();
    if (!t) {
      toast('请先连接一个隧道，再添加路由', 'error');
      return;
    }
    const existing = mode === 'edit' ? routesFor(t.id).find((r) => r.id === id) : null;
    if (mode === 'edit' && !existing) return;
    const route = existing || { host: '', service: '', dns: 'pending', noTLSVerify: false, path: '', hostHeader: '' };
    const html = `
      <aside class="drawer" role="dialog" aria-modal="true" aria-labelledby="route-drawer-title">
        <div class="drawer-head">
          <div>
            <p class="eyebrow">${mode === 'edit' ? '编辑' : '新增'}</p>
            <h2 id="route-drawer-title">${mode === 'edit' ? '编辑路由' : '添加路由'}</h2>
            <p>${escapeHtml(t.name)} · 保存后立即应用到运行中的隧道</p>
          </div>
          <button class="icon-btn" type="button" data-close aria-label="关闭">${icon('x')}</button>
        </div>
        <form id="route-form" class="drawer-body" novalidate>
          <div class="form-grid">
            <div class="field" data-field="host">
              <label for="route-host">域名</label>
              <input class="input mono" id="route-host" name="host" placeholder="app.example.com" value="${escapeHtml(route.host)}" autocomplete="off" spellcheck="false">
              <span class="hint">需要已托管在 Cloudflare 的域名；保存后自动创建 CNAME。</span>
              <span class="error">请输入有效域名，例如 app.example.com</span>
            </div>
            <div class="field" data-field="service">
              <label for="route-service">内网服务</label>
              <input class="input mono" id="route-service" name="service" placeholder="http://127.0.0.1:5666" value="${escapeHtml(route.service)}" autocomplete="off" spellcheck="false">
              <span class="hint">支持 http://、https://、tcp://、ssh://、rdp://、smb:// 和 unix:。</span>
              <span class="error">请输入有效的服务地址，例如 http://127.0.0.1:5666</span>
            </div>
            <label class="check-row">
              <span class="switch"><input id="route-auto-dns" type="checkbox" ${route.dns === 'created' || mode === 'add' ? 'checked' : ''}><i></i></span>
              <span class="check-copy"><strong>自动创建 CNAME</strong><small>指向 ${shortId(t.id)}.cfargotunnel.com；需要 Zone DNS 编辑权限。</small></span>
            </label>
            <label class="check-row">
              <span class="switch"><input id="route-no-tls" type="checkbox" ${route.noTLSVerify ? 'checked' : ''}><i></i></span>
              <span class="check-copy"><strong>不验证内网 HTTPS 证书</strong><small>对应 originRequest.noTLSVerify；仅建议对自签名证书使用。</small></span>
            </label>
            <details class="advanced">
              <summary>高级选项</summary>
              <div class="form-grid">
                <div class="field" data-field="path">
                  <label for="route-path">路径前缀（可选）</label>
                  <input class="input mono" id="route-path" name="path" placeholder="/api" value="${escapeHtml(route.path)}" autocomplete="off" spellcheck="false">
                  <span class="hint">留空表示匹配整个域名；必须以 / 开头。</span>
                  <span class="error">路径必须以 / 开头</span>
                </div>
                <div class="field">
                  <label for="route-host-header">Host 头重写（可选）</label>
                  <input class="input mono" id="route-host-header" name="hostHeader" placeholder="internal.example.local" value="${escapeHtml(route.hostHeader)}" autocomplete="off" spellcheck="false">
                  <span class="hint">对应 originRequest.httpHostHeader。</span>
                </div>
              </div>
            </details>
            <div class="modal-warning">${icon('info')} 规则会立即写入当前隧道的 ingress；原型的“保存”只更新本地演示状态。</div>
          </div>
        </form>
        <div class="drawer-foot">
          <span class="note">首条命中生效<br>兜底 404 自动保留</span>
          <div class="actions">
            <button class="btn ghost" type="button" data-close>取消</button>
            <button class="btn primary" type="submit" form="route-form">${mode === 'edit' ? '保存更改' : '保存并应用'}</button>
          </div>
        </div>
      </aside>`;

    const { layer, close } = openLayer({ html, kind: 'drawer' });
    const form = layer.querySelector('#route-form');
    const hostInput = layer.querySelector('#route-host');
    const serviceInput = layer.querySelector('#route-service');
    const pathInput = layer.querySelector('#route-path');
    const hostHeaderInput = layer.querySelector('#route-host-header');
    const autoDns = layer.querySelector('#route-auto-dns');
    const noTls = layer.querySelector('#route-no-tls');
    const clearError = (input) => input.closest('.field')?.classList.remove('has-error');
    [hostInput, serviceInput, pathInput].forEach((input) => input.addEventListener('input', () => clearError(input)));

    form.addEventListener('submit', (e) => {
      e.preventDefault();
      const host = hostInput.value.trim().toLowerCase();
      const service = serviceInput.value.trim();
      const path = pathInput.value.trim();
      const hostHeader = hostHeaderInput.value.trim();
      let invalid = false;
      if (!/^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)+$/.test(host)) {
        hostInput.closest('.field').classList.add('has-error');
        invalid = true;
      } else if (state.routesByTunnel[t.id].some((r) => r.host === host && r.id !== route.id)) {
        hostInput.closest('.field').classList.add('has-error');
        invalid = true;
      }
      if (!/^(?:https?|tcp|ssh|rdp|smb):\/\/|^unix:/.test(service)) {
        serviceInput.closest('.field').classList.add('has-error');
        invalid = true;
      }
      if (path && !path.startsWith('/')) {
        pathInput.closest('.field').classList.add('has-error');
        invalid = true;
      }
      if (invalid) {
        (layer.querySelector('.field.has-error .input') || hostInput).focus();
        return;
      }

      const list = state.routesByTunnel[t.id];
      const saved = {
        id: route.id || `r-${uid()}`,
        host,
        service,
        dns: autoDns.checked ? 'created' : (route.dns === 'created' ? 'created' : 'pending'),
        noTLSVerify: noTls.checked,
        path,
        hostHeader
      };
      if (mode === 'edit') {
        const index = list.findIndex((r) => r.id === route.id);
        if (index >= 0) list[index] = saved;
      } else {
        list.push(saved);
      }
      pushActivity(`路由 ${host} 已${mode === 'edit' ? '更新' : '添加'}`, 'blue', 'branch');
      toast(autoDns.checked ? '路由已保存 · CNAME 已创建' : '路由已保存 · 等待手动创建 DNS', 'success');
      close();
      safeRefresh();
    });
    setTimeout(() => hostInput.focus(), 120);
  }

  async function deleteRoute(id) {
    const t = activeTunnel();
    if (!t) return;
    const route = routesFor(t.id).find((r) => r.id === id);
    if (!route) return;
    const result = await dialog({
      title: '删除路由？',
      message: `删除 ${route.host} 的转发规则。删除后，该域名将不再指向任何内网服务。`,
      confirmText: '删除路由',
      tone: 'danger',
      checkbox: '同时删除 Cloudflare 上的 DNS 记录'
    });
    if (!result.ok) return;
    state.routesByTunnel[t.id] = routesFor(t.id).filter((r) => r.id !== id);
    pushActivity(`路由 ${route.host} 已删除`, 'danger', 'trash');
    toast(result.checked ? '路由与 DNS 记录已删除' : '路由已删除，DNS 记录保留', 'success');
    safeRefresh();
  }

  /* ---------- logs ---------- */
  function tplLogs() {
    const t = activeTunnel();
    return `
      <section class="page-head reveal" style="--i:0">
        <div>
          <p class="eyebrow">诊断</p>
          <h1>运行日志</h1>
          <p class="sub">来自 cloudflared 的实时输出。日志只在本地保留，不上传云端。</p>
        </div>
        <div class="page-actions">
          <button class="btn ghost" type="button" data-action="download-logs">${icon('download')}下载</button>
          <button class="btn danger" type="button" data-action="clear-logs">${icon('trash')}清空</button>
        </div>
      </section>

      <div class="toolbar reveal" style="--i:1">
        <span class="pill healthy" id="log-live-pill"><span class="dot"></span>实时流</span>
        <button class="btn ghost small" type="button" data-action="toggle-log-pause" id="log-pause-btn">${icon('pause')}暂停</button>
        <div class="segmented" role="group" aria-label="日志级别">
          <button type="button" data-level="all" class="${state.logLevel === 'all' ? 'is-active' : ''}">全部</button>
          <button type="button" data-level="inf" class="${state.logLevel === 'inf' ? 'is-active' : ''}">INF</button>
          <button type="button" data-level="wrn" class="${state.logLevel === 'wrn' ? 'is-active' : ''}">WRN</button>
          <button type="button" data-level="err" class="${state.logLevel === 'err' ? 'is-active' : ''}">ERR</button>
        </div>
        <div class="search">
          ${icon('search')}
          <input id="log-search" type="search" placeholder="搜索日志、域名或连接..." value="${escapeHtml(state.logQuery)}" autocomplete="off">
        </div>
        <div class="spacer"></div>
        <label class="switch-row" style="width:auto;gap:8px;font-size:10px;color:var(--ink-3)">
          自动滚动
          <span class="switch"><input id="auto-scroll" type="checkbox" ${state.autoScroll ? 'checked' : ''}><i></i></span>
        </label>
      </div>

      <div class="logs-layout">
        <section class="card log-console reveal" style="--i:2">
          <div class="log-head">
            <div class="log-head-left"><span class="live-dot"></span>cloudflared · stdout<span class="log-count" id="log-count"></span></div>
            <div class="log-head-right"><span>tail -f</span></div>
          </div>
          <div class="log-body" id="log-body"><div id="log-rows"></div></div>
        </section>
        <aside class="log-side reveal" style="--i:3">
          <section class="card info-card">
            <h3>进程</h3>
            <div class="kv"><span>PID</span><strong class="mono">${t && t.status === 'healthy' ? '5345' : '—'}</strong></div>
            <div class="kv"><span>运行时长</span><strong>${t && t.status === 'healthy' ? '3 天 17 小时' : '—'}</strong></div>
            <div class="kv"><span>架构</span><strong class="mono">amd64</strong></div>
            <div class="kv"><span>cloudflared</span><strong class="mono">2026.10.7</strong></div>
          </section>
          <section class="card info-card">
            <h3>连接</h3>
            <div class="kv"><span>边缘节点</span><strong class="mono">HKG</strong></div>
            <div class="kv"><span>协议</span><strong class="mono">QUIC</strong></div>
            <div class="kv"><span>连接数</span><strong>4 / 4</strong></div>
            <div class="kv"><span>Metrics</span><strong class="mono">:60123</strong></div>
          </section>
          <section class="card info-card">
            <h3>日志级别</h3>
            <div class="legend"><span><i class="inf"></i>INF 信息</span><span><i class="wrn"></i>WRN 警告</span><span><i class="err"></i>ERR 错误</span><span><i class="dbg"></i>DBG 调试</span></div>
            <p class="micro" style="margin:10px 0 0">内存保留 2,000 行，落盘 30 天。点击任意日志可展开 JSON 详情。</p>
          </section>
        </aside>
      </div>`;
  }

  function seedLogs() {
    const seeds = [
      ['INF', 'Registered tunnel connection', 'connIndex=0 location=HKG protocol=quic', 26],
      ['INF', 'Updated to new configuration', 'config_version=7', 23],
      ['INF', 'Request completed', 'GET nas.example.com 200 42ms', 19],
      ['WRN', 'Connection terminated', 'connIndex=1', 15],
      ['INF', 'Retrying connection in 1s', 'connIndex=1', 13],
      ['INF', 'Registered tunnel connection', 'connIndex=1 location=HKG protocol=quic', 11],
      ['ERR', 'Unable to reach the origin service', 'host=ai.example.com error=connection refused', 8],
      ['INF', 'Request completed', 'GET ha.example.com 200 61ms', 5],
      ['DBG', 'Metrics server listening', 'addr=127.0.0.1:60123', 3],
      ['INF', 'Heartbeat acknowledged', 'connIndex=0', 1]
    ];
    seeds.forEach(([level, msg, meta, ago]) => addLog(makeLog(level, msg, meta, ago * 1000), false));
  }

  function makeLog(level, msg, meta = '', agoMs = 0, detail = null) {
    state.logSeq += 1;
    return {
      id: `l${state.logSeq}`,
      date: new Date(Date.now() - agoMs),
      level,
      msg,
      meta,
      detail: detail || {
        level: level.toLowerCase(),
        event: msg,
        meta,
        connIndex: 0,
        location: 'HKG',
        protocol: 'quic',
        config_version: 7
      }
    };
  }

  function randomLog() {
    const pool = [
      ['INF', 'Registered tunnel connection', 'connIndex=0 location=HKG protocol=quic'],
      ['INF', 'Request completed', 'GET nas.example.com 200 42ms'],
      ['INF', 'Request completed', 'GET ha.example.com 200 57ms'],
      ['INF', 'Updated to new configuration', 'config_version=7'],
      ['WRN', 'Connection terminated', 'connIndex=2'],
      ['INF', 'Retrying connection in 1s', 'connIndex=2'],
      ['INF', 'Registered tunnel connection', 'connIndex=2 location=HKG protocol=quic'],
      ['ERR', 'Unable to reach the origin service', 'host=ai.example.com error=connection refused'],
      ['INF', 'Heartbeat acknowledged', 'connIndex=0'],
      ['DBG', 'Metrics scrape completed', 'addr=127.0.0.1:60123']
    ];
    const item = pool[Math.floor(Math.random() * pool.length)];
    return makeLog(item[0], item[1], item[2], 0);
  }

  function addLog(entry, live = true) {
    state.logs.push(entry);
    if (state.logs.length > 300) state.logs.shift();
    if (live && state.page === 'logs' && logMatches(entry)) {
      appendLogRow(entry);
      updateLogCount();
      if (state.autoScroll) scrollLogsToBottom();
    }
  }

  function logMatches(entry) {
    if (state.logLevel !== 'all' && entry.level.toLowerCase() !== state.logLevel) return false;
    const q = state.logQuery.trim().toLowerCase();
    if (!q) return true;
    return `${entry.msg} ${entry.meta}`.toLowerCase().includes(q);
  }

  function renderLogRows() {
    const root = $('#log-rows');
    if (!root) return;
    const entries = state.logs.filter(logMatches);
    root.innerHTML = entries.length
      ? entries.map(logRowHtml).join('')
      : `<div class="log-empty">没有匹配的日志。调整级别或搜索词。</div>`;
    updateLogCount();
  }

  function logRowHtml(entry) {
    return `
      <div class="log-entry" data-log-id="${entry.id}">
        <button class="log-line" type="button" aria-expanded="false">
          <span class="log-time">${formatClock(entry.date)}</span>
          <span class="log-level ${entry.level.toLowerCase()}">${entry.level}</span>
          <span class="log-msg">${escapeHtml(entry.msg)}${entry.meta ? `<span class="meta">${escapeHtml(entry.meta)}</span>` : ''}</span>
        </button>
        <pre class="log-detail" hidden>${escapeHtml(JSON.stringify(entry.detail, null, 2))}</pre>
      </div>`;
  }

  function appendLogRow(entry) {
    const root = $('#log-rows');
    if (!root) return;
    const empty = root.querySelector('.log-empty');
    if (empty) empty.remove();
    root.insertAdjacentHTML('beforeend', logRowHtml(entry));
    const rows = $$('.log-entry', root);
    if (rows.length > 300) rows[0].remove();
  }

  function updateLogCount() {
    const el = $('#log-count');
    if (!el) return;
    const matching = state.logs.filter(logMatches).length;
    el.textContent = `${matching} / ${state.logs.length} 行`;
  }

  function scrollLogsToBottom() {
    const body = $('#log-body');
    if (!body) return;
    lastAutoScrollAt = Date.now();
    body.scrollTop = body.scrollHeight;
  }

  function toggleLogPause() {
    state.logPaused = !state.logPaused;
    updateLogToolbar();
    if (!state.logPaused && state.autoScroll) scrollLogsToBottom();
  }

  function updateLogToolbar() {
    const pill = $('#log-live-pill');
    const btn = $('#log-pause-btn');
    if (pill) {
      if (state.logPaused) {
        pill.className = 'pill warning';
        pill.innerHTML = '<span class="dot"></span>已暂停';
      } else {
        pill.className = 'pill healthy';
        pill.innerHTML = '<span class="dot"></span>实时流';
      }
    }
    if (btn) btn.innerHTML = state.logPaused ? `${icon('play')}继续` : `${icon('pause')}暂停`;
  }

  function clearLogs() {
    state.logs = [];
    renderLogRows();
    toast('日志已清空', 'success');
  }

  function downloadLogs() {
    if (!state.logs.length) {
      toast('暂无可下载的日志', 'error');
      return;
    }
    const text = state.logs.map((l) => `${l.date.toISOString()} ${l.level} ${l.msg} ${l.meta}`.trim()).join('\n');
    const blob = new Blob([text], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'cloudflared-2026-11-05.log';
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
    toast('日志已下载', 'success');
  }

  /* ---------- api / mcp ---------- */
  function tplApi() {
    const credentials = state.credentials.map((c) => `
      <div class="credential-row">
        <span class="credential-icon">${icon('key')}</span>
        <span class="credential-copy"><strong>${escapeHtml(c.name)}</strong><small class="mono">${escapeHtml(c.scopes.join(' · '))}</small></span>
        <span class="pill neutral">${escapeHtml(c.last)}</span>
      </div>`).join('');

    return `
      <section class="page-head reveal" style="--i:0">
        <div>
          <p class="eyebrow">集成</p>
          <h1>接入</h1>
          <p class="sub">通过 HTTP API 或 MCP 把隧道能力接进其他应用。接口层会随 MVP 一起预留，二期开放。</p>
        </div>
        <div class="page-actions"><span class="pill info">二期</span></div>
      </section>

      <div class="banner reveal" style="--i:1">
        ${icon('info')}
        <div><strong>接口层已在架构中预留。</strong> 核心版先交付隧道、路由与日志；API 凭证、HTTP API 和 MCP server 在二期接入，不需要重做数据模型。</div>
      </div>

      <div class="integration-grid">
        <div style="display:flex;flex-direction:column;gap:14px">
          <section class="card endpoint-card reveal" style="--i:2">
            <h3>${icon('braces')}HTTP API</h3>
            <p>给脚本、自动化或自建面板使用。所有调用都在本机完成，不经过第三方。</p>
            <div class="endpoint-row">
              <span class="mono">http://your-nas:19092</span>
              <button class="icon-btn" type="button" data-action="copy-endpoint" data-copy="http://your-nas:19092" aria-label="复制地址">${icon('copy')}</button>
            </div>
            <div class="endpoint-meta">${icon('lock')} Bearer Token · 只监听局域网 · 可绑定 fnOS 网关</div>
            <pre class="code-block"><span class="c"># 查询隧道状态</span>
curl <span class="s">http://your-nas:19092/api/tunnels</span> \\
  -H <span class="s">"Authorization: Bearer tun_..."</span></pre>
          </section>

          <section class="card endpoint-card reveal" style="--i:3">
            <h3>${icon('radio')}MCP Server</h3>
            <p>让 AI 客户端直接读取隧道状态、创建路由或查看日志。</p>
            <div class="endpoint-row">
              <span class="mono">http://your-nas:19093/mcp</span>
              <button class="icon-btn" type="button" data-action="copy-endpoint" data-copy="http://your-nas:19093/mcp" aria-label="复制地址">${icon('copy')}</button>
            </div>
            <div class="endpoint-meta">${icon('plug')} Streamable HTTP · 复用同一套 API 凭证</div>
          </section>
        </div>

        <section class="card credentials reveal" style="--i:4">
          <div class="card-head">
            <div><h3>API 凭证</h3><p>按用途隔离权限，可随时吊销</p></div>
            <button class="btn small ghost" type="button" data-action="new-credential">${icon('plus')}新建</button>
          </div>
          <div style="height:8px"></div>
          ${credentials}
          <div class="card-foot" style="padding-top:12px">
            <span class="micro">Token 只在创建时显示一次；服务端以 0600 保存哈希。</span>
          </div>
        </section>
      </div>`;
  }

  function openCredentialModal() {
    const scopes = [
      ['tunnels:read', '读取隧道与状态'],
      ['tunnels:write', '创建、连接、删除隧道'],
      ['routes:read', '读取 Ingress 规则'],
      ['routes:write', '修改 Ingress 与 DNS'],
      ['logs:read', '读取运行日志']
    ];
    const html = `
      <div class="modal" role="dialog" aria-modal="true" aria-labelledby="credential-title">
        <div class="modal-head">
          <div><p class="eyebrow">API</p><h2 id="credential-title">新建凭证</h2></div>
          <button class="icon-btn" type="button" data-close aria-label="关闭">${icon('x')}</button>
        </div>
        <form id="credential-form" class="modal-body">
          <div class="form-grid">
            <div class="field" data-field="cred-name">
              <label for="credential-name">名称</label>
              <input class="input" id="credential-name" placeholder="Home Assistant" autocomplete="off">
              <span class="hint">用应用或用途命名，方便以后吊销。</span>
              <span class="error">请输入凭证名称</span>
            </div>
            <div class="form-grid" style="gap:8px">
              ${scopes.map(([scope, desc]) => `
                <label class="check-row">
                  <span class="switch"><input type="checkbox" data-scope="${scope}" ${scope.endsWith(':read') ? 'checked' : ''}><i></i></span>
                  <span class="check-copy"><strong>${scope}</strong><small>${desc}</small></span>
                </label>`).join('')}
            </div>
            <div class="modal-warning">${icon('shield')} 原型只生成演示 Token，不会写入任何真实服务。</div>
          </div>
        </form>
        <div class="modal-foot">
          <button class="btn ghost" type="button" data-close>取消</button>
          <button class="btn primary" type="submit" form="credential-form">生成凭证</button>
        </div>
      </div>`;
    const { layer, close } = openLayer({ html });
    const form = layer.querySelector('#credential-form');
    const nameInput = layer.querySelector('#credential-name');
    nameInput.addEventListener('input', () => nameInput.closest('.field').classList.remove('has-error'));
    form.addEventListener('submit', (e) => {
      e.preventDefault();
      const name = nameInput.value.trim();
      if (!name) {
        nameInput.closest('.field').classList.add('has-error');
        nameInput.focus();
        return;
      }
      const selected = $$('[data-scope]:checked', layer).map((el) => el.dataset.scope);
      if (!selected.length) {
        toast('至少选择一个权限范围', 'error');
        return;
      }
      const token = randomToken();
      state.credentials.push({ id: uid(), name, scopes: selected, last: '从未使用' });
      pushActivity(`API 凭证 ${name} 已创建`, 'blue', 'key');
      close();
      safeRefresh();
      setTimeout(() => showTokenModal(token), 180);
    });
    setTimeout(() => nameInput.focus(), 80);
  }

  function showTokenModal(token) {
    const html = `
      <div class="modal" role="dialog" aria-modal="true" aria-labelledby="token-title">
        <div class="modal-head">
          <div><p class="eyebrow">凭证</p><h2 id="token-title">已生成</h2></div>
          <button class="icon-btn" type="button" data-close aria-label="关闭">${icon('x')}</button>
        </div>
        <div class="modal-body">
          <div class="modal-warning">${icon('shield')} Token 只显示一次。关闭后只能吊销并重新生成。</div>
          <div class="endpoint-row"><span class="mono" style="font-size:10px">${escapeHtml(token)}</span><button class="icon-btn" type="button" data-action="copy-endpoint" data-copy="${escapeHtml(token)}" aria-label="复制 Token">${icon('copy')}</button></div>
          <p style="margin-top:12px">客户端在请求头中携带：<span class="mono">Authorization: Bearer &lt;token&gt;</span></p>
        </div>
        <div class="modal-foot"><button class="btn primary" type="button" data-close>我已保存</button></div>
      </div>`;
    openLayer({ html });
    toast('凭证已创建', 'success');
  }

  /* ---------- settings ---------- */
  function tplSettings() {
    const s = state.settings;
    return `
      <section class="page-head reveal" style="--i:0">
        <div>
          <p class="eyebrow">配置</p>
          <h1>设置</h1>
          <p class="sub">账户、连接行为、外观与日志保留。修改后需要保存。</p>
        </div>
        <div class="page-actions">
          <span class="save-state ${state.settingsDirty ? 'is-dirty' : ''}" id="save-state">${state.settingsDirty ? '有未保存的更改' : '已保存'}</span>
          <button class="btn primary" type="button" data-action="save-settings" id="save-settings" ${state.settingsDirty ? '' : 'disabled'}>保存更改</button>
        </div>
      </section>

      <div class="settings-wrap">
        <section class="card settings-card reveal" style="--i:1">
          <div class="card-head"><div><h3>Cloudflare 账户</h3><p>使用最小权限 API Token；应用不会保存账户密码。</p></div></div>
          <div class="setting-row">
            <div class="setting-copy"><strong>Account ID</strong><small>Cloudflare 账户 ID，用于调用 Tunnel API。</small></div>
            <div class="setting-control">
              <input class="input mono" style="width:100%" data-setting="accountId" value="${escapeHtml(s.accountId)}" spellcheck="false">
            </div>
          </div>
          <div class="setting-row">
            <div class="setting-copy"><strong>API Token</strong><small>需要 Account Tunnel 与 Zone DNS 的读写权限。</small></div>
            <div class="setting-control">
              <div class="input-wrap">
                <input class="input mono" id="api-token" type="password" data-setting="token" value="${escapeHtml(s.token)}" spellcheck="false">
                <button class="icon-btn" type="button" data-action="toggle-token" aria-label="显示 Token">${icon('eye')}</button>
              </div>
            </div>
          </div>
          <div class="setting-row">
            <div class="setting-copy"><strong>连接验证</strong><small>验证 Token 是否可访问当前账户与 Zone。</small></div>
            <div class="setting-control" style="justify-content:space-between">
              <span class="verify-line" id="verify-line">${icon('check')}${s.verified ? '已验证 · 3 个 Zone · 权限完整' : '尚未验证'}</span>
              <button class="btn small ghost" type="button" data-action="verify-account" id="verify-account">${icon('shield')}重新验证</button>
            </div>
          </div>
          <div class="setting-row">
            <div class="setting-copy"><strong>权限说明</strong><small>查看应用所需的最小权限集。</small></div>
            <div class="setting-control" style="justify-content:flex-start">
              <button class="link" type="button" data-action="token-help">如何创建最小权限 Token →</button>
            </div>
          </div>
        </section>

        <section class="card settings-card reveal" style="--i:2">
          <div class="card-head"><div><h3>连接</h3><p>cloudflared 与 Cloudflare Edge 的握手方式。</p></div></div>
          <div class="setting-row">
            <div class="setting-copy"><strong>连接协议</strong><small>UDP 受限的网络请切换到 HTTP/2。</small></div>
            <div class="setting-control">
              <div class="segmented" role="group" aria-label="连接协议">
                <button type="button" data-setting-choice="protocol" data-value="quic" class="${s.protocol === 'quic' ? 'is-active' : ''}">QUIC</button>
                <button type="button" data-setting-choice="protocol" data-value="http2" class="${s.protocol === 'http2' ? 'is-active' : ''}">HTTP/2</button>
              </div>
            </div>
          </div>
          <div class="setting-row">
            <div class="setting-copy"><strong>边缘 IP 版本</strong><small>IPv6 可用时通常延迟更低；不确定时选自动。</small></div>
            <div class="setting-control">
              <select class="select" data-setting="edgeIp" style="min-width:150px">
                <option value="auto" ${s.edgeIp === 'auto' ? 'selected' : ''}>自动</option>
                <option value="4" ${s.edgeIp === '4' ? 'selected' : ''}>仅 IPv4</option>
                <option value="6" ${s.edgeIp === '6' ? 'selected' : ''}>仅 IPv6</option>
              </select>
            </div>
          </div>
          <div class="setting-row">
            <div class="setting-copy"><strong>失败时自动重试</strong><small>cloudflared 异常退出后按指数退避重启。</small></div>
            <div class="setting-control"><span class="switch"><input type="checkbox" data-setting="retry" ${s.retry ? 'checked' : ''}><i></i></span></div>
          </div>
        </section>

        <section class="card settings-card reveal" style="--i:3">
          <div class="card-head"><div><h3>启动</h3><p>fnOS 重启或应用重新打开后的行为。</p></div></div>
          <div class="setting-row">
            <div class="setting-copy"><strong>开机自动连接</strong><small>fnOS 启动后自动恢复上一次的隧道连接。</small></div>
            <div class="setting-control"><span class="switch"><input type="checkbox" data-setting="autoConnect" ${s.autoConnect ? 'checked' : ''}><i></i></span></div>
          </div>
          <div class="setting-row">
            <div class="setting-copy"><strong>随应用启动 cloudflared</strong><small>关闭时只在用户点击“连接”后启动进程。</small></div>
            <div class="setting-control"><span class="switch"><input type="checkbox" data-setting="autoStart" ${s.autoStart ? 'checked' : ''}><i></i></span></div>
          </div>
        </section>

        <section class="card settings-card reveal" style="--i:4">
          <div class="card-head"><div><h3>外观</h3><p>默认使用飞牛原生浅色；也可以跟随系统。</p></div></div>
          <div class="setting-row">
            <div class="setting-copy"><strong>主题</strong><small>浅色与深色共用同一套布局和状态色。</small></div>
            <div class="setting-control">
              <div class="segmented" role="group" aria-label="主题">
                <button type="button" data-setting-choice="theme" data-value="system" class="${state.themePref === 'system' ? 'is-active' : ''}">跟随系统</button>
                <button type="button" data-setting-choice="theme" data-value="light" class="${state.themePref === 'light' ? 'is-active' : ''}">浅色</button>
                <button type="button" data-setting-choice="theme" data-value="dark" class="${state.themePref === 'dark' ? 'is-active' : ''}">深色</button>
              </div>
            </div>
          </div>
          <div class="setting-row">
            <div class="setting-copy"><strong>减少动效</strong><small>关闭页面进入、脉冲和打包动画；保留必要的状态过渡。</small></div>
            <div class="setting-control"><span class="switch"><input type="checkbox" data-setting="reduceMotion" ${s.reduceMotion ? 'checked' : ''}><i></i></span></div>
          </div>
        </section>

        <section class="card settings-card reveal" style="--i:5">
          <div class="card-head"><div><h3>日志</h3><p>只保留本地日志，不自动上传。</p></div></div>
          <div class="setting-row">
            <div class="setting-copy"><strong>保留天数</strong><small>超过期限的日志文件会被清理。</small></div>
            <div class="setting-control">
              <select class="select" data-setting="retention" style="min-width:120px">
                <option value="7" ${s.retention === '7' ? 'selected' : ''}>7 天</option>
                <option value="30" ${s.retention === '30' ? 'selected' : ''}>30 天</option>
                <option value="90" ${s.retention === '90' ? 'selected' : ''}>90 天</option>
              </select>
            </div>
          </div>
          <div class="setting-row">
            <div class="setting-copy"><strong>内存行数</strong><small>日志页面保留的最大行数。</small></div>
            <div class="setting-control">
              <select class="select" data-setting="maxLines" style="min-width:120px">
                <option value="1000" ${s.maxLines === '1000' ? 'selected' : ''}>1,000</option>
                <option value="2000" ${s.maxLines === '2000' ? 'selected' : ''}>2,000</option>
                <option value="5000" ${s.maxLines === '5000' ? 'selected' : ''}>5,000</option>
              </select>
            </div>
          </div>
        </section>

        <section class="card settings-card danger-zone reveal" style="--i:6">
          <div class="card-head"><div><h3>危险区</h3><p>这些操作只影响本地配置。</p></div></div>
          <div class="setting-row">
            <div class="setting-copy"><strong>重置本地配置</strong><small>恢复默认账户、连接与外观设置，不删除隧道和路由。</small></div>
            <div class="setting-control" style="justify-content:flex-start">
              <button class="btn danger" type="button" data-action="reset-config">${icon('refresh')}重置配置</button>
            </div>
          </div>
        </section>
      </div>`;
  }

  function bindSettings() {
    $$('#page [data-setting]').forEach((el) => {
      const key = el.dataset.setting;
      const eventName = el.type === 'checkbox' ? 'change' : 'change';
      el.addEventListener(eventName, () => {
        state.settings[key] = el.type === 'checkbox' ? el.checked : el.value;
        if (key === 'reduceMotion') {
          document.documentElement.classList.toggle('reduce-motion', el.checked);
        }
        markDirty();
      });
      if (el.type !== 'checkbox') {
        el.addEventListener('input', () => {
          state.settings[key] = el.value;
          markDirty();
        });
      }
    });

    $$('#page [data-setting-choice]').forEach((btn) => {
      btn.addEventListener('click', () => {
        const key = btn.dataset.settingChoice;
        const value = btn.dataset.value;
        if (key === 'theme') {
          state.themePref = value;
          localStorage.setItem('tunnel-theme', value);
          applyTheme();
        } else {
          state.settings[key] = value;
        }
        updateSegmentedUI();
        markDirty();
      });
    });
  }

  function updateSegmentedUI() {
    $$('#page [data-setting-choice]').forEach((btn) => {
      const key = btn.dataset.settingChoice;
      const value = btn.dataset.value;
      const selected = key === 'theme' ? state.themePref === value : state.settings[key] === value;
      btn.classList.toggle('is-active', selected);
    });
  }

  function markDirty() {
    state.settingsDirty = true;
    updateSaveUI();
  }

  function updateSaveUI() {
    const btn = $('#save-settings');
    const label = $('#save-state');
    if (btn) btn.disabled = !state.settingsDirty;
    if (label) {
      label.textContent = state.settingsDirty ? '有未保存的更改' : '已保存';
      label.classList.toggle('is-dirty', state.settingsDirty);
    }
  }

  function saveSettings() {
    if (!state.settingsDirty) return;
    const btn = $('#save-settings');
    if (btn) {
      btn.disabled = true;
      btn.innerHTML = '<span class="spinner"></span>保存中';
    }
    setTimeout(() => {
      state.settingsDirty = false;
      updateSaveUI();
      if (btn) btn.innerHTML = '保存更改';
      pushActivity('系统设置已更新', 'blue', 'sliders');
      toast('设置已保存', 'success');
    }, 460);
  }

  function verifyAccount(btn) {
    const original = btn.innerHTML;
    btn.disabled = true;
    btn.innerHTML = '<span class="spinner"></span>验证中';
    setTimeout(() => {
      state.settings.verified = true;
      const line = $('#verify-line');
      if (line) line.innerHTML = `${icon('check')}已验证 · 3 个 Zone · 权限完整`;
      btn.disabled = false;
      btn.innerHTML = original;
      toast('账户验证通过 · 3 个 Zone', 'success');
    }, 900);
  }

  function toggleTokenVisibility(btn) {
    const input = $('#api-token');
    if (!input) return;
    const show = input.type === 'password';
    input.type = show ? 'text' : 'password';
    btn.innerHTML = icon(show ? 'eye-off' : 'eye');
    btn.setAttribute('aria-label', show ? '隐藏 Token' : '显示 Token');
  }

  function openTokenHelp() {
    const html = `
      <div class="modal" role="dialog" aria-modal="true" aria-labelledby="token-help-title">
        <div class="modal-head">
          <div><p class="eyebrow">Cloudflare</p><h2 id="token-help-title">最小权限 Token</h2></div>
          <button class="icon-btn" type="button" data-close aria-label="关闭">${icon('x')}</button>
        </div>
        <div class="modal-body">
          <p>在 Cloudflare Dashboard → My Profile → API Tokens → Create Custom Token 中创建：</p>
          <div class="form-grid" style="gap:8px">
            ${[
              ['账户 · Cloudflare Tunnel', 'Read + Edit', '创建/连接/删除隧道'],
              ['区域 · DNS', 'Read + Edit', '创建 CNAME 记录'],
              ['区域 · Zone', 'Read', '列出域名与 Zone'],
              ['账户 · Access（可选）', 'Apps and Policies', '配置 Access 应用']
            ].map(([scope, perm, why]) => `
              <div class="check-row" style="align-items:center">
                <span class="credential-icon">${icon('shield')}</span>
                <span class="check-copy"><strong>${scope}</strong><small>${why}</small></span>
                <span class="pill info" style="margin-left:auto">${perm}</span>
              </div>`).join('')}
          </div>
          <p style="margin-top:14px">不需要 Global API Key，也不要把 Token 写进 URL 或日志。</p>
        </div>
        <div class="modal-foot"><button class="btn primary" type="button" data-close>知道了</button></div>
      </div>`;
    openLayer({ html });
  }

  async function resetConfig() {
    const result = await dialog({
      title: '重置本地配置？',
      message: '账户、连接、外观和日志设置会恢复默认；隧道与路由不会删除。',
      confirmText: '重置配置',
      tone: 'danger'
    });
    if (!result.ok) return;
    state.settings = {
      accountId: 'a1b2c3d4e5f60718',
      token: 'cf_demo_9f3a2b7c4e1d8a6f',
      verified: true,
      protocol: 'quic',
      edgeIp: 'auto',
      retry: true,
      autoConnect: true,
      autoStart: true,
      retention: '30',
      maxLines: '2000',
      reduceMotion: false
    };
    state.themePref = 'light';
    localStorage.setItem('tunnel-theme', 'light');
    document.documentElement.classList.remove('reduce-motion');
    state.settingsDirty = false;
    applyTheme();
    renderPage({ animate: false });
    toast('本地配置已重置', 'success');
  }

  /* ---------- layers ---------- */
  function openLayer({ html, kind = 'modal', onClose } = {}) {
    const root = $('#overlay-root');
    const layer = document.createElement('div');
    layer.className = kind === 'drawer' ? 'layer drawer-layer' : 'layer';
    layer.innerHTML = `<div class="layer-backdrop"></div>${html}`;
    root.appendChild(layer);
    const previousFocus = document.activeElement;
    let closed = false;
    document.body.style.overflow = 'hidden';

    const close = () => {
      if (closed) return;
      closed = true;
      layer.classList.remove('open');
      document.removeEventListener('keydown', onKey);
      setTimeout(() => {
        layer.remove();
        if (!root.querySelector('.layer')) document.body.style.overflow = '';
        if (onClose) onClose();
        if (previousFocus && previousFocus.focus) previousFocus.focus();
      }, 260);
    };

    const onKey = (e) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        close();
      }
      if (e.key === 'Tab') {
        const focusables = $$('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])', layer)
          .filter((el) => !el.disabled && el.offsetParent !== null);
        if (!focusables.length) return;
        const first = focusables[0];
        const last = focusables[focusables.length - 1];
        if (e.shiftKey && document.activeElement === first) {
          e.preventDefault();
          last.focus();
        } else if (!e.shiftKey && document.activeElement === last) {
          e.preventDefault();
          first.focus();
        }
      }
    };

    layer.querySelector('.layer-backdrop').addEventListener('click', close);
    $$('[data-close]', layer).forEach((el) => el.addEventListener('click', close));
    document.addEventListener('keydown', onKey);
    requestAnimationFrame(() => layer.classList.add('open'));
    setTimeout(() => {
      const first = layer.querySelector('input, select, textarea, button:not([data-close])');
      if (first) first.focus();
    }, 80);
    return { layer, close };
  }

  function dialog({ title, message = '', bodyHtml = '', confirmText = '确定', tone = 'primary', checkbox = null }) {
    return new Promise((resolve) => {
      let settled = false;
      const finish = (ok, checked = false) => {
        if (settled) return;
        settled = true;
        resolve({ ok, checked });
      };
      const html = `
        <div class="modal" role="dialog" aria-modal="true" aria-labelledby="dialog-title">
          <div class="modal-head">
            <div><p class="eyebrow">确认操作</p><h2 id="dialog-title">${escapeHtml(title)}</h2></div>
            <button class="icon-btn" type="button" data-close aria-label="关闭">${icon('x')}</button>
          </div>
          <div class="modal-body">
            ${message ? `<p>${escapeHtml(message)}</p>` : ''}
            ${bodyHtml}
            ${checkbox ? `<label class="check-row"><span class="switch"><input id="dialog-check" type="checkbox" checked><i></i></span><span class="check-copy"><strong>${escapeHtml(checkbox)}</strong><small>删除后无法恢复；也可以在 Cloudflare 控制台手动处理。</small></span></label>` : ''}
          </div>
          <div class="modal-foot">
            <button class="btn ghost" type="button" data-close>取消</button>
            <button class="btn ${tone === 'danger' ? 'danger' : 'primary'}" type="button" data-confirm>${escapeHtml(confirmText)}</button>
          </div>
        </div>`;
      const { layer, close } = openLayer({
        html,
        onClose: () => finish(false)
      });
      layer.querySelector('[data-confirm]').addEventListener('click', () => {
        const checked = checkbox ? Boolean(layer.querySelector('#dialog-check')?.checked) : false;
        finish(true, checked);
        close();
      });
    });
  }

  /* ---------- toast / copy ---------- */
  function toast(message, tone = 'success') {
    const root = $('#toast-root');
    const el = document.createElement('div');
    el.className = `toast ${tone === 'error' ? 'error' : ''}`;
    el.innerHTML = `
      ${icon(tone === 'error' ? 'alert' : 'check')}
      <div class="toast-copy">${escapeHtml(message)}</div>
      <button class="toast-close" type="button" aria-label="关闭">${icon('x')}</button>
      <span class="toast-bar"></span>`;
    root.appendChild(el);
    requestAnimationFrame(() => el.classList.add('is-in'));
    const remove = () => {
      el.classList.remove('is-in');
      setTimeout(() => el.remove(), 240);
    };
    el.querySelector('.toast-close').addEventListener('click', remove);
    setTimeout(remove, 3300);
  }

  function copyText(text, label = '已复制') {
    const done = () => toast(label, 'success');
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(done).catch(() => fallbackCopy(text, done));
    } else {
      fallbackCopy(text, done);
    }
  }

  function fallbackCopy(text, done) {
    const area = document.createElement('textarea');
    area.value = text;
    area.style.position = 'fixed';
    area.style.opacity = '0';
    document.body.appendChild(area);
    area.select();
    try {
      document.execCommand('copy');
      done();
    } catch (e) {
      toast('复制失败，请手动选择', 'error');
    }
    area.remove();
  }
})();
