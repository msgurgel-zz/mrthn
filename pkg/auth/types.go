package auth

import (
	"github.com/msgurgel/mrthn/pkg/environment"
)

type Types struct {
	Oauth2 OAuth2
}

func ConfigureTypes(configs *environment.MrthnConfig) Types {
	return Types{
		Oauth2: NewOAuth2(configs),
	}
}
