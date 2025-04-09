package utils

import (
	"log"
	"net/http"
	"os"
	"time"
)

// LoggingMiddleware: 모든 HTTP 요청을 로깅하는 미들웨어
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 요청 시작 시간 기록
		startTime := time.Now()

		// 클라이언트 정보 추출
		clientIP := r.RemoteAddr
		if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			clientIP = forwardedFor
		}

		// 응답 래핑을 통해 상태 코드와 응답 크기 추적
		wrapper := &responseWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // 기본값
		}

		// 요청 방법, 경로, 클라이언트 IP 로깅
		log.Printf("[요청] %s %s FROM %s", r.Method, r.URL.Path, clientIP)

		// 요청 헤더 로깅 (디버깅 목적)
		if os.Getenv("DEBUG") == "true" {
			for name, values := range r.Header {
				log.Printf("[헤더] %s: %s", name, values)
			}
		}

		// 다음 핸들러 호출
		next.ServeHTTP(wrapper, r)

		// 요청 처리 시간 계산
		duration := time.Since(startTime)

		// 응답 정보 로깅
		log.Printf("[응답] %s %s - %d %s - %dms", r.Method, r.URL.Path, wrapper.statusCode, http.StatusText(wrapper.statusCode), duration.Milliseconds())
	})
}

// responseWrapper는 http.ResponseWriter를 래핑하여 상태 코드와 응답 크기를 추적합니다.
type responseWrapper struct {
	http.ResponseWriter
	statusCode int
	size       int
}

// WriteHeader는 상태 코드를 기록하고 원래 ResponseWriter의 WriteHeader를 호출합니다.
func (rw *responseWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write는 응답 크기를 기록하고 원래 ResponseWriter의 Write를 호출합니다.
func (rw *responseWrapper) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// CorsMiddleware: CORS 관련 헤더를 추가하는 미들웨어
func CorsMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 실제 운영환경에서는 허용할 도메인을 제한하세요.
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Fields")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		h.ServeHTTP(w, r)
	})
}
