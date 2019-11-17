package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"

	_ "net/http/pprof"

	"github.com/ginuerzh/gost"
	"github.com/ginuerzh/gost/utils"
	"github.com/go-log/log"
)

var (
	configureFile string
	baseCfg       = &baseConfig{}
)

func init() {
	gost.SetLogger(&gost.LogLogger{})

	var (
		printVersion bool
		fastOpen     bool
	)
	localHost := os.Getenv("SS_LOCAL_HOST")
	localPort := os.Getenv("SS_LOCAL_PORT")
	pluginOptions := os.Getenv("SS_PLUGIN_OPTIONS")
	pluginOptions = strings.ReplaceAll(pluginOptions, "#SS_HOST", os.Getenv("SS_REMOTE_HOST"))
	pluginOptions = strings.ReplaceAll(pluginOptions, "#SS_PORT", os.Getenv("SS_REMOTE_PORT"))

	//简单修复下=变成\=的问题
	pluginOptions = strings.ReplaceAll(pluginOptions, "\=", "=")

	os.Args = append(os.Args, "-L")
	os.Args = append(os.Args, fmt.Sprintf("ss+tcp://rc4-md5:gost@[%s]:%s", localHost, localPort))
	os.Args = append(os.Args, strings.Split(pluginOptions, " ")...)

	flag.Var(&baseCfg.route.ChainNodes, "F", "forward address, can make a forward chain")
	flag.Var(&baseCfg.route.ServeNodes, "L", "listen address, can listen on multiple ports")
	flag.StringVar(&configureFile, "C", "", "configure file")
	flag.BoolVar(&baseCfg.Debug, "D", false, "enable debug log")
	flag.BoolVar(&utils.VpnMode, "V", false, "VPN Mode")
	flag.BoolVar(&fastOpen, "fast-open", false, "fast Open TCP")
	flag.BoolVar(&printVersion, "PV", false, "print version")
	flag.Parse()

	if printVersion {
		fmt.Fprintf(os.Stderr, "gost %s (%s %s/%s)\n",
			gost.Version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	if localHost == "" || localPort == "" {
		fmt.Fprintln(os.Stderr, "Can only be used in the shadowsocks plugin.")
		os.Exit(1)
	}

	if configureFile != "" {
		_, err := parseBaseConfig(configureFile)
		if err != nil {
			log.Log(err)
			os.Exit(1)
		}
	}
	if flag.NFlag() == 0 {
		flag.PrintDefaults()
		os.Exit(0)
	}
}

func main() {
	if os.Getenv("PROFILING") != "" {
		go func() {
			log.Log(http.ListenAndServe("127.0.0.1:16060", nil))
		}()
	}

	// NOTE: as of 2.6, you can use custom cert/key files to initialize the default certificate.
	tlsConfig, err := tlsConfig(defaultCertFile, defaultKeyFile)
	if err != nil {
		// generate random self-signed certificate.
		cert, err := gost.GenCertificate()
		if err != nil {
			log.Log(err)
			os.Exit(1)
		}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}
	gost.DefaultTLSConfig = tlsConfig

	if err := start(); err != nil {
		log.Log(err)
		os.Exit(1)
	}

	select {}
}

func start() error {
	gost.Debug = baseCfg.Debug

	var routers []router
	rts, err := baseCfg.route.GenRouters()
	if err != nil {
		return err
	}
	routers = append(routers, rts...)

	for _, route := range baseCfg.Routes {
		rts, err := route.GenRouters()
		if err != nil {
			return err
		}
		routers = append(routers, rts...)
	}

	if len(routers) == 0 {
		return errors.New("invalid config")
	}
	for i := range routers {
		go routers[i].Serve()
	}

	return nil
}
