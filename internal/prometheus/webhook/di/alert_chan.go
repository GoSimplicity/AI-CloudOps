package di

import "github.com/prometheus/alertmanager/template"

func CreateAlertChan() chan template.Alert {
	alerts := make(chan template.Alert, 1000)
	return alerts
}
