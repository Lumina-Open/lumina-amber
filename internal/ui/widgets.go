package ui

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func colorHex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}

func tintedSVG(base fyne.Resource, c color.Color) fyne.Resource {
	if base == nil {
		return nil
	}
	hex := colorHex(c)
	raw := string(base.Content())
	colored := strings.ReplaceAll(raw, "<path ", fmt.Sprintf(`<path fill="%s" `, hex))
	return fyne.NewStaticResource(fmt.Sprintf("%s_%s.svg", base.Name(), strings.TrimPrefix(hex, "#")), []byte(colored))
}

func darkenOrLighten(c color.Color, lightMode bool) color.Color {
	r, g, b, a := c.RGBA()
	nr, ng, nb, na := uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8)
	delta := 14
	if !lightMode {
		if int(nr)+delta <= 255 {
			nr += uint8(delta)
		}
		if int(ng)+delta <= 255 {
			ng += uint8(delta)
		}
		if int(nb)+delta <= 255 {
			nb += uint8(delta)
		}
		return color.NRGBA{R: nr, G: ng, B: nb, A: na}
	}
	if int(nr)-delta >= 0 {
		nr -= uint8(delta)
	}
	if int(ng)-delta >= 0 {
		ng -= uint8(delta)
	}
	if int(nb)-delta >= 0 {
		nb -= uint8(delta)
	}
	return color.NRGBA{R: nr, G: ng, B: nb, A: na}
}

// StyledText is a lightweight widget wrapping canvas.Text so dynamic text updates
// always trigger clean parent container relayout without overlap or clipping.
type StyledText struct {
	widget.BaseWidget
	text  string
	col   color.Color
	size  float32
	style fyne.TextStyle
	obj   *canvas.Text
}

func newStyledText(text string, col color.Color, size float32, bold bool) *StyledText {
	st := &StyledText{
		text: text,
		col:  col,
		size: size,
	}
	if bold {
		st.style = fyne.TextStyle{Bold: true}
	}
	st.ExtendBaseWidget(st)
	return st
}

func (s *StyledText) SetText(text string) {
	if s.text == text {
		return
	}
	s.text = text
	if s.obj != nil {
		s.obj.Text = text
		s.obj.Refresh()
	}
	s.Refresh()
}

func (s *StyledText) SetColor(col color.Color) {
	s.col = col
	if s.obj != nil {
		s.obj.Color = col
		s.obj.Refresh()
	}
}

func (s *StyledText) CreateRenderer() fyne.WidgetRenderer {
	s.obj = canvas.NewText(s.text, s.col)
	s.obj.TextSize = s.size
	s.obj.TextStyle = s.style
	return &styledTextRenderer{s: s}
}

type styledTextRenderer struct {
	s *StyledText
}

func (r *styledTextRenderer) Layout(size fyne.Size) {
	r.s.obj.Resize(size)
	r.s.obj.Move(fyne.NewPos(0, 0))
}

func (r *styledTextRenderer) MinSize() fyne.Size {
	r.s.obj.Text = r.s.text
	r.s.obj.TextSize = r.s.size
	r.s.obj.TextStyle = r.s.style
	return fyne.MeasureText(r.s.text, r.s.size, r.s.style)
}

func (r *styledTextRenderer) Refresh() {
	r.s.obj.Text = r.s.text
	r.s.obj.Color = r.s.col
	r.s.obj.TextSize = r.s.size
	r.s.obj.TextStyle = r.s.style
	r.s.obj.Refresh()
}

func (r *styledTextRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.s.obj}
}

func (r *styledTextRenderer) Destroy() {}

// ElevatedCard wraps content in an Apple-style rounded card with a crisp 1px hairline border.
type ElevatedCard struct {
	Root fyne.CanvasObject
	bg   *canvas.Rectangle
}

func newElevatedCard(t *LuminaTheme, content fyne.CanvasObject) *ElevatedCard {
	bg := canvas.NewRectangle(t.CardColor())
	bg.CornerRadius = 16
	bg.StrokeColor = t.CardBorderColor()
	bg.StrokeWidth = 1

	padded := container.New(layout.NewCustomPaddedLayout(16, 16, 20, 20), content)
	return &ElevatedCard{
		Root: container.NewStack(bg, padded),
		bg:   bg,
	}
}

func (c *ElevatedCard) ApplyTheme(t *LuminaTheme) {
	c.bg.FillColor = t.CardColor()
	c.bg.StrokeColor = t.CardBorderColor()
	c.bg.Refresh()
}

