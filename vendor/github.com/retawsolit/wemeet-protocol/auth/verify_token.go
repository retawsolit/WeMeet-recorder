package auth

import (
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/retawsolit/wemeet-protocol/wemeet"
)

// VerifyWeMeetAccessToken can be use to verify WeMeet access token
func VerifyWeMeetAccessToken(apiKey, secret, token string, withTime bool) (*wemeet.WeMeetTokenClaims, error) {
	tok, err := jwt.ParseSigned(token, []jose.SignatureAlgorithm{jose.HS256})
	if err != nil {
		return nil, err
	}

	out := jwt.Claims{}
	claims := wemeet.WeMeetTokenClaims{}
	if err = tok.Claims([]byte(secret), &out, &claims); err != nil {
		return nil, err
	}

	exp := jwt.Expected{Issuer: apiKey, Subject: claims.UserId}
	if withTime {
		exp.Time = time.Now().UTC()
	} else {
		// so the token will not be expired
		out.Expiry = nil
		out.NotBefore = nil
		out.IssuedAt = nil
	}

	if err = out.Validate(exp); err != nil {
		return nil, err
	}
	claims.UserId = out.Subject

	return &claims, nil
}
