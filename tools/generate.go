package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// Tipos
// ---------------------------------------------------------------------------

// Estructura del JSON remoto (m3u8_results.json)
type VideoResult struct {
	Title string   `json:"title"`
	Image string   `json:"image"`
	M3U8  []string `json:"m3u8"`
}

// Estructura del JSON local (v/media/local_results.json), generado por downloader.go
type LocalResult struct {
	Title string   `json:"title"`
	Image string   `json:"image"`
	M3U8  []string `json:"m3u8"`
	Local []string `json:"local"`
}

// Datos que se inyectan en window.__VIDEO__
type VideoItem struct {
	ID          string   `json:"id"`
	OriginalKey string   `json:"originalKey"`
	Title       string   `json:"title"`
	Image       string   `json:"image"`
	M3U8        []string `json:"m3u8"`  // remotos (fallback)
	Local       []string `json:"local"` // rutas locales (media/video_N.mp4)
	URL         string   `json:"url"`
}

type PageData struct {
	ID          string
	Title       string
	Description string
	Image       string
	SiteName    string
	RelativeURL string
	JSONData    template.JS
	JSONURL     string
}

// ---------------------------------------------------------------------------
// Configuración
// ---------------------------------------------------------------------------

const (
	jsonRemoteFile = "../m3u8_results.json"
	jsonLocalFile  = "../v/media/local_results.json"
	outputDir      = "../v"
	siteName       = "MyVideos"
)

// ---------------------------------------------------------------------------
// Template (idéntico al tuyo)
// ---------------------------------------------------------------------------

