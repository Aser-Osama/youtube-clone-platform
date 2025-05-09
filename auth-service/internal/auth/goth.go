package auth

import (
	"auth-service/internal/config"

	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/google"
)

func InitGoth(cfg *config.Config) {
	goth.UseProviders(
		google.New(
			cfg.GoogleClientID,
			cfg.GoogleClientSecret,
			cfg.GoogleCallbackURL,
			"email", "profile",
		),
	)
}
