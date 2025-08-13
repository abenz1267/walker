package data

import (
	"bytes"
	"encoding/binary"

	"github.com/abenz1267/elephant/pkg/pb/pb"
	"google.golang.org/protobuf/proto"
)

func Activate(pos uint, query string) {
	item := Items.At(int(pos))

	req := pb.ActivateRequest{
		Qid:        item.Qid,
		Provider:   item.Item.Provider,
		Identifier: item.Item.Identifier,
		Action:     "",
		Arguments:  "",
	}

	b, err := proto.Marshal(&req)
	if err != nil {
		panic(err)
	}

	var buffer bytes.Buffer
	buffer.Write([]byte{1})

	lengthBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBuf, uint32(len(b)))
	buffer.Write(lengthBuf)
	buffer.Write(b)

	_, err = conn.Write(buffer.Bytes())
	if err != nil {
		panic(err)
	}
}
