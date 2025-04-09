// 이 파일을 path.go 또는 utils/root_path.go로 이름 변경 권장
package utils

import (
	"errors"
	"os"
	"path/filepath"
)

// FindProjectRoot 프로젝트 루트 디렉토리를 찾는 함수입니다.
// 함수 이름을 대문자로 시작하여 외부에서 접근 가능하게 만듭니다.
func FindProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		// .env 파일이 있는지 확인
		if _, err := os.Stat(filepath.Join(dir, ".env")); err == nil {
			return dir, nil
		}

		// 상위 디렉토리로 이동
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", errors.New("프로젝트 루트 디렉토리를 찾을 수 없습니다")
}
