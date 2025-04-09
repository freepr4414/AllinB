// seat.go
package tables

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"AllinB/src/consts"
	"AllinB/src/utils"
)

// Seat 구조체는 seat_table의 각 컬럼을 매핑합니다.
// chain_code 필드 삭제됨
type Seat struct {
	AutoIncrement         int    `json:"auto_increment" db:"auto_increment"`
	CompanyCode           int    `json:"company_code"`
	SeatCode              int    `json:"seat_code"`
	SeatTitle             string `json:"seat_title"`
	TitleBackgroundColor  string `json:"title_background_color"`
	TitleTextColor        string `json:"title_text_color"`
	SeatBackgroundColor   string `json:"seat_background_color"`
	SeatTop               int    `json:"seat_top"`
	SeatLeft              int    `json:"seat_left"`
	SeatWidth             int    `json:"seat_width"`
	SeatHeight            int    `json:"seat_height"`
	Gender                int    `json:"gender"`
	Waiting               int    `json:"waiting"`
	Release               int    `json:"release"`
	HideTitle             int    `json:"hide_title"`
	TransparentBackground int    `json:"transparent_background"`
	HideBorder            int    `json:"hide_border"`
	KioskDisabled         int    `json:"kiosk_disabled"`
	PowerControl          int    `json:"power_control"`
	BreakerNumber         int    `json:"breaker_number"`
}

// RegisterSeatRoutes는 seat_table 관련 엔드포인트를 등록합니다.
func RegisterSeatRoutes(r *mux.Router) {
	r.HandleFunc("/seats", GetSeats).Methods("GET")
	r.HandleFunc("/seats/{seat_code}", GetSeat).Methods("GET")
	r.HandleFunc("/seats", CreateSeat).Methods("POST")
	// UpdateSeat은 전체/부분 업데이트를 모두 지원합니다.
	r.HandleFunc("/seats/{seat_code}", UpdateSeat).Methods("PUT")
	r.HandleFunc("/seats/{seat_code}", DeleteSeat).Methods("DELETE")
}

