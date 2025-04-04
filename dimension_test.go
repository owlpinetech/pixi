package pixi

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/owlpinetech/pixi/internal/buffer"
)

func TestDimensionHeaderSize(t *testing.T) {
	headers := allHeaderVariants(Version)

	for _, header := range headers {
		nameLen := rand.Intn(30)
		name := string(make([]byte, nameLen))
		dim := Dimension{
			Name:     name,
			Size:     rand.Int(),
			TileSize: rand.Int(),
		}
		if dim.HeaderSize(header) != 2+nameLen+header.OffsetSize+header.OffsetSize {
			t.Errorf("unexpected dimension header size")
		}
	}
}

func TestDimensionWriteRead(t *testing.T) {
	headers := allHeaderVariants(Version)

	cases := []Dimension{
		{Name: "nameone", Size: 40, TileSize: 20},
		{Name: "", Size: 50, TileSize: 5},
		{Name: "amuchlongernamethanusualwithlotsofcharacters", Size: 20000000, TileSize: 1},
	}

	for _, c := range cases {
		for _, h := range headers {
			buf := buffer.NewBuffer(10)
			err := c.Write(buf, h)
			if err != nil {
				t.Fatal("write dimension", err)
			}

			readBuf := buffer.NewBufferFrom(buf.Bytes())
			readDim := Dimension{}
			err = (&readDim).Read(readBuf, h)
			if err != nil {
				t.Fatal("read dimension", err)
			}

			if !reflect.DeepEqual(c, readDim) {
				t.Errorf("expected read dimension to be %v, got %v for header %v", c, readDim, h)
			}
		}
	}
}

func TestDimensionTiles(t *testing.T) {
	tests := []struct {
		name     string
		size     int
		tileSize int
		want     int
	}{
		{"size same as tile size", 10, 10, 1},
		{"small size, small tile", 100, 10, 10},
		{"medium size, medium tile", 500, 50, 10},
		{"large size, large tile", 2000, 100, 20},
		{"zero size", 0, 10, 0},
		{"tile not multiple", 100, 11, 10},
		{"large multiple", 86400, 21600, 4},
		{"half large multiple", 43200, 21600, 2},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dimension := Dimension{
				Size:     test.size,
				TileSize: test.tileSize,
			}
			got := dimension.Tiles()
			if got != test.want {
				t.Errorf("got %d, want %d", got, test.want)
			}
		})
	}
}
