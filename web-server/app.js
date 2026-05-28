/* ── Tab switching ────────────────────────────────────────── */
function switchTab(tab) {
    document.querySelectorAll('.nav-links li').forEach(el => el.classList.remove('active'));
    document.querySelector(`.nav-links li[onclick="switchTab('${tab}')"]`).classList.add('active');
    document.querySelectorAll('.tab-content').forEach(el => el.classList.remove('active'));
    document.getElementById(`${tab}-tab`).classList.add('active');
}

/* ── Toast helper ─────────────────────────────────────────── */
let _toastTimer = null;
function showToast(msg) {
    const toast = document.getElementById('toast');
    toast.textContent = msg;
    toast.classList.add('show');
    clearTimeout(_toastTimer);
    _toastTimer = setTimeout(() => toast.classList.remove('show'), 2200);
}

/* ── Copy to clipboard ────────────────────────────────────── */
function copyToClip(text, el) {
    navigator.clipboard.writeText(text).then(() => {
        showToast('✓ Copié : ' + text);
        if (el && el.classList) {
            el.classList.add('copied');
            setTimeout(() => el.classList && el.classList.remove('copied'), 1400);
        }
    }).catch(() => {
        showToast('Copie non disponible dans ce contexte');
    });
}

/* ── API status polling ───────────────────────────────────── */
async function fetchStatus() {
    try {
        const res  = await fetch('/api/status');
        const data = await res.json();

        const dot      = document.getElementById('statusDot');
        const text     = document.getElementById('statusText');
        const settings = document.getElementById('serverSettingsGroup');
        const btnStart = document.getElementById('btnStartServer');
        const btnStop  = document.getElementById('btnStopServer');
        const info     = document.getElementById('serverInfo');
        const hostAddr = document.getElementById('hostAddress');

        if (data.status === 'running') {
            dot.classList.add('connected');
            text.textContent = 'CONNECTÉ';
            settings && settings.classList.add('hidden');
            if (data.address) {
                hostAddr.textContent = data.address;
                hostAddr.onclick = () => copyToClip(data.address, hostAddr);
                hostAddr.style.cursor = 'pointer';
            }
            btnStart.classList.add('hidden');
            btnStop.classList.remove('hidden');
            info.classList.remove('hidden');
        } else {
            dot.classList.remove('connected');
            text.textContent = 'DÉCONNECTÉ';
            settings && settings.classList.remove('hidden');
            btnStart.classList.remove('hidden');
            btnStop.classList.add('hidden');
            info.classList.add('hidden');
        }
    } catch {
        /* Backend injoignable */
    }
}

/* ── Start server ─────────────────────────────────────────── */
async function startServer() {
    const btn = document.getElementById('btnStartServer');
    btn.textContent = 'Exposition en cours…';
    btn.disabled = true;

    const localPort = parseInt(document.getElementById('txtLocalPort').value.trim() || '80', 10);
    const allowCaching = document.getElementById('chkAllowCaching').checked;

    try {
        await fetch('/api/server/start', {
            method:  'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                local_port: localPort,
                allow_caching: allowCaching,
            }),
        });
    } catch { /* ignore */ }

    btn.textContent = 'Exposer mon site';
    btn.disabled = false;
    fetchStatus();
}

/* ── Stop node ────────────────────────────────────────────── */
async function stopNode() {
    try { await fetch('/api/stop', { method: 'POST' }); } catch { /* ignore */ }
    fetchStatus();
}

/* ── Boot ─────────────────────────────────────────────────── */
setInterval(fetchStatus, 2500);
fetchStatus();
