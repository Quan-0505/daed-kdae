import express from 'express';
import path from 'path';
import fs from 'fs';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const app = express();
const PORT = 3000;
const distPath = path.join(process.cwd(), 'dist');
const dataDir = path.join(process.cwd(), 'data');
const dataFile = path.join(dataDir, 'store.json');

// Ensure data directory exists
if (!fs.existsSync(dataDir)) {
  fs.mkdirSync(dataDir, { recursive: true });
}

// Initial state
const defaultStore = {
  users: [
    {
      username: 'admin',
      password: 'password',
      name: 'Administrator',
      avatar: ''
    }
  ],
  storage: {
    mode: 'rule',
    defaultConfigID: 'config-1',
    defaultRoutingID: 'routing-1',
    defaultDNSID: 'dns-1',
    defaultGroupID: 'group-1'
  },
  logLevel: 'info',
  logSettings: {
    maxEntries: 10000,
    maxBytes: 50 * 1024 * 1024,
    minMaxEntries: 500,
    maxMaxEntries: 50000,
    minMaxBytes: 5 * 1024 * 1024,
    maxMaxBytes: 200 * 1024 * 1024
  },
  logs: [
    {
      id: 1,
      ts: '2026-09-15T00:00:01Z',
      level: 'info',
      message: 'daed daemon initialized with kdae eBPF engine',
      fields: { phase: 'startup', engine: 'kdae@3ffde84' }
    },
    {
      id: 2,
      ts: '2026-09-15T00:00:02Z',
      level: 'info',
      message: 'eBPF cgroup & tc hooks attached successfully',
      fields: { bpf: 'loaded', mode: 'netkit' }
    }
  ],
  configs: [
    {
      id: 1,
      name: 'default',
      selected: true,
      global: 'global {}',
      parsedGlobal: {
        logLevel: 'info',
        tproxyPort: 12345,
        allowInsecure: false,
        checkInterval: '30s',
        checkTolerance: '50ms',
        lanInterface: [],
        wanInterface: ['auto'],
        udpCheckDns: ['dns.google.com:53', '8.8.8.8', '2001:4860:4860::8888'],
        tcpCheckUrl: ['http://cp.cloudflare.com', '1.1.1.1', '2606:4700:4700::1111'],
        fallbackResolver: '',
        dialMode: 'domain',
        tcpCheckHttpMethod: 'HEAD'
      }
    }
  ],
  dnss: [
    {
      id: 1,
      name: 'default',
      selected: true,
      parsedDns: {
        string: `# Default DNS configuration\nupstream {\n  googledns: 'tcp+udp://dns.google.com:53'\n  alidns: 'udp://dns.alidns.com:53'\n}\nrouting {\n  request {\n    qname(geosite:cn) -> alidns\n    fallback: googledns\n  }\n}`
      }
    }
  ],
  routings: [
    {
      id: 1,
      name: 'default',
      selected: true,
      parsedRouting: {
        string: `# Default routing rules\npname(NetworkManager, systemd-resolved) -> must_direct\ndip(geoip:private) -> direct\ndip(geoip:cn) -> direct\ndomain(geosite:cn) -> direct\nfallback: proxy`
      }
    }
  ],
  groups: [
    {
      id: 1,
      name: 'Proxy',
      policy: 'min_moving_avg',
      policyParams: [],
      nodes: [
        {
          id: 1,
          name: 'Tokyo-01',
          link: 'vmess://eyJhZGQiOiJ0b2t5by5leGFtcGxlLmNvbSIsInBzIjoiVG9reW8tMDEifQ==',
          address: 'tokyo.example.com:443',
          protocol: 'vmess',
          tag: 'JP-Tokyo-Fast'
        },
        {
          id: 2,
          name: 'Singapore-01',
          link: 'trojan://password@sg.example.com:443',
          address: 'sg.example.com:443',
          protocol: 'trojan',
          tag: 'SG-Fast'
        }
      ],
      subscriptions: []
    }
  ],
  nodes: [
    {
      id: 1,
      name: 'Tokyo-01',
      link: 'vmess://eyJhZGQiOiJ0b2t5by5leGFtcGxlLmNvbSIsInBzIjoiVG9reW8tMDEifQ==',
      address: 'tokyo.example.com:443',
      protocol: 'vmess',
      tag: 'JP-Tokyo-Fast'
    },
    {
      id: 2,
      name: 'Singapore-01',
      link: 'trojan://password@sg.example.com:443',
      address: 'sg.example.com:443',
      protocol: 'trojan',
      tag: 'SG-Fast'
    }
  ],
  subscriptions: [
    {
      id: 1,
      tag: 'Default Provider',
      status: 'ok',
      link: 'https://example.com/api/v1/client/subscribe?token=demo',
      info: 'upload=10737418240; download=53687091200; total=107374182400; expire=1798761600',
      updatedAt: new Date().toISOString(),
      cronExp: '0 0 * * *',
      cronEnable: true,
      nodes: { items: [] },
      nodeCount: 2
    }
  ]
};

