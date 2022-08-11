package pine

import "time"

type fieldType int

const (
	interfaceType fieldType = iota
	jsonType
	stringType
	intType
	int8Type
	int16Type
	int32Type
	int64Type
	float32Type
	float64Type
	timeType
	errorType
)

type Field struct {
	tp      fieldType
	key     string
	string  string
	int64   int64
	float64 float64
	value   interface{}
	err     error
}

func Int(key string, val int) Field {
	return Field{tp: intType, key: key, int64: int64(val)}
}
func Int8(key string, val int8) Field {
	return Field{tp: int8Type, key: key, int64: int64(val)}
}
func Int16(key string, val int16) Field {
	return Field{tp: int16Type, key: key, int64: int64(val)}
}
func Int32(key string, val int32) Field {
	return Field{tp: int32Type, key: key, int64: int64(val)}
}
func Int64(key string, val int64) Field {
	return Field{tp: int64Type, key: key, int64: val}
}
func Float32(key string, val float32) Field {
	return Field{tp: float32Type, key: key, float64: float64(val)}
}
func Float64(key string, val float64) Field {
	return Field{tp: float64Type, key: key, float64: val}
}
func String(key string, val string) Field {
	return Field{tp: stringType, key: key, string: val}
}
func Time(key string, val time.Time) Field {
	return Field{tp: timeType, key: key, value: val}
}
func Err(err error) Field {
	return Field{tp: errorType, key: "error", err: err}
}

//nolint:revive,stylecheck
func Json(key string, val interface{}) Field {
	return Field{tp: jsonType, key: key, value: val}
}
func Interface(key string, val interface{}) Field {
	return Field{tp: interfaceType, key: key, value: val}
}
