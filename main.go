package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

type VideoResult struct {
	Title string   `json:"title"`
	Image string   `json:"image"`
	M3U8  []string `json:"m3u8"`
}

var (
	mu           sync.Mutex
	results      = make(map[string]*VideoResult)
	currentVideo string
	notifyCh     chan struct{}
)

const (
	videoURLsFile = "video_urls.txt"
	m3u8TxtFile   = "m3u8_results.txt"
	m3u8JSONFile  = "m3u8_results.json"
)

// isVideoComplete: ya tiene todo lo que queremos (no hace falta reprocesar)
func isVideoComplete(r *VideoResult) bool {
	return r != nil && r.Title != "" && r.Image != "" && len(r.M3U8) > 0
}

func main() {
	binPath := findBraveExe()
	if binPath == "" {
		fmt.Println("❌ No se encontró brave.exe.")
		return
	}
	fmt.Printf("✅ Brave: %s\n", binPath)

	userDataDir := `C:\Users\john\AppData\Local\BraveSoftware\Brave-Browser\User Data`
	if _, err := os.Stat(userDataDir); err != nil {
		fmt.Printf("❌ No existe el perfil: %s\n", userDataDir)
		return
	}
	fmt.Printf("✅ Perfil: %s\n", userDataDir)

	// ---- Cargar estado previo ----
	results = loadExistingResults()
	existingURLs := loadExistingVideoURLs()
	existingSet := make(map[string]bool, len(existingURLs))
	for _, u := range existingURLs {
		existingSet[u] = true
	}
	fmt.Printf("📂 URLs previas: %d | Videos ya en JSON: %d\n",
		len(existingURLs), len(results))

	// ---- Elegir modo ----
	mode := askMode()

	var videoURLsToProcess []string
	switch mode {
	case 1:
		for _, u := range existingURLs {
			if isVideoComplete(results[u]) {
				fmt.Printf("⏭️  Completo: %s\n", u)
				continue
			}
			videoURLsToProcess = append(videoURLsToProcess, u)
		}
		if len(videoURLsToProcess) == 0 {
			fmt.Println("✅ No hay URLs pendientes en video_urls.txt")
			return
		}
	case 2:
		// Modo buscar: se ejecuta después de lanzar el navegador
	default:
		fmt.Println("❌ Opción no válida.")
		return
	}

	fmt.Println("⚠️ CIERRA Brave completamente (incluido el tray). Enter para continuar...")
	fmt.Scanln()

	browser, err := launchBraveWithUserProfile(binPath, userDataDir)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	defer browser.MustClose()

	page := browser.MustPage()
	fmt.Println("✅ Conectado a Brave con TU perfil")

	// --- Router: captura .m3u8 y guarda JSON al instante ---
	router := browser.HijackRequests()
	defer router.MustStop()
	router.MustAdd("*", func(ctx *rod.Hijack) {
		reqURL := ctx.Request.URL().String()
		if strings.Contains(reqURL, ".m3u8") {
			mu.Lock()
			if currentVideo != "" {
				r := results[currentVideo]
				if r == nil {
					r = &VideoResult{}
					results[currentVideo] = r
				}
				exists := false
				for _, u := range r.M3U8 {
					if u == reqURL {
						exists = true
						break
					}
				}
				if !exists {
					r.M3U8 = append(r.M3U8, reqURL)
					fmt.Printf("   🎬 [HLS] %s\n", reqURL)
					_ = appendLine(m3u8TxtFile, currentVideo+"\t"+reqURL)
					saveJSONLocked()
					if notifyCh != nil {
						select {
						case notifyCh <- struct{}{}:
						default:
						}
					}
				}
			}
			mu.Unlock()
		}
		ctx.MustLoadResponse()
	})
	go router.Run()

	listBaseURL := "https://www.prueba.es/" // ⚠️ Base para paginar
	hrefBaseURL := "https://www.prueba.es"  // ⚠️ Base para resolver href

	if mode == 2 {
		videoURLsToProcess = collectVideoURLs(page, listBaseURL, hrefBaseURL, existingSet)
		if len(videoURLsToProcess) == 0 {
			fmt.Println("❌ No se encontraron URLs nuevas.")
			return
		}
	}

	fmt.Printf("\n✅ Total URLs a procesar: %d\n", len(videoURLsToProcess))

	for i, vURL := range videoURLsToProcess {
		processVideo(page, vURL, i+1, len(videoURLsToProcess))
	}

	mu.Lock()
	saveJSONLocked()
	mu.Unlock()
	fmt.Println("\n🏁 Proceso completado.")
}

