package pine

type LevelValue struct {
	Value *Level
}

func NewLevelValue(lvl Level) *LevelValue {
	return &LevelValue{Value: &lvl}
}

func (l LevelValue) SetLevel(lvl Level) {
	*l.Value = lvl
}

func (l LevelValue) GetLevel() Level {
	return *l.Value
}
