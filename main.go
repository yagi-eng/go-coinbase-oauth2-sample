package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gitlab.com/hmlkao/coinbase-go"
	"gitlab.com/hmlkao/coinbase-go/client"
	"golang.org/x/oauth2"
)

const port = 8080
const state = "hogefoo"

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	e := echo.New()
	e.Use(middleware.Logger())
	router(e)
	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", port)))
}

func router(e *echo.Echo) {
	e.GET("", apiKeyVer)
	e.GET("/login", login)
	e.GET("/callback", callback)
}

func newConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("COINBASE_CLIENT_ID"),
		ClientSecret: os.Getenv("COINBASE_CLIENT_SECRET"),
		Scopes:       []string{"wallet:accounts:read"},
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://www.coinbase.com/oauth/authorize",
			TokenURL:  "https://api.coinbase.com/oauth/token",
			AuthStyle: oauth2.AuthStyleAutoDetect,
		},
		RedirectURL: "http://localhost:8080/callback",
	}
}

func login(c echo.Context) error {
	conf := newConfig()
	url := conf.AuthCodeURL(state, oauth2.SetAuthURLParam("account", "all"))
	return c.Redirect(http.StatusMovedPermanently, url)
}

func callback(c echo.Context) error {
	s := c.QueryParam("state")
	if s != state {
		return errors.New("invalid access")
	}

	httpClient, err := createHTTPClient(c.QueryParam("code"))
	if err != nil {
		return err
	}

	config := coinbase.NewConfiguration()
	config.HTTPClient = httpClient
	client := coinbase.NewAPIClient(config)

	ctx := context.Background()
	accounts, res, err := client.AccountsApi.Accounts(ctx)
	if err != nil {
		resBody, _ := ioutil.ReadAll(res.Body)
		if string(resBody) != "" {
			log.Printf("response body: %s", string(resBody))
		}
		log.Fatalf("failed to get accounts: %s", err)
	}
	return c.JSON(http.StatusOK, accounts)

}

func createHTTPClient(code string) (*http.Client, error) {
	ctx := context.Background()
	conf := newConfig()
	cred, err := conf.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}
	return conf.Client(ctx, cred), nil
}

func apiKeyVer(c echo.Context) error {
	config := coinbase.NewConfiguration()
	httpClient := client.GetClient(os.Getenv("COINBASE_KEY"), os.Getenv("COINBASE_SECRET"))
	config.HTTPClient = &httpClient
	client := coinbase.NewAPIClient(config)

	ctx := context.Background()
	accounts, res, err := client.AccountsApi.Accounts(ctx)
	if err != nil {
		resBody, _ := ioutil.ReadAll(res.Body)
		if string(resBody) != "" {
			log.Printf("response body: %s", string(resBody))
		}
		log.Fatalf("failed to get accounts: %s", err)
	}
	return c.JSON(http.StatusOK, accounts)
}
