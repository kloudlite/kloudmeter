package httpServer

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kloudlite/api/common"
	"github.com/kloudlite/kloudmeter/pkg/kv"
)

type HttpMiddleware func(handle http.HandlerFunc) http.HandlerFunc

func NewReadSessionMiddlewareHandler(repo kv.Repo[*common.AuthSession], cookieName string, sessionKeyPrefix string) HttpMiddleware {
	return func(handle http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			cookies := map[string]string{}

			for _, c := range r.Cookies() {
				cookies[c.Name] = c.Value
			}

			nctx := context.WithValue(r.Context(), "http-cookies", cookies)

			req := r.WithContext(nctx)

			cookieValue := cookies[cookieName]

			if cookieValue != "" {
				key := fmt.Sprintf("%s:%s", sessionKeyPrefix, cookieValue)
				sess, err := repo.Get(r.Context(), key)
				if err != nil {
					if !repo.ErrKeyNotFound(err) {
						http.Error(w, err.Error(), http.StatusUnauthorized)
						return
					}
				}

				if sess != nil {
					nctx = context.WithValue(nctx, "session", sess)
					req = req.WithContext(nctx)
				}
			}

			handle(w, req)
		}
	}
}
