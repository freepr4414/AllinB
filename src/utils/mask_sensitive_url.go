package utils

import "strings"

// MaskSensitiveURL은 URL의 민감한 정보를 가립니다.
func MaskSensitiveURL(url string) string {
	if url == "" {
		return ""
	}
	parts := strings.Split(url, "@")
	if len(parts) < 2 {
		return "[비표준 URL 형식]"
	}

	credentials := strings.Split(parts[0], ":")
	if len(credentials) < 3 {
		return "[비표준 URL 형식]"
	}

	return credentials[0] + ":" + credentials[1] + ":******@" + parts[1]
}