const videoHtmlTemplate = `<!DOCTYPE html>
<html lang="es">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.Title}} · {{.SiteName}}</title>
  <meta name="description" content="{{.Description}}">

  <meta property="og:type" content="video.other">
  <meta property="og:site_name" content="{{.SiteName}}">
  <meta property="og:title" content="{{.Title}}">
  <meta property="og:description" content="{{.Description}}">
  <meta property="og:image" content="{{.Image}}">

  <meta name="twitter:card" content="summary_large_image">
  <meta name="twitter:title" content="{{.Title}}">
  <meta name="twitter:description" content="{{.Description}}">
  <meta name="twitter:image" content="{{.Image}}">

  <meta name="theme-color" content="#0a0d14">

  <link rel="stylesheet" href="../../style.css">
  <script src="https://cdn.jsdelivr.net/npm/hls.js@latest"></script>

  <!-- SEO: Schema.org VideoObject -->
  <script type="application/ld+json">
  {
    "@context": "https://schema.org",
    "@type": "VideoObject",
    "name": "{{.Title}}",
    "description": "{{.Description}}",
    "thumbnailUrl": ["{{.Image}}"],
    "uploadDate": "2026-01-01T08:00:00+00:00"
  }
  </script>
</head>
<body data-page="video" data-video-id="{{.ID}}">

  <header>
    <a href="../../index.html" class="brand-link"><h1 id="logo">MyVideos</h1></a>
  </header>

  <main>
    <section id="details-view" class="details-container">
      
      <!-- 1. ANUNCIOS SUPERIORES: Carrusel Horizontal de Banners 320x50 (5 anuncios) -->
      <div class="ads-horizontal-scroll">
        <div class="ad-item">
          <script>
            atOptions = { 'key' : 'f800ed383edcb23d085ab823bc318708', 'format' : 'iframe', 'height' : 50, 'width' : 320, 'params' : {} };
          </script>
          <script src="https://www.highrevenueformat.com/f800ed383edcb23d085ab823bc318708/invoke.js"></script>
        </div>
        <div class="ad-item">
          <script>
            atOptions = { 'key' : 'f800ed383edcb23d085ab823bc318708', 'format' : 'iframe', 'height' : 50, 'width' : 320, 'params' : {} };
          </script>
          <script src="https://www.highrevenueformat.com/f800ed383edcb23d085ab823bc318708/invoke.js"></script>
        </div>
        <div class="ad-item">
          <script>
            atOptions = { 'key' : 'f800ed383edcb23d085ab823bc318708', 'format' : 'iframe', 'height' : 50, 'width' : 320, 'params' : {} };
          </script>
          <script src="https://www.highrevenueformat.com/f800ed383edcb23d085ab823bc318708/invoke.js"></script>
        </div>
        <div class="ad-item">
          <script>
            atOptions = { 'key' : 'f800ed383edcb23d085ab823bc318708', 'format' : 'iframe', 'height' : 50, 'width' : 320, 'params' : {} };
          </script>
          <script src="https://www.highrevenueformat.com/f800ed383edcb23d085ab823bc318708/invoke.js"></script>
        </div>
        <div class="ad-item">
          <script>
            atOptions = { 'key' : 'f800ed383edcb23d085ab823bc318708', 'format' : 'iframe', 'height' : 50, 'width' : 320, 'params' : {} };
          </script>
          <script src="https://www.highrevenueformat.com/f800ed383edcb23d085ab823bc318708/invoke.js"></script>
        </div>
      </div>

      <a href="../../index.html" class="back-btn">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round">
          <path d="m15 18-6-6 6-6"></path>
        </svg>
        Volver al inicio
      </a>

      <!-- Layout principal de reproducción + Banner lateral/inferior estratégico -->
      <div class="player-ad-layout">
        <div class="video-wrapper-container">
          <div class="video-wrapper">
            <video id="video-player" controls autoplay playsinline></video>
          </div>
        </div>

        <!-- 2. ANUNCIO ESTRATÉGICO: Banner 300x250 al lado del reproductor -->
        <div class="strategic-ad-box">
          <script>
            atOptions = {
              'key' : '4b47ce1c3e4f238bc7f7cca8ecccb30f',
              'format' : 'iframe',
              'height' : 250,
              'width' : 300,
              'params' : {}
            };
          </script>
          <script src="https://www.highrevenueformat.com/4b47ce1c3e4f238bc7f7cca8ecccb30f/invoke.js"></script>
        </div>
      </div>

      <!-- 3. NATIVE BANNER (ARRIBA) -->
      <div class="native-banner-container" style="margin: 24px 0; text-align: center;">
        <script async="async" data-cfasync="false" src="https://pl20016627.profitableratecpmnetwork.com/c14d6221cacd09bae7934140e373893e/invoke.js"></script>
        <div id="container-c14d6221cacd09bae7934140e373893e"></div>
      </div>

      <div class="details-info">
        <h1 id="video-title" class="details-title">{{.Title}}</h1>
        <span class="details-label">Opción de Reproducción</span>
        <div id="quality-buttons" class="quality-selector"></div>

        <div class="details-actions">
          <button class="share-btn" id="share-btn">
            Compartir
          </button>
        </div>
      </div>

      <!-- 4. NATIVE BANNER (ABAJO - NUEVO) -->
      <div class="native-banner-container-bottom" style="margin: 30px 0; text-align: center;">
        <script async="async" data-cfasync="false" src="https://pl20016627.profitableratecpmnetwork.com/c14d6221cacd09bae7934140e373893e/invoke.js"></script>
        <div id="container-c14d6221cacd09bae7934140e373893e-bottom"></div>
      </div>

      <h2 class="related-title">Videos recomendados</h2>
      <div id="related-grid" class="grid"></div>
    </section>
  </main>

  <div id="toast" class="toast"></div>

  <script>
    window.__VIDEO__ = {{.JSONData}};
    window.__JSON_URL__ = {{.JSONURL}};
  </script>
  <script src="../../main.js"></script>

  <!-- 5. SOCIAL BAR -->
  <script src="https://pl32079806.profitableratecpmnetwork.com/5c/df/d8/5cdfd8d253e82a91556fb7c9b8240ffc.js"></script>

  <!-- 6. POPUNDER -->
  <script src="https://pl29236831.profitableratecpmnetwork.com/7f/14/36/7f14362c8d3ff443959fa5e3e80d01d8.js"></script>
</body>
</html>`

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
	// 1) Intentar cargar el JSON LOCAL primero (con mp4 descargados)
	items, source, err := loadItems()
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("📂 Fuente de datos: %s (%d videos)\n", source, len(items))

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("❌ No se pudo crear %s/: %v\n", outputDir, err)
		os.Exit(1)
	}

	// 2) Funciones del template
	funcMap := template.FuncMap{
		"safeJSON": func(v interface{}) template.JS {
			b, _ := json.Marshal(v)
			s := string(b)
			s = strings.ReplaceAll(s, "</", "<\\/")
			return template.JS(s)
		},
	}

	tmpl, err := template.New("page").Funcs(funcMap).Parse(videoHtmlTemplate)
	if err != nil {
		fmt.Printf("❌ Error en template: %v\n", err)
		os.Exit(1)
	}

	// 3) Generar una página por video
	var generated int
	for _, item := range items {
		desc := fmt.Sprintf("Ver %s online en %s.", item.Title, siteName)
		relURL := fmt.Sprintf("tools/v/%s.html", item.ID)

		page := PageData{
			ID:          item.ID,
			Title:       item.Title,
			Description: desc,
			Image:       item.Image,
			SiteName:    siteName,
			RelativeURL: relURL,
			JSONData:    funcMap["safeJSON"].(func(interface{}) template.JS)(item),
			JSONURL:     "../../m3u8_results.json",
		}

		outPath := filepath.Join(outputDir, item.ID+".html")
		f, err := os.Create(outPath)
		if err != nil {
			fmt.Printf("⚠️  No se pudo crear %s: %v\n", outPath, err)
			continue
		}
		if err := tmpl.Execute(f, page); err != nil {
			fmt.Printf("⚠️  Error ejecutando template en %s: %v\n", outPath, err)
		}
		f.Close()
		generated++
	}

	fmt.Printf("✅ Se regeneraron %d páginas (local + fallback remoto).\n", generated)
}

