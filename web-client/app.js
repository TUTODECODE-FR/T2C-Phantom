function switchTab(tab) {
    document.querySelectorAll('.nav-links li').forEach(el => el.classList.remove('active'));
    document.querySelector(`.nav-links li[onclick="switchTab('${tab}')"]`).classList.add('active');
    
    document.querySelectorAll('.tab-content').forEach(el => el.classList.remove('active'));
    document.getElementById(`${tab}-tab`).classList.add('active');
}

async function fetchStatus() {
    try {
        const res = await fetch('/api/status');
        const data = await res.json();
        
        const dot = document.getElementById('statusDot');
        const text = document.getElementById('statusText');
        
        if (data.status === 'running') {
            dot.classList.add('connected');
            text.innerText = "CONNECTÉ";
            
            if (data.proxyPort > 0) {
                document.getElementById('btnStartClient').classList.add('hidden');
                document.getElementById('btnStopClient').classList.remove('hidden');
                document.getElementById('clientInfo').classList.remove('hidden');
            }
        } else {
            dot.classList.remove('connected');
            text.innerText = "DÉCONNECTÉ";
            
            document.getElementById('btnStartClient').classList.remove('hidden');
            document.getElementById('btnStopClient').classList.add('hidden');
            document.getElementById('clientInfo').classList.add('hidden');
        }
    } catch (e) {
        console.error("Dashboard backend injoignable");
    }
}

async function startClient() {
    const relay = document.getElementById('clientRelay').value;
    const btn = document.getElementById('btnStartClient');
    btn.innerText = "Lancement...";
    
    await fetch('/api/client/start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ relay_addr: relay })
    });
    
    btn.innerText = "Lancer la connexion sécurisée";
    fetchStatus();
}

async function stopNode() {
    await fetch('/api/stop', { method: 'POST' });
    fetchStatus();
}

setInterval(fetchStatus, 2000);
fetchStatus();
