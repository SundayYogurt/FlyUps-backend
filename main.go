// @title           FlyUps API
// @version         1.0
// @description     FlyUps crowdfunding platform API
// @host            api.flyupapi.dev
// @schemes         https http
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
package main

import (
	"flyup/config"
	"flyup/internal/api"
	_ "flyup/docs"
	"log"
)

func main() {

	cfg, err := config.SetupEnv()
	if err != nil {
		log.Printf("config file is not loaded properly: %v\n", err)
	}

	api.StartServer(cfg)
}
