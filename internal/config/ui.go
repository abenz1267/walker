package config

import (
	"bytes"
	_ "embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/abenz1267/walker/internal/util"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/spf13/viper"
)

//go:embed layout.default.json
var defaultLayout []byte

type UICfg struct {
	UI UI `mapstructure:"ui"`
}

type UI struct {
	Anchors         Anchors `mapstructure:"anchors"`
	Fullscreen      bool    `mapstructure:"fullscreen"`
	IgnoreExclusive bool    `mapstructure:"ignore_exclusive"`
	Window          Window  `mapstructure:"window"`

	// internal
	AlignMap        map[string]gtk.Align         `mapstructure:"-"`
	IconSizeMap     map[string]gtk.IconSize      `mapstructure:"-"`
	IconSizeIntMap  map[string]int               `mapstructure:"-"`
	JustifyMap      map[string]gtk.Justification `mapstructure:"-"`
	OrientationMap  map[string]gtk.Orientation   `mapstructure:"-"`
	ScrollPolicyMap map[string]gtk.PolicyType    `mapstructure:"-"`
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
	CssClasses []string `mapstructure:"css_classes"`
	HAlign     string   `mapstructure:"h_align"`
	HExpand    bool     `mapstructure:"h_expand"`
	Height     int      `mapstructure:"height"`
	Hide       bool     `mapstructure:"hide"`
	Margins    Margins  `mapstructure:"margins"`
	Name       string   `mapstructure:"name"`
	Opacity    float64  `mapstructure:"opacity"`
	VAlign     string   `mapstructure:"v_align"`
	VExpand    bool     `mapstructure:"h_expand"`
	Width      int      `mapstructure:"width"`
}

type BoxWidget struct {
	Widget      `mapstructure:",squash"`
	Orientation string `mapstructure:"orientation"`
	Spacing     int    `mapstructure:"spacing"`
}

type LabelWidget struct {
	Widget  `mapstructure:",squash"`
	Justify string  `mapstructure:"justify"`
	XAlign  float32 `mapstructure:"x_align"`
	YAlign  float32 `mapstructure:"y_align"`
	Wrap    bool    `mapstructure:"wrap"`
}

type ImageWidget struct {
	Widget    `mapstructure:",squash"`
	IconSize  string `mapstructure:"icon_size"`
	PixelSize int    `mapstructure:"pixel_size"`
	Theme     string `mapstructure:"theme"`
}

type Anchors struct {
	Bottom bool `mapstructure:"bottom"`
	Left   bool `mapstructure:"left"`
	Right  bool `mapstructure:"right"`
	Top    bool `mapstructure:"top"`
}

type Margins struct {
	Bottom int `mapstructure:"bottom"`
	End    int `mapstructure:"end"`
	Start  int `mapstructure:"start"`
	Top    int `mapstructure:"top"`
}

type Window struct {
	Widget `mapstructure:",squash"`
	Box    Box `mapstructure:"box"`
}

type Box struct {
	BoxWidget `mapstructure:",squash"`
	Scroll    Scroll        `mapstructure:"scroll"`
	Revert    bool          `mapstructure:"revert"`
	Search    SearchWrapper `mapstructure:"search"`
}

type Scroll struct {
	Widget           `mapstructure:",squash"`
	List             ListWrapper `mapstructure:"list"`
	OverlayScrolling bool        `mapstructure:"overlay_scrolling"`
	HScrollbarPolicy string      `mapstructure:"h_scrollbar_policy"`
	VScrollbarPolicy string      `mapstructure:"v_scrollbar_policy"`
}

type SearchWrapper struct {
	BoxWidget `mapstructure:",squash"`
	Revert    bool          `mapstructure:"revert"`
	Input     SearchWidget  `mapstructure:"input"`
	Prompt    PromptWidget  `mapstructure:"prompt"`
	Spinner   SpinnerWidget `mapstructure:"spinner"`
}

type PromptWidget struct {
	LabelWidget `mapstructure:",squash"`
	Text        string `mapstructure:"text"`
}

type SearchWidget struct {
	Widget `mapstructure:",squash"`
	Icons  bool `mapstructure:"icons"`
}

type SpinnerWidget struct {
	Widget `mapstructure:",squash"`
}

