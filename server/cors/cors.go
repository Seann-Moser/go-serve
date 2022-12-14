package cors

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Cors struct {
	AllowedOrigins     []*regexp.Regexp
	AllowedMethods     []string
	AllowedHeaders     []string
	AllowedCredentials bool
	logger             *zap.Logger
}

const (
	corsAllowedOrigins     = "cors-allowed-origins"
	corsAllowedMethods     = "cors-allowed-methods"
	corsAllowedHeaders     = "cors-allowed-headers"
	corsAllowedCredentials = "cors-allow-credentials"
)

func Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("cors", pflag.ExitOnError)
	fs.StringSlice(corsAllowedOrigins, []string{}, "")
	fs.StringSlice(corsAllowedMethods, []string{}, "")
	fs.StringSlice(corsAllowedHeaders, []string{}, "")
	fs.Bool(corsAllowedCredentials, false, "")
	return fs
}

func NewFromFlags(logger *zap.Logger) (*Cors, error) {
	c := &Cors{
		AllowedOrigins:     []*regexp.Regexp{},
		AllowedMethods:     viper.GetStringSlice(corsAllowedMethods),
		AllowedHeaders:     viper.GetStringSlice(corsAllowedHeaders),
		AllowedCredentials: viper.GetBool(corsAllowedCredentials),
		logger:             logger,
	}
	for _, o := range viper.GetStringSlice(corsAllowedOrigins) {
		exp, err := regexp.Compile(o)
		if err != nil {
			return nil, fmt.Errorf("failed compiling regex origin %s:%w", o, err)
		}
		c.AllowedOrigins = append(c.AllowedOrigins, exp)
	}
	return c, nil
}

func New(origin []string, methods, headers []string, creds bool, logger *zap.Logger) (*Cors, error) {
	c := &Cors{
		AllowedOrigins:     []*regexp.Regexp{},
		AllowedMethods:     methods,
		AllowedHeaders:     headers,
		AllowedCredentials: creds,
		logger:             logger,
	}
	for _, o := range origin {
		exp, err := regexp.Compile(o)
		if err != nil {
			return nil, fmt.Errorf("failed compiling regex origin %s:%w", o, err)
		}
		c.AllowedOrigins = append(c.AllowedOrigins, exp)
	}
	if len(c.AllowedOrigins) == 0 {
		exp, err := regexp.Compile(".*")
		if err != nil {
			return nil, fmt.Errorf("failed compiling regex origin %s:%w", ".*", err)
		}
		c.AllowedOrigins = append(c.AllowedOrigins, exp)
	}
	return c, nil
}

func (c *Cors) Cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin, err := c.matchOrigin(r)
		if err == nil {
			c.setHeaders(w, origin)
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		// Next
		next.ServeHTTP(w, r)
		return
	})
}

func (c *Cors) matchOrigin(r *http.Request) (string, error) {
	origin := getOrigin(r)
	for _, o := range c.AllowedOrigins {
		if o.MatchString(origin) {
			return origin, nil
		}
	}
	return "", fmt.Errorf("invalid origin %s", origin)
}

func getOrigin(r *http.Request) string {
	if v := r.Header.Get("Origin"); v != "" {
		return v
	}
	if v := r.Header.Get("Referer"); v != "" {
		return v
	}
	return ""
}

func (c *Cors) setHeaders(w http.ResponseWriter, origin string) {
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", c.getMethods())
	w.Header().Set("Access-Control-Allow-Headers", c.getHeaders())
	w.Header().Set("Access-Control-Allow-Credentials", strconv.FormatBool(c.AllowedCredentials))
}

func (c *Cors) getMethods() string {
	return getCorsData(c.AllowedMethods)
}

func (c *Cors) getHeaders() string {
	return getCorsData(c.AllowedHeaders)
}

func getCorsData(list []string) string {
	if list == nil {
		return "*"
	}
	if len(list) == 0 {
		return "*"
	}
	return strings.Join(list, ", ")
}
