package utils

import (
	"context"
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
func FetchJSON(ctx context.Context, url string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("error creating request for URL %s: %w", url, err)
	}

	resp, err := http.DefaultClient.Do(req)
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

// FetchJSONWithCookies выполняет запрос к URL с куками и декодирует JSON в целевую структуру
func FetchJSONWithCookies(ctx context.Context, url string, target any, cookies map[string]string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("error creating request for URL %s: %w", url, err)
	}
	if len(cookies) > 0 {
		var cookieHeader string
		for k, v := range cookies {
			if len(cookieHeader) > 0 {
				cookieHeader += "; "
			}
			cookieHeader += k + "=" + v
		}
		req.Header.Set("Cookie", cookieHeader)
	}

	resp, err := http.DefaultClient.Do(req)
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
