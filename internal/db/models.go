// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package db

import (
	"github.com/jackc/pgx/v5/pgtype"
)

type Analytic struct {
	ID            int32
	ShortCode     string
	GeoData       []byte
	UserAgentData []byte
	ReferrerUrl   pgtype.Text
	RecordedAt    pgtype.Timestamptz
	CreatedAt     pgtype.Timestamp
	UpdatedAt     pgtype.Timestamp
}

type Link struct {
	ID             int32
	UserID         int32
	ShortCode      string
	DestinationUrl string
	Title          pgtype.Text
	Notes          pgtype.Text
	CreatedAt      pgtype.Timestamp
	UpdatedAt      pgtype.Timestamp
}

type Subscription struct {
	ID                  int32
	Name                string
	MaxLinksPerMonth    int32
	CanCustomisePath    bool
	CanCreateDuplicates bool
	CanViewAnalytics    bool
	CreatedAt           pgtype.Timestamp
	UpdatedAt           pgtype.Timestamp
}

type User struct {
	ID            int32
	Name          pgtype.Text
	Email         string
	OauthProvider pgtype.Text
	CreatedAt     pgtype.Timestamp
	UpdatedAt     pgtype.Timestamp
}

type UserMonthlyUsage struct {
	ID             int32
	UserID         int32
	LinksCreated   int32
	CycleStartDate pgtype.Date
	CycleEndDate   pgtype.Date
	CreatedAt      pgtype.Timestamp
	UpdatedAt      pgtype.Timestamp
}

type UserSubscription struct {
	UserID         int32
	SubscriptionID int32
	Status         string
	StartDate      pgtype.Timestamp
	EndDate        pgtype.Timestamp
	CreatedAt      pgtype.Timestamp
}
