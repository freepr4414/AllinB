// room.go
package tables

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"AllinB/src/consts"
	"AllinB/src/utils"
)

// Room 구조체는 room_table의 각 컬럼을 매핑합니다.
// chain_code 필드 삭제됨
type Room struct {
	AutoIncrement         int    `json:"auto_increment" db:"auto_increment"`
	CompanyCode           int    `json:"company_code"`
	RoomCode              int    `json:"room_code"`
	RoomTitle             string `json:"room_title"`
	TitleBackgroundColor  string `json:"title_background_color"`
	TitleTextColor        string `json:"title_text_color"`
	RoomBackgroundColor   string `json:"room_background_color"`
	RoomTop               int    `json:"room_top"`
	RoomLeft              int    `json:"room_left"`
	RoomWidth             int    `json:"room_width"`
	RoomHeight            int    `json:"room_height"`
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

// RegisterRoomRoutes는 room_table 관련 엔드포인트를 등록합니다.
func RegisterRoomRoutes(r *mux.Router) {
	r.HandleFunc("/rooms", GetRooms).Methods("GET")
	r.HandleFunc("/rooms/{room_code}", GetRoom).Methods("GET")
	r.HandleFunc("/rooms", CreateRoom).Methods("POST")
	// UpdateRoom은 전체/부분 업데이트를 모두 지원합니다.
	r.HandleFunc("/rooms/{room_code}", UpdateRoom).Methods("PUT")
	r.HandleFunc("/rooms/{room_code}", DeleteRoom).Methods("DELETE")
}

// GetRooms: "X-Fields" 헤더에 지정된 필드만 조회하거나 전체 필드를 조회합니다.
func GetRooms(w http.ResponseWriter, r *http.Request) {
	// 요청 컨텍스트에 10초 타임아웃 설정
	timeout := time.Duration(consts.DEFAULT_QUERY_TIMEOUT) * time.Second
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	allowedFields := []string{
		"auto_increment", "company_code", "room_code", "room_title",
		"title_background_color", "title_text_color", "room_background_color",
		"room_top", "room_left", "room_width", "room_height",
		"gender", "waiting", "release", "hide_title",
		"transparent_background", "hide_border", "kiosk_disabled",
		"power_control", "breaker_number",
	}

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

	query := "SELECT " + strings.Join(fields, ", ") + " FROM room_table"
	rows, err := utils.DB.QueryContext(ctx, query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
			http.Error(w, err.Error(), http.StatusInternalServerError)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetRoom: 단일 room을 전체 필드로 조회합니다.
func GetRoom(w http.ResponseWriter, r *http.Request) {
	timeout := time.Duration(consts.DEFAULT_QUERY_TIMEOUT) * time.Second
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	vars := mux.Vars(r)
	roomCode, err := strconv.Atoi(vars["room_code"])
	if err != nil {
		http.Error(w, "잘못된 room_code", http.StatusBadRequest)
		return
	}

	var room Room
	err = utils.DB.QueryRowContext(ctx, `
		SELECT auto_increment, company_code, room_code, room_title,
		       title_background_color, title_text_color, room_background_color,
		       room_top, room_left, room_width, room_height,
		       gender, waiting, release, hide_title,
		       transparent_background, hide_border, kiosk_disabled,
		       power_control, breaker_number
		FROM room_table WHERE room_code = $1`, roomCode).
		Scan(&room.AutoIncrement, &room.CompanyCode, &room.RoomCode, &room.RoomTitle,
			&room.TitleBackgroundColor, &room.TitleTextColor, &room.RoomBackgroundColor,
			&room.RoomTop, &room.RoomLeft, &room.RoomWidth, &room.RoomHeight,
			&room.Gender, &room.Waiting, &room.Release, &room.HideTitle,
			&room.TransparentBackground, &room.HideBorder, &room.KioskDisabled,
			&room.PowerControl, &room.BreakerNumber)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Room을 찾을 수 없습니다.", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(room)
}

// CreateRoom: 새로운 room을 생성합니다.
func CreateRoom(w http.ResponseWriter, r *http.Request) {
	timeout := time.Duration(consts.DEFAULT_QUERY_TIMEOUT) * time.Second
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	var room Room
	if err := json.NewDecoder(r.Body).Decode(&room); err != nil {
		http.Error(w, "잘못된 요청 데이터", http.StatusBadRequest)
		return
	}

	// 기본값 설정
	if room.RoomWidth == 0 {
		room.RoomWidth = 100
	}
	if room.RoomHeight == 0 {
		room.RoomHeight = 100
	}
	if room.RoomBackgroundColor == "" {
		room.RoomBackgroundColor = "#FFFFFF"
	}
	if room.TitleBackgroundColor == "" {
		room.TitleBackgroundColor = "#000000"
	}
	if room.TitleTextColor == "" {
		room.TitleTextColor = "#FFFFFF"
	}

	query := `
		INSERT INTO room_table 
		(company_code, room_code, room_title, 
		 title_background_color, title_text_color, room_background_color,
		 room_top, room_left, room_width, room_height,
		 gender, waiting, release, hide_title,
		 transparent_background, hide_border, kiosk_disabled,
		 power_control, breaker_number)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
		        $12, $13, $14, $15, $16, $17, $18, $19)
	`
	// 시작 시간 로깅
	startTime := time.Now()
	log.Printf("Room 생성 요청 시작: %+v", room)

	_, err := utils.DB.ExecContext(ctx, query,
		room.CompanyCode, room.RoomCode, room.RoomTitle,
		room.TitleBackgroundColor, room.TitleTextColor, room.RoomBackgroundColor,
		room.RoomTop, room.RoomLeft, room.RoomWidth, room.RoomHeight,
		room.Gender, room.Waiting, room.Release, room.HideTitle,
		room.TransparentBackground, room.HideBorder, room.KioskDisabled,
		room.PowerControl, room.BreakerNumber)

	// 실행 시간 및 오류 로깅
	duration := time.Since(startTime)
	log.Printf("쿼리 실행 시간: %v", duration)

	if err != nil {
		log.Printf("DB 오류: %v", err)
		if strings.Contains(err.Error(), "duplicate key") {
			http.Error(w, "이미 존재하는 room code입니다", http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(room)
}

// UpdateRoom: 제공된 JSON 데이터에 따라 전체 또는 일부 필드만 업데이트합니다.
func UpdateRoom(w http.ResponseWriter, r *http.Request) {
	timeout := time.Duration(consts.DEFAULT_QUERY_TIMEOUT) * time.Second
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	vars := mux.Vars(r)
	roomCode, err := strconv.Atoi(vars["room_code"])
	if err != nil {
		http.Error(w, "잘못된 room_code", http.StatusBadRequest)
		return
	}

	// 요청 본문을 map[string]interface{}로 디코딩하여, 제공된 필드만 업데이트합니다.
	var updateData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		http.Error(w, "잘못된 요청 데이터", http.StatusBadRequest)
		return
	}
	// JSON에 "room_code"가 있다면 URL과 일치하는지 확인 후 제거합니다.
	if v, ok := updateData["room_code"]; ok {
		switch v := v.(type) {
		case float64:
			if int(v) != roomCode {
				http.Error(w, "URL과 body의 room_code가 다릅니다.", http.StatusBadRequest)
				return
			}
		default:
			http.Error(w, "잘못된 room_code 값", http.StatusBadRequest)
			return
		}
		delete(updateData, "room_code")
	}
	if len(updateData) == 0 {
		http.Error(w, "업데이트할 필드가 없습니다.", http.StatusBadRequest)
		return
	}

	allowed := map[string]bool{
		"company_code":           true,
		"room_title":             true,
		"room_background_color":  true,
		"room_top":               true,
		"room_left":              true,
		"room_width":             true,
		"room_height":            true,
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
	args := []interface{}{}
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

	query := "UPDATE room_table SET " + strings.Join(updates, ", ") + " WHERE room_code = $" + strconv.Itoa(idx)
	args = append(args, roomCode)
	_, err = utils.DB.ExecContext(ctx, query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 업데이트 후 비동기 작업 큐에 작업을 넣어 (예: room 업데이트 알림) 백그라운드 처리를 수행합니다.
	job := utils.Job{
		Name: "RoomUpdated",
		Data: map[string]interface{}{
			"room_code": roomCode,
			"time":      time.Now(),
		},
	}
	if utils.EnqueueJobHandler != nil {
		utils.EnqueueJobHandler(job)
	}

	// 업데이트된 room을 조회하여 반환합니다.
	var room Room
	err = utils.DB.QueryRowContext(ctx, `
		SELECT auto_increment, company_code, room_code, room_title,
		       title_background_color, title_text_color, room_background_color,
		       room_top, room_left, room_width, room_height,
		       gender, waiting, release, hide_title,
		       transparent_background, hide_border, kiosk_disabled,
		       power_control, breaker_number
		FROM room_table WHERE room_code = $1`, roomCode).
		Scan(&room.AutoIncrement, &room.CompanyCode, &room.RoomCode, &room.RoomTitle,
			&room.TitleBackgroundColor, &room.TitleTextColor, &room.RoomBackgroundColor,
			&room.RoomTop, &room.RoomLeft, &room.RoomWidth, &room.RoomHeight,
			&room.Gender, &room.Waiting, &room.Release, &room.HideTitle,
			&room.TransparentBackground, &room.HideBorder, &room.KioskDisabled,
			&room.PowerControl, &room.BreakerNumber)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(room)
}

// DeleteRoom: room을 삭제합니다.
func DeleteRoom(w http.ResponseWriter, r *http.Request) {
	timeout := time.Duration(consts.DEFAULT_QUERY_TIMEOUT) * time.Second
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	vars := mux.Vars(r)
	roomCode, err := strconv.Atoi(vars["room_code"])
	if err != nil {
		http.Error(w, "잘못된 room_code", http.StatusBadRequest)
		return
	}
	_, err = utils.DB.ExecContext(ctx, "DELETE FROM room_table WHERE room_code = $1", roomCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
