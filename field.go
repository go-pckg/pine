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
	boolType
)

type Field struct {
	Type    fieldType
	Key     string
	String  string
	Int64   int64
	Float64 float64
	Value   interface{}
	Err     error
}

func Int(key string, val int) Field {
	return Field{Type: intType, Key: key, Int64: int64(val)}
}
func Int8(key string, val int8) Field {
	return Field{Type: int8Type, Key: key, Int64: int64(val)}
}
func Int16(key string, val int16) Field {
	return Field{Type: int16Type, Key: key, Int64: int64(val)}
}
func Int32(key string, val int32) Field {
	return Field{Type: int32Type, Key: key, Int64: int64(val)}
}
func Int64(key string, val int64) Field {
	return Field{Type: int64Type, Key: key, Int64: val}
}
func Float32(key string, val float32) Field {
	return Field{Type: float32Type, Key: key, Float64: float64(val)}
}
func Float64(key string, val float64) Field {
	return Field{Type: float64Type, Key: key, Float64: val}
}
func Bool(key string, val bool) Field {
	var v int64
	if val {
		v = 1
	}
	return Field{Type: boolType, Key: key, Int64: v}
}
func String(key string, val string) Field {
	return Field{Type: stringType, Key: key, String: val}
}
func Time(key string, val time.Time) Field {
	return Field{Type: timeType, Key: key, Value: val}
}
func Err(err error) Field {
	return Field{Type: errorType, Key: "error", Err: err}
}

//nolint:revive,stylecheck
func Json(key string, val interface{}) Field {
	return Field{Type: jsonType, Key: key, Value: val}
}
func Interface(key string, val interface{}) Field {
	return Field{Type: interfaceType, Key: key, Value: val}
}