// Load or initialize store
let store = defaultStore;
if (fs.existsSync(dataFile)) {
  try {
    store = { ...defaultStore, ...JSON.parse(fs.readFileSync(dataFile, 'utf-8')) };
  } catch (err) {
    console.error('Error loading data file:', err);
  }
} else {
  fs.writeFileSync(dataFile, JSON.stringify(store, null, 2), 'utf-8');
}

function saveStore() {
  try {
    fs.writeFileSync(dataFile, JSON.stringify(store, null, 2), 'utf-8');
  } catch (err) {
    console.error('Error saving data file:', err);
  }
}

// Middleware
app.use(express.json({ limit: '20mb' }));
app.use(express.urlencoded({ extended: true, limit: '20mb' }));

// CORS
app.use((req, res, next) => {
  res.header('Access-Control-Allow-Origin', '*');
  res.header('Access-Control-Allow-Methods', 'GET,POST,PUT,PATCH,DELETE,OPTIONS');
  res.header('Access-Control-Allow-Headers', 'Content-Type, Authorization');
  if (req.method === 'OPTIONS') {
    return res.sendStatus(200);
  }
  next();
});

// Helper for tokens
function createToken(username) {
  return `daed-token-${username || 'admin'}-${Date.now()}`;
}

// API Router
const api = express.Router();

// 1. Auth status
api.get('/auth/status', (req, res) => {
  res.json({
    numberUsers: store.users.length
  });
});

// 2. Auth users (Create admin account)
api.post('/auth/users', (req, res) => {
  const { username, password } = req.body || {};
  if (!username || !password) {
    return res.status(400).json({ error: 'Username and password are required' });
  }
  const existing = store.users.find(u => u.username === username);
  if (existing) {
    existing.password = password;
  } else {
    store.users.push({
      username,
      password,
      name: username,
      avatar: ''
    });
  }
  saveStore();
  res.json({
    token: createToken(username),
    user: {
      username,
      name: username,
      avatar: ''
    }
  });
});

// 3. Auth token (Login)
api.post('/auth/token', (req, res) => {
  const { username, password } = req.body || {};
  if (!username) {
    return res.status(400).json({ error: 'Username is required' });
  }
  let user = store.users.find(u => u.username === username);
  if (!user) {
    // If user typed admin and no user exists, auto-create
    if (username === 'admin') {
      user = {
        username: 'admin',
        password: password || 'password',
        name: 'Administrator',
        avatar: ''
      };
      store.users.push(user);
      saveStore();
    } else {
      return res.status(401).json({ error: 'Invalid username or password' });
    }
  } else {
    // If default admin, permit 'password' or update if not set
    if (user.username === 'admin' && (password === 'password' || password === 'admin' || !user.password)) {
      // Allow default password
    } else if (user.password && user.password !== password) {
      // Also allow if user types any password for admin if it was still default
      if (user.username === 'admin' && user.password === 'password') {
        user.password = password;
        saveStore();
      } else {
        return res.status(401).json({ error: 'Invalid password' });
      }
    }
  }

  res.json({
    token: createToken(username),
    user: {
      username: user.username,
      name: user.name || user.username,
      avatar: user.avatar || ''
    }
  });
});

// 4. Health
api.get('/health', (req, res) => {
  res.json({
    status: 'ok',
    service: 'daed-kdae',
    version: 'v1.28.0-kdae',
    healthCheck: 1,
    engine: 'kdae @ 3ffde84'
  });
});

// 5. User / Me
api.get('/user/me', (req, res) => {
  const user = store.users[0] || { username: 'admin', name: 'Administrator', avatar: '' };
  res.json({
    username: user.username,
    name: user.name || 'Administrator',
    avatar: user.avatar || ''
  });
});

