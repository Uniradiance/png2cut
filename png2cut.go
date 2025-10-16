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

// 新增：处理单个 PNG 文件（保存到文件所在目录的 saveSubdir 子目录）
func padPNGFile(srcPath string, saveSubdir string) (bool, error) {
	info, err := os.Stat(srcPath)
	if err != nil {
		return false, err
	}
	if info.IsDir() {
		return false, fmt.Errorf("is a directory: %s", srcPath)
	}
	if !strings.HasSuffix(strings.ToLower(srcPath), ".png") {
		return false, fmt.Errorf("not a png: %s", srcPath)
	}

	f, err := os.Open(srcPath)
	if err != nil {
		return false, err
	}
	img, err := png.Decode(f)
	f.Close()
	if err != nil {
		return false, err
	}

	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()
	newW := w + (w % 2)
	newH := h + (h % 2)

	outDir := filepath.Join(filepath.Dir(srcPath), saveSubdir)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return false, err
	}
	outPath := filepath.Join(outDir, filepath.Base(srcPath))

	if newW == w && newH == h {
		// 尝试直接复制文件以保持原样
		srcBytes, err := os.ReadFile(srcPath)
		if err == nil {
			if err2 := os.WriteFile(outPath, srcBytes, 0o644); err2 == nil {
				return true, nil
			}
		}
		// 回退：重新编码并保存
		of, err := os.Create(outPath)
		if err != nil {
			return false, err
		}
		_ = png.Encode(of, img)
		of.Close()
		return true, nil
	}

	var dst draw.Image
	if hasAlpha(img) {
		dst = image.NewNRGBA(image.Rect(0, 0, newW, newH))
		draw.Draw(dst, image.Rect(0, 0, w, h), img, b.Min, draw.Src)
	} else {
		dstRGBA := image.NewRGBA(image.Rect(0, 0, newW, newH))
		black := &image.Uniform{C: color.RGBA{0, 0, 0, 255}}
		draw.Draw(dstRGBA, dstRGBA.Bounds(), black, image.Point{}, draw.Src)
		draw.Draw(dstRGBA, image.Rect(0, 0, w, h), img, b.Min, draw.Src)
		dst = dstRGBA
	}

	of, err := os.Create(outPath)
	if err != nil {
		return false, err
	}
	_ = png.Encode(of, dst)
	of.Close()
	return true, nil
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
			// 跳过：宽高已是偶数，无需复制或重新编码
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
	fmt.Printf("图片尺寸对其工具，作者：Uniradiance 邮箱：megatronus@sina.cn")
	dir := flag.String("dir", ".", "目标目录 (默认 当前目录)")
	out := flag.String("out", "Texture", "输出子目录名")
	flag.Parse()

	// 优先处理位置参数（拖放时会作为位置参数传入）
	args := flag.Args()
	totalSaved := 0
	if len(args) > 0 {
		for _, p := range args {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			info, err := os.Stat(p)
			if err != nil {
				fmt.Fprintln(os.Stderr, "无法访问:", p, "错误:", err)
				continue
			}
			if info.IsDir() {
				n, err := padPNGsInDir(p, *out)
				if err != nil {
					fmt.Fprintln(os.Stderr, "处理目录失败:", p, "错误:", err)
					continue
				}
				totalSaved += n
			} else {
				ok, err := padPNGFile(p, *out)
				if err != nil {
					fmt.Fprintln(os.Stderr, "跳过文件:", p, "原因:", err)
					continue
				}
				if ok {
					totalSaved++
				}
			}
		}
		fmt.Printf("处理完成：共 %d 个 PNG 文件已保存到各自的子目录 %q\n", totalSaved, *out)
		return
	}

	// 如果没有位置参数，保持原有通过 -dir 参数处理整个目录的行为
	n, err := padPNGsInDir(*dir, *out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
	fmt.Printf("处理完成：%d 个 PNG 文件已保存到子目录 %q\n", n, *out)
}