// HairlineDivider renders a subtle 1px horizontal divider inside grouped cards.
type HairlineDivider struct {
	Root fyne.CanvasObject
	line *canvas.Rectangle
}

func newHairlineDivider(t *LuminaTheme, leftInset float32) *HairlineDivider {
	line := canvas.NewRectangle(t.DividerColor())
	line.SetMinSize(fyne.NewSize(0, 1))
	root := container.New(layout.NewCustomPaddedLayout(4, 4, leftInset, 0), line)
	return &HairlineDivider{Root: root, line: line}
}

func (d *HairlineDivider) ApplyTheme(t *LuminaTheme) {
	d.line.FillColor = t.DividerColor()
	d.line.Refresh()
}

// IconTile renders a squircle icon badge (like iOS Settings / macOS System Settings).
type IconTile struct {
	Root    fyne.CanvasObject
	bg      *canvas.Rectangle
	icon    *canvas.Image
	rawRes  fyne.Resource
	fgFn    func(*LuminaTheme) color.Color
	bgFn    func(*LuminaTheme) color.Color
	size    float32
	iconDim float32
}

func newIconTile(t *LuminaTheme, res fyne.Resource, size, iconDim, radius float32, bgFn, fgFn func(*LuminaTheme) color.Color) *IconTile {
	bg := canvas.NewRectangle(bgFn(t))
	bg.CornerRadius = radius
	bg.SetMinSize(fyne.NewSize(size, size))

	img := canvas.NewImageFromResource(tintedSVG(res, fgFn(t)))
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(iconDim, iconDim))

	root := container.NewStack(bg, container.NewCenter(img))
	return &IconTile{
		Root:    root,
		bg:      bg,
		icon:    img,
		rawRes:  res,
		fgFn:    fgFn,
		bgFn:    bgFn,
		size:    size,
		iconDim: iconDim,
	}
}

func (it *IconTile) SetIcon(t *LuminaTheme, res fyne.Resource) {
	it.rawRes = res
	it.icon.Resource = tintedSVG(res, it.fgFn(t))
	it.icon.Refresh()
}

func (it *IconTile) ApplyTheme(t *LuminaTheme) {
	it.bg.FillColor = it.bgFn(t)
	it.bg.Refresh()
	it.icon.Resource = tintedSVG(it.rawRes, it.fgFn(t))
	it.icon.Refresh()
}

// SoftPill renders an iOS-style tinted status capsule with a colored dot.
type SoftPill struct {
	Root  fyne.CanvasObject
	bg    *canvas.Rectangle
	dot   *canvas.Rectangle
	label *StyledText
}

func newSoftPill(text string, bgColor, fgColor color.Color) *SoftPill {
	bg := canvas.NewRectangle(bgColor)
	bg.CornerRadius = 11

	dot := canvas.NewRectangle(fgColor)
	dot.CornerRadius = 3.5
	dot.SetMinSize(fyne.NewSize(7, 7))

	lbl := newStyledText(text, fgColor, 11.5, true)

	inner := container.NewHBox(container.NewCenter(dot), lbl)
	padded := container.New(layout.NewCustomPaddedLayout(3, 3, 9, 12), inner)
	return &SoftPill{
		Root:  container.NewStack(bg, padded),
		bg:    bg,
		dot:   dot,
		label: lbl,
	}
}

func (p *SoftPill) Set(text string, bgColor, fgColor color.Color) {
	p.label.SetColor(fgColor)
	p.label.SetText(text)
	p.dot.FillColor = fgColor
	p.bg.FillColor = bgColor
	p.dot.Refresh()
	p.bg.Refresh()
}

// PillStyle selects the visual appearance of a PillButton.
type PillStyle int

const (
	PillPrimary PillStyle = iota
	PillTinted
	PillSecondary
	PillDanger
)

// PillButton is a modern macOS/iOS-style rounded button with hover and icon support.
type PillButton struct {
	widget.BaseWidget
	theme    *LuminaTheme
	style    PillStyle
	text     string
	iconRes  fyne.Resource
	trailing fyne.Resource
	minH     float32
	radius   float32
	padH     float32
	padV     float32
	bold     bool
	disabled bool
	hovered  bool
	onTapped func()

	bg      *canvas.Rectangle
	lbl     *canvas.Text
	iconImg *canvas.Image
	trImg   *canvas.Image
}

