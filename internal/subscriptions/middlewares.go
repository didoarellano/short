package subscriptions

import (
	"context"
	"net/http"

	"github.com/didoarellano/short/internal/auth"
)

type key string

const SubscriptionKey key = "subscription"

type UserSubscriptionContext struct {
	Subscription Subscription
	LinksCreated int32
}

func (us *UserSubscriptionService) UserSubscriptionMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, _ := us.sessionStore.Get(r, "session")
			user := session.Values["user"].(auth.UserSession)
			userID := user.UserID
			subscription, _ := us.GetSubscriptionForUser(userID)
			linksCreated, _ := us.GetCurrentUsageForUser(userID)
			ctx := context.WithValue(r.Context(), SubscriptionKey, UserSubscriptionContext{
				Subscription: subscription,
				LinksCreated: linksCreated,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
