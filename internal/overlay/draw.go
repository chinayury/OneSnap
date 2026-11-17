package overlay

import (
	"errors"
	"fmt"
	"image"
	"syscall"
	"unsafe"
)

var (
	gdi32                  = syscall.NewLazyDLL("gdi32.dll")
	procCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	procDeleteObject       = gdi32.NewProc("DeleteObject")
	procSelectObject       = gdi32.NewProc("SelectObject")
	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procDeleteDC           = gdi32.NewProc("DeleteDC")
	procCreateSolidBrush   = gdi32.NewProc("CreateSolidBrush")
	procCreatePen          = gdi32.NewProc("CreatePen")
	procRectangle          = gdi32.NewProc("Rectangle")
	procSetBkMode          = gdi32.NewProc("SetBkMode")
	procSetTextColor       = gdi32.NewProc("SetTextColor")
	procGetStockObject     = gdi32.NewProc("GetStockObject")
	procCreateFontW        = gdi32.NewProc("CreateFontW")
	procTextOutW           = gdi32.NewProc("TextOutW")
	procGetTextExtent32W   = gdi32.NewProc("GetTextExtentPoint32W")
)

// GDI font constants
const (
	FW_BOLD              = 700
	DEFAULT_CHARSET      = 1
	ANTIALIASED_QUALITY  = 4
	CLIP_DEFAULT_PRECIS  = 0
	OUT_DEFAULT_PRECIS   = 0
	DEFAULT_PITCH        = 0
	FF_DONTCARE          = 0
)

// DrawContext manages GDI resources for overlay drawing
type DrawContext struct {
	HMemDC     uintptr
	hBitmap    uintptr
	hOldBitmap uintptr
	pixels     unsafe.Pointer
	width      int
	height     int
	hFont      uintptr // cached font for hint/size text (scaled to DPI)
	fontSize   int
}

// NewDrawContext creates a 32-bit DIB for overlay drawing
func NewDrawContext(screenDC uintptr, width, height int) (*DrawContext, error) {
	hMemDC, _, _ := procCreateCompatibleDC.Call(screenDC)
	if hMemDC == 0 {
		return nil, errors.New("failed to create compatible DC")
	}

	// Create 32-bit DIB section (top-down with negative height)
	bi := BITMAPINFO{
		BmiHeader: BITMAPINFOHEADER{
			BiSize:        uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
			BiWidth:       int32(width),
			BiHeight:      int32(-height), // Top-down DIB
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: BI_RGB,
		},
	}

	var pixels unsafe.Pointer
	hBitmap, _, _ := procCreateDIBSection.Call(
		hMemDC,
		uintptr(unsafe.Pointer(&bi)),
		DIB_RGB_COLORS,
		uintptr(unsafe.Pointer(&pixels)),
		0, 0,
	)
	if hBitmap == 0 {
		procDeleteDC.Call(hMemDC)
		return nil, errors.New("failed to create DIB section")
	}

	hOldBitmap, _, _ := procSelectObject.Call(hMemDC, hBitmap)

	return &DrawContext{
		HMemDC:     hMemDC,
		hBitmap:    hBitmap,
		hOldBitmap: hOldBitmap,
		pixels:     pixels,
		width:      width,
		height:     height,
	}, nil
}

// DrawOverlay renders the selection overlay
func (dc *DrawContext) DrawOverlay(screenshot *image.RGBA, sel *Selection, scaleRatio float64) {
	// Ensure the scaled font exists (scaleRatio is constant per session)
	dc.ensureFont(scaleRatio)

	// 1. Draw screenshot as background
	dc.drawScreenshot(screenshot)

	// 2. Draw semi-transparent dark overlay
	dc.fillOverlay(128) // 50% opacity

	if sel.IsDragging {
		// 3. Calculate normalized selection bounds
		x1, y1 := minInt(sel.StartX, sel.EndX), minInt(sel.StartY, sel.EndY)
		x2, y2 := maxInt(sel.StartX, sel.EndX), maxInt(sel.StartY, sel.EndY)
		w, h := x2-x1, y2-y1

		// 4. Clear the selection area (show screenshot)
		dc.clearRegion(x1, y1, w, h, screenshot)

		// 5. Draw blue border (2px)
		dc.drawSelectionBorder(x1, y1, w, h)

		// 6. Draw corner handles
		dc.drawCornerHandles(x1, y1, w, h)

		// 7. Draw size indicator (physical pixels)
		scaledW := int(float64(w) * scaleRatio)
		scaledH := int(float64(h) * scaleRatio)
		dc.drawSizeIndicator(x1, y2+8, scaledW, scaledH)
	}

	// 8. Draw instructions (Chinese)
	dc.drawInstructions(sel)
}