func newPillButton(t *LuminaTheme, text string, icon fyne.Resource, style PillStyle, onTapped func()) *PillButton {
	b := &PillButton{
		theme:    t,
		style:    style,
		text:     text,
		iconRes:  icon,
		minH:     34,
		radius:   10,
		padH:     14,
		padV:     7,
		bold:     style == PillPrimary,
		onTapped: onTapped,
	}
	b.ExtendBaseWidget(b)
	return b
}

func (b *PillButton) SetHero(height float32) {
	b.minH = height
	b.radius = 12
	b.padH = 22
	b.padV = 10
	b.bold = true
	b.Refresh()
}

func (b *PillButton) SetTrailingIcon(res fyne.Resource) {
	b.trailing = res
	b.Refresh()
}

func (b *PillButton) SetText(text string) {
	b.text = text
	if b.lbl != nil {
		b.lbl.Text = text
		b.lbl.Refresh()
	}
	b.Refresh()
}

func (b *PillButton) SetDisabled(disabled bool) {
	if b.disabled == disabled {
		return
	}
	b.disabled = disabled
	b.Refresh()
}

func (b *PillButton) ApplyTheme(t *LuminaTheme) {
	b.theme = t
	b.Refresh()
}

func (b *PillButton) Tapped(*fyne.PointEvent) {
	if b.disabled || b.onTapped == nil {
		return
	}
	b.onTapped()
}

func (b *PillButton) MouseIn(*desktop.MouseEvent) {
	if b.disabled {
		return
	}
	b.hovered = true
	b.syncColors()
}

func (b *PillButton) MouseMoved(*desktop.MouseEvent) {}

func (b *PillButton) MouseOut() {
	b.hovered = false
	b.syncColors()
}

func (b *PillButton) Cursor() desktop.Cursor {
	if b.disabled {
		return desktop.DefaultCursor
	}
	return desktop.PointerCursor
}

func (b *PillButton) resolveColors() (bg, fg color.Color) {
	t := b.theme
	if b.disabled {
		if t.IsLight() {
			return color.NRGBA{R: 235, G: 235, B: 240, A: 255}, color.NRGBA{R: 165, G: 165, B: 172, A: 255}
		}
		return color.NRGBA{R: 38, G: 38, B: 42, A: 255}, color.NRGBA{R: 99, G: 99, B: 102, A: 255}
	}

	switch b.style {
	case PillPrimary:
		bg = t.AccentColor()
		if b.hovered {
			bg = t.AccentHoverColor()
		}
		fg = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	case PillTinted:
		bg = t.AccentTintColor()
		if b.hovered {
			bg = darkenOrLighten(bg, t.IsLight())
		}
		fg = t.AccentColor()
	case PillDanger:
		bg = t.ErrorTintColor()
		if b.hovered {
			bg = darkenOrLighten(bg, t.IsLight())
		}
		fg = t.ErrorColor()
	default: // PillSecondary
		bg = t.WellColor()
		if b.hovered {
			bg = t.WellHoverColor()
		}
		fg = t.PrimaryTextColor()
	}
	return bg, fg
}

func (b *PillButton) syncColors() {
	if b.bg == nil || b.lbl == nil {
		return
	}
	bg, fg := b.resolveColors()
	b.bg.FillColor = bg
	b.bg.CornerRadius = b.radius
	b.bg.Refresh()

	b.lbl.Color = fg
	b.lbl.Text = b.text
	b.lbl.TextStyle = fyne.TextStyle{Bold: b.bold}
	b.lbl.Refresh()

	if b.iconImg != nil && b.iconRes != nil {
		b.iconImg.Resource = tintedSVG(b.iconRes, fg)
		b.iconImg.Refresh()
	}
	if b.trImg != nil && b.trailing != nil {
		b.trImg.Resource = tintedSVG(b.trailing, fg)
		b.trImg.Refresh()
	}
}

