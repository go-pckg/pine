package pine

type Hook interface {
	Run(ent *Entry, fields []Field) error
}