// ensureFont creates the GDI font once, sized for the display scale
func (dc *DrawContext) ensureFont(scaleRatio float64) {
	if dc.hFont != 0 {
		return
	}
	size := int(float64(16) * scaleRatio)
	if size < 14 {
		size = 14
	}
	dc.fontSize = size

	dc.hFont, _, _ = procCreateFontW.Call(
		uintptr(-size),          // Height (negative = em height)
		0,                       // Width
		0, 0,                    // Escapement, Orientation
		uintptr(FW_BOLD),        // Weight
		0,                       // Italic
		0,                       // Underline
		0,                       // StrikeOut
		uintptr(DEFAULT_CHARSET),// CharSet
		uintptr(OUT_DEFAULT_PRECIS),
		uintptr(CLIP_DEFAULT_PRECIS),
		uintptr(ANTIALIASED_QUALITY),
		0,                       // Quality preset bits (pitch|family)
		uintptr(DEFAULT_PITCH|FF_DONTCARE),
		uintptr(unsafe.Pointer(fontFaceNamePtr())),
	)
}

// fontFaceNamePtr returns a UTF-16 pointer to the font face name
func fontFaceNamePtr() *uint16 {
	face, _ := syscall.UTF16PtrFromString("Microsoft YaHei UI")
	return face
}

// drawScreenshot copies screenshot to pixel buffer
func (dc *DrawContext) drawScreenshot(screenshot *image.RGBA) {
	if screenshot == nil {
		return
	}

	bounds := screenshot.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// Get pixel buffer as slice with correct bounds
	pixelCount := dc.width * dc.height
	pixels := unsafe.Slice((*uint32)(dc.pixels), pixelCount)

	// Copy pixels (convert RGBA to BGRA with premultiplied alpha)
	for y := 0; y < minInt(srcHeight, dc.height); y++ {
		for x := 0; x < minInt(srcWidth, dc.width); x++ {
			srcIdx := y*screenshot.Stride + x*4
			dstIdx := y*dc.width + x

			r := screenshot.Pix[srcIdx]
			g := screenshot.Pix[srcIdx+1]
			b := screenshot.Pix[srcIdx+2]
			a := uint32(255) // Full opacity

			// BGRA format with premultiplied alpha
			pixels[dstIdx] = (a << 24) | (uint32(r) << 16) | (uint32(g) << 8) | uint32(b)
		}
	}
}

// fillOverlay adds semi-transparent overlay
func (dc *DrawContext) fillOverlay(alpha uint8) {
	pixelCount := dc.width * dc.height
	pixels := unsafe.Slice((*uint32)(dc.pixels), pixelCount)

	for i := 0; i < pixelCount; i++ {
		// Blend with black at specified alpha
		existing := pixels[i]
		r := (existing >> 16) & 0xFF
		g := (existing >> 8) & 0xFF
		b := existing & 0xFF

		// Darken by blending with black
		factor := float64(255-alpha) / 255.0
		r = uint32(float64(r) * factor)
		g = uint32(float64(g) * factor)
		b = uint32(float64(b) * factor)

		pixels[i] = (255 << 24) | (r << 16) | (g << 8) | b
	}
}

// clearRegion restores original screenshot in the selection area
func (dc *DrawContext) clearRegion(x, y, w, h int, screenshot *image.RGBA) {
	if screenshot == nil {
		return
	}

	bounds := screenshot.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	pixelCount := dc.width * dc.height
	pixels := unsafe.Slice((*uint32)(dc.pixels), pixelCount)

	for dy := 0; dy < h; dy++ {
		py := y + dy
		if py < 0 || py >= dc.height || py >= srcHeight {
			continue
		}
		for dx := 0; dx < w; dx++ {
			px := x + dx
			if px < 0 || px >= dc.width || px >= srcWidth {
				continue
			}

			srcIdx := py*screenshot.Stride + px*4
			dstIdx := py*dc.width + px

			r := screenshot.Pix[srcIdx]
			g := screenshot.Pix[srcIdx+1]
			b := screenshot.Pix[srcIdx+2]

			pixels[dstIdx] = (255 << 24) | (uint32(r) << 16) | (uint32(g) << 8) | uint32(b)
		}
	}
}

// fillRectPixels fills a rectangle with a premultiplied BGRA color
func (dc *DrawContext) fillRectPixels(x, y, w, h int, color uint32) {
	pixelCount := dc.width * dc.height
	pixels := unsafe.Slice((*uint32)(dc.pixels), pixelCount)

	for dy := 0; dy < h; dy++ {
		py := y + dy
		if py < 0 || py >= dc.height {
			continue
		}
		for dx := 0; dx < w; dx++ {
			px := x + dx
			if px < 0 || px >= dc.width {
				continue
			}
			pixels[py*dc.width+px] = color
		}
	}
}

