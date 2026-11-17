'use strict';
/* T2C Phantom : assistant du premier lancement, puis carnet de sites et bénévolat. */

const $ = (id) => document.getElementById(id);
let state = { status: {}, sites: [], hosting: null, browser: 'default', step: 1 };

async function api(path, opts = {}) {
    const init = { method: opts.method || 'GET', headers: {} };
    if (opts.body !== undefined) {
        init.headers['Content-Type'] = 'application/json';
        init.body = JSON.stringify(opts.body);
    }
    const res = await fetch(path, init);
    const text = await res.text();
    let data;
    try { data = JSON.parse(text); } catch { data = { ok: res.ok, error: text.trim() }; }
    if (!res.ok && data.ok === undefined) data.ok = false;
    return data;
}

function esc(s) {
    return String(s ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}

function fmtBytes(n) {
    if (!n) return '0 o';
    const u = ['o', 'Ko', 'Mo', 'Go'];
    let i = 0;
    while (n >= 1024 && i < u.length - 1) { n /= 1024; i++; }
    return `${n.toFixed(i ? 1 : 0)} ${u[i]}`;
}

function setState(el, kind, html) {
    el.className = 'state ' + (kind || '');
    el.innerHTML = html;
}

/* ── Démarrage ─────────────────────────────────────────────── */
async function boot() {
    const ob = await api('/api/onboarding');
    if (ob.done) {
        showHome();
    } else {
        showWizard(ob);
    }
    pollStatus();
    setInterval(pollStatus, 3000);
}

/* ── Assistant ─────────────────────────────────────────────── */
function showWizard(ob) {
    $('home').hidden = true;
    $('wizard').hidden = false;
    $('btnSettings').hidden = true;
    renderCAState(ob);
    goStep(1);
}

function goStep(n) {
    state.step = n;
    document.querySelectorAll('.step').forEach((s) => { s.hidden = Number(s.dataset.step) !== n; });
    document.querySelectorAll('.stepper li').forEach((li) => {
        const k = Number(li.dataset.step);
        li.classList.toggle('active', k === n);
        li.classList.toggle('done', k < n);
    });
    if (n === 2) loadBrowsers();
    if (n === 3) refreshProxyState();
}

function renderCAState(ob) {
    if (!ob.canSetup) {
        setState($('caState'), 'warn', 'Configuration automatique désactivée sur ce poste (T2C_NO_BROWSER_SETUP).');
        $('btnCA').hidden = true;
        $('btnCASkip').textContent = 'Continuer';
        return;
    }
    if (ob.caInstalled) {
        setState($('caState'), 'ok', 'Certificat déjà installé : ' + esc((ob.caWhere || []).join(', ')));
        $('btnCA').hidden = true;
        $('btnCASkip').textContent = 'Continuer';
        $('btnCASkip').className = 'btn primary';
    }
}

async function installCA() {
    const btn = $('btnCA');
    btn.disabled = true;
    setState($('caState'), '', 'Installation… (ton système peut demander ton mot de passe)');
    const r = await api('/api/setup/ca', { method: 'POST' });
    btn.disabled = false;
    if (r.ok) {
        setState($('caState'), 'ok', 'Certificat installé : ' + esc((r.report?.ca || []).join(', ')));
        setTimeout(() => goStep(2), 700);
    } else {
        const msgs = [...(r.report?.errors || []), ...(r.report?.manual || []), r.error].filter(Boolean);
        setState($('caState'), 'err', 'Installation incomplète. ' + esc(msgs.join(' · ')));
    }
}

async function loadBrowsers() {
    const r = await api('/api/browsers');
    const list = r.browsers || [];
    const hasFirefox = list.some((b) => b.id === 'firefox');
    $('firefoxHint').hidden = hasFirefox;
    const pick = hasFirefox ? 'firefox' : (list.find((b) => b.recommended)?.id || 'default');
    $('browserList').innerHTML = list.map((b) => `
        <label class="browser">
            <input type="radio" name="browser" value="${esc(b.id)}" ${b.id === pick ? 'checked' : ''}>
            <span class="browser-name">${esc(b.name)}</span>
            ${b.recommended ? '<span class="tag">Conseillé</span>' : ''}
        </label>`).join('');
}

async function chooseBrowser() {
    const sel = document.querySelector('input[name="browser"]:checked');
    state.browser = sel ? sel.value : 'default';
    await openURL('https://tutodecode.org/', state.browser);
    goStep(3);
}

async function refreshProxyState() {
    const ob = await api('/api/onboarding');
    if (ob.proxyOn) {
        setState($('proxyState'), 'ok', 'Proxy actif : ' + esc((ob.proxyWhere || []).join(', ')));
        $('btnProxy').textContent = 'Continuer';
        $('btnProxy').onclick = () => goStep(4);
    } else {
        setState($('proxyState'), '', '');
    }
}

async function enableProxy() {
    const btn = $('btnProxy');
    btn.disabled = true;
    setState($('proxyState'), '', 'Activation…');
    const r = await api('/api/setup/proxy', { method: 'POST' });
    btn.disabled = false;
    const rep = r.report || {};
    if (r.ok) {
        const manual = (rep.manual || []).length ? ' À faire : ' + esc(rep.manual.join(' · ')) : '';
        setState($('proxyState'), 'ok', 'Proxy activé : ' + esc((rep.proxy || []).join(', ')) + manual);
        setTimeout(() => goStep(4), 900);
    } else {
        setState($('proxyState'), 'err', 'Échec : ' + esc([...(rep.errors || []), r.error].filter(Boolean).join(' · ')));
    }
}

async function joinNetwork() {
    const btn = $('btnJoin');
    btn.disabled = true;
    setState($('joinState'), '', 'Création de ton identité et connexion… (quelques secondes la première fois)');
    await startClient();
    for (let i = 0; i < 60; i++) {
        const s = await api('/api/status');
        if (s.connected) {
            setState($('joinState'), 'ok', 'Connecté au réseau.');
            await api('/api/onboarding', { method: 'POST' });
            setTimeout(showHome, 600);
            return;
        }
        if (i > 8 && s.lastError) setState($('joinState'), '', 'Connexion en cours : ' + esc(s.lastError));
        await new Promise((r) => setTimeout(r, 1000));
    }
    btn.disabled = false;
    setState($('joinState'), 'err', 'Le réseau ne répond pas pour le moment. Réessaie dans un instant.');
}

/* ── Connexion ─────────────────────────────────────────────── */
async function startClient() {
    let relay = '';
    try { relay = localStorage.getItem('t2c_relay') || ''; } catch { /* stockage indisponible */ }
    await api('/api/client/start', { method: 'POST', body: { relay_addr: relay } });
}

async function pollStatus() {
    const s = await api('/api/status').catch(() => ({}));
    state.status = s;
    const pill = $('statusPill');
    let cls = 'off', txt = 'Déconnecté';
    if (s.status === 'running') {
        cls = s.connected ? 'on' : 'wait';
        txt = s.connected ? 'Connecté' : 'Connexion…';
    }
    pill.className = 'pill ' + cls;
    $('statusText').textContent = txt;
    const anon = s.status === 'running' && s.connected && s.anonymous;
    $('anonPill').hidden = !anon;
    $('anonPill').title = anon ? `Tes demandes traversent ${s.circuitHops} relais : aucun ne connaît à la fois ton adresse et ce que tu consultes.` : '';
    if (!$('home').hidden) renderStatusDetails(s);
}

function renderStatusDetails(s) {
    $('myId').textContent = s.address || '—';
    $('myRelay').textContent = s.relayAddr || '—';
    $('myChunks').textContent = s.status === 'running'
        ? `${s.chunksPeers || 0} morceaux de bénévoles · ${s.chunksOrigin || 0} du serveur · ${s.badChunks || 0} faux rejetés`
        : '—';
    $('lastError').textContent = s.lastError ? 'Dernier message : ' + s.lastError : '';
    if (s.status !== 'running' || !s.connected) {
        $('myAnon').textContent = '—';
    } else if (s.anonymous) {
        $('myAnon').textContent = `actif : circuit à travers ${s.circuitHops} relais, pseudonyme renouvelé toutes les 10 minutes`;
    } else {
        $('myAnon').textContent = (s.relays || 0) < 2
            ? 'inactif : le réseau ne compte qu\'un relais (il s\'activera dès qu\'un deuxième relais rejoindra la fédération)'
            : 'inactif : connexion en cours au relais d\'entrée';
    }
    if (s.catalogVer) $('catalogInfo').textContent = `catalogue signé v${s.catalogVer}`;
}

/* ── Écran principal ───────────────────────────────────────── */
async function showHome() {
    $('wizard').hidden = true;
    $('home').hidden = false;
    $('btnSettings').hidden = false;
    try { $('relayUrl').value = localStorage.getItem('t2c_relay') || ''; } catch { /* rien */ }
    const s = await api('/api/status');
    if (s.status !== 'running') await startClient();
    await loadSites();
    await loadHosting();
    setInterval(loadSites, 30000);
}

async function loadSites() {
    const r = await api('/api/sites');
    state.sites = r.sites || [];
    if (r.browser) state.browser = r.browser;
    if (!state.sites.length) {
        $('siteList').innerHTML = '<p class="muted">Aucun site pour le moment (connexion au réseau en cours ou catalogue vide).</p>';
    } else {
        $('siteList').innerHTML = state.sites.map((s) => `
            <div class="site">
                <div class="site-info">
                    <span class="site-title">${esc(s.title || s.id)}</span>
                    <span class="site-host">${esc(s.alias)} · ${fmtBytes(s.size)}${s.hosted ? ' · <span class="tag">tu l\'héberges</span>' : ''}</span>
                </div>
                <div class="site-actions">
                    <button class="btn small" onclick="testSite('${esc(s.alias)}')">Tester</button>
                    <button class="btn small primary" onclick="openSite('${esc(s.alias)}')">Ouvrir</button>
                </div>
            </div>`).join('');
    }
    if (state.hosting && !$('hostPanel').hidden) renderHostSites();
}

function customHost() {
    let v = $('customUrl').value.trim().toLowerCase().replace(/^https?:\/\//, '').replace(/\/.*$/, '');
    if (v && !v.endsWith('.t2c')) v += '.t2c';
    return v;
}

async function openSite(host) {
    if (!host) return;
    await openURL(`https://${host}/`, state.browser);
}

async function openURL(url, browser) {
    const r = await api('/api/open', { method: 'POST', body: { url, browser: browser || '' } });
    if (!r.ok) showTest(false, 'Impossible d\'ouvrir le navigateur : ' + esc(r.error));
}

async function testSite(host) {
    if (!host) return;
    showTest(null, `Test de ${esc(host)}…`);
    const r = await api('/api/proxy-test?url=' + encodeURIComponent(`https://${host}/`));
    if (r.ok) {
        showTest(true, `${esc(host)} répond (${r.latencyMs ?? '?'} ms) : signature et contenu vérifiés.`);
    } else {
        showTest(false, `${esc(host)} : ${esc(r.error || 'indisponible')}`);
    }
}

function showTest(ok, html) {
    const el = $('testResult');
    el.hidden = false;
    el.className = 'test-result ' + (ok === true ? 'ok' : ok === false ? 'err' : '');
    el.innerHTML = html;
}

/* ── Bénévolat ─────────────────────────────────────────────── */
async function loadHosting() {
    const r = await api('/api/hosting');
    if (!r.ok) return;
    state.hosting = r.hosting;
    $('chkHost').checked = r.volunteer;
    $('hostPanel').hidden = !r.volunteer;
    $('chkHostAll').checked = r.hosting.all;
    $('maxDisk').value = r.hosting.max_disk_mb;
    $('maxUpload').value = r.hosting.max_upload_kbs;
    renderHostSites();
    $('hostStats').innerHTML = `Copies : ${fmtBytes(r.storeBytes)} · envoyé aux autres membres : ${fmtBytes(r.servedBytes)} · sites hébergés : ${(r.hosted || []).length}`;
}

function renderHostSites() {
    const all = $('chkHostAll').checked;
    const chosen = new Set(state.hosting?.sites || []);
    $('hostSites').innerHTML = state.sites.map((s) => `
        <label class="check">
            <input type="checkbox" class="host-site" value="${esc(s.id)}" ${all || chosen.has(s.id) ? 'checked' : ''} ${all ? 'disabled' : ''}>
            ${esc(s.title || s.id)} <span class="muted">(${fmtBytes(s.size)})</span>
        </label>`).join('') || '<p class="muted">Le catalogue se charge…</p>';
}

function toggleHosting() {
    const on = $('chkHost').checked;
    $('hostPanel').hidden = !on;
    if (!on) saveHosting(true);
}

async function saveHosting(off) {
    const body = {
        all: !off && $('chkHostAll').checked,
        sites: off ? [] : [...document.querySelectorAll('.host-site:checked')].map((c) => c.value),
        max_disk_mb: Number($('maxDisk').value) || 0,
        max_upload_kbs: Number($('maxUpload').value) || 0,
    };
    const r = await api('/api/hosting', { method: 'POST', body });
    $('hostSaved').textContent = r.ok ? (off ? 'Bénévolat arrêté, copies supprimées.' : 'Enregistré : les copies se mettent à jour.') : 'Erreur : ' + (r.error || '');
    setTimeout(loadHosting, 1500);
}

/* ── Réglages ──────────────────────────────────────────────── */
function toggleSettings() { $('settings').hidden = !$('settings').hidden; }

async function reconnect() {
    const v = $('relayUrl').value.trim();
    try { v ? localStorage.setItem('t2c_relay', v) : localStorage.removeItem('t2c_relay'); } catch { /* rien */ }
    await api('/api/stop', { method: 'POST' });
    await startClient();
}

async function disconnect() { await api('/api/stop', { method: 'POST' }); }

async function disableProxy() {
    await api('/api/setup', { method: 'DELETE' });
    showTest(false, 'Proxy désactivé : les sites .t2c ne s\'ouvriront plus tant qu\'il n\'est pas réactivé (Refaire l\'assistant).');
}

async function restartWizard() {
    await api('/api/onboarding', { method: 'DELETE' });
    location.reload();
}

$('customUrl')?.addEventListener('keydown', (e) => { if (e.key === 'Enter') openSite(customHost()); });
boot();