// ---------------------------------------------------------------------------
// Carga de datos: local primero, remoto como fallback
// ---------------------------------------------------------------------------

func loadItems() ([]VideoItem, string, error) {
	// --- Intento 1: local_results.json (con .mp4 descargados) ---
	if raw, err := os.ReadFile(jsonLocalFile); err == nil {
		var m map[string]LocalResult
		if err := json.Unmarshal(raw, &m); err == nil && len(m) > 0 {
			items := buildItemsFromLocal(m)
			return items, jsonLocalFile, nil
		}
	}

	// --- Intento 2: m3u8_results.json (solo remotos) ---
	if raw, err := os.ReadFile(jsonRemoteFile); err == nil {
		var m map[string]VideoResult
		if err := json.Unmarshal(raw, &m); err == nil && len(m) > 0 {
			items := buildItemsFromRemote(m)
			return items, jsonRemoteFile, nil
		}
	}

	return nil, "", fmt.Errorf("no se encontró %s ni %s válidos", jsonLocalFile, jsonRemoteFile)
}

// buildItemsFromLocal construye los VideoItem a partir del JSON con .mp4 locales
func buildItemsFromLocal(m map[string]LocalResult) []VideoItem {
	keys := sortedKeys(len(m), func(f func(string)) {
		for k := range m {
			f(k)
		}
	})

	items := make([]VideoItem, 0, len(keys))
	for i, key := range keys {
		v := m[key]
		id := "video_" + strconv.Itoa(i)

		title := strings.TrimSpace(v.Title)
		if title == "" {
			title = "Video " + strconv.Itoa(i+1)
		}

		items = append(items, VideoItem{
			ID:          id,
			OriginalKey: key,
			Title:       title,
			Image:       strings.TrimSpace(v.Image),
			M3U8:        v.M3U8,
			Local:       v.Local,
			URL:         fmt.Sprintf("tools/v/%s.html", id),
		})
	}
	return items
}

// buildItemsFromRemote construye los VideoItem a partir del JSON remoto (sin .mp4)
func buildItemsFromRemote(m map[string]VideoResult) []VideoItem {
	keys := sortedKeys(len(m), func(f func(string)) {
		for k := range m {
			f(k)
		}
	})

	items := make([]VideoItem, 0, len(keys))
	for i, key := range keys {
		v := m[key]
		id := "video_" + strconv.Itoa(i)

		title := strings.TrimSpace(v.Title)
		if title == "" {
			title = "Video " + strconv.Itoa(i+1)
		}

		items = append(items, VideoItem{
			ID:          id,
			OriginalKey: key,
			Title:       title,
			Image:       strings.TrimSpace(v.Image),
			M3U8:        v.M3U8,
			Local:       nil, // no hay locales
			URL:         fmt.Sprintf("tools/v/%s.html", id),
		})
	}
	return items
}

// sortedKeys ordena las claves de un map
func sortedKeys(n int, iter func(func(string))) []string {
	keys := make([]string, 0, n)
	iter(func(k string) { keys = append(keys, k) })
	sort.Strings(keys)
	return keys
}
