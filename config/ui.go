package config

import "github.com/diamondburned/gotk4/pkg/gtk/v4"

type UICfg struct {
	UI *UI `mapstructure:"ui"`
}

type UI struct {
	Anchors         *Anchors `mapstructure:"anchors"`
	Fullscreen      *bool    `mapstructure:"fullscreen"`
	IgnoreExclusive *bool    `mapstructure:"ignore_exclusive"`
	Window          *Window  `mapstructure:"window"`

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
	CssClasses *[]string `mapstructure:"css_classes"`
	HAlign     *string   `mapstructure:"h_align"`
	HExpand    *bool     `mapstructure:"h_expand"`
	Height     *int      `mapstructure:"height"`
	Hide       *bool     `mapstructure:"hide"`
	Margins    *Margins  `mapstructure:"margins"`
	Name       *string   `mapstructure:"name"`
	Opacity    *float64  `mapstructure:"opacity"`
	VAlign     *string   `mapstructure:"v_align"`
	VExpand    *bool     `mapstructure:"h_expand"`
	Width      *int      `mapstructure:"width"`
}

type BoxWidget struct {
	Widget      `mapstructure:",squash"`
	Orientation *string `mapstructure:"orientation"`
	Spacing     *int    `mapstructure:"spacing"`
}

type LabelWidget struct {
	Widget  `mapstructure:",squash"`
	Justify *string  `mapstructure:"justify"`
	XAlign  *float32 `mapstructure:"x_align"`
	YAlign  *float32 `mapstructure:"y_align"`
}

type ImageWidget struct {
	Widget    `mapstructure:",squash"`
	IconSize  *string `mapstructure:"icon_size"`
	PixelSize *int    `mapstructure:"pixel_size"`
	Theme     *string `mapstructure:"theme"`
}

type Anchors struct {
	Bottom *bool `mapstructure:"bottom"`
	Left   *bool `mapstructure:"left"`
	Right  *bool `mapstructure:"right"`
	Top    *bool `mapstructure:"top"`
}

type Margins struct {
	Bottom *int `mapstructure:"bottom"`
	End    *int `mapstructure:"end"`
	Start  *int `mapstructure:"start"`
	Top    *int `mapstructure:"top"`
}

type Window struct {
	Widget `mapstructure:",squash"`
	Box    *Box `mapstructure:"box"`
}

type Box struct {
	BoxWidget `mapstructure:",squash"`
	Scroll    *Scroll        `mapstructure:"scroll"`
	Revert    *bool          `mapstructure:"revert"`
	Search    *SearchWrapper `mapstructure:"search"`
}

type Scroll struct {
	Widget           `mapstructure:",squash"`
	List             *ListWrapper `mapstructure:"list"`
	OverlayScrolling *bool        `mapstructure:"overlay_scrolling"`
	HScrollbarPolicy *string      `mapstructure:"h_scrollbar_policy"`
	VScrollbarPolicy *string      `mapstructure:"v_scrollbar_policy"`
}

type SearchWrapper struct {
	BoxWidget `mapstructure:",squash"`
	Revert    *bool          `mapstructure:"revert"`
	Input     *SearchWidget  `mapstructure:"input"`
	Spinner   *SpinnerWidget `mapstructure:"spinner"`
}

type SearchWidget struct {
	Widget `mapstructure:",squash"`
	Icons  *bool `mapstructure:"icons"`
}

type SpinnerWidget struct {
	Widget `mapstructure:",squash"`
}

type ListWrapper struct {
	Widget      `mapstructure:",squash"`
	Item        *ListItemWidget `mapstructure:"item"`
	Orientation *string         `mapstructure:"orientation"`
	MinHeight   *int            `mapstructure:"min_height"`
	MinWidth    *int            `mapstructure:"min_width"`
	MaxHeight   *int            `mapstructure:"max_height"`
	MaxWidth    *int            `mapstructure:"max_width"`
	AlwaysShow  *bool           `mapstructure:"always_show"`
}

type ListItemWidget struct {
	BoxWidget       `mapstructure:",squash"`
	Revert          *bool        `mapstructure:"revert"`
	ActivationLabel *LabelWidget `mapstructure:"activation_label"`
	Icon            *ImageWidget `mapstructure:"icon"`
	Text            *TextWrapper `mapstructure:"text"`
}

type TextWrapper struct {
	BoxWidget `mapstructure:",squash"`
	Label     *LabelWidget `mapstructure:"label"`
	Revert    *bool        `mapstructure:"revert"`
	Sub       *LabelWidget `mapstructure:"sub"`
}