// drawSelectionBorder draws a 2px blue border
func (dc *DrawContext) drawSelectionBorder(x, y, w, h int) {
	// Windows blue: #0078D7 -> premultiplied BGRA (alpha 255)
	blue := uint32((255 << 24) | (0xD7 << 16) | (0x78 << 8) | 0x00)
	dc.fillRectPixels(x, y, w, 2, blue)                 // Top
	dc.fillRectPixels(x, y+h-2, w, 2, blue)             // Bottom
	dc.fillRectPixels(x, y, 2, h, blue)                 // Left
	dc.fillRectPixels(x+w-2, y, 2, h, blue)             // Right
}

// drawCornerHandles draws 6x6 blue squares at corners
func (dc *DrawContext) drawCornerHandles(x, y, w, h int) {
	blue := uint32((255 << 24) | (0xD7 << 16) | (0x78 << 8) | 0x00)
	handleSize := 6

	corners := []struct{ cx, cy int }{
		{x - handleSize/2, y - handleSize/2},         // Top-left
		{x + w - handleSize/2, y - handleSize/2},     // Top-right
		{x - handleSize/2, y + h - handleSize/2},     // Bottom-left
		{x + w - handleSize/2, y + h - handleSize/2}, // Bottom-right
	}

	for _, corner := range corners {
		dc.fillRectPixels(corner.cx, corner.cy, handleSize, handleSize, blue)
	}
}

// drawSizeIndicator draws the "W x H" pill with GDI text
func (dc *DrawContext) drawSizeIndicator(x, y, w, h int) {
	text := fmt.Sprintf("%d x %d", w, h)
	dc.drawPillText(x, y, text, 0xD77800) // Windows blue pill (#0078D7 as COLORREF BGR)
}

// drawInstructions draws the instruction pill at top center (Chinese)
func (dc *DrawContext) drawInstructions(sel *Selection) {
	var text string
	if sel.IsDragging && sel.SpaceHeld {
		text = "空格 + 拖动：调整选区位置"
	} else {
		text = "拖动选取区域 · 空格移动选区 · Esc 取消"
	}

	pillX := (dc.width - dc.measurePill(text)) / 2
	dc.drawPillText(pillX, 16, text, 0x000000) // Black pill
}

// measurePill returns the pill width for the given text
func (dc *DrawContext) measurePill(text string) int {
	w := dc.textExtent(text)
	return w + 24
}

// drawPillText draws a rounded pill background with GDI-rendered text on top.
// pillColor is a COLORREF (0x00BBGGRR).
func (dc *DrawContext) drawPillText(x, y int, text string, pillColor COLORREF) {
	const padX = 12
	const padY = 5

	textW := dc.textExtent(text)
	textH := dc.fontSize + 4
	pillW := textW + padX*2
	pillH := textH + padY*2

	// Keep the pill inside the overlay bounds
	if x+pillW >= dc.width {
		x = dc.width - pillW - 4
	}
	if x < 4 {
		x = 4
	}
	if y+pillH >= dc.height {
		y = y - pillH - 8
		if y < 4 {
			y = 4
		}
	}

	// Pill background: opaque BGRA
	bg := colorRefToBGRA(pillColor, 235)
	dc.fillRectPixels(x, y, pillW, pillH, bg)

	// Text stamped with proper alpha over the pill
	dc.stampText(x+padX, y+padY, text, 0xFFFFFF)
}

// textExtent measures text width in pixels using the cached font
func (dc *DrawContext) textExtent(text string) int {
	if dc.hFont == 0 || text == "" {
		return len(text) * dc.fontSize
	}
	hTmpDC, _, _ := procCreateCompatibleDC.Call(dc.HMemDC)
	if hTmpDC == 0 {
		return len(text) * dc.fontSize
	}
	defer procDeleteDC.Call(hTmpDC)

	oldFont, _, _ := procSelectObject.Call(hTmpDC, dc.hFont)
	defer procSelectObject.Call(hTmpDC, oldFont)

	utf16, err := syscall.UTF16FromString(text)
	if err != nil {
		return len(text) * dc.fontSize
	}
	var size SIZE
	procGetTextExtent32W.Call(
		hTmpDC,
		uintptr(unsafe.Pointer(&utf16[0])),
		uintptr(len(utf16)-1), // Exclude null terminator
		uintptr(unsafe.Pointer(&size)),
	)
	if size.Cx <= 0 {
		return len(text) * dc.fontSize
	}
	return int(size.Cx)
}