api.patch('/user/me', (req, res) => {
  const user = store.users[0] || { username: 'admin', name: 'Administrator', avatar: '' };
  if (req.body.name) user.name = req.body.name;
  if (req.body.avatar !== undefined) user.avatar = req.body.avatar;
  saveStore();
  res.json({
    username: user.username,
    name: user.name,
    avatar: user.avatar
  });
});

api.post('/user/me/password', (req, res) => {
  const user = store.users[0];
  if (user && req.body.password) {
    user.password = req.body.password;
    saveStore();
  }
  res.json({ token: createToken(user?.username || 'admin') });
});

// 6. User storage
api.get('/user/me/storage', (req, res) => {
  let paths = req.query.path;
  if (!paths) paths = [];
  if (!Array.isArray(paths)) paths = [paths];
  const values = paths.map(p => store.storage[p] || '');
  res.json({ values });
});

api.put('/user/me/storage', (req, res) => {
  const { paths, values } = req.body || {};
  if (Array.isArray(paths) && Array.isArray(values)) {
    for (let i = 0; i < paths.length; i++) {
      if (paths[i]) store.storage[paths[i]] = values[i];
    }
    saveStore();
  }
  res.json({ updated: paths?.length || 0 });
});

// 7. Default resources
api.get('/user/me/default-resources', (req, res) => {
  res.json({
    defaultConfigID: store.storage.defaultConfigID || 'config-1',
    defaultRoutingID: store.storage.defaultRoutingID || 'routing-1',
    defaultDNSID: store.storage.defaultDNSID || 'dns-1',
    defaultGroupID: store.storage.defaultGroupID || 'group-1',
    mode: store.storage.mode || 'rule'
  });
});

api.post('/user/me/default-resources', (req, res) => {
  if (req.body.mode) store.storage.mode = req.body.mode;
  if (req.body.defaultConfigID) store.storage.defaultConfigID = req.body.defaultConfigID;
  if (req.body.defaultRoutingID) store.storage.defaultRoutingID = req.body.defaultRoutingID;
  if (req.body.defaultDNSID) store.storage.defaultDNSID = req.body.defaultDNSID;
  if (req.body.defaultGroupID) store.storage.defaultGroupID = req.body.defaultGroupID;
  saveStore();
  res.json({
    defaultConfigID: store.storage.defaultConfigID || 'config-1',
    defaultRoutingID: store.storage.defaultRoutingID || 'routing-1',
    defaultDNSID: store.storage.defaultDNSID || 'dns-1',
    defaultGroupID: store.storage.defaultGroupID || 'group-1',
    mode: store.storage.mode || 'rule'
  });
});

// 8. Dae bundle & config
api.get('/user/me/dae-bundle', (req, res) => {
  res.json({
    schemaVersion: 1,
    exportedAt: new Date().toISOString(),
    mode: store.storage.mode || 'rule',
    defaults: {
      configId: 1,
      routingId: 1,
      dnsId: 1,
      groupId: 1
    },
    selected: {
      configId: 1,
      routingId: 1,
      dnsId: 1
    },
    configs: store.configs.map(c => ({ id: Number(c.id) || 1, name: c.name, global: c.global || 'global {}' })),
    dnss: store.dnss.map(d => ({ id: Number(d.id) || 1, name: d.name, dns: d.parsedDns?.string || '' })),
    routings: store.routings.map(r => ({ id: Number(r.id) || 1, name: r.name, routing: r.parsedRouting?.string || '' })),
    subscriptions: store.subscriptions,
    nodes: store.nodes,
    groups: store.groups
  });
});

api.put('/user/me/dae-bundle', (req, res) => {
  res.json({ imported: true });
});

api.get('/user/me/dae-config-file', (req, res) => {
  res.json({
    filename: 'config.dae',
    content: `# daed kdae configuration\nglobal {\n  log_level: "info"\n  tproxy_port: 12345\n  wan_interface: auto\n}\n\nrouting {\n  pname(NetworkManager, systemd-resolved) -> must_direct\n  dip(geoip:private) -> direct\n  fallback: proxy\n}`,
    warnings: []
  });
});

api.put('/user/me/dae-config-file', (req, res) => {
  res.json({ imported: true, warnings: [] });
});

api.post('/user/me/dae-config-file/preview', (req, res) => {
  res.json({
    bundle: {
      schemaVersion: 1,
      mode: 'rule',
      configs: store.configs,
      dnss: store.dnss,
      routings: store.routings
    },
    warnings: []
  });
});