func (b *PillButton) CreateRenderer() fyne.WidgetRenderer {
	bgCol, fgCol := b.resolveColors()
	b.bg = canvas.NewRectangle(bgCol)
	b.bg.CornerRadius = b.radius

	b.lbl = canvas.NewText(b.text, fgCol)
	b.lbl.TextSize = 13
	if b.minH >= 40 {
		b.lbl.TextSize = 14.5
	}
	b.lbl.TextStyle = fyne.TextStyle{Bold: b.bold}

	row := container.NewHBox()
	if b.iconRes != nil {
		b.iconImg = canvas.NewImageFromResource(tintedSVG(b.iconRes, fgCol))
		b.iconImg.FillMode = canvas.ImageFillContain
		dim := float32(16)
		if b.minH >= 40 {
			dim = 18
		}
		b.iconImg.SetMinSize(fyne.NewSize(dim, dim))
		row.Add(container.NewCenter(b.iconImg))
	}
	row.Add(container.NewCenter(b.lbl))
	if b.trailing != nil {
		b.trImg = canvas.NewImageFromResource(tintedSVG(b.trailing, fgCol))
		b.trImg.FillMode = canvas.ImageFillContain
		b.trImg.SetMinSize(fyne.NewSize(14, 14))
		row.Add(container.NewCenter(b.trImg))
	}

	padded := container.New(layout.NewCustomPaddedLayout(b.padV, b.padV, b.padH, b.padH), container.NewCenter(row))
	root := container.NewStack(b.bg, padded)
	return &pillButtonRenderer{b: b, root: root}
}

type pillButtonRenderer struct {
	b    *PillButton
	root fyne.CanvasObject
}

func (r *pillButtonRenderer) Layout(size fyne.Size) {
	r.root.Resize(size)
}

func (r *pillButtonRenderer) MinSize() fyne.Size {
	ms := r.root.MinSize()
	if ms.Height < r.b.minH {
		ms.Height = r.b.minH
	}
	return ms
}

func (r *pillButtonRenderer) Refresh() {
	r.b.syncColors()
	r.root.Refresh()
}

func (r *pillButtonRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.root}
}

func (r *pillButtonRenderer) Destroy() {}

// SegmentedControl implements an Apple-style horizontal segmented selector.
type SegmentedControl struct {
	theme    *LuminaTheme
	options  []string
	selected int
	onChange func(index int)

	trackBg *canvas.Rectangle
	items   []*segmentItem
	Root    fyne.CanvasObject
}

type segmentItem struct {
	widget.BaseWidget
	parent  *SegmentedControl
	index   int
	title   string
	bg      *canvas.Rectangle
	lbl     *canvas.Text
	hovered bool
}

func newSegmentedControl(t *LuminaTheme, options []string, initial int, onChange func(int)) *SegmentedControl {
	sc := &SegmentedControl{
		theme:    t,
		options:  options,
		selected: initial,
		onChange: onChange,
	}
	sc.trackBg = canvas.NewRectangle(t.WellColor())
	sc.trackBg.CornerRadius = 9

	row := container.NewGridWithColumns(len(options))
	for i, opt := range options {
		it := &segmentItem{
			parent: sc,
			index:  i,
			title:  opt,
		}
		it.ExtendBaseWidget(it)
		sc.items = append(sc.items, it)
		row.Add(it)
	}

	padded := container.New(layout.NewCustomPaddedLayout(3, 3, 3, 3), row)
	sc.Root = container.NewStack(sc.trackBg, padded)
	return sc
}

func (sc *SegmentedControl) SetSelected(idx int) {
	if idx < 0 || idx >= len(sc.options) {
		return
	}
	sc.selected = idx
	for _, it := range sc.items {
		it.syncColors()
	}
}

func (sc *SegmentedControl) ApplyTheme(t *LuminaTheme) {
	sc.theme = t
	sc.trackBg.FillColor = t.WellColor()
	sc.trackBg.Refresh()
	for _, it := range sc.items {
		it.syncColors()
	}
}

func (s *segmentItem) Tapped(*fyne.PointEvent) {
	if s.parent.selected == s.index {
		return
	}
	s.parent.SetSelected(s.index)
	if s.parent.onChange != nil {
		s.parent.onChange(s.index)
	}
}

func (s *segmentItem) MouseIn(*desktop.MouseEvent) {
	s.hovered = true
	s.syncColors()
}

func (s *segmentItem) MouseMoved(*desktop.MouseEvent) {}

func (s *segmentItem) MouseOut() {
	s.hovered = false
	s.syncColors()
}

func (s *segmentItem) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func (s *segmentItem) syncColors() {
	if s.bg == nil || s.lbl == nil {
		return
	}
	t := s.parent.theme
	active := s.parent.selected == s.index
	if active {
		s.bg.FillColor = t.CardColor()
		s.lbl.Color = t.PrimaryTextColor()
		s.lbl.TextStyle = fyne.TextStyle{Bold: true}
	} else {
		if s.hovered {
			s.bg.FillColor = t.WellHoverColor()
		} else {
			s.bg.FillColor = color.Transparent
		}
		s.lbl.Color = t.MutedTextColor()
		s.lbl.TextStyle = fyne.TextStyle{}
	}
	s.bg.Refresh()
	s.lbl.Refresh()
}