// stampText renders text with GDI into a small DIB, then blends the glyph
// coverage into the main pixel buffer with correct premultiplied alpha
// (GDI does not maintain the alpha channel on 32bpp DIBs, so we cannot
// TextOut directly onto the layered window surface).
func (dc *DrawContext) stampText(x, y int, text string, colorRef COLORREF) {
	if dc.hFont == 0 || text == "" || dc.pixels == nil {
		return
	}

	utf16, err := syscall.UTF16FromString(text)
	if err != nil {
		return
	}
	charCount := len(utf16) - 1
	if charCount <= 0 {
		return
	}

	textW := dc.textExtent(text)
	textH := dc.fontSize + 8
	if textW <= 0 || textH <= 0 {
		return
	}

	// Create a small 32bpp top-down DIB for glyph rendering
	bi := BITMAPINFO{
		BmiHeader: BITMAPINFOHEADER{
			BiSize:        uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
			BiWidth:       int32(textW),
			BiHeight:      int32(-textH),
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: BI_RGB,
		},
	}
	var smallPixels unsafe.Pointer
	hSmallDC, _, _ := procCreateCompatibleDC.Call(dc.HMemDC)
	if hSmallDC == 0 {
		return
	}
	defer procDeleteDC.Call(hSmallDC)

	hSmallBmp, _, _ := procCreateDIBSection.Call(
		hSmallDC,
		uintptr(unsafe.Pointer(&bi)),
		DIB_RGB_COLORS,
		uintptr(unsafe.Pointer(&smallPixels)),
		0, 0,
	)
	if hSmallBmp == 0 {
		return
	}
	defer procDeleteObject.Call(hSmallBmp)

	oldFont, _, _ := procSelectObject.Call(hSmallDC, dc.hFont)
	defer procSelectObject.Call(hSmallDC, oldFont)

	// White glyphs on transparent (zeroed) background; grayscale AA coverage
	procSetBkMode.Call(hSmallDC, TRANSPARENT)
	procSetTextColor.Call(hSmallDC, 0xFFFFFF) // COLORREF white
	procTextOutW.Call(
		hSmallDC,
		0, 0,
		uintptr(unsafe.Pointer(&utf16[0])),
		uintptr(charCount),
	)

	// Blend glyph coverage into the main DIB
	r := colorRef & 0xFF
	g := (colorRef >> 8) & 0xFF
	b := (colorRef >> 16) & 0xFF

	pixels := unsafe.Slice((*uint32)(dc.pixels), dc.width*dc.height)
	small := unsafe.Slice((*uint32)(smallPixels), textW*textH)

	for dy := 0; dy < textH; dy++ {
		py := y + dy
		if py < 0 || py >= dc.height {
			continue
		}
		for dx := 0; dx < textW; dx++ {
			px := x + dx
			if px < 0 || px >= dc.width {
				continue
			}

			// Grayscale coverage from the glyph bitmap (white on black)
			s := small[dy*textW+dx]
			lum := maxU32((s>>16)&0xFF, maxU32((s>>8)&0xFF, s&0xFF))
			if lum == 0 {
				continue
			}

			// Premultiplied alpha: rgb scaled by coverage
			pr := uint32(float64(r) * float64(lum) / 255.0)
			pg := uint32(float64(g) * float64(lum) / 255.0)
			pb := uint32(float64(b) * float64(lum) / 255.0)

			dst := &pixels[py*dc.width+px]
			*dst = (lum << 24) | (pr << 16) | (pg << 8) | pb
		}
	}
}

// Cleanup releases GDI resources
func (dc *DrawContext) Cleanup() {
	if dc.hFont != 0 {
		procDeleteObject.Call(dc.hFont)
		dc.hFont = 0
	}
	if dc.hOldBitmap != 0 {
		procSelectObject.Call(dc.HMemDC, dc.hOldBitmap)
	}
	if dc.hBitmap != 0 {
		procDeleteObject.Call(dc.hBitmap)
	}
	if dc.HMemDC != 0 {
		procDeleteDC.Call(dc.HMemDC)
	}
}

// COLORREF is a Windows COLORREF (0x00BBGGRR)
type COLORREF uint32

// colorRefToBGRA converts a COLORREF to a premultiplied BGRA pixel value
func colorRefToBGRA(c COLORREF, alpha uint32) uint32 {
	r := uint32(c) & 0xFF
	g := (uint32(c) >> 8) & 0xFF
	b := (uint32(c) >> 16) & 0xFF
	if alpha > 255 {
		alpha = 255
	}
	// Premultiplied: scale rgb by alpha
	factor := float64(alpha) / 255.0
	pr := uint32(float64(r) * factor)
	pg := uint32(float64(g) * factor)
	pb := uint32(float64(b) * factor)
	return (alpha << 24) | (pr << 16) | (pg << 8) | pb
}

// Helper functions
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxU32(a, b uint32) uint32 {
	if a > b {
		return a
	}
	return b
}
