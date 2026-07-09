package facebook

type Provider struct{}

func (Provider) ID() string {
	return "facebook"
}

func (Provider) Name() string {
	return "Facebook"
}

func (Provider) Status() string {
	return "planned"
}

func (Provider) Description() string {
	return "Prepared for a future provider module once the Snapchat indexing flow reaches parity."
}
