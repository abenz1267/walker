[![GoDoc](https://godoc.org/github.com/KarpelesLab/weak?status.svg)](https://godoc.org/github.com/KarpelesLab/weak)

# weakref map in go 1.18

This is a weakref map for Go 1.18, with some inspiration from [xeus2001's weakref implementation](https://github.com/xeus2001/go-weak).

This provides both a `weak.Ref` object to store weak references, and a `weak.Map` object for maps.

## Usage

```go
import "github.com/KarpelesLab/weak"

var m = weak.NewMap[uint64, Object]()

// instanciate/get an object
func Get(id uint64) (*Object, error) {
	// try to get from cache
	var obj *Object
	if obj = m.Get(id); obj == nil {
		// create new
		obj = m.Set(id, &Object{id: id}) // this will return an existing object if already existing
	}

	obj.initOnce.Do(obj.init) // use sync.Once to ensure init happens only once
	return obj, obj.err
}

func main() {
	obj, err := Get(1234)
	// ...
}
```

As to the `Object` implementation, it could look like:

```go
type Object struct {
	id       uint64
	initOnce sync.Once
	f        *os.File
	err      error
}

func (o *Object) init() {
	o.f, o.err = os.Open(fmt.Sprintf("/tmp/file%d.bin", o.id))
}

func (o *Object) Destroy() {
	if o.f != nil {
		o.f.Close()
	}
}
```

### File example

Simple example that allows opening multiple files without having to care about closing these and reading random data using `ReadAt`.

```go
import {
	"github.com/KarpelesLab/weak"
	"os"
	"sync"
}

type File struct {
	*os.File
	err  error
	open sync.Once
}

var fileCache = weak.NewMap[string, File]()

func Open(filepath string) (io.ReaderAt, error) (
	var f *File
	if f = fileCache.Get(filepath); f == nil {
		f = fileCache.Set(filepath, &File{})
	}
	f.open.Do(func() {
		f.File, f.err = os.Open(filepath)
	})
	return f, f.err
}

func (f *File) Destroy() {
	if f.File != nil {
		f.File.Close()
	}
}
```