type ListWrapper struct {
	Widget      `mapstructure:",squash"`
	Item        ListItemWidget `mapstructure:"item"`
	Grid        bool           `mapstructure:"grid"`
	Orientation string         `mapstructure:"orientation"`
	MinHeight   int            `mapstructure:"min_height"`
	MaxHeight   int            `mapstructure:"max_height"`
	MaxWidth    int            `mapstructure:"max_width"`
	MinWidth    int            `mapstructure:"min_width"`
	AlwaysShow  bool           `mapstructure:"always_show"`
}

type ListItemWidget struct {
	BoxWidget       `mapstructure:",squash"`
	Revert          bool                  `mapstructure:"revert"`
	ActivationLabel ActivationLabelWidget `mapstructure:"activation_label"`
	Icon            ImageWidget           `mapstructure:"icon"`
	Text            TextWrapper           `mapstructure:"text"`
}

type ActivationLabelWidget struct {
	LabelWidget  `mapstructure:",squash"`
	Overlay      bool `mapstructure:"overlay"`
	HideModifier bool `mapstructure:"hide_modifier"`
}

type TextWrapper struct {
	BoxWidget `mapstructure:",squash"`
	Label     LabelWidget `mapstructure:"label"`
	Revert    bool        `mapstructure:"revert"`
	Sub       LabelWidget `mapstructure:"sub"`
}

func GetLayout(theme string, base []string) *UI {
	layout, layoutFt := getLayout(theme)

	layoutCfg := viper.New()

	defs := viper.New()
	defs.SetConfigType("json")

	err := defs.ReadConfig(bytes.NewBuffer(defaultLayout))
	if err != nil {
		log.Panicln(err)
	}

	if base != nil && len(base) > 0 {
		inherit(base, defs)
	}

	for k, v := range defs.AllSettings() {
		layoutCfg.SetDefault(k, v)
	}

	layoutCfg.SetConfigType(layoutFt)

	err = layoutCfg.ReadConfig(bytes.NewBuffer(layout))
	if err != nil {
		log.Panicln(err)
	}

	ui := &UICfg{}
	err = layoutCfg.Unmarshal(ui)
	if err != nil {
		log.Panic(err)
	}

	return &ui.UI
}

func inherit(themes []string, cfg *viper.Viper) {
	for _, v := range themes {
		layout, layoutFt := getLayout(v)

		defs := viper.New()
		defs.SetConfigType(layoutFt)

		err := defs.ReadConfig(bytes.NewBuffer(layout))
		if err != nil {
			log.Panicln(err)
		}

		cfg.MergeConfig(bytes.NewBuffer(layout))
	}
}

func createLayoutFile(data []byte) {
	ft := "json"

	et := os.Getenv("WALKER_CONFIG_TYPE")

	if et != "" {
		ft = et
	}

	layout := viper.New()
	layout.SetConfigType("json")

	err := layout.ReadConfig(bytes.NewBuffer(data))
	if err != nil {
		log.Panicln(err)
	}

	layout.AddConfigPath(util.ThemeDir())

	layout.SetConfigType(ft)
	layout.SetConfigName(viper.GetString("theme"))

	wErr := layout.SafeWriteConfig()
	if wErr != nil {
		log.Println(wErr)
	}
}

func getLayout(theme string) ([]byte, string) {
	var layout []byte
	layoutFt := "json"

	file := filepath.Join(util.ThemeDir(), fmt.Sprintf("%s.json", theme))

	path := fmt.Sprintf("%s/", util.ThemeDir())

	filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		switch d.Name() {
		case fmt.Sprintf("%s.json", theme):
			layoutFt = "json"
			file = path
		case fmt.Sprintf("%s.toml", theme):
			layoutFt = "toml"
			file = path
		case fmt.Sprintf("%s.yaml", theme):
			layoutFt = "yaml"
			file = path
		}

		return nil
	})

	if _, err := os.Stat(file); err == nil {
		layout, err = os.ReadFile(file)
		if err != nil {
			log.Panicln(err)
		}
	} else {
		layoutFt = "json"

		switch theme {
		case "kanagawa":
			layout, err = Themes.ReadFile("themes/kanagawa.json")
			if err != nil {
				log.Panicln(err)
			}
		case "bare":
			layout, err = Themes.ReadFile("themes/bare.json")
			if err != nil {
				log.Panicln(err)
			}
		case "catppuccin":
			layout, err = Themes.ReadFile("themes/catppuccin.json")
			if err != nil {
				log.Panicln(err)
			}
		default:
			log.Printf("layout file for theme '%s' not found\n", theme)
			os.Exit(1)
		}

		createLayoutFile(layout)
	}

	return layout, layoutFt
}
