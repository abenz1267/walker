package data

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/abenz1267/elephant/pkg/pb/pb"
	"github.com/abenz1267/walker/internal/config"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"google.golang.org/protobuf/proto"
)

var (
	socket   = filepath.Join(os.TempDir(), "elephant.sock")
	provider string
)

func InputChanged(input *gtk.Entry) {
	if input.Text() == "" {
		provider = ""
	}

	query(input.Text())
}

var conn net.Conn

func Init() {
	var err error

	conn, err = net.Dial("unix", socket)
	if err != nil {
		panic(err)
	}
}

func StartListening() {
	reader := bufio.NewReader(conn)

	for {
		header, err := reader.Peek(5)
		if err != nil {
			if err == io.EOF {
				continue
			}
			panic(err)
		}

		if header[0] == 255 {
			msg := make([]byte, 5)
			_, err = io.ReadFull(reader, msg)
			if err != nil {
				panic(err)
			}

			continue
		}

		if header[0] == 254 {
			msg := make([]byte, 5)
			_, err = io.ReadFull(reader, msg)
			if err != nil {
				panic(err)
			}

			glib.IdleAdd(func() {
				Items.Splice(0, Items.Len())
			})

			continue
		}

		length := binary.BigEndian.Uint32(header[1:5])

		msg := make([]byte, 5+length)
		_, err = io.ReadFull(reader, msg)
		if err != nil {
			panic(err)
		}

		payload := msg[5:]

		resp := pb.QueryResponse{}
		if err := proto.Unmarshal(payload, &resp); err != nil {
			panic(err)
		}

		glib.IdleAdd(func() {
			// async item response
			if header[0] == 1 {
				for i := 0; i < Items.Len(); i++ {
					item := Items.At(i)
					if item.Item.Identifier == resp.Item.Identifier {
						if resp.Item.Text == "%DELETE%" {
							Items.Splice(i, 1)
						} else {
							Items.Splice(i, 1, []pb.QueryResponse{resp}...)
						}

						break
					}
				}
			} else {
				if Items.Len() > 0 {
					i := Items.At(Items.Len() - 1)

					if resp.Qid > i.Qid || (resp.Qid == i.Qid && resp.Iid > i.Iid) {
						Items.Splice(0, Items.Len())
					}
				}

				Items.Splice(Items.Len(), 0, resp)
			}
		})
	}
}

func query(text string) {
	for _, v := range config.LoadedConfig.Providers.Prefixes {
		if strings.HasPrefix(text, v.Prefix) {
			provider = v.Provider
			text = strings.TrimPrefix(text, v.Prefix)
			break
		}
	}

	req := pb.QueryRequest{
		Query:      text,
		Maxresults: int32(50),
	}

	if provider != "" {
		req.Providers = append(req.Providers, provider)
	} else {
		if text == "" {
			req.Providers = config.LoadedConfig.Providers.Empty
		} else {
			req.Providers = config.LoadedConfig.Providers.Default
		}
	}

	b, err := proto.Marshal(&req)
	if err != nil {
		panic(err)
	}

	var buffer bytes.Buffer
	buffer.Write([]byte{0})

	lengthBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBuf, uint32(len(b)))
	buffer.Write(lengthBuf)
	buffer.Write(b)

	_, err = conn.Write(buffer.Bytes())
	if err != nil {
		panic(err)
	}
}
