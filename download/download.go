// downloader.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
)

const (
	m3u8JSONFile = "../m3u8_results.json" // ajusta según ejecutes
	downloadDir  = "../v/media"
	ytdlpBin     = "yt-dlp" // o ruta completa: "C:\\tools\\yt-dlp.exe"
)

type VideoResult struct {
	Title string   `json:"title"`
	Image string   `json:"image"`
	M3U8  []string `json:"m3u8"`
}

type localResult struct {
	Title string   `json:"title"`
	Image string   `json:"image"`
	M3U8  []string `json:"m3u8"`
	Local []string `json:"local"`
}

func main() {
	if _, err := exec.LookPath(ytdlpBin); err != nil {
		fmt.Println("❌ yt-dlp no está en PATH. Instálalo o pon la ruta completa en ytdlpBin.")
		os.Exit(1)
	}

	raw, err := os.ReadFile(m3u8JSONFile)
	if err != nil {
		fmt.Printf("❌ No se pudo leer %s: %v\n", m3u8JSONFile, err)
		os.Exit(1)
	}

	var data map[string]VideoResult
	if err := json.Unmarshal(raw, &data); err != nil {
		fmt.Printf("❌ JSON inválido: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		fmt.Printf("❌ No se pudo crear %s: %v\n", downloadDir, err)
		os.Exit(1)
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make(map[string]localResult, len(data))
	ok, fail := 0, 0

	for i, key := range keys {
		v := data[key]
		id := "video_" + strconv.Itoa(i)
		fmt.Printf("\n[%d/%d] 🎥 %s\n", i+1, len(keys), truncate(v.Title, 70))

		lr := localResult{Title: v.Title, Image: v.Image, M3U8: v.M3U8}

		if len(v.M3U8) == 0 {
			fmt.Println("   ⚠️ Sin m3u8, se omite")
			out[key] = lr
			continue
		}

		outFile := filepath.Join(downloadDir, id+".mp4")
		if fileExists(outFile) && fileSize(outFile) > 1024 {
			fmt.Printf("   ⏭️  Ya existe: %s\n", outFile)
			lr.Local = append(lr.Local, "media/"+id+".mp4")
			ok++
			out[key] = lr
			continue
		}

		// Probar cada m3u8 hasta que uno funcione
		downloaded := false
		for _, m3u8URL := range v.M3U8 {
			if err := downloadWithYtdlp(m3u8URL, outFile); err != nil {
				fmt.Printf("   ❌ Fallo: %v\n", err)
				continue
			}
			lr.Local = append(lr.Local, "media/"+id+".mp4")
			downloaded = true
			break
		}

		if downloaded {
			ok++
		} else {
			fail++
			fmt.Println("   ⚠️ No se pudo descargar ningún m3u8")
		}
		out[key] = lr
	}

	pretty, _ := json.MarshalIndent(out, "", "  ")
	outPath := filepath.Join(downloadDir, "local_results.json")
	if err := os.WriteFile(outPath, pretty, 0644); err != nil {
		fmt.Printf("❌ No se pudo guardar %s: %v\n", outPath, err)
	} else {
		fmt.Printf("\n✅ JSON local guardado en %s\n", outPath)
	}
	fmt.Printf("🏁 OK: %d | Fallidos: %d\n", ok, fail)
}

// downloadWithYtdlp llama a yt-dlp para bajar un m3u8 a MP4
func downloadWithYtdlp(m3u8URL, outFile string) error {
	fmt.Printf("   ⬇️  yt-dlp: %s\n", truncate(m3u8URL, 90))

	// Comando base
	args := []string{
		"--no-playlist",
		"--concurrent-fragments", "8", // descarga paralela de segmentos
		"-f", "bv*+ba/b", // mejor video+audio, o lo que haya
		"--merge-output-format", "mp4", // forzar mp4
		"--postprocessor-args", "ffmpeg:-movflags +faststart", // optimizar reproducción
		"-o", outFile, // salida exacta
		"--no-warnings",
		"--newline", // progreso en líneas (opcional)
	}

	// Headers críticos para streams protegidos
	// Ajusta el Referer al dominio real de tu sitio fuente si es necesario
	args = append(args,
		"--add-header", "Referer: https://www.prueba.es/",
		"--add-header", "Origin: https://www.prueba.es",
		"--user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120 Safari/537.36",
	)

	// Si necesitas cookies de sesión, descomenta y ajusta:
	// args = append(args, "--cookies", "cookies.txt")

	args = append(args, m3u8URL)

	cmd := exec.Command(ytdlpBin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// ------------------------------------------------------------
// Utilidades
// ------------------------------------------------------------
func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func fileSize(p string) int64 {
	fi, err := os.Stat(p)
	if err != nil {
		return 0
	}
	return fi.Size()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
