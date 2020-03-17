package auth

import (
	"github.com/msgurgel/marathon/pkg/environment"
)

type Types struct {
	Oauth2 OAuth2
}

func (a *Types) GetAuthTypes(configs *environment.MarathonConfig) {
	a.Oauth2 = NewOAuth2(configs)
}