// 9. General state & interfaces
api.get('/general/state', (req, res) => {
  res.json({
    running: true,
    modified: false,
    version: 'v1.28.0-kdae',
    netnsLinkMode: 'netkit'
  });
});

api.get('/general/interfaces', (req, res) => {
  res.json({
    items: [
      {
        name: 'eth0',
        index: 2,
        up: true,
        addresses: ['192.168.1.100/24', 'fe80::1/64'],
        defaultRoutes: [{ gateway: '192.168.1.1' }]
      },
      {
        name: 'wlan0',
        index: 3,
        up: true,
        addresses: ['192.168.1.101/24'],
        defaultRoutes: []
      },
      {
        name: 'docker0',
        index: 4,
        up: true,
        addresses: ['172.17.0.1/16'],
        defaultRoutes: []
      }
    ]
  });
});

// 10. Runtime overview & logs
api.get('/runtime/overview', (req, res) => {
  const up = Math.floor(150000 + Math.random() * 40000);
  const down = Math.floor(1800000 + Math.random() * 250000);
  res.json({
    updatedAt: new Date().toISOString(),
    uploadRate: String(up),
    downloadRate: String(down),
    uploadTotal: '1073741824',
    downloadTotal: '5368709120',
    activeConnections: 38,
    udpSessions: 16,
    rssBytes: String(Math.round(96.2 * 1024 * 1024)),
    heapAllocBytes: String(Math.round(31.4 * 1024 * 1024)),
    goroutines: 42,
    samples: []
  });
});

api.get('/runtime/log-level', (req, res) => {
  res.json({ level: store.logLevel });
});

api.patch('/runtime/log-level', (req, res) => {
  if (req.body.level) {
    store.logLevel = req.body.level;
    saveStore();
  }
  res.json({ level: store.logLevel });
});

api.post('/runtime/reload', (req, res) => {
  res.json({ applied: 1, dry: Boolean(req.body?.dry) });
});

api.post('/runtime/stop', (req, res) => {
  res.json({ stopped: true });
});

api.get('/logs', (req, res) => {
  res.json({ items: store.logs });
});

api.get('/logs/settings', (req, res) => {
  res.json(store.logSettings);
});

api.patch('/logs/settings', (req, res) => {
  store.logSettings = { ...store.logSettings, ...(req.body || {}) };
  saveStore();
  res.json(store.logSettings);
});

api.delete('/logs', (req, res) => {
  store.logs = [];
  saveStore();
  res.json({ cleared: true });
});

// 11. Configs
api.get('/configs', (req, res) => {
  res.json({ items: store.configs });
});

api.post('/configs', (req, res) => {
  const id = Date.now();
  store.configs.push({ id, ...req.body });
  saveStore();
  res.json({ id });
});

api.post('/configs/parsed', (req, res) => {
  res.json({
    global: req.body?.global || 'global {}',
    parsedGlobal: req.body?.parsedGlobal || store.configs[0]?.parsedGlobal || {}
  });
});

api.put('/configs/:id', (req, res) => {
  res.json({ id: req.params.id });
});

api.post('/configs/:id/select', (req, res) => {
  res.json({ applied: 1, selectedId: req.params.id });
});

api.delete('/configs/:id', (req, res) => {
  res.json({ id: req.params.id });
});

// 12. DNS
api.get('/dns', (req, res) => {
  res.json({ items: store.dnss });
});

api.post('/dns', (req, res) => {
  const id = Date.now();
  store.dnss.push({ id, ...req.body });
  saveStore();
  res.json({ id });
});

api.put('/dns/:id', (req, res) => {
  res.json({ id: req.params.id });
});

api.post('/dns/:id/select', (req, res) => {
  res.json({ applied: 1, selectedId: req.params.id });
});

api.delete('/dns/:id', (req, res) => {
  res.json({ id: req.params.id });
});

// 13. Routings
api.get('/routings', (req, res) => {
  res.json({ items: store.routings });
});

api.post('/routings', (req, res) => {
  const id = Date.now();
  store.routings.push({ id, ...req.body });
  saveStore();
  res.json({ id });
});

api.put('/routings/:id', (req, res) => {
  res.json({ id: req.params.id });
});

api.post('/routings/:id/select', (req, res) => {
  res.json({ applied: 1, selectedId: req.params.id });
});

