package main

import (
	"context"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"time"
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type UI struct {
	client       *HomebridgeClient
	width        int32
	height       int32
	bgColor      color.RGBA
	highlight    color.RGBA
	textColor    color.RGBA
	subtleColor  color.RGBA
	tileOn       color.RGBA
	tileOff      color.RGBA
	tileBorder   color.RGBA
	fontFace     font.Face
	lineHeight   int
	notice       string
	lastRefresh  time.Time
	refreshEvery time.Duration
	busy         bool
	totalCount   int
	toggleCount  int
}

func NewUI(client *HomebridgeClient, width, height int32) *UI {
	face := loadUIFont(24)
	return &UI{
		client:       client,
		width:        width,
		height:       height,
		bgColor:      color.RGBA{R: 12, G: 14, B: 18, A: 255},
		highlight:    color.RGBA{R: 0, G: 120, B: 212, A: 255},
		textColor:    color.RGBA{R: 240, G: 240, B: 240, A: 255},
		subtleColor:  color.RGBA{R: 120, G: 130, B: 140, A: 255},
		tileOn:       color.RGBA{R: 28, G: 140, B: 90, A: 255},
		tileOff:      color.RGBA{R: 28, G: 32, B: 40, A: 255},
		tileBorder:   color.RGBA{R: 48, G: 56, B: 68, A: 255},
		fontFace:     face,
		lineHeight:   face.Metrics().Height.Ceil(),
		refreshEvery: 10 * time.Second,
	}
}

func (ui *UI) Run(ctx context.Context) error {
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		return err
	}
	defer sdl.Quit()

	window, err := sdl.CreateWindow("hmbrg", 0, 0, ui.width, ui.height, sdl.WINDOW_SHOWN|sdl.WINDOW_BORDERLESS)
	if err != nil {
		return err
	}
	defer window.Destroy()
	window.SetPosition(0, 0)

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		return err
	}
	defer renderer.Destroy()

	texture, err := renderer.CreateTexture(sdl.PIXELFORMAT_ABGR8888, sdl.TEXTUREACCESS_STREAMING, ui.width, ui.height)
	if err != nil {
		return err
	}
	defer texture.Destroy()

	accessories := []ToggleAccessory{}
	selected := 0
	ui.lastRefresh = time.Now()

	reqCh := make(chan uiRequest, 4)
	respCh := make(chan uiResponse, 4)
	go ui.worker(ctx, reqCh, respCh)
	ui.queueRefresh(reqCh, "initial refresh")

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		select {
		case resp := <-respCh:
			ui.busy = false
			if resp.err != nil {
				ui.notice = resp.err.Error()
			} else {
				if resp.notice != "" {
					ui.notice = resp.notice
				} else {
					ui.notice = ""
				}
				if resp.accessories != nil {
					accessories = resp.accessories
					ui.totalCount = resp.total
					ui.toggleCount = resp.toggleable
					selected = clampIndex(selected, len(accessories))
				}
			}
		default:
		}

		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch e := event.(type) {
			case *sdl.QuitEvent:
				return nil
			case *sdl.KeyboardEvent:
				if e.State == sdl.PRESSED {
					if e.Keysym.Sym == sdl.K_ESCAPE || e.Keysym.Sym == sdl.K_HOME {
						return nil
					}
					if e.Keysym.Sym == sdl.K_UP {
						selected = moveGrid(selected, len(accessories), -1, 0, ui.gridCols())
					}
					if e.Keysym.Sym == sdl.K_DOWN {
						selected = moveGrid(selected, len(accessories), 1, 0, ui.gridCols())
					}
					if e.Keysym.Sym == sdl.K_LEFT {
						selected = moveGrid(selected, len(accessories), 0, -1, ui.gridCols())
					}
					if e.Keysym.Sym == sdl.K_RIGHT {
						selected = moveGrid(selected, len(accessories), 0, 1, ui.gridCols())
					}
					if e.Keysym.Sym == sdl.K_RETURN || e.Keysym.Sym == sdl.K_SPACE {
						ui.queueToggle(reqCh, accessories, selected)
					}
				}
			}
		}

		if ui.shouldRefresh() {
			ui.queueRefresh(reqCh, "auto refresh")
		}

		img := ui.render(accessories, selected)
		if len(img.Pix) == 0 {
			return errors.New("render buffer is empty")
		}
		if err := texture.Update(nil, unsafe.Pointer(&img.Pix[0]), img.Stride); err != nil {
			return err
		}
		renderer.Clear()
		renderer.Copy(texture, nil, nil)
		renderer.Present()
		sdl.Delay(16)
	}
}

