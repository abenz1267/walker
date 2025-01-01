package config

import (
	_ "embed"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/abenz1267/walker/internal/util"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml/v2"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
)

//go:embed layout.default.toml
var defaultLayout []byte

//go:embed themes/default.toml
var defaultThemeLayout []byte

type UICfg struct {
	UI UI `koanf:"ui"`
}

type UI struct {
	Anchors         Anchors `koanf:"anchors"`
	Fullscreen      bool    `koanf:"fullscreen"`
	IgnoreExclusive bool    `koanf:"ignore_exclusive"`
	Window          Window  `koanf:"window"`

	// internal
	AlignMap        map[string]gtk.Align         `koanf:"-"`
	IconSizeMap     map[string]gtk.IconSize      `koanf:"-"`
	IconSizeIntMap  map[string]int               `koanf:"-"`
	JustifyMap      map[string]gtk.Justification `koanf:"-"`
	OrientationMap  map[string]gtk.Orientation   `koanf:"-"`
	ScrollPolicyMap map[string]gtk.PolicyType    `koanf:"-"`
}

func (u *UI) InitUnitMaps() {
	u.AlignMap = make(map[string]gtk.Align)
	u.AlignMap["fill"] = gtk.AlignFill
	u.AlignMap["start"] = gtk.AlignStart
	u.AlignMap["end"] = gtk.AlignEnd
	u.AlignMap["center"] = gtk.AlignCenter
	u.AlignMap["baseline"] = gtk.AlignBaseline
	u.AlignMap["baseline_fill"] = gtk.AlignBaselineFill
	u.AlignMap["baseline_center"] = gtk.AlignBaselineCenter

	u.IconSizeMap = make(map[string]gtk.IconSize)
	u.IconSizeMap["inherit"] = gtk.IconSizeInherit
	u.IconSizeMap["normal"] = gtk.IconSizeNormal
	u.IconSizeMap["large"] = gtk.IconSizeLarge

	u.IconSizeIntMap = make(map[string]int)
	u.IconSizeIntMap["inherit"] = -1
	u.IconSizeIntMap["normal"] = 16
	u.IconSizeIntMap["large"] = 32
	u.IconSizeIntMap["larger"] = 64
	u.IconSizeIntMap["largest"] = 128

	u.JustifyMap = make(map[string]gtk.Justification)
	u.JustifyMap["left"] = gtk.JustifyLeft
	u.JustifyMap["right"] = gtk.JustifyRight
	u.JustifyMap["center"] = gtk.JustifyCenter
	u.JustifyMap["fill"] = gtk.JustifyFill

	u.OrientationMap = make(map[string]gtk.Orientation)
	u.OrientationMap["horizontal"] = gtk.OrientationHorizontal
	u.OrientationMap["vertical"] = gtk.OrientationVertical

	u.ScrollPolicyMap = make(map[string]gtk.PolicyType)
	u.ScrollPolicyMap["never"] = gtk.PolicyNever
	u.ScrollPolicyMap["always"] = gtk.PolicyAlways
	u.ScrollPolicyMap["automatic"] = gtk.PolicyAutomatic
	u.ScrollPolicyMap["external"] = gtk.PolicyExternal
}

type Widget struct {
	CssClasses []string `koanf:"css_classes"`
	HAlign     string   `koanf:"h_align"`
	HExpand    bool     `koanf:"h_expand"`
	Height     int      `koanf:"height"`
	Hide       bool     `koanf:"hide"`
	Margins    Margins  `koanf:"margins"`
	Name       string   `koanf:"name"`
	Opacity    float64  `koanf:"opacity"`
	VAlign     string   `koanf:"v_align"`
	VExpand    bool     `koanf:"h_expand"`
	Width      int      `koanf:"width"`
}

type BoxWidget struct {
	Widget      `koanf:",squash"`
	Orientation string `koanf:"orientation"`
	Spacing     int    `koanf:"spacing"`
}

type LabelWidget struct {
	Widget  `koanf:",squash"`
	Justify string  `koanf:"justify"`
	XAlign  float32 `koanf:"x_align"`
	YAlign  float32 `koanf:"y_align"`
	Wrap    bool    `koanf:"wrap"`
}

type ImageWidget struct {
	Widget    `koanf:",squash"`
	Icon      string `koanf:"icon"`
	IconSize  string `koanf:"icon_size"`
	PixelSize int    `koanf:"pixel_size"`
	Theme     string `koanf:"theme"`
}

type Anchors struct {
	Bottom bool `koanf:"bottom"`
	Left   bool `koanf:"left"`
	Right  bool `koanf:"right"`
	Top    bool `koanf:"top"`
}

type Margins struct {
	Bottom int `koanf:"bottom"`
	End    int `koanf:"end"`
	Start  int `koanf:"start"`
	Top    int `koanf:"top"`
}

type Window struct {
	Widget `koanf:",squash"`
	Box    Box `koanf:"box"`
}

