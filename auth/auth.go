package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type contextKey struct {
	name string
}

const jwtkey = "gkhsession"

var (
	claimsKey = &contextKey{"USER_CLAIMS"}
)

type AuthKey struct {
	URL                 string
	AuthorizationHeader string
	Cookie              string
}

// AuthKey always initiate through function NewAuthKey
func NewAuthKey() AuthKey {
	var a AuthKey
	a.URL = jwtkey
	a.AuthorizationHeader = jwtkey
	a.Cookie = jwtkey
	return a
}

func getJWT(r *http.Request, a AuthKey) (string, error) {
	jwtToken := r.URL.Query().Get(a.URL)

	if jwtToken == "" {
		jwtToken = r.Header.Get(a.AuthorizationHeader)
		if jwtToken != "" && (len(jwtToken) < 7 || !strings.HasPrefix(jwtToken, "Bearer ")) {
			return "", ErrJWT
		}
	}

	if jwtToken == "" {
		cookie, err := r.Cookie(a.Cookie)
		if err != nil {
			return "", err
		}
		jwtToken = "Bearer " + cookie.Value
	}

	return jwtToken, nil
}

type UserClaims struct {
	jwt.StandardClaims
	ID uuid.UUID `json:"userID"`
}

var (
	ErrJWT = errors.New("jwt token")
	ErrId  = errors.New("no id in token")
)

func decAuthToken(jwtToken string, jwtSecretDecoded []byte) (*jwt.Token, *UserClaims, error) {
	token, err := jwt.ParseWithClaims(jwtToken, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecretDecoded, nil
	})
	if err != nil {
		return token, nil, err
	}

	var claims *UserClaims

	if token != nil {
		ok := false

		if claims, ok = token.Claims.(*UserClaims); ok && token.Valid {
			return token, claims, nil
		}
	}

	if err != nil {
		return token, claims, err
	}

	return token, claims, ErrJWT
}

func extractChiToken(ctx context.Context, jwtToken string, jwtSecretDecoded []byte) context.Context {
	trimToken := strings.TrimPrefix(jwtToken, "Bearer ")

	token, claims, err := decAuthToken(trimToken, jwtSecretDecoded)
	if err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).Msg("extracting chi token")
		return ctx
	}
	ctx = WithTokenAndClaims(ctx, token, claims)
	zerolog.Ctx(ctx).Debug().Str("token", token.Raw).Msg("extracting chi token")
	return ctx
}

func WithTokenAndClaims(ctx context.Context, token *jwt.Token, claims *UserClaims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

func AuthChi(a AuthKey, jwtSecretDecoded []byte) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			jwtToken, err := getJWT(r, a)
			if err != nil {
				zerolog.Ctx(ctx).Warn().Err(err).Msg("get jwt token")
			}
			nctx := extractChiToken(ctx, jwtToken, jwtSecretDecoded)
			next.ServeHTTP(w, r.WithContext(nctx))
		}
		return http.HandlerFunc(fn)
	}
}

func UserFromContext(ctx context.Context) (*UserClaims, bool) {
	user, ok := ctx.Value(claimsKey).(*UserClaims)
	return user, ok
}
