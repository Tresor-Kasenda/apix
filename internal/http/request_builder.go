package apixhttp

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type FormField struct {
	Key   string
	Value string
}

func BuildMultipartForm(fields []FormField) ([]byte, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for _, field := range fields {
		key := strings.TrimSpace(field.Key)
		if key == "" {
			_ = writer.Close()
			return nil, "", fmt.Errorf("multipart field key cannot be empty")
		}

		if strings.HasPrefix(field.Value, "@") {
			path := strings.TrimSpace(strings.TrimPrefix(field.Value, "@"))
			if path == "" {
				_ = writer.Close()
				return nil, "", fmt.Errorf("multipart field %q references an empty file path", key)
			}

			file, err := os.Open(path)
			if err != nil {
				_ = writer.Close()
				return nil, "", fmt.Errorf("opening multipart file %q: %w", path, err)
			}

			part, err := writer.CreateFormFile(key, filepath.Base(path))
			if err != nil {
				_ = file.Close()
				_ = writer.Close()
				return nil, "", fmt.Errorf("creating multipart field %q: %w", key, err)
			}

			if _, err := io.Copy(part, file); err != nil {
				_ = file.Close()
				_ = writer.Close()
				return nil, "", fmt.Errorf("copying multipart file %q: %w", path, err)
			}
			_ = file.Close()
			continue
		}

		if err := writer.WriteField(key, field.Value); err != nil {
			_ = writer.Close()
			return nil, "", fmt.Errorf("writing multipart field %q: %w", key, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("finalizing multipart body: %w", err)
	}

	return buf.Bytes(), writer.FormDataContentType(), nil
}

func BuildURLEncodedForm(fields []FormField) (string, string, error) {
	values := url.Values{}
	for _, field := range fields {
		key := strings.TrimSpace(field.Key)
		if key == "" {
			return "", "", fmt.Errorf("urlencoded field key cannot be empty")
		}
		values.Add(key, field.Value)
	}

	return values.Encode(), "application/x-www-form-urlencoded", nil
}
