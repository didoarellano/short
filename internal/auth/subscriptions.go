package auth

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
)

type Subscription struct {
	Status              string
	Name                string
	MaxLinksPerMonth    int32
	CanCustomisePath    bool
	CanCreateDuplicates bool
}

func (ah *AuthHandler) GetSubscriptionForUser(userID int32) (Subscription, error) {
	ctx := context.Background()
	var subscription Subscription

	s, err := ah.redisClient.Get(ctx, fmt.Sprintf("user:%d:subscription", userID)).Result()
	if err != redis.Nil {
		err = json.Unmarshal([]byte(s), &subscription)
		if err != nil {
			return subscription, err
		}
	} else {
		sub, err := ah.queries.GetUserSubscription(ctx, userID)
		if err != nil {
			return subscription, err
		}

		subscription = Subscription(sub)
		b, err := json.Marshal(subscription)
		if err != nil {
			return subscription, err
		}

		ah.redisClient.Set(ctx, fmt.Sprintf("user:%d:subscription", userID), string(b), 0)
	}

	return subscription, nil
}
