package media

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStripJPEGExifSegment(t *testing.T) {
	input := []byte{0xff, 0xd8, 0xff, 0xe1, 0x00, 0x0c, 'E', 'x', 'i', 'f', 0, 0, 'G', 'P', 'S', '!', 0xff, 0xda, 0, 2, 1, 2, 0xff, 0xd9}
	var output bytes.Buffer
	stripped, err := stripJPEGAPP1(bytes.NewReader(input), &output)
	if err != nil || !stripped {
		t.Fatalf("strip JPEG = %v, %v", stripped, err)
	}
	if bytes.Contains(output.Bytes(), []byte("Exif")) || !bytes.HasSuffix(output.Bytes(), []byte{0xff, 0xd9}) {
		t.Fatalf("JPEG metadata was not stripped: %x", output.Bytes())
	}
}

func TestStripWebPExifChunkAndUpdatesRIFFLength(t *testing.T) {
	var input bytes.Buffer
	input.WriteString("RIFF")
	input.Write([]byte{0, 0, 0, 0})
	input.WriteString("WEBP")
	input.WriteString("VP8X")
	input.Write([]byte{4, 0, 0, 0})
	input.Write([]byte{1, 2, 3, 4})
	input.WriteString("EXIF")
	input.Write([]byte{3, 0, 0, 0})
	input.Write([]byte("gps"))
	input.WriteByte(0)
	binary.LittleEndian.PutUint32(input.Bytes()[4:8], uint32(input.Len()-8))
	sourcePath := filepath.Join(t.TempDir(), "source.webp")
	if err := os.WriteFile(sourcePath, input.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	destination, stripped, err := stripImageMetadata(sourcePath, t.TempDir(), "image/webp")
	if err != nil || !stripped {
		t.Fatalf("strip WebP = %v, %v", stripped, err)
	}
	output, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(output), "EXIF") || int(binary.LittleEndian.Uint32(output[4:8])) != len(output)-8 || !bytes.Contains(output, []byte("VP8X")) {
		t.Fatalf("WebP EXIF/RIFF cleanup invalid: %x", output)
	}
}