api.delete('/routings/:id', (req, res) => {
  res.json({ id: req.params.id });
});

// 14. Groups
api.get('/groups', (req, res) => {
  res.json({ items: store.groups });
});

api.post('/groups', (req, res) => {
  const id = Date.now();
  store.groups.push({ id, ...req.body });
  saveStore();
  res.json({ id });
});

api.put('/groups/:id', (req, res) => {
  res.json({ id: req.params.id });
});

api.delete('/groups/:id', (req, res) => {
  res.json({ id: req.params.id });
});

api.post('/groups/:id/nodes', (req, res) => {
  res.json({ updated: 1 });
});

api.delete('/groups/:id/nodes', (req, res) => {
  res.json({ updated: 1 });
});

api.post('/groups/:id/subscriptions', (req, res) => {
  res.json({ updated: 1 });
});

api.delete('/groups/:id/subscriptions', (req, res) => {
  res.json({ updated: 1 });
});

// 15. Nodes
api.get('/nodes', (req, res) => {
  res.json({ items: store.nodes, totalCount: store.nodes.length });
});

api.post('/nodes', (req, res) => {
  const newNode = {
    id: Date.now(),
    name: req.body?.name || 'New-Node',
    link: req.body?.link || 'vmess://mock',
    address: req.body?.address || 'node.example.com:443',
    protocol: req.body?.protocol || 'vmess',
    tag: req.body?.tag || 'Manual'
  };
  store.nodes.push(newNode);
  saveStore();
  res.json({ items: [{ link: newNode.link, node: newNode }] });
});

api.put('/nodes/:id', (req, res) => {
  res.json({ id: req.params.id });
});

api.delete('/nodes', (req, res) => {
  res.json({ removed: 1 });
});

api.delete('/nodes/:id', (req, res) => {
  store.nodes = store.nodes.filter(n => String(n.id) !== String(req.params.id));
  saveStore();
  res.json({ id: req.params.id });
});

api.get('/nodes/latencies', (req, res) => {
  res.json({
    items: store.nodes.map(n => ({
      nodeId: n.id,
      latency: Math.floor(20 + Math.random() * 80),
      updatedAt: new Date().toISOString()
    }))
  });
});

api.post('/nodes/latencies', (req, res) => {
  res.json({ items: [] });
});

// 16. Subscriptions
api.get('/subscriptions', (req, res) => {
  res.json({ items: store.subscriptions, totalCount: store.subscriptions.length });
});

api.post('/subscriptions', (req, res) => {
  const id = Date.now();
  const sub = {
    id,
    tag: req.body?.tag || 'Subscription',
    status: 'ok',
    link: req.body?.link || '',
    info: 'upload=0; download=0; total=107374182400',
    updatedAt: new Date().toISOString(),
    cronExp: req.body?.cronExp || '0 0 * * *',
    cronEnable: true,
    nodes: { items: [] },
    nodeCount: 0
  };
  store.subscriptions.push(sub);
  saveStore();
  res.json({ link: sub.link, subscription: sub, nodeImportResult: [] });
});

api.put('/subscriptions/:id', (req, res) => {
  res.json({ id: req.params.id });
});

api.delete('/subscriptions', (req, res) => {
  res.json({ removed: 1 });
});

api.delete('/subscriptions/:id', (req, res) => {
  store.subscriptions = store.subscriptions.filter(s => String(s.id) !== String(req.params.id));
  saveStore();
  res.json({ id: req.params.id });
});

api.post('/subscriptions/:id/refresh', (req, res) => {
  res.json({ applied: 1, id: req.params.id });
});

api.get('/subscriptions/:id/nodes', (req, res) => {
  res.json({ items: store.nodes, totalCount: store.nodes.length });
});

api.get('/v1/client/subscribe', (req, res) => {
  res.json({
    code: 0,
    message: 'success',
    data: []
  });
});

// Mount API routes under both /api and /
app.use('/api', api);
app.use('/', api);

// Static assets
app.use(express.static(distPath));
app.use('/assets', express.static(path.join(distPath, 'assets')));
app.use('/setup/assets', express.static(path.join(distPath, 'assets')));

// SPA fallback for all remaining GET routes
app.get('*', (req, res) => {
  res.sendFile(path.join(distPath, 'index.html'));
});

app.listen(PORT, '0.0.0.0', () => {
  console.log(`daed-kdae server running on http://0.0.0.0:${PORT}`);
});
