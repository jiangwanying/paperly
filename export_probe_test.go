package main

import (
	"bytes"
	"image/jpeg"
	"image/png"
	"os"
	"testing"

	"paperly/internal/editor"
)

func TestProbeRealPage(t *testing.T) {
	src := `# 高斯积分

正态分布的归一化常数取自高斯积分：

$$
\int_{-\infty}^{\infty} e^{-x^2}\,dx = \sqrt{\pi}
$$

于是密度函数是 $f(x) = \frac{1}{\sqrt{2\pi}} e^{-x^2/2}$。

## 推导

把积分平方，换成极坐标：

$$
\left(\int e^{-x^2}dx\right)^2 = \int\!\!\int e^{-(x^2+y^2)}\,dx\,dy = \pi
$$

- 第一步：平方
- 第二步：极坐标
- 第三步：开方

| 量 | 值 |
| --- | --- |
| 积分 | √π |
| 方差 | 1 |
`
	w := float32(pdfPageW-2*pdfSide) / pointsPerDIP
	h := float32(pdfPageH-pdfTop-pdfFoot) / pointsPerDIP
	ex, err := editor.NewExporter(src, "", editor.NewShared(), editor.PageOptions{Width: w, Height: h, Scale: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("pages=%d, page dips=%.0fx%.0f", ex.Pages(), w, h)
	p := ex.Page(0)
	f, _ := os.Create("/tmp/real-page.png")
	png.Encode(f, p.Image)
	f.Close()
	var buf bytes.Buffer
	jpeg.Encode(&buf, p.Image, &jpeg.Options{Quality: 88})
	t.Logf("jpeg=%d KB", buf.Len()/1024)
	if err := writePDF("/tmp/paperly-real.pdf", []pdfPage{{jpeg: buf.Bytes(), headings: p.Heading}}); err != nil {
		t.Fatal(err)
	}
}