type uiRequest struct {
	kind     string
	uniqueID string
	desired  bool
	name     string
	reason   string
}

type uiResponse struct {
	accessories []ToggleAccessory
	notice      string
	err         error
	total       int
	toggleable  int
}

func (ui *UI) worker(ctx context.Context, reqCh <-chan uiRequest, respCh chan<- uiResponse) {
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-reqCh:
			switch req.kind {
			case "refresh":
				accs, err := ui.client.GetAccessories(ctx)
				total := len(accs)
				toggleable := 0
				for _, item := range accs {
					if item.Toggleable {
						toggleable++
					}
				}
				respCh <- uiResponse{accessories: accs, err: err, total: total, toggleable: toggleable}
			case "toggle":
				if err := ui.client.SetAccessoryOn(ctx, req.uniqueID, req.desired); err != nil {
					respCh <- uiResponse{err: err}
					continue
				}
				accs, err := ui.client.GetAccessories(ctx)
				total := len(accs)
				toggleable := 0
				for _, item := range accs {
					if item.Toggleable {
						toggleable++
					}
				}
				if err != nil {
					respCh <- uiResponse{accessories: accs, notice: "toggle succeeded, refresh failed", err: nil, total: total, toggleable: toggleable}
				} else {
					notice := "Set " + req.name + " "
					if req.desired {
						notice += "on"
					} else {
						notice += "off"
					}
					respCh <- uiResponse{accessories: accs, notice: notice, total: total, toggleable: toggleable}
				}
			}
		}
	}
}

func (ui *UI) queueRefresh(reqCh chan<- uiRequest, reason string) {
	if ui.busy {
		return
	}
	ui.busy = true
	ui.lastRefresh = time.Now()
	reqCh <- uiRequest{kind: "refresh", reason: reason}
}

func (ui *UI) queueToggle(reqCh chan<- uiRequest, accessories []ToggleAccessory, selected int) {
	if ui.busy {
		return
	}
	if len(accessories) == 0 {
		ui.notice = "no toggleable accessories"
		return
	}
	acc := accessories[selected]
	if !acc.Toggleable {
		ui.notice = "accessory is read-only"
		return
	}
	ui.busy = true
	reqCh <- uiRequest{kind: "toggle", uniqueID: acc.UniqueID, desired: !acc.On, name: acc.Name}
}

func (ui *UI) shouldRefresh() bool {
	if ui.busy {
		return false
	}
	return time.Since(ui.lastRefresh) > ui.refreshEvery
}

func (ui *UI) render(accessories []ToggleAccessory, selected int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, int(ui.width), int(ui.height)))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: ui.bgColor}, image.Point{}, draw.Src)

	padding := 16
	headerY := padding + ui.lineHeight
	summaryY := headerY + ui.lineHeight
	listTop := summaryY + ui.lineHeight/2
	header := "Homebridge"
	ui.drawText(img, padding, headerY, header, ui.textColor)
	if ui.busy {
		ui.drawText(img, int(ui.width)-140, headerY, "Loading...", ui.subtleColor)
	}
	if ui.totalCount > 0 {
		summary := "Showing " + itoa(ui.toggleCount) + " of " + itoa(ui.totalCount)
		ui.drawText(img, padding, summaryY, summary, ui.subtleColor)
	}

	if ui.notice != "" {
		ui.drawText(img, padding, int(ui.height)-6, ui.notice, ui.subtleColor)
	}

	if len(accessories) == 0 {
		ui.drawText(img, padding, listTop, "No toggleable accessories found", ui.subtleColor)
		return img
	}

	ui.renderTiles(img, accessories, selected, padding, listTop)

	return img
}

func (ui *UI) drawText(img *image.RGBA, x, y int, text string, col color.Color) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: ui.fontFace,
		Dot:  fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)},
	}
	d.DrawString(text)
}

func (ui *UI) fillRect(img *image.RGBA, x, y, w, h int, col color.Color) {
	if w <= 0 || h <= 0 {
		return
	}
	rect := image.Rect(x, y, x+w, y+h).Intersect(img.Bounds())
	if rect.Empty() {
		return
	}
	draw.Draw(img, rect, &image.Uniform{C: col}, image.Point{}, draw.Src)
}