// GetSeats: "X-Fields" 헤더에 지정된 필드만 조회하거나 전체 필드를 조회합니다.
// URL 쿼리 파라미터를 통해 필터링 기능도 지원합니다.
func GetSeats(w http.ResponseWriter, r *http.Request) {
	// 요청 컨텍스트에 10초 타임아웃 설정
	timeout := time.Duration(consts.DEFAULT_QUERY_TIMEOUT) * time.Second
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	allowedFields := []string{
		"auto_increment", "company_code", "seat_code", "seat_title",
		"title_background_color", "title_text_color", "seat_background_color",
		"seat_top", "seat_left", "seat_width", "seat_height",
		"gender", "waiting", "release", "hide_title",
		"transparent_background", "hide_border", "kiosk_disabled",
		"power_control", "breaker_number",
	}

	// 필드 선택 처리
	fieldsHeader := r.Header.Get("X-Fields")
	var fields []string
	if fieldsHeader != "" {
		requested := strings.Split(fieldsHeader, ",")
		allowedSet := make(map[string]bool)
		for _, f := range allowedFields {
			allowedSet[f] = true
		}
		for _, f := range requested {
			f = strings.TrimSpace(f)
			if allowedSet[f] {
				fields = append(fields, f)
			}
		}
		if len(fields) == 0 {
			fields = allowedFields
		}
	} else {
		fields = allowedFields
	}

	// 필터링 조건 처리
	filters := []string{}
	args := []interface{}{}
	paramIdx := 1

	// 지원하는 필터 파라미터 목록 (chain_code 제거됨)
	filterParams := map[string]string{
		"company_code":   "company_code",
		"seat_code":      "seat_code",
		"gender":         "gender",
		"waiting":        "waiting",
		"release":        "release",
		"kiosk_disabled": "kiosk_disabled",
		"power_control":  "power_control",
	}

	// URL 쿼리 파라미터에서 필터 조건 추출
	for param, dbField := range filterParams {
		if value := r.URL.Query().Get(param); value != "" {
			filters = append(filters, fmt.Sprintf("%s = $%d", dbField, paramIdx))
			args = append(args, value)
			paramIdx++
		}
	}

	// 검색 기능 추가 (seat_title에 대한 부분 검색)
	if search := r.URL.Query().Get("search"); search != "" {
		filters = append(filters, fmt.Sprintf("seat_title LIKE $%d", paramIdx))
		args = append(args, "%"+search+"%")
		paramIdx++
	}

	// 쿼리 구성
	query := "SELECT " + strings.Join(fields, ", ") + " FROM seat_table"
	if len(filters) > 0 {
		query += " WHERE " + strings.Join(filters, " AND ")
	}

	// 정렬 옵션 처리
	if sort := r.URL.Query().Get("sort"); sort != "" {
		direction := "ASC"
		if strings.HasPrefix(sort, "-") {
			sort = sort[1:]
			direction = "DESC"
		}

		// 허용된 정렬 필드인지 확인
		allowedSortFields := map[string]bool{
			"seat_code":      true,
			"seat_title":     true,
			"auto_increment": true,
		}

		if allowedSortFields[sort] {
			query += fmt.Sprintf(" ORDER BY %s %s", sort, direction)
		}
	} else {
		// 기본 정렬은 seat_code 기준
		query += " ORDER BY seat_code ASC"
	}

	// 로깅 추가
	log.Printf("실행 쿼리: %s, 인자: %v", query, args)

	// 쿼리 실행
	var rows *sql.Rows
	var err error
	if len(args) > 0 {
		rows, err = utils.DB.QueryContext(ctx, query, args...)
	} else {
		rows, err = utils.DB.QueryContext(ctx, query)
	}

	if err != nil {
		log.Printf("데이터베이스 쿼리 오류: %v", err)
		http.Error(w, "데이터 조회 중 오류가 발생했습니다", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// 결과 처리
	columns, err := rows.Columns()
	if err != nil {
		log.Printf("컬럼 정보 조회 오류: %v", err)
		http.Error(w, "데이터 처리 중 오류가 발생했습니다", http.StatusInternalServerError)
		return
	}

	result := []map[string]interface{}{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			log.Printf("행 스캔 오류: %v", err)
			http.Error(w, "데이터 처리 중 오류가 발생했습니다", http.StatusInternalServerError)
			return
		}

		rowMap := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]
			if b, ok := val.([]byte); ok {
				v = string(b)
			} else {
				v = val
			}
			rowMap[col] = v
		}
		result = append(result, rowMap)
	}

	// 결과 반환
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("JSON 인코딩 오류: %v", err)
		http.Error(w, "응답 생성 중 오류가 발생했습니다", http.StatusInternalServerError)
		return
	}
}

// GetSeat: 단일 seat를 전체 필드로 조회합니다.
func GetSeat(w http.ResponseWriter, r *http.Request) {
	timeout := time.Duration(consts.DEFAULT_QUERY_TIMEOUT) * time.Second
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	vars := mux.Vars(r)
	seatCode, err := strconv.Atoi(vars["seat_code"])
	if err != nil {
		http.Error(w, "잘못된 seat_code", http.StatusBadRequest)
		return
	}

	var seat Seat
	err = utils.DB.QueryRowContext(ctx, `
        SELECT auto_increment, company_code, seat_code, seat_title,
               title_background_color, title_text_color, seat_background_color,
               seat_top, seat_left, seat_width, seat_height,
               gender, waiting, release, hide_title,
               transparent_background, hide_border, kiosk_disabled,
               power_control, breaker_number
        FROM seat_table WHERE seat_code = $1`, seatCode).
		Scan(&seat.AutoIncrement, &seat.CompanyCode, &seat.SeatCode, &seat.SeatTitle,
			&seat.TitleBackgroundColor, &seat.TitleTextColor, &seat.SeatBackgroundColor,
			&seat.SeatTop, &seat.SeatLeft, &seat.SeatWidth, &seat.SeatHeight,
			&seat.Gender, &seat.Waiting, &seat.Release, &seat.HideTitle,
			&seat.TransparentBackground, &seat.HideBorder, &seat.KioskDisabled,
			&seat.PowerControl, &seat.BreakerNumber)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Seat를 찾을 수 없습니다.", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(seat)
}

