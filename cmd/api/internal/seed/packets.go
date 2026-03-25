package seed

import (
	"log/slog"

	"gorm.io/gorm"
	"toggly.com/m/cmd/api/internal/domain"
)

var defaultPackets = []domain.Packet{
	{
		Name:        "Free",
		Description: "Get started with the essentials — no credit card required.",
		IsPopular:   false,
		Tier:        domain.BillingTierFree,
		Amount:      0,
		Currency:    "usd",
		Interval:    domain.BillingIntervalFree,
		TrialDays:   0,
		Limits: domain.BillingLimits{
			MaxProjects:        1,
			MaxEnvironments:    2,
			MaxFlags:           3,
			MaxAPIKeys:         2,
			MaxTeamMembers:     1,
			MonthlyEvaluations: 10000,
		},
	},
	{
		Name:        "Pro",
		Description: "Everything you need to scale your feature flagging.",
		IsPopular:   true,
		Tier:        domain.BillingTierPro,
		Amount:      2900,
		Currency:    "usd",
		Interval:    domain.BillingIntervalFree, // month
		TrialDays:   14,
		Limits: domain.BillingLimits{
			MaxProjects:        20,
			MaxEnvironments:    10,
			MaxFlags:           200,
			MaxAPIKeys:         20,
			MaxTeamMembers:     10,
			MonthlyEvaluations: 1000000,
		},
	},
}

// Packets inserts the default packets if the table is empty.
func Packets(db *gorm.DB, log *slog.Logger) {
	var count int64
	if err := db.Model(&domain.Packet{}).Count(&count).Error; err != nil {
		log.Error("seed: count packets", "err", err)
		return
	}
	if count > 0 {
		log.Info("seed: packets already seeded, skipping")
		return
	}

	for i := range defaultPackets {
		if err := db.Create(&defaultPackets[i]).Error; err != nil {
			log.Error("seed: insert packet", "name", defaultPackets[i].Name, "err", err)
			return
		}
		log.Info("seed: inserted packet", "name", defaultPackets[i].Name)
	}
}
