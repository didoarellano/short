package subscriptions

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/didoarellano/short/internal/db"
	"github.com/didoarellano/short/internal/session"
	"github.com/go-redis/redis/v8"
)

type UserSubscriptionService struct {
	queries      *db.Queries
	sessionStore session.SessionStore
	redisClient  *redis.Client
}

type Subscription struct {
	Status              string
	Name                string
	MaxLinksPerMonth    int32
	CanCustomiseSlug    bool
	CanCreateDuplicates bool
	CanViewAnalytics    bool
}

func NewUserSubscriptionService(q *db.Queries, s session.SessionStore, r *redis.Client) *UserSubscriptionService {
	return &UserSubscriptionService{
		queries:      q,
		redisClient:  r,
		sessionStore: s,
	}
}

func (us *UserSubscriptionService) GetSubscriptionForUser(userID int32) (Subscription, error) {
	ctx := context.Background()
	var subscription Subscription

	s, err := us.redisClient.Get(ctx, fmt.Sprintf("user:%d:subscription", userID)).Result()
	if err != redis.Nil {
		err = json.Unmarshal([]byte(s), &subscription)
		if err != nil {
			return subscription, err
		}
	} else {
		sub, err := us.queries.GetUserSubscription(ctx, userID)
		if err != nil {
			return subscription, err
		}

		subscription = Subscription(sub)
		b, err := json.Marshal(subscription)
		if err != nil {
			return subscription, err
		}

		us.redisClient.Set(ctx, fmt.Sprintf("user:%d:subscription", userID), string(b), 0)
	}

	return subscription, nil
}

func (us *UserSubscriptionService) GetCurrentUsageForUser(userID int32) (int32, error) {
	var links_created int32
	var e error

	key := fmt.Sprintf("user:%d:links_created", userID)
	ctx := context.Background()

	s, err := us.redisClient.Get(ctx, key).Result()

	if err != redis.Nil {
		i, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			e = err
		}
		links_created = int32(i)
	} else {
		links_created, err = us.queries.GetUserCurrentUsage(ctx, userID)
		if err != nil {
			e = err
		}
		us.redisClient.Set(ctx, key, links_created, 0)
	}

	return links_created, e
}

func (us *UserSubscriptionService) SetCachedCurrentUsageForUser(userID, value int32) {
	key := fmt.Sprintf("user:%d:links_created", userID)
	ctx := context.Background()
	us.redisClient.Set(ctx, key, value, 0)
}
