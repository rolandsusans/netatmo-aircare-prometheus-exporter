package main

import (
	"log"
	"time"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/kasko/netatmo-exporter/netatmo"
)

const (
	staleDataThreshold = 30 * time.Minute
)

var (
	netatmoUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "netatmo_up",
		Help: "Zero if there was an error scraping the Netatmo API.",
	})

	varLabels = []string{
		"module",
		"station",
	}

	prefix = "netatmo_aircare_"

	updatedDesc = prometheus.NewDesc(
		prefix+"updated",
		"Timestamp of last update",
		varLabels,
		nil)

	tempDesc = prometheus.NewDesc(
		prefix+"temperature_celsius",
		"Temperature measurement in celsius",
		varLabels,
		nil)

	humidityDesc = prometheus.NewDesc(
		prefix+"humidity_percent",
		"Relative humidity measurement in percent",
		varLabels,
		nil)

	cotwoDesc = prometheus.NewDesc(
		prefix+"co2_ppm",
		"Carbondioxide measurement in parts per million",
		varLabels,
		nil)

	noiseDesc = prometheus.NewDesc(
		prefix+"noise_db",
		"Noise measurement in decibels",
		varLabels,
		nil)

	pressureDesc = prometheus.NewDesc(
		prefix+"pressure_mb",
		"Atmospheric pressure measurement in millibar",
		varLabels,
		nil)
	wifiDesc = prometheus.NewDesc(
		prefix+"wifi_signal_strength",
		"Wifi signal strength (86: bad, 71: avg, 56: good)",
		varLabels,
		nil)
	rfDesc = prometheus.NewDesc(
		prefix+"rf_signal_strength",
		"RF signal strength (90: lowest, 60: highest)",
		varLabels,
		nil)
	absolutePressureDesc = prometheus.NewDesc(
		prefix+"absolute_pressure",
		"Absolute pressure",
		varLabels,
		nil)
    lastMeasureUtcDesc = prometheus.NewDesc(
        prefix+"last_measure_utc",
        "Measurement time UTC",
        varLabels,
        nil)
    healthIndexDesc = prometheus.NewDesc(
        prefix+"health_index",
        "Health index: 0 = Healthy,1 = Fine,2 = Fair,3 = Poor,4 = Unhealthy",
        varLabels,
        nil)

)

type netatmoCollector struct {
	client *netatmo.Client
}

func (m *netatmoCollector) Describe(dChan chan<- *prometheus.Desc) {
	dChan <- updatedDesc
	dChan <- tempDesc
	dChan <- humidityDesc
	dChan <- cotwoDesc
	dChan <- absolutePressureDesc
	dChan <- lastMeasureUtcDesc
	dChan <- healthIndexDesc
}

func (m *netatmoCollector) Collect(mChan chan<- prometheus.Metric) {
	devices, err := m.client.Read()
	if err != nil {
		netatmoUp.Set(0)
		mChan <- netatmoUp
		return
	}
	netatmoUp.Set(1)
	mChan <- netatmoUp

	for _, dev := range devices.Devices() {
		stationName := dev.StationName
		collectData(mChan, dev, stationName)

		for _, module := range dev.LinkedModules {
			collectData(mChan, module, stationName)
		}
	}
}

func collectData(ch chan<- prometheus.Metric, device *netatmo.Device, stationName string) {
	moduleName := device.ModuleName
	data := device.DashboardData

	if data.LastMeasure == nil {
		return
	}

	date := time.Unix(*data.LastMeasure, 0)
	if time.Since(date) > staleDataThreshold {
		return
	}

	sendMetric(ch, updatedDesc, prometheus.CounterValue, float64(date.UTC().Unix()), moduleName, stationName)

	if data.Temperature != nil {
		sendMetric(ch, tempDesc, prometheus.GaugeValue, float64(*data.Temperature), moduleName, stationName)
	}

	if data.Humidity != nil {
		sendMetric(ch, humidityDesc, prometheus.GaugeValue, float64(*data.Humidity), moduleName, stationName)
	}

	if data.CO2 != nil {
		sendMetric(ch, cotwoDesc, prometheus.GaugeValue, float64(*data.CO2), moduleName, stationName)
	}

	if data.Noise != nil {
		sendMetric(ch, noiseDesc, prometheus.GaugeValue, float64(*data.Noise), moduleName, stationName)
	}

	if data.Pressure != nil {
		sendMetric(ch, pressureDesc, prometheus.GaugeValue, float64(*data.Pressure), moduleName, stationName)
	}

    if data.AbsolutePressure != nil {
        sendMetric(ch, absolutePressureDesc, prometheus.GaugeValue, float64(*data.AbsolutePressure), moduleName, stationName)
    }

	if data.HealthIdx != nil {
		sendMetric(ch, healthIndexDesc, prometheus.GaugeValue, float64(*data.HealthIdx), moduleName, stationName)
	}

    if data.LastMeasure != nil {
        sendMetric(ch, lastMeasureUtcDesc, prometheus.GaugeValue, float64(*data.LastMeasure), moduleName, stationName)
    }

	if device.WifiStatus != nil {
		sendMetric(ch, wifiDesc, prometheus.GaugeValue, float64(*device.WifiStatus), moduleName, stationName)
	}
	if device.RFStatus != nil {
		sendMetric(ch, rfDesc, prometheus.GaugeValue, float64(*device.RFStatus), moduleName, stationName)
	}
}

func sendMetric(ch chan<- prometheus.Metric, desc *prometheus.Desc, valueType prometheus.ValueType, value float64, moduleName string, stationName string) {
	m, err := prometheus.NewConstMetric(desc, valueType, value, moduleName, stationName)
	if err != nil {
		log.Printf("Error creating %s metric: %s", updatedDesc.String(), err)
	}
	ch <- m
}