// ------------------------------------------------------------
// Input del usuario
// ------------------------------------------------------------
func askMode() int {
	fmt.Println("\n¿Qué quieres hacer?")
	fmt.Println("  [1] Procesar las URLs existentes en video_urls.txt")
	fmt.Println("  [2] Buscar nuevas URLs y procesarlas")
	fmt.Print("Opción [1/2]: ")
	var s string
	fmt.Scanln(&s)
	s = strings.TrimSpace(s)
	if s == "2" {
		return 2
	}
	return 1
}

// ------------------------------------------------------------
// Carga de archivos previos
// ------------------------------------------------------------
func loadExistingVideoURLs() []string {
	data, err := os.ReadFile(videoURLsFile)
	if err != nil {
		return nil
	}
	var urls []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			urls = append(urls, line)
		}
	}
	return urls
}

func loadExistingResults() map[string]*VideoResult {
	m := make(map[string]*VideoResult)

	// Prioridad 1: JSON enriquecido
	if data, err := os.ReadFile(m3u8JSONFile); err == nil {
		if err := json.Unmarshal(data, &m); err == nil && len(m) > 0 {
			return m
		}
	}

	// Fallback: TXT plano (sin títulos)
	data, err := os.ReadFile(m3u8TxtFile)
	if err != nil {
		return m
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		if m[parts[0]] == nil {
			m[parts[0]] = &VideoResult{}
		}
		m[parts[0]].M3U8 = append(m[parts[0]].M3U8, parts[1])
	}
	return m
}

// ------------------------------------------------------------
// Lanzar Brave con perfil real + auto-cleanup
// ------------------------------------------------------------
func launchBraveWithUserProfile(binPath, userDataDir string) (*rod.Browser, error) {
	l := launcher.New().
		Bin(binPath).
		UserDataDir(userDataDir).
		Headless(false).
		Leakless(true).
		Set("disable-blink-features", "AutomationControlled").
		Set("no-first-run").
		Set("no-default-browser-check")

	u, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("error al lanzar Brave con perfil: %w", err)
	}
	fmt.Println("🚀 Brave lanzado con tu perfil (auto-cleanup activado)")
	return rod.New().ControlURL(u).MustConnect(), nil
}

// ------------------------------------------------------------
// Fase 1: paginación
// ------------------------------------------------------------
func collectVideoURLs(page *rod.Page, listBaseURL, hrefBaseURL string, existingSet map[string]bool) []string {
	var newURLs []string
	seen := make(map[string]bool, len(existingSet))
	for k := range existingSet {
		seen[k] = true
	}
	const maxPages = 200

	for pageNum := 1; pageNum <= maxPages; pageNum++ {
		var target string
		if pageNum == 1 {
			target = listBaseURL + "#_videos"
		} else {
			target = fmt.Sprintf("%s#_videos,page-%d", listBaseURL, pageNum)
		}
		fmt.Printf("\n📄 Página %d: %s\n", pageNum, target)

		page.MustNavigate("about:blank")
		if err := page.Navigate(target); err != nil {
			fmt.Printf("⚠️ Navigate error: %v\n", err)
			break
		}
		_ = page.Timeout(30 * time.Second).WaitLoad()

		if _, err := page.Timeout(15 * time.Second).Element("div.thumb"); err != nil {
			fmt.Printf("🏁 Sin 'div.thumb' en página %d (%v)\n", pageNum, err)
			break
		}

		links, err := page.Elements("div.thumb a[href]")
		if err != nil || len(links) == 0 {
			fmt.Printf("🏁 Sin links en página %d\n", pageNum)
			break
		}

		newCount := 0
		for _, link := range links {
			href, err := link.Attribute("href")
			if err != nil || href == nil {
				continue
			}
			h := strings.TrimSpace(*href)
			if h == "" || h == "#" || strings.HasPrefix(h, "javascript:") {
				continue
			}
			full := resolveURL(hrefBaseURL, h)
			if !seen[full] {
				seen[full] = true
				newURLs = append(newURLs, full)
				newCount++
				_ = appendLine(videoURLsFile, full)
				fmt.Printf("  💾 %s\n", full)
			}
		}
		fmt.Printf("  → %d nuevas (total nuevo: %d)\n", newCount, len(newURLs))

		if newCount == 0 {
			fmt.Println("🏁 Fin de paginación (sin URLs nuevas)")
			break
		}

		fmt.Println("⏳ Esperando 15s antes de la siguiente página…")
		time.Sleep(15 * time.Second)
	}
	return newURLs
}

