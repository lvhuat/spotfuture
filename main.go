package main

import (
	"bytes"
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/lvhuat/textformatter"
	"github.com/sirupsen/logrus"
)

var (
	log = logrus.WithFields(logrus.Fields{})
)

var cfgFile = flag.String("cfg", "config.json", "基本配置文件")
var testMode = flag.Bool("test", false, "仅打印不会下单，不会执行网格")
var mf = flag.Bool("mf", false, "仅监控保证金率")

type WaitGroupExecutor struct {
	mutex sync.Mutex
	sync.WaitGroup
	errs []error
}

func (wg *WaitGroupExecutor) Run(fn func() error) {
	go func() {
		defer wg.Done()
		err := fn()
		if err == nil {
			return
		}
		wg.mutex.Lock()
		defer wg.mutex.Unlock()
		wg.errs = append(wg.errs, err)
	}()
}

func (wg *WaitGroupExecutor) Err() error {
	if len(wg.errs) == 0 {
		return nil
	}

	return fmt.Errorf("%v", wg.errs)
}

type Record struct {
	Spot    string
	Futures struct {
		Future string

		Max     float64
		MaxTime float64

		Min     float64
		MinTime float64
	}
}

func main() {
	logrus.SetFormatter(&textformatter.TextFormatter{})

	flag.Parse()

	if *cfgFile != "" {
		loadBaseConfigAndAssign(*cfgFile)
	}

	buffer := bytes.NewBuffer(nil)
	var lastReport time.Time
	for {
		time.Sleep(time.Second)
		wg := WaitGroupExecutor{}
		for _, m := range cfg.CheckMarkets {
			var spot *MarketItem
			var futures []*MarketItem
			wg.Add(len(m.Futures) + 1)
			wg.Run(func() error {
				resp, err := client.getMarket(m.Spot)
				if err != nil {
					return err
				}
				spot = resp
				return nil
			})

			for _, fut := range m.Futures {
				wg.Run(func() error {
					resp, err := client.getMarket(fut)
					if err != nil {
						return err
					}
					futures = append(futures, resp)
					return nil
				})
			}
			wg.Wait()

			if err := wg.Err(); err != nil {
				logrus.WithError(err).Errorln("GetMarketFailed")
				continue
			}

			for _, future := range futures {
				open := (future.Bid - spot.Ask) / spot.Ask
				close := (future.Ask - spot.Bid) / spot.Bid
				fmt.Fprintln(buffer, future.Name, "open", open, "close", close)
			}
		}
		fmt.Printf(string(buffer.Bytes()))

		now := time.Now()
		if now.Hour() != lastReport.Hour() {
			lastReport = now
			SendDingtalkText(cfg.Ding, "告警\n"+string(buffer.Bytes()))
		}
	}
}
