package media

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const DefaultMaxImageBytes int64 = 8 << 20 // 8 MiB

type AVIFProcessor struct {
	RootDir       string
	MaxImageBytes int64
}

type ConvertOptions struct {
	Subdir string
	Width  int
	Height int
	Prefix string
}

type Result struct {
	Filename string `json:"filename"`
	URLPath  string `json:"url_path"`
	Size     int64  `json:"size"`
}

func NewAVIFProcessor(rootDir string) AVIFProcessor {
	return AVIFProcessor{RootDir: rootDir, MaxImageBytes: DefaultMaxImageBytes}
}

func (p AVIFProcessor) ConvertUpload(ctx context.Context, file multipart.File, header *multipart.FileHeader, opts ConvertOptions) (Result, error) {
	if file == nil || header == nil {
		return Result{}, errors.New("image file required")
	}
	if p.RootDir == "" {
		p.RootDir = "storage/uploads"
	}
	if p.MaxImageBytes <= 0 {
		p.MaxImageBytes = DefaultMaxImageBytes
	}
	if opts.Width <= 0 || opts.Height <= 0 {
		return Result{}, errors.New("target width and height required")
	}
	if opts.Subdir == "" {
		opts.Subdir = "images"
	}
	if opts.Prefix == "" {
		opts.Prefix = "image"
	}

	limited := http.MaxBytesReader(nil, file, p.MaxImageBytes+1)
	tmp, err := os.CreateTemp("", "upload-*"+safeExt(header.Filename))
	if err != nil {
		return Result{}, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	defer tmp.Close()

	buf := make([]byte, 512)
	n, readErr := io.ReadFull(limited, buf)
	if readErr != nil && !errors.Is(readErr, io.ErrUnexpectedEOF) && !errors.Is(readErr, io.EOF) {
		return Result{}, readErr
	}
	buf = buf[:n]
	if !isAllowedImage(http.DetectContentType(buf)) {
		return Result{}, errors.New("unsupported image type")
	}
	if _, err := tmp.Write(buf); err != nil {
		return Result{}, err
	}
	written, err := io.Copy(tmp, limited)
	if err != nil {
		return Result{}, err
	}
	if int64(len(buf))+written > p.MaxImageBytes {
		return Result{}, errors.New("image too large")
	}
	if err := tmp.Close(); err != nil {
		return Result{}, err
	}

	id, err := randomID(12)
	if err != nil {
		return Result{}, err
	}
	subdir := cleanSubdir(opts.Subdir)
	outDir := filepath.Join(p.RootDir, subdir)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return Result{}, err
	}
	filename := fmt.Sprintf("%s-%s.avif", cleanName(opts.Prefix), id)
	outPath := filepath.Join(outDir, filename)
	vf := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:color=white@0,format=yuva420p", opts.Width, opts.Height, opts.Width, opts.Height)
	cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-hide_banner", "-loglevel", "error", "-i", tmpPath, "-vf", vf, "-frames:v", "1", "-c:v", "libaom-av1", "-crf", "35", "-b:v", "0", outPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return Result{}, fmt.Errorf("convert image to avif: %w: %s", err, strings.TrimSpace(string(out)))
	}
	st, err := os.Stat(outPath)
	if err != nil {
		return Result{}, err
	}
	return Result{Filename: filename, URLPath: "/uploads/" + subdir + "/" + filename, Size: st.Size()}, nil
}

func (p AVIFProcessor) Serve(w http.ResponseWriter, r *http.Request, path string) {
	if p.RootDir == "" {
		p.RootDir = "storage/uploads"
	}
	f, err := os.Open(filepath.Join(p.RootDir, cleanPath(path)))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	// ServeContent supports Range requests, ETag, If-None-Match, and
	// If-Modified-Since — enabling CDN- and browser-level caching.
	http.ServeContent(w, r, filepath.Base(path), st.ModTime(), f)
}

func cleanPath(v string) string {
	v = strings.Trim(strings.ReplaceAll(v, "\\", "/"), "/")
	parts := strings.Split(v, "/")
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" && part != "." && part != ".." {
			cleaned = append(cleaned, part)
		}
	}
	return strings.Join(cleaned, "/")
}

func randomID(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func isAllowedImage(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/png", "image/gif", "image/webp", "image/avif":
		return true
	default:
		return false
	}
}

func safeExt(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".avif":
		return ext
	default:
		return ".img"
	}
}

func cleanSubdir(v string) string {
	v = strings.Trim(strings.ReplaceAll(v, "\\", "/"), "/")
	parts := strings.Split(v, "/")
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = cleanName(part)
		if part != "" && part != "." && part != ".." {
			cleaned = append(cleaned, part)
		}
	}
	if len(cleaned) == 0 {
		return "images"
	}
	return strings.Join(cleaned, "/")
}

func cleanName(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	var b strings.Builder
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else if r == ' ' || r == '.' {
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "image"
	}
	return out
}

func ContextWithConvertTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, 30*time.Second)
}