type Box struct {
	BoxWidget `koanf:",squash"`
	Scroll    Scroll        `koanf:"scroll"`
	AiScroll  AiScroll      `koanf:"ai_scroll"`
	Revert    bool          `koanf:"revert"`
	Search    SearchWrapper `koanf:"search"`
	Bar       BarWrapper    `koanf:"bar"`
}

type AiScroll struct {
	Widget           `koanf:",squash"`
	List             AiListWrapper `koanf:"list"`
	OverlayScrolling bool          `koanf:"overlay_scrolling"`
	HScrollbarPolicy string        `koanf:"h_scrollbar_policy"`
	VScrollbarPolicy string        `koanf:"v_scrollbar_policy"`
}

type AiListWrapper struct {
	BoxWidget `koanf:",squash"`
	Item      LabelWidget `koanf:"item"`
}

type BarWrapper struct {
	BoxWidget `koanf:",squash"`
	Position  string          `koanf:"position"`
	Entry     BarEntryWrapper `koanf:"entry"`
}

type BarEntryWrapper struct {
	BoxWidget `koanf:",squash"`
	Icon      ImageWidget `koanf:"icon"`
	Label     LabelWidget `koanf:"label"`
}

type Scroll struct {
	Widget           `koanf:",squash"`
	List             ListWrapper `koanf:"list"`
	OverlayScrolling bool        `koanf:"overlay_scrolling"`
	HScrollbarPolicy string      `koanf:"h_scrollbar_policy"`
	VScrollbarPolicy string      `koanf:"v_scrollbar_policy"`
}

type SearchWrapper struct {
	BoxWidget `koanf:",squash"`
	Revert    bool          `koanf:"revert"`
	Input     SearchWidget  `koanf:"input"`
	Prompt    PromptWidget  `koanf:"prompt"`
	Clear     ImageWidget   `koanf:"clear"`
	Spinner   SpinnerWidget `koanf:"spinner"`
}

type PromptWidget struct {
	LabelWidget `koanf:",squash"`
	ImageWidget `koanf:",squash"`
	Text        string `koanf:"text"`
	Icon        string `koanf:"icon"`
}

type SearchWidget struct {
	Widget `koanf:",squash"`
}

type SpinnerWidget struct {
	Widget `koanf:",squash"`
}

type ListWrapper struct {
	AlwaysShow  bool           `koanf:"always_show"`
	Grid        bool           `koanf:"grid"`
	Item        ListItemWidget `koanf:"item"`
	MarkerColor string         `koanf:"marker_color"`
	MaxHeight   int            `koanf:"max_height"`
	MaxWidth    int            `koanf:"max_width"`
	MinHeight   int            `koanf:"min_height"`
	MinWidth    int            `koanf:"min_width"`
	Orientation string         `koanf:"orientation"`
	Placeholder LabelWidget    `koanf:"placeholder"`
	Widget      `koanf:",squash"`
}

type ListItemWidget struct {
	BoxWidget       `koanf:",squash"`
	Revert          bool                  `koanf:"revert"`
	ActivationLabel ActivationLabelWidget `koanf:"activation_label"`
	Icon            ImageWidget           `koanf:"icon"`
	Text            TextWrapper           `koanf:"text"`
}

type ActivationLabelWidget struct {
	LabelWidget  `koanf:",squash"`
	Overlay      bool `koanf:"overlay"`
	HideModifier bool `koanf:"hide_modifier"`
}

type TextWrapper struct {
	BoxWidget `koanf:",squash"`
	Label     LabelWidget `koanf:"label"`
	Revert    bool        `koanf:"revert"`
	Sub       LabelWidget `koanf:"sub"`
}

func init() {
	os.MkdirAll(util.ThemeDir(), 0755)
	os.WriteFile(filepath.Join(util.ThemeDir(), "default.toml"), defaultThemeLayout, 0o600)
}

func GetLayout(theme string, base []string) *UI {
	layout := koanf.New(".")

	err := layout.Load(rawbytes.Provider(defaultLayout), toml.Parser())
	if err != nil {
		log.Panicln(err)
	}

	if base == nil {
		base = []string{}
	}

	base = append(base, theme)

	for _, v := range base {
		tomlFile := filepath.Join(util.ThemeDir(), fmt.Sprintf("%s.toml", v))
		jsonFile := filepath.Join(util.ThemeDir(), fmt.Sprintf("%s.json", v))
		yamlFile := filepath.Join(util.ThemeDir(), fmt.Sprintf("%s.yaml", v))

		if util.FileExists(tomlFile) {
			err := layout.Load(file.Provider(tomlFile), toml.Parser())
			if err != nil {
				return nil
			}
		} else if util.FileExists(jsonFile) {
			err := layout.Load(file.Provider(jsonFile), json.Parser())
			if err != nil {
				return nil
			}
		} else if util.FileExists(yamlFile) {
			err := layout.Load(file.Provider(yamlFile), yaml.Parser())
			if err != nil {
				return nil
			}
		} else {
			slog.Error("layout", "not found", v)
		}
	}

	ui := &UICfg{}

	err = layout.Unmarshal("", ui)
	if err != nil {
		slog.Error("layout", "error", err)
		return nil
	}

	return &ui.UI
}
