/* ── Config / constants ───────────────────────────────────── */
const LS_ONBOARD = 't2c_onboard_v4_accept_ca';
const LS_CA_ACCEPTED = 't2c_tutodecode_ca_accepted_v1';
const LS_ASSO_SITES = 't2c_asso_sites_v1';
const LS_CA_FP = 't2c_tutodecode_ca_fp_v1';
let appConfig = {
    gatewayHost: 't2cphantom.eu.org',
    gatewayIP: 'relais',
    gatewayMultiaddr: '',
    cloudflareReady: false,
    proxyPort: 8080,
};

/* ── Tab switching ────────────────────────────────────────── */
function switchTab(tab) {
    document.querySelectorAll('.nav-links li').forEach(el => el.classList.remove('active'));
    const nav = document.querySelector(`.nav-links li[onclick="switchTab('${tab}')"]`);
    if (nav) nav.classList.add('active');
    document.querySelectorAll('.tab-content').forEach(el => el.classList.remove('active'));
    const pane = document.getElementById(`${tab}-tab`);
    if (pane) pane.classList.add('active');
    if (tab === 'network') refreshNetwork();
}

/* ── Relay toggle ─────────────────────────────────────────── */
function toggleRelayInput() {
    const chk = document.getElementById('chkUseRelay');
    const wrapper = document.getElementById('relayInputWrapper');
    if (wrapper) wrapper.classList.toggle('hidden', !chk.checked);
}

/* ── Toast ────────────────────────────────────────────────── */
let _toastTimer = null;
function showToast(msg) {
    const toast = document.getElementById('toast');
    toast.textContent = msg;
    toast.classList.add('show');
    clearTimeout(_toastTimer);
    _toastTimer = setTimeout(() => toast.classList.remove('show'), 2400);
}

function copyToClip(text, el) {
    navigator.clipboard.writeText(text).then(() => {
        showToast('✓ Copié : ' + text);
        if (el && el.classList) {
            el.classList.add('copied');
            setTimeout(() => el.classList && el.classList.remove('copied'), 1400);
        }
    }).catch(() => showToast('Copie non disponible'));
}

/* ── Onboarding : demander → télécharger → accepter CA ───── */
function wireCAAcceptCheckbox() {
    const chk = document.getElementById('chkAcceptCA');
    const btn = document.getElementById('btnAcceptCA');
    if (!chk || !btn) return;
    chk.addEventListener('change', () => { btn.disabled = !chk.checked; });
}

function maybeShowOnboard() {
    const modal = document.getElementById('onboardModal');
    if (!modal) return;
    wireCAAcceptCheckbox();
    // Tant que la CA n’est pas acceptée, on redemande à chaque lancement
    if (localStorage.getItem(LS_CA_ACCEPTED) === '1') {
        modal.classList.add('hidden');
        return;
    }
    modal.classList.remove('hidden');
}

function promptAcceptCA() {
    const modal = document.getElementById('onboardModal');
    if (!modal) return;
    const chk = document.getElementById('chkAcceptCA');
    const btn = document.getElementById('btnAcceptCA');
    if (chk) chk.checked = false;
    if (btn) btn.disabled = true;
    modal.classList.remove('hidden');
}

function finishOnboard(accepted) {
    localStorage.setItem(LS_ONBOARD, '1');
    const modal = document.getElementById('onboardModal');
    if (modal) modal.classList.add('hidden');
    if (accepted) {
        showToast('CA acceptée — importe tutodecode-ca.crt dans Firefox (Autorités)');
        refreshAssoSites(false);
    } else {
        showToast('Sans CA acceptée, Firefox affichera « non sécurisé » — c’est normal');
    }
}

function downloadTutodecodeCA() {
    const a = document.createElement('a');
    a.href = '/api/asso-ca.crt?ts=' + Date.now();
    a.download = 'tutodecode-ca.crt';
    document.body.appendChild(a);
    a.click();
    a.remove();
}