func (s *segmentItem) CreateRenderer() fyne.WidgetRenderer {
	s.bg = canvas.NewRectangle(color.Transparent)
	s.bg.CornerRadius = 7
	s.lbl = canvas.NewText(s.title, s.parent.theme.PrimaryTextColor())
	s.lbl.TextSize = 12
	s.syncColors()

	padded := container.New(layout.NewCustomPaddedLayout(4, 4, 12, 12), container.NewCenter(s.lbl))
	root := container.NewStack(s.bg, padded)
	return widget.NewSimpleRenderer(root)
}

// FilterChip is a pill toggle button used for Photo / Video media filters.
type FilterChip struct {
	widget.BaseWidget
	theme    *LuminaTheme
	label    string
	iconRes  fyne.Resource
	checked  bool
	hovered  bool
	onChange func(bool)

	bg      *canvas.Rectangle
	chkImg  *canvas.Image
	iconImg *canvas.Image
	lbl     *canvas.Text
}

func newFilterChip(t *LuminaTheme, label string, icon fyne.Resource, checked bool, onChange func(bool)) *FilterChip {
	fc := &FilterChip{
		theme:    t,
		label:    label,
		iconRes:  icon,
		checked:  checked,
		onChange: onChange,
	}
	fc.ExtendBaseWidget(fc)
	return fc
}

func (fc *FilterChip) SetChecked(v bool) {
	fc.checked = v
	fc.syncColors()
}

func (fc *FilterChip) ApplyTheme(t *LuminaTheme) {
	fc.theme = t
	fc.syncColors()
}

func (fc *FilterChip) Tapped(*fyne.PointEvent) {
	fc.checked = !fc.checked
	fc.syncColors()
	if fc.onChange != nil {
		fc.onChange(fc.checked)
	}
}

func (fc *FilterChip) MouseIn(*desktop.MouseEvent) {
	fc.hovered = true
	fc.syncColors()
}

func (fc *FilterChip) MouseMoved(*desktop.MouseEvent) {}

func (fc *FilterChip) MouseOut() {
	fc.hovered = false
	fc.syncColors()
}

func (fc *FilterChip) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func (fc *FilterChip) syncColors() {
	if fc.bg == nil || fc.lbl == nil {
		return
	}
	t := fc.theme
	var bgCol, fgCol, borderCol color.Color
	if fc.checked {
		bgCol = t.AccentTintColor()
		fgCol = t.AccentColor()
		c := color.NRGBAModel.Convert(fgCol).(color.NRGBA)
		borderCol = color.NRGBA{R: c.R, G: c.G, B: c.B, A: 65}
		if fc.chkImg != nil {
			fc.chkImg.Show()
			fc.chkImg.Resource = tintedSVG(resIconCheck, fgCol)
			fc.chkImg.Refresh()
		}
	} else {
		bgCol = t.WellColor()
		if fc.hovered {
			bgCol = t.WellHoverColor()
		}
		fgCol = t.MutedTextColor()
		borderCol = color.Transparent
		if fc.chkImg != nil {
			fc.chkImg.Hide()
		}
	}
	fc.bg.FillColor = bgCol
	fc.bg.StrokeColor = borderCol
	fc.bg.Refresh()

	fc.lbl.Color = fgCol
	fc.lbl.TextStyle = fyne.TextStyle{Bold: fc.checked}
	fc.lbl.Refresh()

	if fc.iconImg != nil {
		fc.iconImg.Resource = tintedSVG(fc.iconRes, fgCol)
		fc.iconImg.Refresh()
	}
}

func (fc *FilterChip) CreateRenderer() fyne.WidgetRenderer {
	fc.bg = canvas.NewRectangle(fc.theme.WellColor())
	fc.bg.CornerRadius = 9
	fc.bg.StrokeWidth = 1

	fc.chkImg = canvas.NewImageFromResource(tintedSVG(resIconCheck, fc.theme.AccentColor()))
	fc.chkImg.FillMode = canvas.ImageFillContain
	fc.chkImg.SetMinSize(fyne.NewSize(13, 13))

	fc.iconImg = canvas.NewImageFromResource(tintedSVG(fc.iconRes, fc.theme.AccentColor()))
	fc.iconImg.FillMode = canvas.ImageFillContain
	fc.iconImg.SetMinSize(fyne.NewSize(14, 14))

	fc.lbl = canvas.NewText(fc.label, fc.theme.PrimaryTextColor())
	fc.lbl.TextSize = 12

	fc.syncColors()

	row := container.NewHBox(container.NewCenter(fc.iconImg), container.NewCenter(fc.lbl), container.NewCenter(fc.chkImg))
	padded := container.New(layout.NewCustomPaddedLayout(5, 5, 10, 10), row)
	return widget.NewSimpleRenderer(container.NewStack(fc.bg, padded))
}

