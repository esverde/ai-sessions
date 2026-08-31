package session

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func ReadJSONLLines(path string, visit func(map[string]any) error) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	// 只读文件,Close 的错误没有可采取的行动,显式丢弃以示有意为之。
	defer func() { _ = file.Close() }()

	reader := bufio.NewReaderSize(file, 64*1024)
	for {
		line, readErr := reader.ReadBytes('\n')
		line = bytes.TrimSpace(line)
		if len(line) > 0 {
			var value map[string]any
			if err := json.Unmarshal(line, &value); err == nil && value != nil {
				if err := visit(value); err != nil {
					return err
				}
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				return nil
			}
			return fmt.Errorf("read JSONL %s: %w", path, readErr)
		}
	}
}

func ReadTailLines(path string, maxBytes, maxLines int) ([][]byte, error) {
	if maxBytes < 1024 {
		maxBytes = 1024
	}
	if maxLines < 1 {
		maxLines = 1
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	// 只读文件,Close 的错误没有可采取的行动,显式丢弃以示有意为之。
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	offset := info.Size() - int64(maxBytes)
	if offset < 0 {
		offset = 0
	}
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	if offset > 0 {
		if index := bytes.IndexByte(data, '\n'); index >= 0 {
			data = data[index+1:]
		} else {
			data = nil
		}
	}
	rawLines := bytes.Split(data, []byte{'\n'})
	result := make([][]byte, 0, maxLines)
	for index := len(rawLines) - 1; index >= 0 && len(result) < maxLines; index-- {
		line := bytes.TrimSpace(rawLines[index])
		if len(line) == 0 {
			continue
		}
		copyLine := append([]byte(nil), line...)
		result = append(result, copyLine)
	}
	for left, right := 0, len(result)-1; left < right; left, right = left+1, right-1 {
		result[left], result[right] = result[right], result[left]
	}
	return result, nil
}
