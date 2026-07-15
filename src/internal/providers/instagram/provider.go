package instagram

type Provider struct{}

func (Provider) ID() string   { return "instagram" }
func (Provider) Name() string { return "Instagram" }
func (Provider) Status() string { return "active" }
func (Provider) Description() string {
	return "Indexes Instagram 'Your Instagram Activity' exports — DMs, photos, and videos."
}
