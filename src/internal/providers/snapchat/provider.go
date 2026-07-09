package snapchat

type Provider struct{}

func (Provider) ID() string {
	return "snapchat"
}

func (Provider) Name() string {
	return "Snapchat"
}

func (Provider) Status() string {
	return "active-first-importer"
}

func (Provider) Description() string {
	return "First provider target. The MVP desktop shell will index Snapchat zip exports without extracting everything."
}
