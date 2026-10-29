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

type VideoResult struct {
	Title string   `json:"title"`
	Image string   `json:"image"`
	M3U8  []string `json:"m3u8"`
}

type VideoItem struct {
	ID          string   `json:"id"`
	OriginalKey string   `json:"originalKey"`
	Title       string   `json:"title"`
	Image       string   `json:"image"`
	M3U8        []string `json:"m3u8"`
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

const (
	jsonFile  = "m3u8_results.json"
	outputDir = "tools/v"
	siteName  = "MyVideos"
)

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

        <!-- 2. ANUNCIO ESTRATÉGICO: Banner 300x250 al lado del reproductor (o debajo en móviles) -->
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

      <!-- 3. ANUNCIOS ESTRATÉGICOS: Native Banner -->
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

  <!-- 4. SOCIAL BAR -->
  <script src="https://pl32079806.profitableratecpmnetwork.com/5c/df/d8/5cdfd8d253e82a91556fb7c9b8240ffc.js"></script>

  <!-- 5. POPUNDER -->
  <script src="https://pl29236831.profitableratecpmnetwork.com/7f/14/36/7f14362c8d3ff443959fa5e3e80d01d8.js"></script>
</body>
</html>`

func main() {
	raw, err := os.ReadFile(jsonFile)
	if err != nil {
		fmt.Printf("❌ No se pudo leer %s: %v\n", jsonFile, err)
		os.Exit(1)
	}

	var data map[string]VideoResult
	if err := json.Unmarshal(raw, &data); err != nil {
		fmt.Printf("❌ JSON inválido: %v\n", err)
		os.Exit(1)
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("❌ No se pudo crear %s/: %v\n", outputDir, err)
		os.Exit(1)
	}

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

	var videoList []VideoItem

	for i, key := range keys {
		video := data[key]
		id := "video_" + strconv.Itoa(i)

		title := strings.TrimSpace(video.Title)
		if title == "" {
			title = "Video " + strconv.Itoa(i+1)
		}

		image := strings.TrimSpace(video.Image)
		desc := fmt.Sprintf("Ver %s online en %s.", title, siteName)
		relURL := fmt.Sprintf("tools/v/%s.html", id)

		item := VideoItem{
			ID:          id,
			OriginalKey: key,
			Title:       title,
			Image:       image,
			M3U8:        video.M3U8,
			URL:         relURL,
		}
		videoList = append(videoList, item)

		page := PageData{
			ID:          id,
			Title:       title,
			Description: desc,
			Image:       image,
			SiteName:    siteName,
			RelativeURL: relURL,
			JSONData:    funcMap["safeJSON"].(func(interface{}) template.JS)(item),
			JSONURL:     "../../m3u8_results.json",
		}

		outPath := filepath.Join(outputDir, id+".html")
		f, err := os.Create(outPath)
		if err == nil {
			tmpl.Execute(f, page)
			f.Close()
		}
	}

	fmt.Printf("✅ Se regeneraron %d páginas de detalles con todos los anuncios integrados estratégicamente.\n", len(videoList))
}
