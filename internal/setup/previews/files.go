package previews

import (
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/abenz1267/elephant/pkg/pb/pb"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type FilesPreviewHandler struct{}

var throttler = NewLatestOnlyThrottler(50 * time.Millisecond)

func (*FilesPreviewHandler) Handle(item *pb.QueryResponse_Item, preview *gtk.Box, builder *gtk.Builder) {
	throttler.Execute(item.Text, preview, builder)
}

type FilePreview struct {
	*gtk.Box
	previewArea *gtk.Stack
	currentFile string
}

func NewFilePreview(builder *gtk.Builder) *FilePreview {
	fp := &FilePreview{}

	fp.Box = builder.GetObject("PreviewBox").Cast().(*gtk.Box)

	fp.previewArea = builder.GetObject("PreviewStack").Cast().(*gtk.Stack)

	return fp
}

func (fp *FilePreview) PreviewFile(filePath string) error {
	fp.currentFile = filePath

	mimeType := fp.detectMimeType(filePath)

	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return fp.previewImage(filePath)
	case strings.HasPrefix(mimeType, "text/"):
		return fp.previewText(filePath)
	default:
		return fp.previewGeneric(filePath, mimeType)
	}
}

func (fp *FilePreview) detectMimeType(filePath string) string {
	// First try by extension
	mimeType := mime.TypeByExtension(filepath.Ext(filePath))
	if mimeType != "" {
		return mimeType
	}

	// If that fails, try reading file header
	file, err := os.Open(filePath)
	if err != nil {
		return "application/octet-stream"
	}
	defer file.Close()

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		return "application/octet-stream"
	}

	return mime.TypeByExtension(filepath.Ext(filePath))
}

func (fp *FilePreview) previewImage(filePath string) error {
	picture := gtk.NewPicture()
	picture.SetFilename(filePath)
	picture.SetCanShrink(true)
	picture.SetContentFit(gtk.ContentFitContain)

	scrolled := gtk.NewScrolledWindow()
	scrolled.SetChild(picture)
	scrolled.SetPolicy(gtk.PolicyAutomatic, gtk.PolicyAutomatic)

	fp.previewArea.AddChild(scrolled)
	fp.previewArea.SetVisibleChild(scrolled)

	return nil
}

func (fp *FilePreview) previewText(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Limit content size for large files
	maxSize := 1024 * 1024 // 1MB
	if len(content) > maxSize {
		content = content[:maxSize]
		content = append(content, []byte("\n\n[File truncated...]")...)
	}

	textView := gtk.NewTextView()
	textView.SetEditable(false)
	textView.SetMonospace(true)
	textView.SetWrapMode(gtk.WrapWord)

	buffer := textView.Buffer()
	buffer.SetText(string(content))

	scrolled := gtk.NewScrolledWindow()
	scrolled.SetChild(textView)
	scrolled.SetPolicy(gtk.PolicyAutomatic, gtk.PolicyAutomatic)

	fp.previewArea.AddChild(scrolled)
	fp.previewArea.SetVisibleChild(scrolled)

	return nil
}

func (fp *FilePreview) previewGeneric(filePath string, mimeType string) error {
	box := gtk.NewBox(gtk.OrientationVertical, 10)
	box.SetHAlign(gtk.AlignCenter)
	box.SetVAlign(gtk.AlignCenter)

	// Generic file icon
	icon := gtk.NewImageFromIconName("text-x-generic")
	icon.SetIconSize(gtk.IconSizeLarge)
	box.Append(icon)

	// File type
	typeLabel := gtk.NewLabel(fmt.Sprintf("Type: %s", mimeType))
	box.Append(typeLabel)

	// File size
	info, err := os.Stat(filePath)
	if err == nil {
		sizeLabel := gtk.NewLabel(fmt.Sprintf("Size: %d bytes", info.Size()))
		box.Append(sizeLabel)

		modLabel := gtk.NewLabel(fmt.Sprintf("Modified: %s", info.ModTime().Format("2006-01-02 15:04:05")))
		box.Append(modLabel)
	}

	fp.previewArea.AddChild(box)
	fp.previewArea.SetVisibleChild(box)

	return nil
}

type LatestOnlyThrottler struct {
	ticker     *time.Ticker
	latestCall string
	hasCall    bool
	preview    *gtk.Box
	builder    *gtk.Builder
	mu         sync.Mutex
	stop       chan struct{}
	once       sync.Once
}

func NewLatestOnlyThrottler(interval time.Duration) *LatestOnlyThrottler {
	t := &LatestOnlyThrottler{
		ticker: time.NewTicker(interval),
		stop:   make(chan struct{}),
	}

	go t.run()
	return t
}

func (t *LatestOnlyThrottler) run() {
	for {
		select {
		case <-t.stop:
			return
		case <-t.ticker.C:
			t.mu.Lock()
			if t.hasCall && t.latestCall != "" {
				t.hasCall = false
				t.mu.Unlock()

				glib.IdleAdd(func() {
					f := NewFilePreview(t.builder)
					f.PreviewFile(t.latestCall)

					for t.preview.FirstChild() != nil {
						t.preview.Remove(t.preview.FirstChild())
					}

					t.preview.Append(f)
					t.preview.SetVisible(true)
				})
			} else {
				t.mu.Unlock()
			}
		}
	}
}

func (t *LatestOnlyThrottler) Execute(file string, preview *gtk.Box, builder *gtk.Builder) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.latestCall = file
	t.builder = builder
	t.preview = preview
	t.hasCall = true
}

func (t *LatestOnlyThrottler) Stop() {
	t.once.Do(func() {
		close(t.stop)
		t.ticker.Stop()
	})
}
