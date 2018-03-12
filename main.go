package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

type MetricValue struct {
	Endpoint  string      `json:"endpoint"`
	Metric    string      `json:"metric"`
	Value     interface{} `json:"value"`
	Step      int64       `json:"step"`
	Type      string      `json:"counterType"`
	Tags      string      `json:"tags"`
	Timestamp int64       `json:"timestamp"`
}

const (
	Guage   string = "GUAGE"
	Counter string = "COUNTER"
	Step    int64  = 60

	Interval time.Duration = time.Second * 60

	Falcon_url string = "http://127.0.0.1:1988/v1/push"
)

var Hostname string = hostname()

func hostname() string {
	name, err := os.Hostname()
	if err == nil {
		return name
	}

	return ""
}

func now_ts() int64 {
	return time.Now().Unix()
}

func NewMetric(metric string, value interface{}) MetricValue {
	return MetricValue{
		Endpoint:  Hostname,
		Metric:    metric,
		Value:     value,
		Step:      Step,
		Type:      Guage,
		Tags:      "",
		Timestamp: now_ts(),
	}
}

func GetMemStats() []MetricValue {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	payload := []MetricValue{}
	payload = append(payload, NewMetric("Mem.Alloc", int64(mem.Alloc)))
	payload = append(payload, NewMetric("Mem.TotalAlloc", int64(mem.TotalAlloc)))
	payload = append(payload, NewMetric("Mem.Sys", int64(mem.Sys)))
	payload = append(payload, NewMetric("Mem.Lookups", int64(mem.Lookups)))
	payload = append(payload, NewMetric("Mem.Mallocs", int64(mem.Mallocs)))
	payload = append(payload, NewMetric("Mem.Frees", int64(mem.Frees)))
	payload = append(payload, NewMetric("Mem.HeapAlloc", int64(mem.HeapAlloc)))
	payload = append(payload, NewMetric("Mem.HeapSys", int64(mem.HeapSys)))
	payload = append(payload, NewMetric("Mem.HeapIdle", int64(mem.HeapIdle)))
	payload = append(payload, NewMetric("Mem.HeapInuse", int64(mem.HeapInuse)))
	payload = append(payload, NewMetric("Mem.HeapReleased", int64(mem.HeapReleased)))
	payload = append(payload, NewMetric("Mem.HeapObjects", int64(mem.HeapObjects)))
	payload = append(payload, NewMetric("Mem.StackInuse", int64(mem.StackInuse)))
	payload = append(payload, NewMetric("Mem.StackSys", int64(mem.StackSys)))
	payload = append(payload, NewMetric("Mem.MSpanInuse", int64(mem.MSpanInuse)))
	payload = append(payload, NewMetric("Mem.MSpanSys", int64(mem.MSpanSys)))
	payload = append(payload, NewMetric("Mem.MCacheInuse", int64(mem.MCacheInuse)))
	payload = append(payload, NewMetric("Mem.MCacheSys", int64(mem.MCacheSys)))
	payload = append(payload, NewMetric("Mem.BuckHashSys", int64(mem.BuckHashSys)))
	payload = append(payload, NewMetric("Mem.GCSys", int64(mem.GCSys)))
	payload = append(payload, NewMetric("Mem.OtherSys", int64(mem.OtherSys)))
	payload = append(payload, NewMetric("Mem.NextGC", int64(mem.NextGC)))
	payload = append(payload, NewMetric("Mem.LastGC", int64(mem.LastGC)))
	payload = append(payload, NewMetric("Mem.PauseTotalNs", int64(mem.PauseTotalNs)))
	payload = append(payload, NewMetric("Mem.NumGC", int64(mem.NumGC)))
	payload = append(payload, NewMetric("Mem.GCCPUFraction", mem.GCCPUFraction))

	return payload
}

func send2falcon(items []MetricValue) {

	if len(items) == 0 {
		log.Println("no items given")
		return
	}

	content, err := json.Marshal(items)
	if err != nil {
		log.Println("json Marshal faild:", items)
		return
	}

	data := bytes.NewBuffer(content)

	client := http.Client{
		Timeout: time.Second * 3,
	}
	resp, err := client.Post(Falcon_url, "application/json", data)

	if err != nil {
		log.Println("post data", items, "to falcon url", Falcon_url, "failed...")
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("read response body failed:", err)
		return
	}

	log.Println("send data to falcon_url:", Falcon_url, ", response is:", string(body[:]))
}

func process_signal(pid int) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	log.Println(pid, "register signal notify")

	ticker := time.NewTicker(Interval)

	for {

		select {

		case s := <-sigs:
			log.Println("recv", s)

			switch s {

			case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				log.Println("gracefull shut down")
				log.Println(pid, "exit")
				os.Exit(0)

			default:
				log.Println("couldn't process signal:", s)
			}

		case <-ticker.C:
			log.Println("begin collect mem stats and send to falcon")
			items := GetMemStats()
			send2falcon(items)
			log.Println("end   collect mem stats and send to falcon")
		}
	}
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func main() {
	process_signal(os.Getpid())
}
