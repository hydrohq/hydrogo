package media

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxUploadParts      = 16
	maxUploadFieldBytes = 4096
)

type multipartUpload struct {
	Path     string
	Filename string
	Fields   map[string]string
}

func readMultipartUpload(r *http.Request, directory string, maxFileBytes int64) (result multipartUpload, err error) {
	result.Fields = make(map[string]string)
	temporary, err := os.CreateTemp(directory, "request-upload-*")
	if err != nil {
		return result, err
	}
	result.Path = temporary.Name()
	defer func() {
		if closeErr := temporary.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(result.Path)
		}
	}()
	if maxFileBytes <= 0 {
		return result, ErrFileTooLarge
	}

	var fileFound bool
	var reader *multipart.Reader
	if r.MultipartForm != nil {
		if values := r.MultipartForm.Value; values != nil {
			for key, entries := range values {
				if len(entries) > 1 || (len(entries) == 1 && len(entries[0]) > maxUploadFieldBytes) {
					return result, errors.New("invalid upload field")
				}
				if len(entries) == 1 {
					result.Fields[key] = entries[0]
				}
			}
		}
		files := r.MultipartForm.File["file"]
		if len(files) != 1 {
			return result, errors.New("file field is required")
		}
		input, openErr := files[0].Open()
		if openErr != nil {
			return result, openErr
		}
		result.Filename = filepath.Base(strings.ReplaceAll(files[0].Filename, `\`, "/"))
		err = copyValidatedUpload(temporary, input, result.Filename, maxFileBytes)
		_ = input.Close()
		if err != nil {
			return result, err
		}
		fileFound = true
	} else {
		reader, err = r.MultipartReader()
		if err != nil {
			return result, errors.New("invalid multipart upload")
		}
		for partCount := 0; ; partCount++ {
			part, nextErr := reader.NextPart()
			if errors.Is(nextErr, io.EOF) {
				break
			}
			if nextErr != nil {
				return result, errors.New("incomplete multipart upload")
			}
			if partCount >= maxUploadParts {
				_ = part.Close()
				return result, errors.New("too many upload fields")
			}
			name := part.FormName()
			if name == "file" && part.FileName() != "" {
				if fileFound {
					_ = part.Close()
					return result, errors.New("multiple files are not allowed")
				}
				result.Filename = filepath.Base(strings.ReplaceAll(part.FileName(), `\`, "/"))
				if err = copyValidatedUpload(temporary, part, result.Filename, maxFileBytes); err != nil {
					_ = part.Close()
					return result, err
				}
				fileFound = true
			} else {
				if name == "" || part.FileName() != "" {
					_ = part.Close()
					return result, errors.New("invalid upload field")
				}
				if _, exists := result.Fields[name]; exists {
					_ = part.Close()
					return result, errors.New("duplicate upload field")
				}
				value, readErr := io.ReadAll(io.LimitReader(part, maxUploadFieldBytes+1))
				if readErr != nil || len(value) > maxUploadFieldBytes {
					_ = part.Close()
					return result, errors.New("invalid upload field")
				}
				result.Fields[name] = string(value)
			}
			if closeErr := part.Close(); closeErr != nil {
				return result, errors.New("incomplete multipart upload")
			}
		}
	}
	if !fileFound {
		return result, errors.New("file field is required")
	}
	if err = temporary.Sync(); err != nil {
		return result, err
	}
	return result, nil
}

func copyValidatedUpload(destination *os.File, source io.Reader, filename string, maxBytes int64) error {
	limited := &io.LimitedReader{R: source, N: maxBytes + 1}
	header := make([]byte, 512)
	n, err := io.ReadFull(limited, header)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return fmt.Errorf("read upload header: %w", err)
	}
	header = header[:n]
	contentType := http.DetectContentType(header)
	if index := strings.IndexByte(contentType, ';'); index >= 0 {
		contentType = strings.TrimSpace(contentType[:index])
	}
	if !isDocumentFilename(filename) {
		if _, allowed := AllowedMimeTypes[contentType]; !allowed {
			return fmt.Errorf("%w: %s", ErrDisallowedMimeType, contentType)
		}
	}
	written, err := destination.Write(header)
	if err != nil {
		return err
	}
	remaining, err := io.Copy(destination, limited)
	if err != nil {
		return err
	}
	if int64(written)+remaining > maxBytes {
		return ErrFileTooLarge
	}
	return nil
}