// CreateSeat: 새로운 seat을 생성합니다.
func CreateSeat(w http.ResponseWriter, r *http.Request) {
	timeout := time.Duration(consts.DEFAULT_QUERY_TIMEOUT) * time.Second
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	var seat Seat
	if err := json.NewDecoder(r.Body).Decode(&seat); err != nil {
		http.Error(w, "잘못된 요청 데이터", http.StatusBadRequest)
		return
	}

	// 기본값 설정
	if seat.SeatWidth == 0 {
		seat.SeatWidth = 100
	}
	if seat.SeatHeight == 0 {
		seat.SeatHeight = 100
	}
	if seat.SeatBackgroundColor == "" {
		seat.SeatBackgroundColor = "#FFFFFF"
	}
	if seat.TitleBackgroundColor == "" {
		seat.TitleBackgroundColor = "#000000"
	}
	if seat.TitleTextColor == "" {
		seat.TitleTextColor = "#FFFFFF"
	}

	query := `
        INSERT INTO seat_table 
        (company_code, seat_code, seat_title, 
         title_background_color, title_text_color, seat_background_color,
         seat_top, seat_left, seat_width, seat_height,
         gender, waiting, release, hide_title,
         transparent_background, hide_border, kiosk_disabled,
         power_control, breaker_number)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
                $12, $13, $14, $15, $16, $17, $18, $19)
    `
	// 시작 시간 로깅
	startTime := time.Now()
	log.Printf("Seat 생성 요청 시작: %+v", seat)

	_, err := utils.DB.ExecContext(ctx, query,
		seat.CompanyCode, seat.SeatCode, seat.SeatTitle,
		seat.TitleBackgroundColor, seat.TitleTextColor, seat.SeatBackgroundColor,
		seat.SeatTop, seat.SeatLeft, seat.SeatWidth, seat.SeatHeight,
		seat.Gender, seat.Waiting, seat.Release, seat.HideTitle,
		seat.TransparentBackground, seat.HideBorder, seat.KioskDisabled,
		seat.PowerControl, seat.BreakerNumber)

	// 실행 시간 및 오류 로깅
	duration := time.Since(startTime)
	log.Printf("쿼리 실행 시간: %v", duration)

	if err != nil {
		log.Printf("DB 오류: %v", err)
		if strings.Contains(err.Error(), "duplicate key") {
			http.Error(w, "이미 존재하는 seat code입니다", http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(seat)
}

// UpdateSeat: 제공된 JSON 데이터에 따라 전체 또는 일부 필드만 업데이트합니다.
func UpdateSeat(w http.ResponseWriter, r *http.Request) {
	timeout := time.Duration(consts.DEFAULT_QUERY_TIMEOUT) * time.Second
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	vars := mux.Vars(r)
	seatCode, err := strconv.Atoi(vars["seat_code"])
	if err != nil {
		http.Error(w, "잘못된 seat_code", http.StatusBadRequest)
		return
	}

	// 요청 본문을 map[string]interface{}로 디코딩하여, 제공된 필드만 업데이트합니다.
	var updateData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		http.Error(w, "잘못된 요청 데이터", http.StatusBadRequest)
		return
	}
	// JSON에 "seat_code"가 있다면 URL과 일치하는지 확인 후 제거합니다.
	if v, ok := updateData["seat_code"]; ok {
		switch v := v.(type) {
		case float64:
			if int(v) != seatCode {
				http.Error(w, "URL과 body의 seat_code가 다릅니다.", http.StatusBadRequest)
				return
			}
		default:
			http.Error(w, "잘못된 seat_code 값", http.StatusBadRequest)
			return
		}
		delete(updateData, "seat_code")
	}
	if len(updateData) == 0 {
		http.Error(w, "업데이트할 필드가 없습니다.", http.StatusBadRequest)
		return
	}

	allowed := map[string]bool{
		"company_code":           true,
		"seat_title":             true,
		"seat_background_color":  true,
		"seat_top":               true,
		"seat_left":              true,
		"seat_width":             true,
		"seat_height":            true,
		"title_background_color": true,
		"title_text_color":       true,
		"gender":                 true,
		"waiting":                true,
		"release":                true,
		"hide_title":             true,
		"transparent_background": true,
		"hide_border":            true,
		"kiosk_disabled":         true,
		"power_control":          true,
		"breaker_number":         true,
	}
	updates := []string{}
	args := []interface{}{} // 올바른 방식으로 빈 인터페이스 슬라이스 초기화
	idx := 1
	for key, value := range updateData {
		if !allowed[key] {
			continue
		}
		updates = append(updates, key+" = $"+strconv.Itoa(idx))
		args = append(args, value)
		idx++
	}
	if len(updates) == 0 {
		http.Error(w, "유효한 업데이트 필드가 없습니다.", http.StatusBadRequest)
		return
	}

	query := "UPDATE seat_table SET " + strings.Join(updates, ", ") + " WHERE seat_code = $" + strconv.Itoa(idx)
	args = append(args, seatCode)
	_, err = utils.DB.ExecContext(ctx, query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 업데이트 후 비동기 작업 큐에 작업을 넣어 (예: seat 업데이트 알림) 백그라운드 처리를 수행합니다.
	job := utils.Job{
		Name: "SeatUpdated",
		Data: map[string]interface{}{
			"seat_code": seatCode,
			"time":      time.Now(),
		},
	}
	if utils.EnqueueJobHandler != nil {
		utils.EnqueueJobHandler(job)
	}

	// 업데이트된 seat을 조회하여 반환합니다.
	var seat Seat
	err = utils.DB.QueryRowContext(ctx, `
        SELECT auto_increment, company_code, seat_code, seat_title,
               title_background_color, title_text_color, seat_background_color,
               seat_top, seat_left, seat_width, seat_height,
               gender, waiting, release, hide_title,
               transparent_background, hide_border, kiosk_disabled,
               power_control, breaker_number
        FROM seat_table WHERE seat_code = $1`, seatCode).
		Scan(&seat.AutoIncrement, &seat.CompanyCode, &seat.SeatCode, &seat.SeatTitle,
			&seat.TitleBackgroundColor, &seat.TitleTextColor, &seat.SeatBackgroundColor,
			&seat.SeatTop, &seat.SeatLeft, &seat.SeatWidth, &seat.SeatHeight,
			&seat.Gender, &seat.Waiting, &seat.Release, &seat.HideTitle,
			&seat.TransparentBackground, &seat.HideBorder, &seat.KioskDisabled,
			&seat.PowerControl, &seat.BreakerNumber)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(seat)
}

// DeleteSeat: seat을 삭제합니다.
func DeleteSeat(w http.ResponseWriter, r *http.Request) {
	timeout := time.Duration(consts.DEFAULT_QUERY_TIMEOUT) * time.Second
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	vars := mux.Vars(r)
	seatCode, err := strconv.Atoi(vars["seat_code"])
	if err != nil {
		http.Error(w, "잘못된 seat_code", http.StatusBadRequest)
		return
	}
	_, err = utils.DB.ExecContext(ctx, "DELETE FROM seat_table WHERE seat_code = $1", seatCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
