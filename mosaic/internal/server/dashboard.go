package server

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Mosaic</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #0d1117;
            color: #e6edf3;
            min-height: 100vh;
        }
        .header {
            background: linear-gradient(135deg, #238636 0%, #1f6feb 100%);
            padding: 1rem 1.5rem;
            display: flex;
            align-items: center;
            justify-content: space-between;
        }
        .header h1 {
            font-size: 1.5rem;
            font-weight: 600;
        }
        .header .status {
            font-size: 0.875rem;
            opacity: 0.9;
        }
        .container {
            max-width: 1000px;
            margin: 0 auto;
            padding: 1.5rem;
        }
        .grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 1.5rem;
        }
        @media (max-width: 768px) {
            .grid { grid-template-columns: 1fr; }
        }
        .card {
            background: #161b22;
            border: 1px solid #30363d;
            border-radius: 8px;
            overflow: hidden;
        }
        .card-header {
            background: #21262d;
            padding: 0.75rem 1rem;
            border-bottom: 1px solid #30363d;
            font-weight: 600;
            font-size: 0.875rem;
        }
        .card-body {
            padding: 1rem;
        }
        .full-width { grid-column: 1 / -1; }
        
        /* Preview */
        .preview {
            background: #000;
            border-radius: 4px;
            padding: 0.5rem;
            text-align: center;
        }
        .preview canvas {
            image-rendering: pixelated;
            max-width: 100%;
            height: auto;
        }
        .preview-info {
            margin-top: 0.5rem;
            font-size: 0.75rem;
            color: #8b949e;
        }
        .preview-app {
            color: #58a6ff;
            font-weight: 500;
        }
        
        /* Controls */
        .control-row {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 0.75rem 0;
            border-bottom: 1px solid #21262d;
        }
        .control-row:last-child { border-bottom: none; }
        .control-label { color: #8b949e; }
        .control-value { font-family: monospace; color: #58a6ff; }
        
        /* Toggle */
        .toggle {
            position: relative;
            width: 44px;
            height: 22px;
        }
        .toggle input { opacity: 0; width: 0; height: 0; }
        .toggle .slider {
            position: absolute;
            cursor: pointer;
            inset: 0;
            background: #30363d;
            border-radius: 22px;
            transition: 0.2s;
        }
        .toggle .slider:before {
            position: absolute;
            content: "";
            height: 16px;
            width: 16px;
            left: 3px;
            bottom: 3px;
            background: #e6edf3;
            border-radius: 50%;
            transition: 0.2s;
        }
        .toggle input:checked + .slider { background: #238636; }
        .toggle input:checked + .slider:before { transform: translateX(22px); }
        
        /* Buttons */
        button {
            background: #238636;
            color: white;
            border: none;
            padding: 0.5rem 1rem;
            border-radius: 6px;
            cursor: pointer;
            font-size: 0.875rem;
            font-weight: 500;
        }
        button:hover { background: #2ea043; }
        button:disabled { background: #21262d; color: #484f58; cursor: not-allowed; }
        button.secondary { background: #21262d; }
        button.secondary:hover { background: #30363d; }
        button.danger { background: #da3633; }
        button.danger:hover { background: #f85149; }
        
        /* Input */
        input[type="text"], input[type="search"] {
            background: #0d1117;
            border: 1px solid #30363d;
            color: #e6edf3;
            padding: 0.5rem 0.75rem;
            border-radius: 6px;
            font-size: 0.875rem;
            width: 100%;
        }
        input:focus { outline: none; border-color: #58a6ff; }
        
        /* Range slider */
        input[type="range"] {
            width: 100%;
            height: 6px;
            background: #30363d;
            border-radius: 3px;
            outline: none;
            -webkit-appearance: none;
        }
        input[type="range"]::-webkit-slider-thumb {
            -webkit-appearance: none;
            width: 16px;
            height: 16px;
            background: #58a6ff;
            border-radius: 50%;
            cursor: pointer;
        }
        
        /* App list */
        .app-list {
            max-height: 300px;
            overflow-y: auto;
        }
        .app-item {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 0.75rem;
            border-bottom: 1px solid #21262d;
        }
        .app-item:last-child { border-bottom: none; }
        .app-item:hover { background: #21262d; }
        .app-name { font-weight: 500; }
        .app-meta { font-size: 0.75rem; color: #8b949e; margin-top: 0.25rem; }
        .app-actions { display: flex; gap: 0.5rem; }
        .app-actions button { padding: 0.25rem 0.5rem; font-size: 0.75rem; }
        
        /* Search */
        .search-box {
            display: flex;
            gap: 0.5rem;
            margin-bottom: 1rem;
        }
        .search-box input { flex: 1; }
        
        /* Empty state */
        .empty {
            text-align: center;
            padding: 2rem;
            color: #8b949e;
        }
        
        /* Loading */
        .loading {
            text-align: center;
            padding: 1rem;
            color: #8b949e;
        }
        
        /* Error */
        .error {
            background: #3d1f1f;
            border: 1px solid #f85149;
            color: #f85149;
            padding: 0.75rem;
            border-radius: 6px;
            margin-bottom: 1rem;
        }
        
        /* Tabs */
        .tabs {
            display: flex;
            border-bottom: 1px solid #30363d;
            margin-bottom: 1rem;
        }
        .tab {
            padding: 0.75rem 1rem;
            cursor: pointer;
            color: #8b949e;
            border-bottom: 2px solid transparent;
            margin-bottom: -1px;
        }
        .tab:hover { color: #e6edf3; }
        .tab.active { color: #58a6ff; border-bottom-color: #58a6ff; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🎨 Mosaic</h1>
        <div style="display:flex;align-items:center;gap:1rem;">
            <select id="displaySelect" style="background:#21262d;color:#e6edf3;border:1px solid #30363d;padding:0.5rem;border-radius:6px;font-size:0.875rem;">
                <option value="default">Loading...</option>
            </select>
            <div class="status" id="headerStatus">Loading...</div>
        </div>
    </div>
    
    <div class="container">
        <div id="errorBox" class="error" style="display:none;"></div>
        
        <div class="grid">
            <!-- Preview -->
            <div class="card">
                <div class="card-header">Display Preview</div>
                <div class="card-body">
                    <div class="preview">
                        <canvas id="display" width="64" height="32" style="width:320px;height:160px;"></canvas>
                    </div>
                    <div class="preview-info">
                        Now showing: <span class="preview-app" id="currentApp">-</span>
                    </div>
                </div>
            </div>
            
            <!-- Controls -->
            <div class="card">
                <div class="card-header">Display Controls</div>
                <div class="card-body">
                    <div class="control-row">
                        <span class="control-label">Power</span>
                        <label class="toggle">
                            <input type="checkbox" id="power">
                            <span class="slider"></span>
                        </label>
                    </div>
                    <div class="control-row">
                        <span class="control-label">Auto-Rotate</span>
                        <label class="toggle">
                            <input type="checkbox" id="rotation">
                            <span class="slider"></span>
                        </label>
                    </div>
                    <div class="control-row">
                        <span class="control-label">Brightness</span>
                        <span class="control-value" id="brightnessValue">80%</span>
                    </div>
                    <input type="range" id="brightness" min="0" max="100" value="80">
                    <div style="margin-top:1rem; display:flex; gap:0.5rem;">
                        <button id="skipBtn">Skip →</button>
                    </div>
                </div>
            </div>
            
            <!-- Rotation Apps -->
            <div class="card full-width">
                <div class="card-header">Apps in Rotation</div>
                <div class="card-body">
                    <div class="app-list" id="rotationApps">
                        <div class="loading">Loading...</div>
                    </div>
                </div>
            </div>
            
            <!-- App Browser -->
            <div class="card full-width">
                <div class="card-header">App Browser</div>
                <div class="card-body">
                    <div class="tabs">
                        <div class="tab active" data-tab="installed">Installed</div>
                        <div class="tab" data-tab="community">Community</div>
                        <div class="tab" data-tab="config">Configure</div>
                    </div>
                    
                    <div id="installedTab">
                        <div class="app-list" id="installedApps">
                            <div class="loading">Loading...</div>
                        </div>
                    </div>
                    
                    <div id="configTab" style="display:none;">
                        <div style="margin-bottom:1rem;">
                            <label style="display:block;margin-bottom:0.5rem;font-weight:500;">Select App</label>
                            <select id="configAppSelect" style="background:#0d1117;border:1px solid #30363d;color:#e6edf3;padding:0.5rem;border-radius:6px;font-size:0.875rem;width:100%;">
                                <option value="">Choose an app to configure...</option>
                            </select>
                        </div>
                        <div id="configPanel" style="display:none;">
                            <div id="configFields"></div>
                            <div style="display:flex;gap:0.5rem;margin-top:1rem;">
                                <button id="saveConfigBtn" style="flex:1;">Save Configuration</button>
                                <button id="resetConfigBtn" class="secondary" style="flex:1;">Reset</button>
                            </div>
                            <div id="configStatus" style="font-size:0.75rem;color:#8b949e;margin-top:0.5rem;"></div>
                        </div>
                    </div>
                    
                    <div id="communityTab" style="display:none;">
                        <div class="search-box">
                            <input type="search" id="appSearch" placeholder="Search 887 community apps...">
                            <button id="searchBtn">Search</button>
                        </div>
                        <div class="app-list" id="communityApps">
                            <div class="loading">Loading apps...</div>
                        </div>
                    </div>
                    
                    <!-- Upload custom app -->
                    <div style="margin-top:1rem;padding-top:1rem;border-top:1px solid #30363d;">
                        <div style="font-weight:500;margin-bottom:0.5rem;">Upload Custom App</div>
                        <div style="display:flex;gap:0.5rem;align-items:center;">
                            <input type="file" id="starFile" accept=".star" style="flex:1;">
                            <button id="uploadBtn">Upload .star</button>
                        </div>
                        <div id="uploadStatus" style="font-size:0.75rem;color:#8b949e;margin-top:0.5rem;"></div>
                    </div>
                </div>
            </div>
        </div>
    </div>
    
    <script>
        const canvas = document.getElementById('display');
        const ctx = canvas.getContext('2d');
        let currentDisplayId = 'default';
        
        // Fetch and populate display selector
        async function fetchDisplays() {
            try {
                const displays = await api('api/displays');
                const select = document.getElementById('displaySelect');
                if (!displays || displays.length === 0) {
                    select.innerHTML = '<option value="">No displays connected</option>';
                    currentDisplayId = '';
                    document.getElementById('headerStatus').textContent = 'Waiting for display...';
                } else {
                    select.innerHTML = displays.map(d => 
                        '<option value="' + d.id + '">' + d.name + ' (' + d.width + 'x' + d.height + ')</option>'
                    ).join('');
                    currentDisplayId = displays[0].id;
                }
            } catch (e) {
                console.error('Failed to fetch displays:', e);
                document.getElementById('displaySelect').innerHTML = '<option value="">Error loading</option>';
            }
        }
        
        document.getElementById('displaySelect').addEventListener('change', (e) => {
            currentDisplayId = e.target.value;
            fetchStatus();
            fetchRotation();
        });
        
        // Get base URL for API calls
        function getBaseUrl() {
            let base = window.location.href.split('?')[0];
            if (!base.endsWith('/')) base += '/';
            return base;
        }
        
        // API helper
        async function api(path, options = {}) {
            try {
                const url = new URL(path, getBaseUrl()).href;
                const resp = await fetch(url, {
                    ...options,
                    headers: { 'Content-Type': 'application/json', ...options.headers },
                });
                if (!resp.ok) throw new Error('API error: ' + resp.status);
                return await resp.json();
            } catch (e) {
                console.error('API error:', e);
                throw e;
            }
        }
        
        // Show error
        function showError(msg) {
            const box = document.getElementById('errorBox');
            box.textContent = msg;
            box.style.display = 'block';
            setTimeout(() => box.style.display = 'none', 5000);
        }
        
        // Fetch and render frame
        async function fetchFrame() {
            try {
                const resp = await fetch(getBaseUrl() + 'frame?display=' + currentDisplayId);
                const buffer = await resp.arrayBuffer();
                const pixels = new Uint8Array(buffer);
                const width = parseInt(resp.headers.get('X-Frame-Width') || '64');
                const height = parseInt(resp.headers.get('X-Frame-Height') || '32');
                const appName = resp.headers.get('X-App-Name') || 'unknown';
                
                canvas.width = width;
                canvas.height = height;
                const imageData = ctx.createImageData(width, height);
                for (let i = 0; i < width * height; i++) {
                    imageData.data[i * 4] = pixels[i * 3];
                    imageData.data[i * 4 + 1] = pixels[i * 3 + 1];
                    imageData.data[i * 4 + 2] = pixels[i * 3 + 2];
                    imageData.data[i * 4 + 3] = 255;
                }
                ctx.putImageData(imageData, 0, 0);
                document.getElementById('currentApp').textContent = appName;
            } catch (e) {
                console.error('Frame error:', e);
            }
        }
        
        // Fetch status for current display
        async function fetchStatus() {
            if (!currentDisplayId) return;
            try {
                const display = await api('api/displays/' + currentDisplayId);
                document.getElementById('headerStatus').textContent = (display.power ? 'On' : 'Off') + ' • ' + display.current_app;
                document.getElementById('power').checked = display.power !== false;
                document.getElementById('rotation').checked = display.rotation_enabled !== false;
                document.getElementById('brightness').value = display.brightness || 80;
                document.getElementById('brightnessValue').textContent = (display.brightness || 80) + '%';
            } catch (e) {
                document.getElementById('headerStatus').textContent = 'Disconnected';
            }
        }
        
        // Fetch rotation apps for current display
        async function fetchRotation() {
            if (!currentDisplayId) {
                document.getElementById('rotationApps').innerHTML = '<div class="empty">Connect a display first</div>';
                return;
            }
            try {
                const data = await api('api/displays/' + currentDisplayId + '/rotation');
                const container = document.getElementById('rotationApps');
                if (!data.apps || data.apps.length === 0) {
                    container.innerHTML = '<div class="empty">No apps in rotation. Add some from the App Browser below.</div>';
                    return;
                }
                container.innerHTML = data.apps.map(app => 
                    '<div class="app-item">' +
                        '<div><div class="app-name">' + (app.name || app.id) + '</div></div>' +
                        '<div class="app-actions">' +
                            '<button class="danger" onclick="removeFromRotation(\'' + app.id + '\')">Remove</button>' +
                        '</div>' +
                    '</div>'
                ).join('');
            } catch (e) {
                document.getElementById('rotationApps').innerHTML = '<div class="error">Failed to load rotation</div>';
            }
        }
        
        // Fetch installed apps
        async function fetchInstalledApps() {
            try {
                const apps = await api('api/apps');
                const container = document.getElementById('installedApps');
                if (!apps || apps.length === 0) {
                    container.innerHTML = '<div class="empty">No apps installed. Browse Community apps to install some.</div>';
                    return;
                }
                container.innerHTML = apps.map(app => 
                    '<div class="app-item">' +
                        '<div><div class="app-name">' + (app.name || app.id) + '</div>' +
                        '<div class="app-meta">' + (app.summary || '') + '</div></div>' +
                        '<div class="app-actions">' +
                            '<button onclick="addToRotation(\'' + app.id + '\')">Add</button>' +
                            '<button class="danger" onclick="uninstallApp(\'' + app.id + '\')">Uninstall</button>' +
                        '</div>' +
                    '</div>'
                ).join('');
            } catch (e) {
                document.getElementById('installedApps').innerHTML = '<div class="error">Failed to load apps</div>';
            }
        }
        
        // Fetch community apps
        async function fetchCommunityApps(query) {
            const container = document.getElementById('communityApps');
            container.innerHTML = '<div class="loading">Loading...</div>';
            try {
                let url = 'api/apps/community';
                if (query) url = 'api/apps/community/search?q=' + encodeURIComponent(query);
                const apps = await api(url);
                if (!apps || apps.length === 0) {
                    container.innerHTML = '<div class="empty">No apps found</div>';
                    return;
                }
                container.innerHTML = apps.slice(0, 100).map(app => 
                    '<div class="app-item">' +
                        '<div><div class="app-name">' + (app.name || app.id) + '</div>' +
                        '<div class="app-meta">' + (app.summary || app.author || app.id) + '</div></div>' +
                        '<div class="app-actions">' +
                            '<button onclick="installApp(\'' + app.id + '\')">Install</button>' +
                        '</div>' +
                    '</div>'
                ).join('');
            } catch (e) {
                container.innerHTML = '<div class="error">Failed to load community apps: ' + e.message + '</div>';
            }
        }
        
        // Actions
        async function addToRotation(appId) {
            if (!currentDisplayId) {
                showError('No display connected');
                return;
            }
            try {
                await api('api/displays/' + currentDisplayId + '/rotation/apps', { method: 'POST', body: JSON.stringify({ app_id: appId }) });
                fetchRotation();
            } catch (e) { showError('Failed to add to rotation: ' + e.message); }
        }
        
        async function removeFromRotation(appId) {
            if (!currentDisplayId) return;
            try {
                await api('api/displays/' + currentDisplayId + '/rotation/apps/' + appId, { method: 'DELETE' });
                fetchRotation();
            } catch (e) { showError('Failed to remove from rotation'); }
        }
        
        async function installApp(appId) {
            try {
                await api('api/apps/install', { method: 'POST', body: JSON.stringify({ app_id: appId }) });
                fetchInstalledApps();
                fetchCommunityApps(document.getElementById('appSearch').value);
            } catch (e) { showError('Failed to install app: ' + e.message); }
        }
        
        async function uninstallApp(appId) {
            if (!confirm('Uninstall ' + appId + '?')) return;
            try {
                await api('api/apps/' + appId, { method: 'DELETE' });
                fetchInstalledApps();
                fetchRotation();
            } catch (e) { showError('Failed to uninstall: ' + e.message); }
        }
        
        // Event handlers
        document.getElementById('power').addEventListener('change', async (e) => {
            await api('api/displays/' + currentDisplayId + '/power', { method: 'PUT', body: JSON.stringify({ power: e.target.checked }) });
        });
        
        document.getElementById('rotation').addEventListener('change', async (e) => {
            await api('api/displays/' + currentDisplayId + '/rotation', { method: 'PUT', body: JSON.stringify({ enabled: e.target.checked }) });
        });
        
        let brightnessTimeout;
        document.getElementById('brightness').addEventListener('input', (e) => {
            document.getElementById('brightnessValue').textContent = e.target.value + '%';
            clearTimeout(brightnessTimeout);
            brightnessTimeout = setTimeout(async () => {
                await api('api/displays/' + currentDisplayId + '/brightness', { method: 'PUT', body: JSON.stringify({ brightness: parseInt(e.target.value) }) });
            }, 200);
        });
        
        document.getElementById('skipBtn').addEventListener('click', async () => {
            await api('api/displays/' + currentDisplayId + '/skip', { method: 'POST' });
        });
        
        document.getElementById('searchBtn').addEventListener('click', () => {
            fetchCommunityApps(document.getElementById('appSearch').value);
        });
        
        document.getElementById('appSearch').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') fetchCommunityApps(e.target.value);
        });
        
        document.getElementById('uploadBtn').addEventListener('click', async () => {
            const fileInput = document.getElementById('starFile');
            const status = document.getElementById('uploadStatus');
            
            if (!fileInput.files.length) {
                status.textContent = 'Please select a .star file';
                status.style.color = '#f85149';
                return;
            }
            
            const file = fileInput.files[0];
            const reader = new FileReader();
            
            reader.onload = async (e) => {
                try {
                    status.textContent = 'Uploading...';
                    status.style.color = '#8b949e';
                    
                    const content = e.target.result;
                    const appId = file.name.replace('.star', '');
                    
                    await api('api/apps/upload', {
                        method: 'POST',
                        body: JSON.stringify({
                            id: appId,
                            name: appId,
                            source: content
                        })
                    });
                    
                    status.textContent = 'Uploaded successfully! App: ' + appId;
                    status.style.color = '#238636';
                    fileInput.value = '';
                    fetchInstalledApps();
                } catch (err) {
                    status.textContent = 'Upload failed: ' + err.message;
                    status.style.color = '#f85149';
                }
            };
            
            reader.readAsText(file);
        });
        
        // Tabs
        document.querySelectorAll('.tab').forEach(tab => {
            tab.addEventListener('click', () => {
                document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
                tab.classList.add('active');
                const tabName = tab.dataset.tab;
                document.getElementById('installedTab').style.display = tabName === 'installed' ? 'block' : 'none';
                document.getElementById('communityTab').style.display = tabName === 'community' ? 'block' : 'none';
                document.getElementById('configTab').style.display = tabName === 'config' ? 'block' : 'none';
                if (tabName === 'community') fetchCommunityApps();
                if (tabName === 'config') populateConfigSelect();
            });
        });
        
        // App Configuration
        let currentConfigApp = null;
        
        async function populateConfigSelect() {
            try {
                const apps = await api('api/apps');
                const select = document.getElementById('configAppSelect');
                select.innerHTML = '<option value="">Choose an app to configure...</option>';
                if (apps && apps.length > 0) {
                    apps.forEach(app => {
                        const option = document.createElement('option');
                        option.value = app.id;
                        option.textContent = app.name || app.id;
                        select.appendChild(option);
                    });
                }
            } catch (e) {
                console.error('Failed to load apps for config:', e);
            }
        }
        
        document.getElementById('configAppSelect').addEventListener('change', async (e) => {
            const appId = e.target.value;
            currentConfigApp = appId;
            if (!appId) {
                document.getElementById('configPanel').style.display = 'none';
                return;
            }
            
            try {
                const app = await api('api/apps/' + appId);
                // Generate config form
                const fieldsDiv = document.getElementById('configFields');
                fieldsDiv.innerHTML = '';
                
                if (!app.schema_json || app.schema_json.length === 0) {
                    fieldsDiv.innerHTML = '<div style="color:#8b949e;padding:1rem;text-align:center;">This app has no configuration options.</div>';
                    document.getElementById('configPanel').style.display = 'block';
                    return;
                }
                
                try {
                    const schema = JSON.parse(atob(app.schema_json));
                    let html = '';
                    const config = app.config || {};
                    
                    // Pixlet schema format uses "schema" array, not "fields"
                    const fields = schema.schema || schema.fields || [];
                    if (fields.length > 0) {
                        fields.forEach(field => {
                            const value = config[field.id] || field.default || '';
                            const required = field.required ? ' required' : '';
                            
                            html += '<div style="margin-bottom:1rem;">';
                            html += '<label style="display:block;margin-bottom:0.25rem;font-weight:500;font-size:0.875rem;">' + (field.name || field.id) + '</label>';
                            
                            if (field.description) {
                                html += '<div style="font-size:0.75rem;color:#8b949e;margin-bottom:0.5rem;">' + field.description + '</div>';
                            }
                            
                            if (field.type === 'onoff' || field.type === 'toggle') {
                                // Boolean toggle - use checkbox, value stored as "true"/"false"
                                const checked = value === 'true' || value === true ? ' checked' : '';
                                html += '<label style="display:flex;align-items:center;gap:0.5rem;cursor:pointer;">';
                                html += '<input type="checkbox" id="config_' + field.id + '"' + checked + ' style="width:1.25rem;height:1.25rem;">';
                                html += '<span style="font-size:0.875rem;">Enabled</span></label>';
                            } else if (field.type === 'select' && field.options) {
                                html += '<select id="config_' + field.id + '" style="background:#0d1117;border:1px solid #30363d;color:#e6edf3;padding:0.5rem;border-radius:6px;font-size:0.875rem;width:100%;"' + required + '>';
                                field.options.forEach(opt => {
                                    const selected = value === opt.value ? ' selected' : '';
                                    html += '<option value="' + opt.value + '"' + selected + '>' + opt.display + '</option>';
                                });
                                html += '</select>';
                            } else if (field.type === 'dropdown' && field.options) {
                                // Pixlet uses "dropdown" type
                                html += '<select id="config_' + field.id + '" style="background:#0d1117;border:1px solid #30363d;color:#e6edf3;padding:0.5rem;border-radius:6px;font-size:0.875rem;width:100%;"' + required + '>';
                                field.options.forEach(opt => {
                                    const optVal = opt.value || opt;
                                    const optDisplay = opt.display || opt.text || opt;
                                    const selected = value === optVal ? ' selected' : '';
                                    html += '<option value="' + optVal + '"' + selected + '>' + optDisplay + '</option>';
                                });
                                html += '</select>';
                            } else if (field.type === 'number') {
                                html += '<input type="number" id="config_' + field.id + '" value="' + value + '" style="background:#0d1117;border:1px solid #30363d;color:#e6edf3;padding:0.5rem;border-radius:6px;font-size:0.875rem;width:100%;"' + required + '>';
                            } else if (field.type === 'color') {
                                html += '<input type="color" id="config_' + field.id + '" value="' + (value || '#ffffff') + '" style="width:100%;height:2.5rem;border:1px solid #30363d;border-radius:6px;cursor:pointer;">';
                            } else if (field.type === 'location') {
                                // Location picker - needs JSON object with lat, lng, timezone
                                // Try to parse existing value or show empty fields
                                let locLat = '', locLng = '', locTz = '';
                                try {
                                    if (value) {
                                        const loc = typeof value === 'string' ? JSON.parse(value) : value;
                                        locLat = loc.lat || '';
                                        locLng = loc.lng || '';
                                        locTz = loc.timezone || '';
                                    }
                                } catch(e) {}
                                html += '<div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem;">';
                                html += '<input type="text" id="config_' + field.id + '_lat" value="' + locLat + '" placeholder="Latitude (e.g. 40.7128)" style="background:#0d1117;border:1px solid #30363d;color:#e6edf3;padding:0.5rem;border-radius:6px;font-size:0.875rem;">';
                                html += '<input type="text" id="config_' + field.id + '_lng" value="' + locLng + '" placeholder="Longitude (e.g. -74.0060)" style="background:#0d1117;border:1px solid #30363d;color:#e6edf3;padding:0.5rem;border-radius:6px;font-size:0.875rem;">';
                                html += '</div>';
                                html += '<input type="text" id="config_' + field.id + '_tz" value="' + locTz + '" placeholder="Timezone (e.g. America/New_York)" style="background:#0d1117;border:1px solid #30363d;color:#e6edf3;padding:0.5rem;border-radius:6px;font-size:0.875rem;width:100%;margin-top:0.5rem;">';
                            } else {
                                // Default: text input (also handles 'text', 'typeahead', etc.)
                                html += '<input type="text" id="config_' + field.id + '" value="' + value + '" style="background:#0d1117;border:1px solid #30363d;color:#e6edf3;padding:0.5rem;border-radius:6px;font-size:0.875rem;width:100%;"' + required + '>';
                            }
                            html += '</div>';
                        });
                    } else {
                        html = '<div style="color:#8b949e;padding:1rem;text-align:center;">This app has no configuration options.</div>';
                    }
                    
                    fieldsDiv.innerHTML = html;
                    document.getElementById('configPanel').style.display = 'block';
                } catch (parseErr) {
                    fieldsDiv.innerHTML = '<div style="color:#f85149;">Invalid schema format</div>';
                }
            } catch (e) {
                console.error('Failed to load app config:', e);
                document.getElementById('configFields').innerHTML = '<div style="color:#f85149;">Failed to load app configuration</div>';
            }
        });
        
        document.getElementById('saveConfigBtn').addEventListener('click', async () => {
            if (!currentConfigApp) return;
            
            const configData = {};
            const locationFields = {};
            const inputs = document.getElementById('configFields').querySelectorAll('input, select');
            inputs.forEach(input => {
                const key = input.id.replace('config_', '');
                // Handle location fields (lat, lng, tz) - combine into JSON object
                if (key.endsWith('_lat') || key.endsWith('_lng') || key.endsWith('_tz')) {
                    const baseKey = key.replace(/_lat$|_lng$|_tz$/, '');
                    if (!locationFields[baseKey]) locationFields[baseKey] = {};
                    if (key.endsWith('_lat')) locationFields[baseKey].lat = parseFloat(input.value) || 0;
                    else if (key.endsWith('_lng')) locationFields[baseKey].lng = parseFloat(input.value) || 0;
                    else if (key.endsWith('_tz')) locationFields[baseKey].timezone = input.value || 'UTC';
                }
                // Handle checkboxes (boolean values) - save as "true"/"false" strings
                else if (input.type === 'checkbox') {
                    configData[key] = input.checked ? 'true' : 'false';
                } else {
                    configData[key] = input.value;
                }
            });
            // Merge location fields as JSON strings
            Object.keys(locationFields).forEach(key => {
                configData[key] = JSON.stringify(locationFields[key]);
            });
            
            const status = document.getElementById('configStatus');
            try {
                status.textContent = 'Saving...';
                status.style.color = '#8b949e';
                await api('api/apps/' + currentConfigApp + '/config', {
                    method: 'PUT',
                    body: JSON.stringify(configData)
                });
                status.textContent = 'Configuration saved!';
                status.style.color = '#238636';
                setTimeout(() => status.textContent = '', 3000);
            } catch (e) {
                status.textContent = 'Failed to save: ' + e.message;
                status.style.color = '#f85149';
            }
        });
        
        document.getElementById('resetConfigBtn').addEventListener('click', () => {
            document.getElementById('configAppSelect').value = '';
            document.getElementById('configPanel').style.display = 'none';
            document.getElementById('configStatus').textContent = '';
        });
        
        // Initial load
        fetchDisplays().then(() => {
            fetchFrame();
            fetchStatus();
            fetchRotation();
            fetchInstalledApps();
        });
        
        // Refresh
        setInterval(fetchFrame, 1000);
        setInterval(fetchStatus, 5000);
    </script>
</body>
</html>`