async function acceptAndDownloadCA() {
    const chk = document.getElementById('chkAcceptCA');
    if (!chk || !chk.checked) {
        showToast('Coche d’abord l’acceptation de la CA asso');
        return;
    }
    downloadTutodecodeCA();
    localStorage.setItem(LS_CA_ACCEPTED, '1');
    localStorage.setItem('t2c_prefer_firefox', '1');
    // mémoriser empreinte locale du fichier pour détecter un vrai changement de CA
    try {
        const res = await fetch('/api/asso-ca.crt');
        const buf = await res.arrayBuffer();
        const hash = await crypto.subtle.digest('SHA-256', buf);
        const fp = [...new Uint8Array(hash)].map(b => b.toString(16).padStart(2, '0')).join('');
        localStorage.setItem(LS_CA_FP, fp);
    } catch { /* ignore */ }
    const out = document.getElementById('certValidateResult');
    if (out) {
        out.classList.remove('hidden');
        out.innerHTML = `<span class="ok">✓ CA tutodecode acceptée (source asso)</span><br>
            <span class="muted">Firefox → Certificats → Autorités → Importer tutodecode-ca.crt → trust websites.<br>
            « Non sécurisé » disparaît ensuite sur tous les .t2c asso.<br>
            Le serv refuse tout autre certificat (pas le bon nom / pas tutodecode).</span>`;
    }
    finishOnboard(true);
}

/* ── Sites asso : 1 CA pour tous (pas de re-téléchargement à chaque site) ─ */
async function refreshAssoSites(announceNew) {
    const line = document.getElementById('assoSitesLine');
    const fpEl = document.getElementById('officialCertFP');
    const out = document.getElementById('certValidateResult');
    try {
        const res = await fetch('/api/asso-sites');
        const data = await res.json();
        const sites = data.sites || [];
        const prev = JSON.parse(localStorage.getItem(LS_ASSO_SITES) || '[]');
        const prevSet = new Set(prev);
        const added = sites.filter(s => !prevSet.has(s));
        localStorage.setItem(LS_ASSO_SITES, JSON.stringify(sites));

        if (line) {
            line.textContent = sites.length
                ? ('Sites asso HTTPS (1 CA pour tous) : ' + sites.join(', '))
                : 'Aucun site asso HTTPS pour l’instant';
        }
        if (fpEl) {
            const accepted = localStorage.getItem(LS_CA_ACCEPTED) === '1';
            fpEl.textContent = (accepted ? 'CA acceptée · ' : 'CA pas encore acceptée · ')
                + (data.serverCAFingerprint
                    ? ('source asso pin ' + data.serverCAFingerprint.slice(0, 16) + '…')
                    : 'tutodecode CA');
        }
        if (announceNew && added.length) {
            if (out) {
                out.classList.remove('hidden');
                out.innerHTML = `<span class="ok">Nouveau site asso : ${esc(added.join(', '))}</span><br>
                    <span class="muted">Pas besoin d’une nouvelle CA — celle déjà acceptée couvre tous les .t2c asso.
                    « Non sécurisé » = Firefox n’a pas encore importé tutodecode-ca.crt.</span>`;
            }
            showToast('Nouveau site asso — même CA tutodecode');
            // Si jamais accepté : redemander
            if (localStorage.getItem(LS_CA_ACCEPTED) !== '1') {
                promptAcceptCA();
            }
        }
        return data;
    } catch (e) {
        if (line) line.textContent = 'Catalogue asso injoignable';
        return null;
    }
}