// ------------------------------------------------------------
// Fase 2: procesar un video
//   - extrae lo que falte (título / imagen)
//   - si ya tiene m3u8, no espera m3u8 nuevo
//   - si no tiene m3u8, espera el m3u8
//   - 5s antes del siguiente
//
// ------------------------------------------------------------
func processVideo(page *rod.Page, videoURL string, idx, total int) {
	fmt.Printf("\n[%d/%d] 🎥 %s\n", idx, total, videoURL)

	mu.Lock()
	r := results[videoURL]
	if r == nil {
		r = &VideoResult{}
		results[videoURL] = r
	}
	needsM3U8 := len(r.M3U8) == 0
	needsTitle := r.Title == ""
	needsImage := r.Image == ""
	mu.Unlock()

	if !needsM3U8 && !needsTitle && !needsImage {
		fmt.Println("   ✅ Ya completo, se salta")
		return
	}

	localNotify := make(chan struct{}, 1)
	mu.Lock()
	currentVideo = videoURL
	notifyCh = localNotify
	mu.Unlock()

	page.MustNavigate("about:blank")
	if err := page.Navigate(videoURL); err != nil {
		fmt.Printf("   ⚠️ Navigate error: %v\n", err)
		clearCurrent()
		return
	}
	_ = page.Timeout(30 * time.Second).WaitLoad()

	// --- Título ---
	if needsTitle {
		title := extractTitle(page)
		fmt.Printf("   📝 Título: %s\n", title)
		mu.Lock()
		if title != "" {
			results[videoURL].Title = title
		}
		saveJSONLocked()
		mu.Unlock()
	} else {
		fmt.Println("   ⏭️  Título ya presente")
	}

	// --- Imagen ---
	if needsImage {
		img := extractImage(page, videoURL)
		if img != "" {
			fmt.Printf("   🖼️ Imagen: %s\n", img)
		} else {
			fmt.Println("   🖼️ Imagen no encontrada")
		}
		mu.Lock()
		if img != "" {
			results[videoURL].Image = img
		}
		saveJSONLocked()
		mu.Unlock()
	} else {
		fmt.Println("   ⏭️  Imagen ya presente")
	}

	// --- M3U8 ---
	if !needsM3U8 {
		fmt.Println("   ⏭️  M3U8 ya presente, no se espera")
		clearCurrent()
		fmt.Println("   ⏳ Esperando 5s antes del siguiente video…")
		time.Sleep(5 * time.Second)
		return
	}

	gotIt := false
	select {
	case <-localNotify:
		gotIt = true
		fmt.Println("   ✅ .m3u8 capturado automáticamente")
	case <-time.After(10 * time.Second):
		fmt.Println("   ⏳ Sin .m3u8 tras 10s, intentando clic en Play…")
		if err := clickPlay(page); err == nil {
			fmt.Println("   👆 Clic realizado, esperando…")
		}
		select {
		case <-localNotify:
			gotIt = true
			fmt.Println("   ✅ .m3u8 capturado tras clic")
		case <-time.After(20 * time.Second):
			fmt.Println("   ❌ Timeout esperando .m3u8")
		}
	}

	if gotIt {
		mu.Lock()
		n := len(results[videoURL].M3U8)
		mu.Unlock()
		fmt.Printf("   📊 Total m3u8: %d\n", n)
	}

	clearCurrent()

	fmt.Println("   ⏳ Esperando 5s antes del siguiente video…")
	time.Sleep(5 * time.Second)
}

