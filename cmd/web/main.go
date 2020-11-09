package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wvh/urn-harvester/internal/version"
	"github.com/wvh/urn-harvester/pkg/api"
)

const (
	httpPort = "8080"
	httpHost = "localhost"
)

var (
	appName = version.Id + "-" + "web"
	env     = "dev"
	_isDev  = func() bool { return false }()
)

func start(srv *http.Server) error {
	log.Println("server starting")

	idleConnsClosed := make(chan struct{})
	wasGraceful := true
	go func() {
		// sigterm is sent by the os, systemd, docker or kubernetes on shutdown;
		// sigint typically orignates from console ctrl-c
		sigShutdown := make(chan os.Signal, 1)
		signal.Notify(sigShutdown, syscall.SIGTERM, syscall.SIGINT)

		reason := <-sigShutdown
		signal.Stop(sigShutdown)
		close(sigShutdown)

		//atomic.StoreInt64(&c.healthy, 0)
		log.Println("server shutting down, reason:", reason)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			wasGraceful = false
			if err == context.DeadlineExceeded {
				log.Printf("forced shutdown: %v", err)
			} else {
				log.Printf("error during shutdown: %v", err)
			}
		}
		close(idleConnsClosed)
	}()

	log.Printf("serving http on port %s\n", httpPort)
	//atomic.StoreInt64(&c.healthy, time.Now().UnixNano())

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Println("ListenAndServe:", err)
		return err
		//panic("http: " + err.Error())
		// error opening or closing listener
		//log.Fatalf("HTTP server ListenAndServe: %v", err)
	}

	<-idleConnsClosed
	log.Println("server stopped", func(g bool) string {
		if g {
			return "gracefully"
		}
		return "forcefully"
	}(wasGraceful))
	return nil
}

var (
	// startup errors; annotate on use
	errStartup = errors.New("error")
	// runtime errors; annotate on use
	errFatal = errors.New("fatal")
)

func run(args []string, env string) error {
	// set logger as early as possible
	logger := makeLogger(os.Stdout, appName, env)
	logger.Log(
		"version", version.Version,
		"env", env,
		"state", "starting",
	)

	api, err := api.New()
	if err != nil {
		return fmt.Errorf("%w: %v", errStartup, err)
	}

	router := http.NewServeMux()
	router.HandleFunc("/", helloHandler)
	router.HandleFunc("/version", handleVersion())
	router.HandleFunc("/health", handleHealth())
	router.Handle("/api", api)

	srv := http.Server{
		Addr: ":" + httpPort,
		//Handler:        authoritiveHostOnly(httpHost, logMiddleware(sublogger(logger, "request"))(router)),
		Handler: logMiddleware(sublogger(logger, "request"))(router),
		//ErrorLog:       newStdlogAdapter(logger),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if err := start(&srv); err != nil {
		return fmt.Errorf("%w: %v", errFatal, err)
	}

	return nil
}

func main() {
	fmt.Println("args:", os.Args)

	if err := run(os.Args, os.Getenv("ENVIRONMENT")); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(func(err error) int {
			if errors.Is(err, errFatal) {
				return 2
			}
			return 1
		}(err))
	}
}
