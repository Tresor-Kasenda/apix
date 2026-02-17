package apixhttp

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildMultipartForm(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "avatar.txt")
	if err := os.WriteFile(filePath, []byte("avatar-content"), 0o644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	body, contentType, err := BuildMultipartForm([]FormField{
		{Key: "type", Value: "avatar"},
		{Key: "file", Value: "@" + filePath},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		t.Fatalf("parsing content type: %v", err)
	}
	if mediaType != "multipart/form-data" {
		t.Fatalf("expected multipart/form-data, got %q", mediaType)
	}

	reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	values := map[string]string{}
	filenames := map[string]string{}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("reading multipart part: %v", err)
		}

		data, err := io.ReadAll(part)
		if err != nil {
			t.Fatalf("reading multipart part body: %v", err)
		}

		values[part.FormName()] = string(data)
		if part.FileName() != "" {
			filenames[part.FormName()] = part.FileName()
		}
	}

	if values["type"] != "avatar" {
		t.Fatalf("expected type=avatar, got %q", values["type"])
	}
	if values["file"] != "avatar-content" {
		t.Fatalf("expected file content avatar-content, got %q", values["file"])
	}
	if filenames["file"] != "avatar.txt" {
		t.Fatalf("expected filename avatar.txt, got %q", filenames["file"])
	}
}

func TestBuildURLEncodedForm(t *testing.T) {
	t.Parallel()

	encoded, contentType, err := BuildURLEncodedForm([]FormField{
		{Key: "email", Value: "test@example.com"},
		{Key: "note", Value: "hello world"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if contentType != "application/x-www-form-urlencoded" {
		t.Fatalf("unexpected content type: %q", contentType)
	}

	values, err := url.ParseQuery(encoded)
	if err != nil {
		t.Fatalf("parsing encoded form: %v", err)
	}

	if values.Get("email") != "test@example.com" {
		t.Fatalf("expected email field, got %q", values.Get("email"))
	}
	if values.Get("note") != "hello world" {
		t.Fatalf("expected note field, got %q", values.Get("note"))
	}
}
