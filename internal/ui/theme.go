package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// ThemeMode selects between macOS Light, macOS Dark, and AMOLED Pure Black palettes.
type ThemeMode string

const (
	ThemeModeLight  ThemeMode = "light"
	ThemeModeDark   ThemeMode = "dark"
	ThemeModeAMOLED ThemeMode = "amoled"
)

// LuminaTheme implements fyne.Theme with an Apple macOS/iOS-inspired palette.
type LuminaTheme struct {
	mode ThemeMode
}

// NewLuminaTheme creates a LuminaTheme for the requested mode.
func NewLuminaTheme(mode ThemeMode) *LuminaTheme {
	switch mode {
	case ThemeModeDark, "slate":
		return &LuminaTheme{mode: ThemeModeDark}
	case ThemeModeAMOLED:
		return &LuminaTheme{mode: ThemeModeAMOLED}
	default:
		return &LuminaTheme{mode: ThemeModeLight}
	}
}

func (t *LuminaTheme) Mode() ThemeMode {
	return t.mode
}

func (t *LuminaTheme) IsLight() bool {
	return t.mode == ThemeModeLight
}

func (t *LuminaTheme) BackgroundColor() color.Color {
	switch t.mode {
	case ThemeModeAMOLED:
		return color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	case ThemeModeDark:
		return color.NRGBA{R: 22, G: 22, B: 24, A: 255} // #161618 macOS dark canvas
	default:
		return color.NRGBA{R: 245, G: 245, B: 247, A: 255} // #F5F5F7 Apple warm light gray
	}
}

func (t *LuminaTheme) CardColor() color.Color {
	switch t.mode {
	case ThemeModeAMOLED:
		return color.NRGBA{R: 17, G: 17, B: 19, A: 255} // #111113
	case ThemeModeDark:
		return color.NRGBA{R: 34, G: 34, B: 37, A: 255} // #222225
	default:
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255} // #FFFFFF
	}
}

func (t *LuminaTheme) CardBorderColor() color.Color {
	switch t.mode {
	case ThemeModeAMOLED:
		return color.NRGBA{R: 38, G: 38, B: 42, A: 255} // #26262A
	case ThemeModeDark:
		return color.NRGBA{R: 54, G: 54, B: 58, A: 255} // #36363A
	default:
		return color.NRGBA{R: 229, G: 229, B: 234, A: 255} // #E5E5EA
	}
}

func (t *LuminaTheme) WellColor() color.Color {
	switch t.mode {
	case ThemeModeAMOLED:
		return color.NRGBA{R: 26, G: 26, B: 29, A: 255}
	case ThemeModeDark:
		return color.NRGBA{R: 44, G: 44, B: 48, A: 255} // #2C2C30
	default:
		return color.NRGBA{R: 242, G: 242, B: 247, A: 255} // #F2F2F7
	}
}

func (t *LuminaTheme) WellHoverColor() color.Color {
	switch t.mode {
	case ThemeModeAMOLED:
		return color.NRGBA{R: 38, G: 38, B: 43, A: 255}
	case ThemeModeDark:
		return color.NRGBA{R: 58, G: 58, B: 63, A: 255}
	default:
		return color.NRGBA{R: 229, G: 229, B: 234, A: 255}
	}
}

func (t *LuminaTheme) DividerColor() color.Color {
	switch t.mode {
	case ThemeModeAMOLED:
		return color.NRGBA{R: 36, G: 36, B: 40, A: 255}
	case ThemeModeDark:
		return color.NRGBA{R: 52, G: 52, B: 56, A: 255}
	default:
		return color.NRGBA{R: 232, G: 232, B: 237, A: 255}
	}
}

func (t *LuminaTheme) PrimaryTextColor() color.Color {
	if t.IsLight() {
		return color.NRGBA{R: 29, G: 29, B: 31, A: 255} // #1D1D1F
	}
	return color.NRGBA{R: 245, G: 245, B: 247, A: 255} // #F5F5F7
}

func (t *LuminaTheme) MutedTextColor() color.Color {
	if t.IsLight() {
		return color.NRGBA{R: 110, G: 110, B: 115, A: 255} // #6E6E73
	}
	return color.NRGBA{R: 152, G: 152, B: 157, A: 255} // #98989D
}

func (t *LuminaTheme) AccentColor() color.Color {
	if t.IsLight() {
		return color.NRGBA{R: 0, G: 122, B: 255, A: 255} // #007AFF Apple Blue
	}
	return color.NRGBA{R: 10, G: 132, B: 255, A: 255} // #0A84FF Apple Dark Blue
}

func (t *LuminaTheme) AccentHoverColor() color.Color {
	if t.IsLight() {
		return color.NRGBA{R: 0, G: 102, B: 219, A: 255}
	}
	return color.NRGBA{R: 45, G: 150, B: 255, A: 255}
}

func (t *LuminaTheme) AccentTintColor() color.Color {
	if t.IsLight() {
		return color.NRGBA{R: 229, G: 241, B: 255, A: 255}
	}
	return color.NRGBA{R: 20, G: 44, B: 74, A: 255}
}

func (t *LuminaTheme) SuccessColor() color.Color {
	if t.IsLight() {
		return color.NRGBA{R: 26, G: 143, B: 62, A: 255}
	}
	return color.NRGBA{R: 48, G: 209, B: 88, A: 255} // #30D158
}

