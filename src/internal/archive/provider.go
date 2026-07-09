package archive

type Provider interface {
	ID() string
	Name() string
	Status() string
	Description() string
}
