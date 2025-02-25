package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/kosttiik/semesterly_backend/internal/models"
)

// AppendError добавляет ошибку в срез
func AppendError(mu *sync.Mutex, errors *[]string, errMsg string) {
	mu.Lock()
	*errors = append(*errors, errMsg)
	mu.Unlock()
}

// FetchJSON выполняет запрос к URL и декодирует JSON в целевую структуру
func FetchJSON(url string, target any) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error fetching URL %s: %w", url, err)
	}
	defer resp.Body.Close()

	// Проверка Content-Type
	if !strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		return fmt.Errorf("invalid content type for URL %s", url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("error unmarshalling JSON: %w", err)
	}

	return nil
}

// ExtractGroupUUIDs извлекает UUID групп из дерева
func ExtractGroupUUIDs(children []models.Child) []string {
	var uuids []string
	for _, child := range children {
		if child.NodeType != nil && *child.NodeType == "group" {
			uuids = append(uuids, child.UUID)
		}
		if len(child.Children) > 0 {
			uuids = append(uuids, ExtractGroupUUIDs(child.Children)...)
		}
	}
	return uuids
}
