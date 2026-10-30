// Configuración básica
const JSON_URL_HOME = 'm3u8_results.json';
const ITEMS_PER_PAGE = 12;
const SITE_NAME = 'MyVideos';

const pageMode  = document.body.dataset.page || 'home';
let videosData  = {};
let hlsInstance = null;
let currentPage  = 1;
let searchQuery  = '';
let activeLetter = '';

function getVideoTitle(v) {
    if (!v) return 'Video sin título';
    if (v.title && v.title.trim() !== '') return v.title;
    const parts = (v.originalKey || '').split('/').filter(Boolean);
    return parts.pop() || 'Video sin título';
}

function assignIds(data) {
    const keys = Object.keys(data).sort();
    keys.forEach((key, i) => {
        data[key].id = `video_${i}`;
        data[key].originalKey = key;
    });
}

function getVideoUrl(id) {
    return pageMode === 'video' ? `./${id}.html` : `v/${id}.html`;
}

function getJsonUrl() {
    return pageMode === 'video'
        ? (window.__JSON_URL__ || '../../m3u8_results.json')
        : JSON_URL_HOME;
}

function showToast(msg) {
    const el = document.getElementById('toast');
    if (!el) return;
    el.textContent = msg;
    el.classList.add('show');
    clearTimeout(showToast._t);
    showToast._t = setTimeout(() => el.classList.remove('show'), 2200);
}