/* ── Load gateway config ──────────────────────────────────── */
async function loadConfig() {
    try {
        const res = await fetch('/api/config');
        appConfig = await res.json();
        const input = document.getElementById('txtRelayAddr');
        if (input && appConfig.gatewayMultiaddr) {
            input.value = appConfig.gatewayMultiaddr;
        }
        const gwLabel = document.getElementById('gatewayLabel');
        if (gwLabel) {
            gwLabel.textContent = appConfig.cloudflareReady
                ? appConfig.gatewayHost
                : appConfig.gatewayIP + ' (CF en attente)';
        }
        const cfBadge = document.getElementById('cfBadge');
        if (cfBadge) {
            cfBadge.textContent = appConfig.cloudflareReady ? 'Cloudflare OK' : 'IP directe';
            cfBadge.className = 'option-badge ' + (appConfig.cloudflareReady ? 'privacy' : 'warning');
        }
        const desc = document.getElementById('gatewayDesc');
        if (desc) {
            desc.textContent = appConfig.cloudflareReady
                ? `Connexion via ${appConfig.gatewayHost} (Cloudflare) — IP VPS/serv masquées`
                : `Domaine CF pas encore délégué — fallback ${appConfig.gatewayIP}:443`;
        }
    } catch { /* ignore */ }
}

/* ── Status polling ───────────────────────────────────────── */
async function fetchStatus() {
    try {
        const res = await fetch('/api/status');
        const data = await res.json();

        const dot = document.getElementById('statusDot');
        const text = document.getElementById('statusText');
        const settings = document.getElementById('clientSettingsGroup');
        const btnStart = document.getElementById('btnStartClient');
        const btnStop = document.getElementById('btnStopClient');
        const info = document.getElementById('clientInfo');
        const idEl = document.getElementById('clientIdValue');

        if (data.status === 'running') {
            dot.classList.add('connected');
            text.textContent = data.seeding ? 'SEEDER ACTIF' : 'CONNECTÉ';
            settings && settings.classList.add('hidden');
            if (data.proxyPort > 0) {
                btnStart && btnStart.classList.add('hidden');
                btnStop && btnStop.classList.remove('hidden');
                info && info.classList.remove('hidden');
            }
            if (idEl && data.address) idEl.textContent = data.address;
            const peersEl = document.getElementById('peersCount');
            if (peersEl) peersEl.textContent = String(data.peers || 0);
            const roleEl = document.getElementById('seederRole');
            if (roleEl) {
                roleEl.textContent = [
                    data.seeding ? 'seeding ON' : 'seeding OFF',
                    data.hopRelay ? 'hop ON' : 'hop OFF',
                ].join(' · ');
            }
        } else {
            dot.classList.remove('connected');
            text.textContent = 'DÉCONNECTÉ';
            settings && settings.classList.remove('hidden');
            btnStart && btnStart.classList.remove('hidden');
            btnStop && btnStop.classList.add('hidden');
            info && info.classList.add('hidden');
        }
    } catch { /* backend down */ }
}

/* ── Start / stop ─────────────────────────────────────────── */
async function startClient() {
    const btn = document.getElementById('btnStartClient');
    btn.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polygon points="5 3 19 12 5 21 5 3"/></svg> Connexion…';
    btn.disabled = true;

    const enableSeeding = document.getElementById('chkEnableSeeding').checked;
    const useRelay = document.getElementById('chkUseRelay').checked;
    const relayAddr = useRelay ? document.getElementById('txtRelayAddr').value.trim() : '';
    const isRelay = document.getElementById('chkIsRelay').checked;

    try {
        await fetch('/api/client/start', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                disable_seeding: !enableSeeding,
                use_relay: useRelay,
                relay_addr: relayAddr,
                is_relay: isRelay,
            }),
        });
        showToast('Connexion OK — accepte la CA tutodecode pour HTTPS asso');
        setTimeout(() => {
            refreshAssoSites(false);
            if (localStorage.getItem(LS_CA_ACCEPTED) !== '1') promptAcceptCA();
        }, 1200);
    } catch {
        showToast('Erreur de démarrage');
    }

    btn.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polygon points="5 3 19 12 5 21 5 3"/></svg> Lancer la connexion sécurisée';
    btn.disabled = false;
    fetchStatus();
}

async function stopNode() {
    try { await fetch('/api/stop', { method: 'POST' }); } catch { /* ignore */ }
    fetchStatus();
}

