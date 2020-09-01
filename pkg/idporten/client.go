package idporten

type Client struct{}

func (c Client) Register() {
	panic("implement me")
}

func (c Client) List() {
	panic("implement me")
}

func (c Client) Update() {
	panic("implement me")
}

func (c Client) Delete() {
	panic("implement me")
}

func (c Client) RegisterKeys() {
	panic("implement me")
}

func NewClient() Client {
	return Client{}
}
