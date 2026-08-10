package boundary

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const maxPayloadSize = 16 << 20

func EncodeBoundaryPayload(path json.RawMessage) ([]byte, error) {
	var value []any
	if err := json.Unmarshal(path, &value); err != nil {
		return nil, fmt.Errorf("invalid boundary JSON: %w", err)
	}
	if len(value) == 0 {
		return nil, errors.New("boundary JSON must be a non-empty array")
	}
	compact, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal boundary JSON: %w", err)
	}

	var output bytes.Buffer
	writer := gzip.NewWriter(&output)
	if _, err := writer.Write(compact); err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("compress boundary JSON: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close boundary gzip: %w", err)
	}
	return output.Bytes(), nil
}

func DecodeBoundaryPayload(reader io.Reader) (json.RawMessage, error) {
	if reader == nil {
		return nil, errors.New("boundary payload reader is required")
	}
	compressed, err := gzip.NewReader(reader)
	if err != nil {
		return nil, fmt.Errorf("open boundary gzip: %w", err)
	}
	defer compressed.Close()

	limited := io.LimitReader(compressed, maxPayloadSize+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read boundary gzip: %w", err)
	}
	if len(raw) > maxPayloadSize {
		return nil, fmt.Errorf("boundary payload exceeds %d bytes", maxPayloadSize)
	}
	var value []any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("invalid boundary JSON: %w", err)
	}
	if len(value) == 0 {
		return nil, errors.New("boundary JSON must be a non-empty array")
	}
	compact, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal boundary JSON: %w", err)
	}
	return json.RawMessage(compact), nil
}