/* ── Testeur URL .t2c (sites) — pas .tdc (cours) ─────────── */
async function testSiteUrl() {
    const input = document.getElementById('txtTestUrl');
    const out = document.getElementById('testResult');
    let raw = (input.value || '').trim();
    if (!raw) {
        showToast('Entre une URL .t2c (site réseau)');
        return;
    }
    if (/\.tdc(\/|:|$)/i.test(raw) || /\.tdc$/i.test(raw.replace(/^https?:\/\//, ''))) {
        showToast('.tdc = cours, pas un site — utilise .t2c');
        return;
    }
    out.classList.remove('hidden');
    out.innerHTML = '<span class="muted">Test via proxy local…</span>';

    try {
        const res = await fetch('/api/proxy-test?url=' + encodeURIComponent(raw));
        const data = await res.json();
        if (data.ok) {
            out.innerHTML = `<span class="ok">✓ Accessible</span> · HTTP ${data.status} · ${data.latencyMs} ms<br>
                <span class="muted">Vu via ${data.seenVia} · host ${data.host}</span>`;
        } else if (!data.proxyReady) {
            out.innerHTML = `<span class="bad">✗ Proxy inactif</span><br><span class="muted">${data.error || ''}<br>${data.hint || ''}</span>`;
        } else {
            out.innerHTML = `<span class="bad">✗ Injoignable</span><br><span class="muted">${data.error || 'erreur'} · ${data.latencyMs || '?'} ms</span>`;
        }
    } catch (e) {
        out.innerHTML = `<span class="bad">✗ Erreur</span> <span class="muted">${e.message}</span>`;
    }
}

/* ── Page réseau (carnet + catalogue cache 60s) ───────────── */
async function refreshNetwork() {
    const box = document.getElementById('networkLive');
    if (!box) return;
    try {
        const res = await fetch('/api/network');
        const data = await res.json();

        const idLine = document.getElementById('netClientId');
        if (idLine) idLine.textContent = data.clientID || '— (déconnecté)';
        const peersLine = document.getElementById('netPeers');
        if (peersLine) peersLine.textContent = String(data.peers || 0);
        const stamp = document.getElementById('netStamp');
        if (stamp) stamp.textContent = data.polledAt || '';

        const hostedList = document.getElementById('hostedSitesList');
        if (hostedList) {
            const hosted = data.hostedSites || [];
            if (hosted.length === 0) {
                hostedList.innerHTML = '<li class="muted">Aucun site en cache seeder pour l’instant (visite un .t2c via le proxy)</li>';
            } else {
                hostedList.innerHTML = hosted.map(s =>
                    `<li><code class="site-alias">${esc(s.alias)}</code>
                     <span class="pill-mini">${esc(s.status || 'seeding')}</span></li>`
                ).join('');
            }
        }

        const localList = document.getElementById('localSitesList');
        if (localList) {
            if (!data.sites || data.sites.length === 0) {
                localList.innerHTML = '<li class="muted">Aucun alias dans le carnet local</li>';
            } else {
                localList.innerHTML = data.sites.map(s =>
                    `<li><code class="site-alias">${esc(s.alias)}</code> → <span class="muted">${esc(s.target)}</span>
                     <button class="mini-btn" onclick="document.getElementById('txtTestUrl').value='http://${esc(s.alias)}';testSiteUrl();switchTab('tools')">Tester</button></li>`
                ).join('');
            }
        }

        const caBox = document.getElementById('assoCABox');
        if (caBox) {
            const ca = data.assoCA || {};
            if (ca.display && ca.fingerprintSha256) {
                caBox.className = 'asso-ca-box';
                caBox.innerHTML = `<div class="ok">CA locale T2C Phantom (ne signe que les sites du catalogue signé)</div>
                    <div class="mono" style="word-break:break-all;margin-top:8px">SHA-256 ${esc(ca.fingerprintSha256)}</div>
                    <div class="muted" style="margin-top:6px">${esc(ca.note || '')}</div>
                    <details style="margin-top:8px"><summary>PEM</summary><pre class="mono" style="white-space:pre-wrap;font-size:11px">${esc(ca.pem || '')}</pre></details>`;
            } else {
                caBox.className = 'asso-ca-box muted';
                caBox.textContent = 'CA locale pas encore créée — lance la connexion';
            }
        }

        const abuseBox = document.getElementById('abuseBox');
        if (abuseBox) {
            const ab = data.abuse || {};
            const evs = (ab.events || []).slice(-12).reverse();
            abuseBox.className = 'asso-ca-box';
            abuseBox.innerHTML = `<div>signalés: <b>${esc(String(ab.signaled||0))}</b> · throttlés: <b>${esc(String(ab.throttled||0))}</b>
                <div class="muted" style="margin-top:4px">${esc(ab.note||'')}</div></div>
                <ul class="site-list" style="margin-top:8px">${evs.length ? evs.map(e =>
                    `<li><span class="pill-mini">${esc(e.kind||'')}</span>
                     <code class="site-alias">${esc(e.detail||'')}</code>
                     <span class="muted">×${esc(String(e.strikes||0))} · ${esc(e.client||'')}</span></li>`
                ).join('') : '<li class="muted">Aucun abus pour l’instant</li>'}</ul>`;
        }

        const remoteList = document.getElementById('remoteSitesList');
        if (remoteList) {
            const remote = data.remoteCatalog || [];
            if (remote.length === 0) {
                remoteList.innerHTML = '<li class="muted">Catalogue signé vide ou relais injoignable</li>';
            } else {
                remoteList.innerHTML = remote.map(s => {
                    const alias = s.alias || s.name || s.id || JSON.stringify(s);
                    const asso = (s.asso || s.https) ? '<span class="pill-mini ok">HTTPS asso</span>' : '<span class="pill-mini">tiers</span>';
                    return `<li><code class="site-alias">${esc(String(alias))}</code>
                        ${asso}
                        <span class="pill-mini">${esc(s.source || 'remote')}</span></li>`;
                }).join('');
            }
        }

        const seedersList = document.getElementById('seedersList');
        if (seedersList) {
            const seeders = data.seeders || [];
            if (seeders.length === 0) {
                seedersList.innerHTML = '<li class="muted">Aucun bénévole rencontré pour l’instant</li>';
            } else {
                seedersList.innerHTML = seeders.map(s =>
                    `<li><code class="site-alias">${esc(s.peerID || '')}</code>
                     <span class="pill-mini">${esc(s.role || 'peer')}</span>
                     <span class="muted">${esc(s.via || '')}</span></li>`
                ).join('');
            }
        }

        const note = document.getElementById('netNote');
        if (note) note.textContent = data.note || '';
    } catch {
        box && (document.getElementById('netNote').textContent = 'Impossible de charger /api/network');
    }
}

function esc(s) {
    return String(s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}

/* ── Auto-start seeder dès l’ouverture de l’app ───────────── */
async function autoStartSeeder() {
    try {
        const res = await fetch('/api/status');
        const data = await res.json();
        if (data.status === 'running') return;
        await startClient();
    } catch { /* dashboard pas prêt */ }
}

/* ── Boot ─────────────────────────────────────────────────── */
maybeShowOnboard();
loadConfig().then(() => autoStartSeeder()).then(() => setTimeout(() => {
    refreshAssoSites(localStorage.getItem(LS_CA_ACCEPTED) === '1');
    if (localStorage.getItem(LS_CA_ACCEPTED) !== '1') promptAcceptCA();
}, 2000));
setInterval(fetchStatus, 2500);
setInterval(() => {
    const net = document.getElementById('network-tab');
    if (net && net.classList.contains('active')) refreshNetwork();
    refreshAssoSites(true);
}, 60000);
fetchStatus();
