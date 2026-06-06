/* ── Tab switching ────────────────────────────────────────── */
function switchTab(tab) {
    document.querySelectorAll('.nav-links li').forEach(el => el.classList.remove('active'));
    document.querySelector(`.nav-links li[onclick="switchTab('${tab}')"]`).classList.add('active');
    document.querySelectorAll('.tab-content').forEach(el => el.classList.remove('active'));
    document.getElementById(`${tab}-tab`).classList.add('active');
}

/* ── Relay toggle ─────────────────────────────────────────── */
function toggleRelayInput() {
    const chk     = document.getElementById('chkUseRelay');
    const wrapper = document.getElementById('relayInputWrapper');
    wrapper.classList.toggle('hidden', !chk.checked);
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
        const settings = document.getElementById('clientSettingsGroup');
        const btnStart = document.getElementById('btnStartClient');
        const btnStop  = document.getElementById('btnStopClient');
        const info     = document.getElementById('clientInfo');

        if (data.status === 'running') {
            dot.classList.add('connected');
            text.textContent = 'CONNECTÉ';
            settings && settings.classList.add('hidden');
            if (data.proxyPort > 0) {
                btnStart.classList.add('hidden');
                btnStop.classList.remove('hidden');
                info.classList.remove('hidden');
            }
        } else {
            dot.classList.remove('connected');
            text.textContent = 'DÉCONNECTÉ';
            settings && settings.classList.remove('hidden');
            btnStart.classList.remove('hidden');
            btnStop.classList.add('hidden');
            info.classList.add('hidden');
        }
    } catch {
        /* Backend injoignable — UI reste en état déconnecté */
    }
}

/* ── Start client ─────────────────────────────────────────── */
async function startClient() {
    const btn = document.getElementById('btnStartClient');
    btn.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polygon points="5 3 19 12 5 21 5 3"/></svg> Connexion…';
    btn.disabled = true;

    const enableSeeding = document.getElementById('chkEnableSeeding').checked;
    const useRelay      = document.getElementById('chkUseRelay').checked;
    const relayAddr     = useRelay ? document.getElementById('txtRelayAddr').value.trim() : '';
    const isRelay       = document.getElementById('chkIsRelay').checked;

    try {
        await fetch('/api/client/start', {
            method:  'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                disable_seeding: !enableSeeding,
                use_relay:        useRelay,
                relay_addr:       relayAddr,
                is_relay:         isRelay,
            }),
        });
    } catch { /* ignore */ }

    btn.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polygon points="5 3 19 12 5 21 5 3"/></svg> Lancer la connexion sécurisée';
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
