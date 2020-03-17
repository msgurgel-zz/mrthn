package auth

import (
	"github.com/msgurgel/marathon/pkg/environment"
)

type Types struct {
	Oauth2 OAuth2
}

func ConfigureTypes(configs *environment.MarathonConfig) Types {
	return Types{
		Oauth2: NewOAuth2(configs),
	}
}
