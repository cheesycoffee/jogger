# Jogger

**Jogger** is a lightweight tracing-aware logging library for Go. It builds on top of `uber-go/zap` and uses concepts inspired by `Jaeger` and `OpenTracing` to trace spans and contextual logs across a request lifecycle.

---

## ✨ Features

* Zap-based structured logging
* Span-based tracing abstraction
* Context propagation with request IDs
* Environment-specific encoders (production & development)

---

## 🧩 Compatibility

- **Go 1.12 or later**  
  This library is compatible with Go 1.12+ and does not rely on generics or Go modules features introduced after that version.

---

## 📦 Installation

```bash
go get github.com/cheesycoffee/jogger
```

---

## 🚀 Usage

### Add manually request ID to context or via middlware
manually added :
```go
ctx := context.Background()
ctx = jogger.WithRequestID(ctx, "abc-123")
```

REST middleware :
```go
    func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Extract or generate request ID
		var incomingID string

		// check existing request id within header
		requestIDHeaders := []string{"X-Request-ID", "X-Correlation-ID", "X-Amzn-Trace-Id", "uber-trace-id"}
		for _, k := range requestIDHeaders {
			incomingID = r.Header.Get(k)
			if incomingID != "" {
				break
			}
		}

		// generate new request id if does not exist in header
		if incomingID == "" {
			incomingID = uuid.New().String()
		}

		ctx := jogger.WithRequestID(r.Context(), incomingID)

		// Read and restore body
		var bodyMap map[string]interface{}
		if r.Body != nil {
			bodyBytes, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			_ = json.Unmarshal(bodyBytes, &bodyMap)

			// remove sensitive data from loggin
			credentials := []string{"password", "creditCard"}
			for _, k := range credentials {
				delete(bodyMap, k)
			}
		}

		// Sanitize Authorization header
		headers := make(map[string]interface{})
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			headers["Authorization"] = "Bearer ***redacted***"
		} else if strings.HasPrefix(authHeader, "Basic ") {
			headers["Authorization"] = "Basic ***redacted***"
		}

		jogger.Info(ctx, "Incoming Request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Any("body", bodyMap),
			zap.Any("headers", headers),
			zap.String("requestID", incomingID),
		)

		// Replace context in request
		r = r.WithContext(ctx)

		// Capture response status
		rr := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		err := catchPanic(func() {
			next.ServeHTTP(rr, r)
		})

		duration := time.Since(start).Milliseconds()

		jogger.Info(ctx, "Completed Request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", rr.statusCode),
			zap.Int64("durationMs", duration),
			zap.String("requestID", incomingID),
			zap.Error(err),
		)
	})
}

// responseRecorder wraps http.ResponseWriter to capture status code
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

// catchPanic recovers from panics and returns an error
func catchPanic(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", e)
		}
	}()
	fn()
	return nil
}
```

### Start using span

```go
func (r *repository) GetAllUsers(ctx context.Context, userParam UserParam) (userList []model.User, err error) {
    span, ctx := jogger.StartSpan(ctx, "Repository:GetAllUsers")
    defer span.Finish(&err) // pass error pointer to capture the last named var result
    

    query := fmt.Sprintf(`SELECT id, name, created_at, updated_at FROM users LIMIT %d OFFSET %d`, userParam.PageSize, userParam.Offset)
    span.SetTag("query", query)

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var user model.User
		if err = rows.Scan(&user.ID, &user.Name, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return
		}
		userList = append(userList, user)
	}

	err = rows.Err()

    return
}
```

### 3. Log messages with context

```go
jogger.Info(ctx, "retrieving users", zap.Int("user_count", 42))
jogger.Warn(ctx, "slow response")
jogger.Error(ctx, "query failed", zap.Error(err))
```

---

## 📁 Example Console Log Output

* Logs include `requestID`, `spanID`, `duration`, and any custom tags

```log
2025-05-12T08:19:25.126+0700    INFO    Incoming Request        {"requestID": "39164279-e9ad-4312-9cc5-fb0002ec80cb", "method": "GET", "path": "/v1/users", "body": null, "headers": {}, "requestID": "39164279-e9ad-4312-9cc5-fb0002ec80cb"}
2025-05-12T08:19:25.126+0700    ERROR   span finished with error        {"span": "Repository:GetAllUsers", "spanID": "7a888a3d-a94b-4a0c-b6f6-021a398b7de1", "requestID": "39164279-e9ad-4312-9cc5-fb0002ec80cb", "duration": 0.000000201}
2025-05-12T08:19:25.126+0700    ERROR   span finished with error        {"span": "Usecase:GetAllUsers", "spanID": "1cde182f-d89f-4676-b0dc-bdf674b27f41", "requestID": "39164279-e9ad-4312-9cc5-fb0002ec80cb", "duration": 0.000148596, "error": "sql: Scan error on column index 2, name \"created_at\": unsupported Scan, storing driver.Value type int64 into type *time.Time"}
2025-05-12T08:19:25.126+0700    ERROR   span finished with error        {"span": "handlerRest:GetAllUsers", "spanID": "19d75057-f909-443d-b8d6-35bb82816499", "requestID": "39164279-e9ad-4312-9cc5-fb0002ec80cb", "duration": 0.000165527}
2025-05-12T08:19:25.126+0700    INFO    Completed Request       {"requestID": "39164279-e9ad-4312-9cc5-fb0002ec80cb", "method": "GET", "path": "/v1/users", "status": 500, "durationMs": 0, "requestID": "39164279-e9ad-4312-9cc5-fb0002ec80cb"}
```

---

## 🧪 Testing

Run unit tests:

```bash
go test -v
```

---