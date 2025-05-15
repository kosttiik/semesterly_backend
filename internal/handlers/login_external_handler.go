package handlers

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"maps"

	"github.com/labstack/echo/v4"
)

// Ожидаемый payload от фронтенда
type LoginExternalRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Ответ, отправляемый на фронтенд
type LoginExternalResponse struct {
	Cookies    map[string]string `json:"cookies"`
	LastName   string            `json:"lastName"`
	FirstName  string            `json:"firstName"`
	MiddleName string            `json:"middleName"`
	Photo      string            `json:"photo"`
}

// Структуры для ответа внешнего API
type externalProfileResponse struct {
	LastName   string `json:"lastName"`
	FirstName  string `json:"firstName"`
	MiddleName string `json:"middleName"`
	// Остальные поля не нужны (возможно, пока что...)
}

type externalPhotoResponse struct {
	Photo string `json:"photo"`
}

// LoginExternalHandler логинит пользователя во внешний портал и возвращает сессионные куки
func (a *App) LoginExternalHandler(c echo.Context) error {
	var req LoginExternalRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}
	if req.Username == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Username and password required"})
	}

	ctx := c.Request().Context()

	// Шаг 1: Получаем страницу логина, чтобы вытащить execution
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Не следуем за редиректами автоматически
			return http.ErrUseLastResponse
		},
	}

	// Добавляем стартовые куки, которые могут понадобиться
	initialCookies := map[string]string{
		"width": "2048",
		"_csrf": base64.URLEncoding.EncodeToString([]byte("47ddcd561d141f56eb1d46a35981c446b454b33118eec30da836906c500d26fea")),
	}

	cookies := make(map[string]string)
	maps.Copy(cookies, initialCookies)

	// Общие заголовки для всех запросов
	commonHeaders := map[string]string{
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36",
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"Accept-Language": "ru,en-US;q=0.9,en;q=0.8",
	}

	// Добавить заголовки и куки к запросу
	addHeadersAndCookies := func(req *http.Request) {
		for k, v := range commonHeaders {
			req.Header.Set(k, v)
		}
		for name, value := range cookies {
			cookie := &http.Cookie{
				Name:  name,
				Value: value, // TGC и другие специальные куки не трогаем
				Path:  "/",
			}
			// Домен только для определённых кук
			if name != "TGC" && strings.Contains(req.URL.Host, "bmstu.ru") {
				cookie.Domain = "bmstu.ru"
			}
			req.AddCookie(cookie)
		}
	}

	// Вспомогательная функция: логировать ответ для отладки
	debugResponse := func(resp *http.Response, step string) {
		location := resp.Header.Get("Location")
		if location != "" {
			log.Printf("%s: status=%d, Location=%s", step, resp.StatusCode, location)
		} else {
			log.Printf("%s: status=%d", step, resp.StatusCode)
		}
	}

	loginURL := "https://proxy.bmstu.ru:8443/cas/login?service=https%3A%2F%2Fproxy.bmstu.ru%3A8443%2Fcas%2Foauth2.0%2FcallbackAuthorize%3Fclient_name%3DCasOAuthClient%26client_id%3DEU"
	loginReq, _ := http.NewRequestWithContext(ctx, "GET", loginURL, nil)
	addHeadersAndCookies(loginReq)
	resp, err := client.Do(loginReq)
	if err != nil {
		log.Printf("Failed to reach login page: %v", err)
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "Failed to reach login page"})
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	execution := extractExecution(string(body))
	if execution == "" {
		log.Printf("Failed to extract execution token")
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "Failed to extract execution token"})
	}

	// Шаг 2: Отправляем логин и пароль на сервер
	form := url.Values{}
	form.Set("username", req.Username)
	form.Set("password", req.Password)
	form.Set("execution", execution)
	form.Set("_eventId", "submit")
	form.Set("geolocation", "")

	loginReq, _ = http.NewRequestWithContext(ctx, "POST", loginURL, strings.NewReader(form.Encode()))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, cookie := range resp.Cookies() {
		loginReq.AddCookie(cookie)
	}
	addHeadersAndCookies(loginReq)

	loginResp, err := client.Do(loginReq)
	if err != nil {
		log.Printf("Failed to submit login: %v", err)
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "Failed to submit login"})
	}
	defer loginResp.Body.Close()

	// Шаг 3: Проверяем редирект с ticket
	location := loginResp.Header.Get("Location")
	if !strings.Contains(location, "ticket=") {
		b, _ := io.ReadAll(loginResp.Body)
		log.Printf("Login failed: %s", string(b))
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Login failed", "details": string(b)})
	}

	// Обновляем куки из ответа логина
	for _, cookie := range loginResp.Cookies() {
		cookies[cookie.Name] = cookie.Value
	}

	// Шаг 4: Следуем по редиректу callbackAuthorize
	callbackReq, err := http.NewRequestWithContext(ctx, "GET", location, nil)
	if err != nil {
		log.Printf("Failed to create callback request: %v", err)
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "Failed to create callback request"})
	}
	addHeadersAndCookies(callbackReq)
	log.Printf("Callback request URL: %s", location)

	callbackResp, err := client.Do(callbackReq)
	if err != nil {
		log.Printf("Callback request failed: %v", err)
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "Failed to follow callback"})
	}
	defer callbackResp.Body.Close()

	// Обновляем куки и получаем следующий location
	for _, cookie := range callbackResp.Cookies() {
		log.Printf("Got cookie from callback: %s", cookie.Name)
		cookies[cookie.Name] = cookie.Value
	}

	// Шаг 5: Следуем по редиректу на OAuth2 authorize
	location = callbackResp.Header.Get("Location")
	debugResponse(callbackResp, "Callback")

	if location == "" || location == "/" {
		// Генерируем state для OAuth2
		state := base64.RawURLEncoding.EncodeToString([]byte(time.Now().String()))

		// Собираем OAuth2 URL с нужными параметрами
		location = "https://proxy.bmstu.ru:8443/cas/oauth2.0/authorize?" +
			"client_id=EU&" +
			"response_type=code&" +
			"client_name=CasOAuthClient&" +
			"state=" + state + "&" +
			"redirect_uri=" + url.QueryEscape("https://lks.bmstu.ru/portal3/login1/mail?back=https://lks.bmstu.ru/profile")

		log.Printf("Constructed OAuth2 URL: %s", location)
	} else if !strings.HasPrefix(location, "http") {
		if !strings.HasPrefix(location, "/cas/") {
			location = "/cas/oauth2.0/authorize" + location
		}
		location = "https://proxy.bmstu.ru:8443" + location
	}
	log.Printf("OAuth2 authorize URL: %s", location)

	authorizeReq, err := http.NewRequestWithContext(ctx, "GET", location, nil)
	if err != nil {
		log.Printf("Failed to create authorize request: %v", err)
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "Failed to create authorize request"})
	}
	authorizeReq.Header.Set("Referer", "https://proxy.bmstu.ru:8443/cas/login")
	authorizeReq.Header.Set("Cache-Control", "no-cache")
	addHeadersAndCookies(authorizeReq)

	authorizeResp, err := client.Do(authorizeReq)
	if err != nil {
		log.Printf("Failed to follow OAuth2 authorize: %v", err)
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "Failed to follow OAuth2 authorize"})
	}
	defer authorizeResp.Body.Close()
	debugResponse(authorizeResp, "Authorize")

	// Шаг 6: Финальный редирект на портал с code
	location = authorizeResp.Header.Get("Location")
	if !strings.Contains(location, "code=") {
		log.Printf("Missing OAuth2 code in Location header")
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "Missing OAuth2 code"})
	}

	finalReq, err := http.NewRequestWithContext(ctx, "GET", location, nil)
	if err != nil {
		log.Printf("Failed to create final request: %v", err)
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "Failed to create final request"})
	}
	// Специальные заголовки для запроса к порталу
	finalReq.Host = "lks.bmstu.ru"
	finalReq.Header.Set("Origin", "https://lks.bmstu.ru")
	finalReq.Header.Set("Referer", "https://proxy.bmstu.ru:8443/")
	addHeadersAndCookies(finalReq)

	finalResp, err := client.Do(finalReq)
	if err != nil {
		log.Printf("Failed to complete login: %v", err)
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "Failed to complete login"})
	}
	defer finalResp.Body.Close()
	debugResponse(finalResp, "Final")

	// Сохраняем итоговые куки портала
	for _, cookie := range finalResp.Cookies() {
		if cookie.Value != "" && (strings.HasPrefix(cookie.Name, "__portal3_") ||
			cookie.Name == "PHPSESSID" || cookie.Name == "sessionid") {
			cookies[cookie.Name] = cookie.Value
		}
	}

	// Проверяем наличие нужных кук портала
	if _, ok := cookies["__portal3_login"]; !ok {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "Missing portal login cookie"})
	}
	if _, ok := cookies["__portal3_info"]; !ok {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "Missing portal info cookie"})
	}

	// Получаем профиль и фото пользователя
	profileURL := "https://lks.bmstu.ru/lks-back/api/v1/student"
	photoURL := "https://lks.bmstu.ru/lks-back/api/v1/student/photo"

	// Собираем куки для запроса к API
	apiCookies := []*http.Cookie{
		{Name: "width", Value: "2048", Path: "/", Domain: "lks.bmstu.ru"},
		{Name: "__portal3_login", Value: cookies["__portal3_login"], Path: "/", Domain: "lks.bmstu.ru"},
		{Name: "__portal3_info", Value: cookies["__portal3_info"], Path: "/", Domain: "lks.bmstu.ru"},
	}

	// Вспомогательная функция для запроса к API
	doAPIRequest := func(url string) ([]byte, error) {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", commonHeaders["User-Agent"])
		req.Header.Set("Accept", "application/json")
		for _, ck := range apiCookies {
			req.AddCookie(ck)
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		return io.ReadAll(resp.Body)
	}

	// Получаем профиль
	var profile externalProfileResponse
	profileBody, err := doAPIRequest(profileURL)
	if err != nil {
		log.Printf("Failed to fetch profile: %v", err)
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "Failed to fetch profile"})
	}
	if err := json.Unmarshal(profileBody, &profile); err != nil {
		log.Printf("Failed to parse profile: %v", err)
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "Failed to parse profile"})
	}

	// Получаем фото
	var photo externalPhotoResponse
	photoBody, err := doAPIRequest(photoURL)
	if err != nil {
		log.Printf("Failed to fetch photo: %v", err)
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "Failed to fetch photo"})
	}
	if err := json.Unmarshal(photoBody, &photo); err != nil {
		log.Printf("Failed to parse photo: %v", err)
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "Failed to parse photo"})
	}

	return c.JSON(http.StatusOK, LoginExternalResponse{
		Cookies:    cookies,
		LastName:   profile.LastName,
		FirstName:  profile.FirstName,
		MiddleName: profile.MiddleName,
		Photo:      photo.Photo,
	})
}

// extractExecution вытаскивает execution из HTML страницы логина
func extractExecution(html string) string {
	const marker = `name="execution" value="`
	idx := strings.Index(html, marker)
	if idx == -1 {
		return ""
	}
	start := idx + len(marker)
	end := strings.Index(html[start:], `"`)
	if end == -1 {
		return ""
	}
	return html[start : start+end]
}