func clampIndex(idx int, length int) int {
	if length <= 0 {
		return 0
	}
	if idx < 0 {
		return 0
	}
	if idx >= length {
		return length - 1
	}
	return idx
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	buf := [20]byte{}
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func (ui *UI) gridCols() int {
	width := int(ui.width)
	padding := 20
	gap := 16
	tileW := 220
	usable := width - padding*2 + gap
	cols := usable / (tileW + gap)
	if cols < 1 {
		cols = 1
	}
	if cols > 4 {
		cols = 4
	}
	return cols
}

func moveGrid(idx, length, dRow, dCol, cols int) int {
	if length == 0 {
		return 0
	}
	if cols < 1 {
		cols = 1
	}
	row := idx / cols
	col := idx % cols
	row += dRow
	col += dCol
	if col < 0 {
		col = 0
	}
	if col >= cols {
		col = cols - 1
	}
	if row < 0 {
		row = 0
	}
	newIdx := row*cols + col
	if newIdx >= length {
		newIdx = length - 1
	}
	return newIdx
}

func (ui *UI) renderTiles(img *image.RGBA, accessories []ToggleAccessory, selected int, padding int, listTop int) {
	height := int(ui.height)
	gap := 16
	tileW := 220
	tileH := 140

	cols := ui.gridCols()
	rows := 1
	usableH := height - listTop - 40
	if usableH > tileH {
		rows = (usableH + gap) / (tileH + gap)
		if rows < 1 {
			rows = 1
		}
	}
	tilesPerPage := cols * rows
	page := 0
	if tilesPerPage > 0 {
		page = selected / tilesPerPage
	}
	start := page * tilesPerPage
	end := start + tilesPerPage
	if end > len(accessories) {
		end = len(accessories)
	}

	pulse := 0.6 + 0.4*float64(time.Now().UnixNano()%1_000_000_000)/1_000_000_000
	highlight := scaleColor(ui.highlight, pulse)

	for i := start; i < end; i++ {
		index := i - start
		row := index / cols
		col := index % cols
		x := padding + col*(tileW+gap)
		y := listTop + row*(tileH+gap)

		acc := accessories[i]
		bg := ui.tileOff
		if acc.OnKnown && acc.On {
			bg = ui.tileOn
		}
		if !acc.Toggleable {
			bg = ui.tileOff
		}
		ui.fillRect(img, x, y, tileW, tileH, bg)
		ui.drawRect(img, x, y, tileW, tileH, ui.tileBorder)

		if i == selected {
			ui.drawRect(img, x-2, y-2, tileW+4, tileH+4, highlight)
		}

		nameColor := ui.textColor
		stateColor := ui.subtleColor
		if !acc.Toggleable {
			nameColor = ui.subtleColor
		}
		state := "off"
		if !acc.OnKnown {
			state = "n/a"
		} else if acc.On {
			state = "on"
			stateColor = ui.textColor
		}
		if !acc.Toggleable {
			state = "read-only"
		}

		nameY := y + ui.lineHeight + 8
		typeY := nameY + ui.lineHeight + 2
		stateY := y + tileH - 6
		ui.drawText(img, x+12, nameY, truncate(acc.Name, 20), nameColor)
		ui.drawText(img, x+12, typeY, acc.HumanType, ui.subtleColor)
		ui.drawText(img, x+12, stateY, state, stateColor)
	}
}

func loadUIFont(size float64) font.Face {
	tt, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return basicfont.Face7x13
	}
	face, err := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return basicfont.Face7x13
	}
	return face
}

func (ui *UI) drawRect(img *image.RGBA, x, y, w, h int, col color.Color) {
	ui.fillRect(img, x, y, w, 2, col)
	ui.fillRect(img, x, y+h-2, w, 2, col)
	ui.fillRect(img, x, y, 2, h, col)
	ui.fillRect(img, x+w-2, y, 2, h, col)
}

func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func scaleColor(c color.RGBA, scale float64) color.RGBA {
	if scale < 0.2 {
		scale = 0.2
	}
	if scale > 1.0 {
		scale = 1.0
	}
	return color.RGBA{
		R: uint8(float64(c.R) * scale),
		G: uint8(float64(c.G) * scale),
		B: uint8(float64(c.B) * scale),
		A: c.A,
	}
}
