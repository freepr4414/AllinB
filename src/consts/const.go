package consts

// 데이터베이스 작업 관련 상수
const (
	// ShortQueryTimeout은 간단한 쿼리 실행 시 타임아웃입니다.
	SHORT_QUERY_TIMEOUT int = 5

	// DefaultQueryTimeout은 DB 쿼리 실행 시 기본 타임아웃입니다.
	DEFAULT_QUERY_TIMEOUT int = 10

	// LongQueryTimeout은 복잡한 쿼리나 대량 데이터 작업 시 타임아웃입니다.
	LONG_QUERY_TIMEOUT int = 30

	//----------------------------------------------------------
	// ShortWorkTimeout은 간단한 실행 시 타임아웃입니다.
	SHORT_WORK_TIMEOUT int = 5

	// DefaultWorkTimeout은 실행 시 기본 타임아웃입니다.
	DEFAULT_WORK_TIMEOUT int = 10

	// LongWorkTimeout은 복잡한  작업 시 타임아웃입니다.
	LONG_WORK_TIMEOUT int = 30
)
