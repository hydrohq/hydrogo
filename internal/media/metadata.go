package media

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func stripImageMetadata(sourcePath, directory, mimeType string) (string, bool, error) {
	if mimeType != "image/jpeg" && mimeType != "image/webp" {
		return "", false, nil
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return "", false, err
	}
	defer source.Close()
	target, err := os.CreateTemp(directory, "metadata-clean-*")
	if err != nil {
		return "", false, err
	}
	targetPath := target.Name()
	clean, stripErr := false, error(nil)
	if mimeType == "image/jpeg" {
		clean, stripErr = stripJPEGAPP1(source, target)
	} else {
		clean, stripErr = stripWebPExif(source, target)
	}
	if stripErr != nil {
		_ = target.Close()
		_ = os.Remove(targetPath)
		return "", false, stripErr
	}
	if !clean {
		_ = target.Close()
		_ = os.Remove(targetPath)
		return "", false, nil
	}
	if err := target.Sync(); err != nil {
		_ = target.Close()
		_ = os.Remove(targetPath)
		return "", false, err
	}
	if err := target.Close(); err != nil {
		_ = os.Remove(targetPath)
		return "", false, err
	}
	return targetPath, true, nil
}

func stripJPEGAPP1(source io.Reader, target io.Writer) (bool, error) {
	reader := bufio.NewReader(source)
	var soi [2]byte
	if _, err := io.ReadFull(reader, soi[:]); err != nil || soi != [2]byte{0xff, 0xd8} {
		return false, ErrInvalidImageData
	}
	if _, err := target.Write(soi[:]); err != nil {
		return false, err
	}
	stripped := false
	for {
		prefix := make([]byte, 0, 4)
		first, err := reader.ReadByte()
		if err != nil || first != 0xff {
			return false, ErrInvalidImageData
		}
		prefix = append(prefix, first)
		for {
			b, err := reader.ReadByte()
			if err != nil {
				return false, ErrInvalidImageData
			}
			prefix = append(prefix, b)
			if b != 0xff {
				break
			}
		}
		marker := prefix[len(prefix)-1]
		if marker == 0xd9 || marker == 0xd8 || marker == 0x01 || marker >= 0xd0 && marker <= 0xd7 {
			if _, err := target.Write(prefix); err != nil {
				return false, err
			}
			if marker == 0xd9 {
				_, err := io.Copy(target, reader)
				return stripped, err
			}
			continue
		}
		var lengthBytes [2]byte
		if _, err := io.ReadFull(reader, lengthBytes[:]); err != nil {
			return false, ErrInvalidImageData
		}
		length := int(binary.BigEndian.Uint16(lengthBytes[:]))
		if length < 2 {
			return false, ErrInvalidImageData
		}
		payloadLength := length - 2
		if marker == 0xe1 {
			var signature [6]byte
			readLength := min(payloadLength, len(signature))
			if _, err := io.ReadFull(reader, signature[:readLength]); err != nil {
				return false, ErrInvalidImageData
			}
			isExif := readLength == len(signature) && string(signature[:]) == "Exif\x00\x00"
			if isExif {
				if _, err := io.CopyN(io.Discard, reader, int64(payloadLength-readLength)); err != nil {
					return false, ErrInvalidImageData
				}
				stripped = true
				continue
			}
			if _, err := target.Write(prefix); err != nil {
				return false, err
			}
			if _, err := target.Write(lengthBytes[:]); err != nil {
				return false, err
			}
			if _, err := target.Write(signature[:readLength]); err != nil {
				return false, err
			}
			if _, err := io.CopyN(target, reader, int64(payloadLength-readLength)); err != nil {
				return false, err
			}
		} else {
			if _, err := target.Write(prefix); err != nil {
				return false, err
			}
			if _, err := target.Write(lengthBytes[:]); err != nil {
				return false, err
			}
			if _, err := io.CopyN(target, reader, int64(payloadLength)); err != nil {
				return false, ErrInvalidImageData
			}
		}
		if marker == 0xda {
			_, err := io.Copy(target, reader)
			return stripped, err
		}
	}
}

func stripWebPExif(source io.ReadSeeker, target io.WriteSeeker) (bool, error) {
	var header [12]byte
	if _, err := io.ReadFull(source, header[:]); err != nil || string(header[:4]) != "RIFF" || string(header[8:]) != "WEBP" {
		return false, ErrInvalidImageData
	}
	if _, err := target.Write(header[:]); err != nil {
		return false, err
	}
	stripped := false
	for {
		var chunk [8]byte
		_, err := io.ReadFull(source, chunk[:])
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return false, ErrInvalidImageData
		}
		size := int64(binary.LittleEndian.Uint32(chunk[4:]))
		padding := size % 2
		if string(chunk[:4]) == "EXIF" {
			if _, err := io.CopyN(io.Discard, source, size+padding); err != nil {
				return false, ErrInvalidImageData
			}
			stripped = true
			continue
		}
		if _, err := target.Write(chunk[:]); err != nil {
			return false, err
		}
		if _, err := io.CopyN(target, source, size+padding); err != nil {
			return false, ErrInvalidImageData
		}
	}
	if !stripped {
		return false, nil
	}
	end, err := target.Seek(0, io.SeekCurrent)
	if err != nil || end < 8 || end-8 > int64(^uint32(0)) {
		return false, ErrInvalidImageData
	}
	if _, err := target.Seek(4, io.SeekStart); err != nil {
		return false, err
	}
	var size [4]byte
	binary.LittleEndian.PutUint32(size[:], uint32(end-8))
	if _, err := target.Write(size[:]); err != nil {
		return false, err
	}
	_, err = target.Seek(0, io.SeekEnd)
	return stripped, err
}

func replaceSanitizedFile(originalPath, sanitizedPath string) error {
	if strings.TrimSpace(filepath.Base(sanitizedPath)) == "" {
		return errors.New("invalid sanitized image path")
	}
	if err := os.Remove(originalPath); err != nil {
		return err
	}
	if err := os.Rename(sanitizedPath, originalPath); err != nil {
		return err
	}
	return nil
}
