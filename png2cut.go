package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func hasAlpha(img image.Image) bool {
	switch img.(type) {
	case *image.NRGBA, *image.NRGBA64, *image.RGBA, *image.RGBA64:
		return true
	default:
		return false
	}
}

func padPNGsInDir(targetDir string, saveSubdir string) (int, error) {
	info, err := os.Stat(targetDir)
	if err != nil {
		return 0, err
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("not a directory: %s", targetDir)
	}

	ents, err := os.ReadDir(targetDir)
	if err != nil {
		return 0, err
	}

	var files []string
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(strings.ToLower(name), ".png") {
			files = append(files, name)
		}
	}
	if len(files) == 0 {
		return 0, nil
	}
	sort.Strings(files)

	outDir := filepath.Join(targetDir, saveSubdir)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return 0, err
	}

	saved := 0
	for _, fname := range files {
		srcPath := filepath.Join(targetDir, fname)
		f, err := os.Open(srcPath)
		if err != nil {
			continue
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			continue
		}

		b := img.Bounds()
		w := b.Dx()
		h := b.Dy()
		newW := w + (w % 2)
		newH := h + (h % 2)

		outPath := filepath.Join(outDir, fname)
		if newW == w && newH == h {
			// 直接复制文件以保持原样（避免重新编码带来的细微变化）
			// 如果复制失败，再尝试用 png.Encode 写入
			srcBytes, err := os.ReadFile(srcPath)
			if err == nil {
				_ = os.WriteFile(outPath, srcBytes, 0o644)
				saved++
				continue
			}
			// 回退：重新编码并保存
			of, err := os.Create(outPath)
			if err != nil {
				continue
			}
			_ = png.Encode(of, img)
			of.Close()
			saved++
			continue
		}

		var dst draw.Image
		if hasAlpha(img) {
			dst = image.NewNRGBA(image.Rect(0, 0, newW, newH))
			// 默认零值就是透明，直接复制到左上角
			draw.Draw(dst, image.Rect(0, 0, w, h), img, b.Min, draw.Src)
		} else {
			// 无透明通道：用不透明黑色填充背景，再把原图覆盖上去
			dstRGBA := image.NewRGBA(image.Rect(0, 0, newW, newH))
			black := &image.Uniform{C: color.RGBA{0, 0, 0, 255}}
			draw.Draw(dstRGBA, dstRGBA.Bounds(), black, image.Point{}, draw.Src)
			draw.Draw(dstRGBA, image.Rect(0, 0, w, h), img, b.Min, draw.Src)
			dst = dstRGBA
		}

		of, err := os.Create(outPath)
		if err != nil {
			continue
		}
		_ = png.Encode(of, dst)
		of.Close()
		saved++
	}

	return saved, nil
}

func main() {
	dir := flag.String("dir", ".", "目标目录 (默认 当前目录)")
	out := flag.String("out", "Texture", "输出子目录名")
	flag.Parse()

	n, err := padPNGsInDir(*dir, *out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
	fmt.Printf("处理完成：%d 个 PNG 文件已保存到子目录 %q\n", n, *out)
}
