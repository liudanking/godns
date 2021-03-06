package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"
)

const (
	PANIC_MAX = 5
	INTERVAL  = 5 //Minute
)

var (
	Configuration Settings
	optConf       = flag.String("c", "./config.json", "config file")
	optCommand    = flag.String("s", "", "send signal to a master process: stop, quit, reopen, reload")
	optHelp       = flag.Bool("h", false, "this help")
	panicCount    = 0
)

func usage() {
	log.Println("[command] -c=[config file path]")
	flag.PrintDefaults()
}
func main() {
	flag.Parse()
	if *optHelp {
		usage()
		return
	}

	Configuration = LoadSettings(*optConf)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGUSR2, syscall.SIGTERM)

	go dns_loop()

	for s := range sig {
		switch s {
		case syscall.SIGHUP, syscall.SIGUSR2:
			// do some custom jobs while reload/hotupdate
		case syscall.SIGTERM:
			// do some clean up and exit
			return
		}
	}
}

func dns_loop() {
	defer func() {
		if err := recover(); err != nil {
			panicCount++
			log.Printf("Recovered in %v: %s\n", err, string(debug.Stack()))
			if panicCount < PANIC_MAX {
				log.Println("Got panic in goroutine, will start a new one... :", panicCount)
				go dns_loop()
			}
		}
	}()

	for {

		domain_id := get_domain(Configuration.Domain)

		if domain_id == -1 {
			continue
		}

		currentIP, err := get_currentIP(Configuration.IP_Url)

		if err != nil {
			log.Println("get_currentIP:", err)
			continue
		}

		sub_domain_id, ip := get_subdomain(domain_id, Configuration.Sub_domain)

		if sub_domain_id == "" || ip == "" {
			log.Println("sub_domain:", sub_domain_id, ip)
			continue
		}

		log.Println("currentIp is:", currentIP)

		//Continue to check the IP of sub-domain
		if len(ip) > 0 && !strings.Contains(currentIP, ip) {
			log.Println("Start to update record IP...")
			update_ip(domain_id, sub_domain_id, Configuration.Sub_domain, currentIP)
		} else {
			log.Println("Current IP is same with domain IP, no need to update...")
		}

		//Interval is 5 minutes
		time.Sleep(time.Minute * INTERVAL)
	}

	log.Printf("Loop %d exited...\n", panicCount)
}