function createCard(item) {
    const a = document.createElement('a');
    a.className = 'card';
    a.href = getVideoUrl(item.id);

    const title = getVideoTitle(item);
    const hasImage = item.image && item.image.trim() !== '';

    a.innerHTML = `
        <div class="card-thumb">
          ${hasImage
            ? `<img src="${item.image}" alt="${title.replace(/"/g, '&quot;')}" loading="lazy" onerror="this.style.display='none'">`
            : '<span class="play-icon">▶</span>'}
        </div>
        <div class="card-body">
          <h3 class="card-title">${title}</h3>
        </div>
    `;
    return a;
}

// ---------------------------------------------------------------------------
// Reproducción: local primero, HLS remoto como fallback
// ---------------------------------------------------------------------------

function playLocal(url) {
    destroyHls();
    const videoPlayer = document.getElementById('video-player');
    if (!videoPlayer) return;

    videoPlayer.pause();
    videoPlayer.removeAttribute('src');
    videoPlayer.load();
    videoPlayer.src = url;
    videoPlayer.play().catch(() => {});
}

function playM3U8(url, fallbackList) {
    destroyHls();
    const videoPlayer = document.getElementById('video-player');
    if (!videoPlayer) return;

    videoPlayer.pause();
    videoPlayer.removeAttribute('src');
    videoPlayer.load();

    if (window.Hls && Hls.isSupported()) {
        hlsInstance = new Hls({ enableWorker: true });
        hlsInstance.loadSource(url);
        hlsInstance.attachMedia(videoPlayer);

        hlsInstance.on(Hls.Events.ERROR, (_e, data) => {
            if (data.fatal) {
                console.warn('HLS fatal, intentando fallback:', data);
                const next = (fallbackList || []).find(u => u !== url);
                if (next) {
                    playM3U8(next, (fallbackList || []).filter(u => u !== next));
                } else {
                    showToast('No se pudo reproducir este video');
                }
            }
        });
    } else if (videoPlayer.canPlayType('application/vnd.apple.mpegurl')) {
        videoPlayer.src = url;
        videoPlayer.play().catch(() => {});
    } else {
        showToast('Tu navegador no soporta HLS');
    }
}

function destroyHls() {
    if (hlsInstance) {
        hlsInstance.destroy();
        hlsInstance = null;
    }
}

function renderDetails(video) {
    const titleEl = document.getElementById('video-title');
    if (titleEl) titleEl.textContent = getVideoTitle(video);

    const qualityButtons = document.getElementById('quality-buttons');
    if (!qualityButtons) return;

    qualityButtons.innerHTML = '';

    const locals = Array.isArray(video.local) ? video.local.filter(Boolean) : [];
    const remotes = Array.isArray(video.m3u8) ? video.m3u8.filter(Boolean) : [];

    if (locals.length === 0 && remotes.length === 0) {
        const msg = document.createElement('span');
        msg.style.color = '#888';
        msg.textContent = 'Sin fuentes disponibles';
        qualityButtons.appendChild(msg);
        return;
    }

    const options = [];
    locals.forEach((src, i) => {
        options.push({ label: `Local ${i + 1}`, type: 'local', src });
    });
    remotes.forEach((src, i) => {
        options.push({ label: `HLS ${i + 1}`, type: 'hls', src });
    });

    function activate(opt, btn) {
        qualityButtons.querySelectorAll('.btn-quality').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        if (opt.type === 'local') {
            playLocal(opt.src);
        } else {
            playM3U8(opt.src, remotes);
        }
    }

    options.forEach((opt, i) => {
        const btn = document.createElement('button');
        btn.className = `btn-quality ${i === 0 ? 'active' : ''}`;
        btn.textContent = opt.label;
        btn.onclick = () => activate(opt, btn);
        qualityButtons.appendChild(btn);
    });

    // Autoplay del primero (local si existe)
    if (options[0].type === 'local') {
        playLocal(options[0].src);
    } else {
        playM3U8(options[0].src, remotes);
    }
}

// ---------------------------------------------------------------------------
// Home / filtros / paginación (sin cambios)
// ---------------------------------------------------------------------------

function getFilteredVideos() {
    let items = Object.values(videosData);
    if (activeLetter) {
        items = items.filter(v => getVideoTitle(v).trim().toUpperCase().startsWith(activeLetter));
    }
    if (searchQuery.trim()) {
        const q = searchQuery.toLowerCase().trim();
        items = items.filter(v => getVideoTitle(v).toLowerCase().includes(q));
    }
    return items;
}

function getPageNumbers(current, total) {
    if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1);
    if (current <= 4) return [1, 2, 3, 4, 5, '...', total];
    if (current >= total - 3) return [1, '...', total - 4, total - 3, total - 2, total - 1, total];
    return [1, '...', current - 1, current, current + 1, '...', total];
}

function renderPagination(totalPages, totalItems) {
    const pagTop = document.getElementById('pagination-top');
    const pagBot = document.getElementById('pagination');
    if (pagTop) pagTop.innerHTML = '';
    if (pagBot) pagBot.innerHTML = '';
    if (totalItems === 0) return;

    const start = (currentPage - 1) * ITEMS_PER_PAGE + 1;
    const end = Math.min(currentPage * ITEMS_PER_PAGE, totalItems);

    const buildInto = (container) => {
        if (!container) return;

        const info = document.createElement('span');
        info.className = 'page-info';
        info.textContent = `${start}–${end} de ${totalItems}`;
        container.appendChild(info);

        if (totalPages <= 1) return;

        const goTo = (p) => {
            currentPage = p;
            renderGrid();
            window.scrollTo({ top: 0, behavior: 'smooth' });
        };

        const prev = document.createElement('button');
        prev.className = 'page-btn';
        prev.innerHTML = '‹';
        prev.disabled = currentPage === 1;
        prev.onclick = () => goTo(currentPage - 1);
        container.appendChild(prev);

        getPageNumbers(currentPage, totalPages).forEach(p => {
            if (p === '...') {
                const sp = document.createElement('span');
                sp.className = 'page-ellipsis';
                sp.textContent = '…';
                container.appendChild(sp);
            } else {
                const btn = document.createElement('button');
                btn.className = `page-btn ${p === currentPage ? 'active' : ''}`;
                btn.textContent = p;
                btn.onclick = () => goTo(p);
                container.appendChild(btn);
            }
        });

        const next = document.createElement('button');
        next.className = 'page-btn';
        next.innerHTML = '›';
        next.disabled = currentPage === totalPages;
        next.onclick = () => goTo(currentPage + 1);
        container.appendChild(next);
    };

    buildInto(pagTop);
    buildInto(pagBot);
}

function renderTopMenu() {
    const topMenu = document.getElementById('top-menu');
    if (!topMenu) return;
    topMenu.innerHTML = '';

    const letters = new Set();
    Object.values(videosData).forEach(v => {
        const t = getVideoTitle(v).trim();
        if (t) letters.add(t[0].toUpperCase());
    });
    const sorted = Array.from(letters).sort((a, b) => a.localeCompare(b, 'es'));

    const all = document.createElement('button');
    all.className = `menu-chip ${!activeLetter ? 'active' : ''}`;
    all.textContent = 'Todos';
    all.onclick = () => { activeLetter = ''; currentPage = 1; renderGrid(); };
    topMenu.appendChild(all);

    sorted.forEach(letter => {
        const chip = document.createElement('button');
        chip.className = `menu-chip ${activeLetter === letter ? 'active' : ''}`;
        chip.textContent = letter;
        chip.onclick = () => { activeLetter = letter; currentPage = 1; renderGrid(); };
        topMenu.appendChild(chip);
    });
}

function renderGrid() {
    const cardsGrid = document.getElementById('cards-grid');
    if (!cardsGrid) return;

    renderTopMenu();
    cardsGrid.innerHTML = '';

    const filtered = getFilteredVideos();
    const totalItems = filtered.length;
    const totalPages = Math.max(1, Math.ceil(totalItems / ITEMS_PER_PAGE));

    if (currentPage > totalPages) currentPage = totalPages;
    if (currentPage < 1) currentPage = 1;

    const start = (currentPage - 1) * ITEMS_PER_PAGE;
    const pageItems = filtered.slice(start, start + ITEMS_PER_PAGE);

    if (pageItems.length === 0) {
        cardsGrid.innerHTML = '<div class="empty-state">No se encontraron videos.</div>';
    } else {
        pageItems.forEach(item => cardsGrid.appendChild(createCard(item)));
    }

    renderPagination(totalPages, totalItems);
}

async function initHome() {
    try {
        const res = await fetch(JSON_URL_HOME);
        videosData = await res.json();
        assignIds(videosData);

        const searchInput = document.getElementById('search-input');
        if (searchInput) {
            searchInput.addEventListener('input', (e) => {
                searchQuery = e.target.value;
                currentPage = 1;
                renderGrid();
            });
        }
        renderGrid();
    } catch (err) {
        console.error('Error cargando m3u8_results.json:', err);
    }
}

async function initVideo() {
    const video = window.__VIDEO__;
    if (!video) return;

    renderDetails(video);

    // Compartir
    const shareBtn = document.getElementById('share-btn');
    if (shareBtn) {
        shareBtn.addEventListener('click', async () => {
            try {
                await navigator.clipboard.writeText(location.href);
                showToast('Enlace copiado');
            } catch (e) {
                showToast('No se pudo copiar');
            }
        });
    }

    try {
        const res = await fetch(getJsonUrl());
        const data = await res.json();
        assignIds(data);

        const relatedGrid = document.getElementById('related-grid');
        if (relatedGrid) {
            Object.values(data)
                .filter(v => v.id !== video.id)
                .slice(0, 4)
                .forEach(rel => relatedGrid.appendChild(createCard(rel)));
        }
    } catch (e) {
        console.error(e);
    }
}

if (pageMode === 'video') {
    initVideo();
} else {
    initHome();
}