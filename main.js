
const JSON_URL = 'https://johncliver.github.io/xxx/m3u8_results.json';

let videosData = {};
let hlsInstance = null;

// Elementos del DOM
const gridView = document.getElementById('grid-view');
const detailsView = document.getElementById('details-view');
const videoPlayer = document.getElementById('video-player');
const videoTitle = document.getElementById('video-title');
const qualityButtons = document.getElementById('quality-buttons');
const relatedGrid = document.getElementById('related-grid');
const logo = document.getElementById('logo');

// Cargar Datos
async function loadVideos() {
    try {
        const res = await fetch(JSON_URL);
        videosData = await res.json();

        // Asignar IDs/Hashes únicos basados en las claves del JSON
        let index = 0;
        for (const key in videosData) {
            videosData[key].id = `video_${index++}`;
            videosData[key].originalKey = key;
        }

        // Escuchar cambios de Hash en la URL
        window.addEventListener('hashchange', handleRoute);

        // Evaluar la ruta inicial
        handleRoute();

    } catch (err) {
        console.error('Error cargando el JSON:', err);
    }
}

// Manejo de Enrutamiento por Hash (#)
function handleRoute() {
    const hash = window.location.hash.replace('#', '');

    if (!hash) {
        renderGrid();
    } else {
        // Buscar el video con ese ID/hash
        const videoEntry = Object.values(videosData).find(v => v.id === hash);
        if (videoEntry) {
            renderDetails(videoEntry);
        } else {
            renderGrid();
        }
    }
}

// Renderizar Vista Principal (Grid)
function renderGrid() {
    // Detener video si estaba en reproducción
    destroyHls();

    detailsView.classList.add('hidden');
    gridView.classList.remove('hidden');
    gridView.innerHTML = '';

    Object.values(videosData).forEach(item => {
        const card = createCard(item);
        gridView.appendChild(card);
    });
}

// Crear Card de Video
function createCard(item) {
    const card = document.createElement('div');
    card.className = 'card';

    const title = item.title && item.title.trim() !== ''
        ? item.title
        : item.originalKey.split('/').filter(Boolean).pop() || 'Video sin título';

    card.innerHTML = `
        <div class="card-thumb">▶</div>
        <div class="card-body">
          <div class="card-title">${title}</div>
        </div>
      `;

    card.onclick = () => {
        window.location.hash = item.id;
    };

    return card;
}

// Renderizar Vista de Detalles
function renderDetails(videoItem) {
    gridView.classList.add('hidden');
    detailsView.classList.remove('hidden');

    const title = videoItem.title && videoItem.title.trim() !== ''
        ? videoItem.title
        : videoItem.originalKey.split('/').filter(Boolean).pop() || 'Video sin título';

    videoTitle.textContent = title;

    // Renderizar Botones de Calidad (.m3u8)
    qualityButtons.innerHTML = '';
    if (videoItem.m3u8 && videoItem.m3u8.length > 0) {
        videoItem.m3u8.forEach((m3u8Url, index) => {
            const btn = document.createElement('button');
            btn.className = `btn-quality ${index === 0 ? 'active' : ''}`;

            // Tratar de inferir la resolución del nombre de archivo (ej. 720p, 250p)
            const qualityMatch = m3u8Url.match(/hls-(\d+p)/);
            btn.textContent = qualityMatch ? qualityMatch[1] : `Opción ${index + 1}`;

            btn.onclick = () => {
                document.querySelectorAll('.btn-quality').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                playM3U8(m3u8Url);
            };

            qualityButtons.appendChild(btn);
        });

        // Reproducir la primera opción por defecto
        playM3U8(videoItem.m3u8[0]);
    }

    // Renderizar Videos Relacionados (Excluyendo el actual)
    relatedGrid.innerHTML = '';
    Object.values(videosData)
        .filter(v => v.id !== videoItem.id)
        .slice(0, 4) // Mostrar hasta 4 relacionados
        .forEach(related => {
            relatedGrid.appendChild(createCard(related));
        });

    window.scrollTo(0, 0);
}

// Reproducción de m3u8 con HLS.js
function playM3U8(url) {
    destroyHls();

    if (Hls.isSupported()) {
        hlsInstance = new Hls();
        hlsInstance.loadSource(url);
        hlsInstance.attachMedia(videoPlayer);
    } else if (videoPlayer.canPlayType('application/vnd.apple.mpegurl')) {
        // Soporte nativo de Safari / iOS
        videoPlayer.src = url;
    }
}

function destroyHls() {
    if (hlsInstance) {
        hlsInstance.destroy();
        hlsInstance = null;
    }
    videoPlayer.pause();
    videoPlayer.removeAttribute('src');
    videoPlayer.load();
}

// Volver a Home
logo.onclick = () => {
    window.location.hash = '';
};

// Iniciar App
loadVideos();