// AppleProgressBar renders a sleek 8px capsule progress bar.
type AppleProgressBar struct {
	widget.BaseWidget
	theme *LuminaTheme
	value float64
	track *canvas.Rectangle
	fill  *canvas.Rectangle
}

func newAppleProgressBar(t *LuminaTheme) *AppleProgressBar {
	p := &AppleProgressBar{theme: t}
	p.ExtendBaseWidget(p)
	return p
}

func (p *AppleProgressBar) SetValue(v float64) {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	p.value = v
	p.Refresh()
}

func (p *AppleProgressBar) ApplyTheme(t *LuminaTheme) {
	p.theme = t
	p.Refresh()
}

func (p *AppleProgressBar) CreateRenderer() fyne.WidgetRenderer {
	p.track = canvas.NewRectangle(p.theme.WellColor())
	p.track.CornerRadius = 4

	p.fill = canvas.NewRectangle(p.theme.AccentColor())
	p.fill.CornerRadius = 4

	return &appleProgressRenderer{p: p}
}

type appleProgressRenderer struct {
	p *AppleProgressBar
}

func (r *appleProgressRenderer) Layout(size fyne.Size) {
	r.p.track.Resize(size)
	r.p.track.Move(fyne.NewPos(0, 0))
	fillW := float32(r.p.value) * size.Width
	if r.p.value > 0 && fillW < 8 {
		fillW = 8
	}
	r.p.fill.Resize(fyne.NewSize(fillW, size.Height))
	r.p.fill.Move(fyne.NewPos(0, 0))
}

func (r *appleProgressRenderer) MinSize() fyne.Size {
	return fyne.NewSize(120, 8)
}

func (r *appleProgressRenderer) Refresh() {
	r.p.track.FillColor = r.p.theme.WellColor()
	r.p.fill.FillColor = r.p.theme.AccentColor()
	r.p.track.Refresh()
	r.p.fill.Refresh()
	r.Layout(r.p.Size())
}

func (r *appleProgressRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.p.track, r.p.fill}
}

func (r *appleProgressRenderer) Destroy() {}

// MetricTile displays a numeric counter inside a soft-tinted rounded card.
type MetricTile struct {
	Root   fyne.CanvasObject
	bg     *canvas.Rectangle
	valTxt *canvas.Text
	lblTxt *canvas.Text
	bgFn   func(*LuminaTheme) color.Color
	fgFn   func(*LuminaTheme) color.Color
}

func newMetricTile(t *LuminaTheme, title string, initial string, bgFn, fgFn func(*LuminaTheme) color.Color) *MetricTile {
	bg := canvas.NewRectangle(bgFn(t))
	bg.CornerRadius = 12

	valTxt := canvas.NewText(initial, fgFn(t))
	valTxt.TextSize = 22
	valTxt.TextStyle = fyne.TextStyle{Bold: true}

	lblTxt := canvas.NewText(title, t.MutedTextColor())
	lblTxt.TextSize = 11.5

	body := container.NewVBox(valTxt, lblTxt)
	padded := container.New(layout.NewCustomPaddedLayout(10, 10, 14, 14), body)
	return &MetricTile{
		Root:   container.NewStack(bg, padded),
		bg:     bg,
		valTxt: valTxt,
		lblTxt: lblTxt,
		bgFn:   bgFn,
		fgFn:   fgFn,
	}
}

func (m *MetricTile) SetValue(v string) {
	m.valTxt.Text = v
	m.valTxt.Refresh()
}

func (m *MetricTile) ApplyTheme(t *LuminaTheme) {
	m.bg.FillColor = m.bgFn(t)
	m.bg.Refresh()
	m.valTxt.Color = m.fgFn(t)
	m.valTxt.Refresh()
	m.lblTxt.Color = t.MutedTextColor()
	m.lblTxt.Refresh()
}