func (t *LuminaTheme) SuccessTintColor() color.Color {
	if t.IsLight() {
		return color.NRGBA{R: 230, G: 248, B: 235, A: 255}
	}
	return color.NRGBA{R: 20, G: 51, B: 30, A: 255}
}

func (t *LuminaTheme) WarningColor() color.Color {
	if t.IsLight() {
		return color.NRGBA{R: 196, G: 104, B: 0, A: 255}
	}
	return color.NRGBA{R: 255, G: 159, B: 10, A: 255} // #FF9F0A
}

func (t *LuminaTheme) WarningTintColor() color.Color {
	if t.IsLight() {
		return color.NRGBA{R: 255, G: 244, B: 224, A: 255}
	}
	return color.NRGBA{R: 56, G: 38, B: 14, A: 255}
}

func (t *LuminaTheme) ErrorColor() color.Color {
	if t.IsLight() {
		return color.NRGBA{R: 215, G: 42, B: 34, A: 255}
	}
	return color.NRGBA{R: 255, G: 69, B: 58, A: 255} // #FF453A
}

func (t *LuminaTheme) ErrorTintColor() color.Color {
	if t.IsLight() {
		return color.NRGBA{R: 255, G: 233, B: 232, A: 255}
	}
	return color.NRGBA{R: 61, G: 24, B: 24, A: 255}
}

func (t *LuminaTheme) IndigoColor() color.Color {
	if t.IsLight() {
		return color.NRGBA{R: 88, G: 86, B: 214, A: 255} // #5856D6
	}
	return color.NRGBA{R: 94, G: 92, B: 230, A: 255} // #5E5CE6
}

func (t *LuminaTheme) IndigoTintColor() color.Color {
	if t.IsLight() {
		return color.NRGBA{R: 236, G: 236, B: 252, A: 255}
	}
	return color.NRGBA{R: 34, G: 33, B: 66, A: 255}
}

func (t *LuminaTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return t.BackgroundColor()
	case theme.ColorNameInputBackground:
		return t.WellColor()
	case theme.ColorNameInputBorder:
		return t.CardBorderColor()
	case theme.ColorNameOverlayBackground, theme.ColorNameMenuBackground:
		return t.CardColor()
	case theme.ColorNameSeparator:
		return t.DividerColor()
	case theme.ColorNameButton:
		return t.WellColor()
	case theme.ColorNameHover:
		return t.WellHoverColor()
	case theme.ColorNamePressed:
		return t.CardBorderColor()
	case theme.ColorNameDisabledButton:
		if t.IsLight() {
			return color.NRGBA{R: 235, G: 235, B: 240, A: 255}
		}
		return color.NRGBA{R: 38, G: 38, B: 42, A: 255}
	case theme.ColorNameDisabled:
		if t.IsLight() {
			return color.NRGBA{R: 174, G: 174, B: 178, A: 255}
		}
		return color.NRGBA{R: 99, G: 99, B: 102, A: 255}
	case theme.ColorNamePrimary, theme.ColorNameHyperlink:
		return t.AccentColor()
	case theme.ColorNameForegroundOnPrimary:
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	case theme.ColorNameFocus:
		c := t.AccentColor().(color.NRGBA)
		return color.NRGBA{R: c.R, G: c.G, B: c.B, A: 90}
	case theme.ColorNameSelection:
		return t.AccentTintColor()
	case theme.ColorNameForeground:
		return t.PrimaryTextColor()
	case theme.ColorNamePlaceHolder:
		return t.MutedTextColor()
	case theme.ColorNameError:
		return t.ErrorColor()
	case theme.ColorNameSuccess:
		return t.SuccessColor()
	case theme.ColorNameWarning:
		return t.WarningColor()
	case theme.ColorNameScrollBar:
		return t.CardBorderColor()
	case theme.ColorNameScrollBarBackground:
		return color.Transparent
	case theme.ColorNameShadow:
		if t.IsLight() {
			return color.NRGBA{R: 0, G: 0, B: 0, A: 22}
		}
		return color.NRGBA{R: 0, G: 0, B: 0, A: 90}
	}

	if t.IsLight() {
		return theme.DefaultTheme().Color(name, theme.VariantLight)
	}
	return theme.DefaultTheme().Color(name, theme.VariantDark)
}

func (t *LuminaTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *LuminaTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *LuminaTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 13.5
	case theme.SizeNameCaptionText:
		return 11.5
	case theme.SizeNameSubHeadingText:
		return 15
	case theme.SizeNameHeadingText:
		return 20
	case theme.SizeNameButtonRadius:
		return 10
	case theme.SizeNameInputRadius:
		return 10
	case theme.SizeNameCardRadius:
		return 16
	case theme.SizeNameDialogRadius:
		return 16
	case theme.SizeNamePopupRadius:
		return 12
	case theme.SizeNameMenuRadius:
		return 10
	case theme.SizeNameSelectionRadius:
		return 8
	case theme.SizeNameInputBorder:
		return 1
	case theme.SizeNamePadding:
		return 8
	case theme.SizeNameInnerPadding:
		return 10
	case theme.SizeNameScrollBarRadius:
		return 4
	}
	return theme.DefaultTheme().Size(name)
}
