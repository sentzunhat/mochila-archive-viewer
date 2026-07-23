package facebook

type Provider struct{}

func (Provider) ID() string {
	return "facebook"
}

func (Provider) Name() string {
	return "Facebook"
}

func (Provider) Status() string {
	return "active"
}

func (Provider) Description() string {
	return "Facebook / Messenger — export indexer + conversation parser."
}
