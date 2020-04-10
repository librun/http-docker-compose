package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"

	goflag "github.com/jessevdk/go-flags"
	"github.com/labstack/echo"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type compose struct {
	Services map[string]struct{}
}

// ServiceName name of service.
const ServiceName = "Middleware docker-compose"

// ServiceVersion version
var ServiceVersion = "0.0.0" // nolint

func main() {
	var entry = logrus.NewEntry(&logrus.Logger{
		Formatter: &logrus.JSONFormatter{},
		Level:     logrus.InfoLevel,
		Out:       os.Stderr,
	}).WithField("service", ServiceName).
		WithField("version", ServiceVersion)

	var flagData struct {
		Config string `short:"c" long:"config" default:"docker-compose.yml" description:"Path to docker-compose file." required:"true"`
		Port   int    `short:"p" long:"port" default:"80" description:"Port for http service." required:"true"`
		Print  bool   `long:"print" description:"Print config file and exit."`
	}

	_, err := goflag.Parse(&flagData)
	if err != nil {
		entry.Errorf("parse flag error: %s", err)
		return
	}

	composeServices, err := getFile(flagData.Config)
	if err != nil {
		entry.Errorf("parse file error: %s", err)
		return
	}

	e := echo.New()

	e.PUT("/pull/:name", func(c echo.Context) (err error) {
		name := c.Param("name")
		if _, ok := composeServices.Services[name]; !ok {
			return getResult(c, 200, "", errors.New("No such service"))
		}

		data, err := runCommand("pull", name)
		if err != nil {
			return getResult(c, 500, string(data), err)
		}

		return getResult(c, 200, string(data), err)
	})

	e.PUT("/up/:name", func(c echo.Context) (err error) {
		name := c.Param("name")
		if _, ok := composeServices.Services[name]; !ok {
			return getResult(c, 200, "", errors.New("No such service"))
		}

		data, err := runCommand("up", "-d", name)

		if err != nil {
			return getResult(c, 500, string(data), err)
		}

		return getResult(c, 200, string(data), err)
	})

	e.PUT("/ps/:name", func(c echo.Context) (err error) {
		name := c.Param("name")
		if _, ok := composeServices.Services[name]; !ok {
			return getResult(c, 200, "", errors.New("No such service"))
		}

		data, err := runCommand("ps", name)
		if err != nil {
			return getResult(c, 500, string(data), err)
		}

		return getResult(c, 200, string(data), err)
	})

	e.PUT("/logs/:name", func(c echo.Context) (err error) {
		name := c.Param("name")
		if _, ok := composeServices.Services[name]; !ok {
			return getResult(c, 200, "", errors.New("No such service"))
		}

		data, err := runCommand("logs", name)
		if err != nil {
			return getResult(c, 500, string(data), err)
		}

		return getResult(c, 200, string(data), err)
	})

	entry.Fatal(e.Start(":" + strconv.Itoa(flagData.Port)))
}

func getFile(fileName string) (*compose, error) {

	configFile, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var composeFile compose

	if err := yaml.Unmarshal(configFile, &composeFile); err != nil {
		return nil, fmt.Errorf("yaml unmarshal error: %s", err)
	}

	return &composeFile, nil
}

func runCommand(args ...string) ([]byte, error) { // args ...string
	cmd := exec.Command("docker-compose", args...)
	cmd.Dir = "./"

	return cmd.Output()
}

func getResult(c echo.Context, code int, output string, err error) error {
	var r struct {
		Output string  `json:"output"`
		Error  *string `json:"error"`
	}

	r.Output = output
	if err != nil {
		errString := err.Error()
		r.Error = &errString
	}

	return c.JSON(code, r)
}
