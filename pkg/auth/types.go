package auth

import (
	"github.com/msgurgel/marathon/pkg/environment"
)

type Types struct {
	Oauth2 Oauth2
}

func (a *Types) Init(configs *environment.MarathonConfig) {
	a.Oauth2 = NewOAuth2(configs)
}
