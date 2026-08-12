package domain

type SiteStatus string

const (
	SiteStatusActive      SiteStatus = "active"
	SiteStatusOffline     SiteStatus = "offline"
	SiteStatusMaintenance SiteStatus = "maintenance"
)

type Site struct {
	ID   string
	Name string
	Code string
}