func extractTitle(page *rod.Page) string {
	if el, err := page.Timeout(8 * time.Second).Element("div.video-title strong"); err == nil {
		if t, e := el.Text(); e == nil {
			if s := strings.TrimSpace(t); s != "" {
				return s
			}
		}
	}
	if el, err := page.Timeout(2 * time.Second).Element("div.video-title"); err == nil {
		if t, e := el.Text(); e == nil {
			return strings.TrimSpace(t)
		}
	}
	return ""
}

// extrae la URL absoluta del img dentro de div.video-pic
func extractImage(page *rod.Page, videoURL string) string {
	el, err := page.Timeout(8 * time.Second).Element("div.video-pic img")
	if err != nil {
		return ""
	}
	// Intenta src, luego data-src, data-original, srcset (lazy load)
	for _, attr := range []string{"src", "data-src", "data-original", "data-lazy-src"} {
		if v, e := el.Attribute(attr); e == nil && v != nil {
			s := strings.TrimSpace(*v)
			if s != "" && !strings.HasPrefix(s, "data:") {
				return resolveURL(videoURL, s)
			}
		}
	}
	// Fallback: srcset (primera URL)
	if v, e := el.Attribute("srcset"); e == nil && v != nil {
		first := strings.Split(strings.TrimSpace(*v), ",")[0]
		first = strings.Fields(first)[0]
		if first != "" {
			return resolveURL(videoURL, first)
		}
	}
	return ""
}

func clearCurrent() {
	mu.Lock()
	currentVideo = ""
	notifyCh = nil
	mu.Unlock()
}

func clickPlay(page *rod.Page) error {
	el, err := page.Timeout(10 * time.Second).Element("#anc-tst-play-btn")
	if err != nil {
		return fmt.Errorf("no se encontró el botón de Play: %w", err)
	}
	if err := el.WaitVisible(); err != nil {
		el.MustEval(`() => {
			this.style.display = 'block';
			this.style.visibility = 'visible';
		}`)
	}
	if err := el.Click(proto.InputMouseButtonLeft, 1); err != nil {
		if span, e := page.Element(`#anc-tst-play-btn span[title="Play"]`); e == nil {
			if err2 := span.Click(proto.InputMouseButtonLeft, 1); err2 != nil {
				el.MustEval(`() => this.click()`)
			}
		} else {
			el.MustEval(`() => this.click()`)
		}
	}
	return nil
}

// ------------------------------------------------------------
// Escritura incremental y JSON
// ------------------------------------------------------------
func appendLine(path, line string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line + "\n")
	return err
}

// saveJSONLocked asume que mu está tomado.
func saveJSONLocked() {
	if len(results) == 0 {
		return
	}
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(m3u8JSONFile, data, 0644)
}

// ------------------------------------------------------------
// Utilidades
// ------------------------------------------------------------
func resolveURL(base, href string) string {
	bu, err := url.Parse(base)
	if err != nil {
		return href
	}
	ru, err := url.Parse(href)
	if err != nil {
		return href
	}
	return bu.ResolveReference(ru).String()
}

func findBraveExe() string {
	candidates := []string{
		`C:\Program Files\BraveSoftware\Brave-Browser\Application\brave.exe`,
		`C:\Program Files (x86)\BraveSoftware\Brave-Browser\Application\brave.exe`,
		`C:\Users\john\AppData\Local\BraveSoftware\Brave-Browser\Application\brave.exe`,
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	patterns := []string{
		`C:\Program Files\BraveSoftware\**\brave.exe`,
		`C:\Program Files (x86)\BraveSoftware\**\brave.exe`,
		`C:\Users\john\AppData\Local\BraveSoftware\**\brave.exe`,
	}
	for _, p := range patterns {
		if matches, _ := filepath.Glob(p); len(matches) > 0 {
			return matches[0]
		}
	}
	return ""
}
