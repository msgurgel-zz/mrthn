package auth

type Types struct {
	Oauth2 Oauth2
}

func (a *Types) Init() {
	a.Oauth2 = NewOAuth2()
}
