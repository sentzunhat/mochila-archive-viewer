package instagram

type Provider struct{}

func (Provider) ID() string {
	return "instagram"
}

func (Provider) Name() string {
	return "Instagram"
}

func (Provider) Status() string {
	return "planned"
}

func (Provider) Description() string {
	return "Reserved for the post-Snapchat importer lane and shared archive interfaces."
}
