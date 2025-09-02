package promutil

import (
	"context"

	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	Namespace   = "qa"
	Subsystem   = "alilo"
	RunningName = "load_testing_running"
	RunningHelp = "The metric shows that the test was run."
)

// gaugeStorage Конкурентно-способная мапа, на случай если запуск и остановка будут в разных рутинах
// RLock\RUnlock при чтении
// Lock\Unlock при записи
//var gaugeStorage = &gaugeMapSafe{m: make(map[int32]*prometheus.GaugeVec)}
//
//type gaugeMapSafe struct {
//	mu sync.RWMutex
//	m  map[int32]*prometheus.GaugeVec
//}
//
//func (gm *gaugeMapSafe) Append(runID int32, gauge *prometheus.GaugeVec) {
//	gm.mu.Lock()
//	defer gm.mu.Unlock()
//
//	gaugeStorage.m[runID] = gauge
//	gm.m[runID] = gauge
//}
//
//func (gm *gaugeMapSafe) GetGauge(runID int32) (*prometheus.GaugeVec, bool) {
//	gm.mu.RLock()
//	defer gm.mu.RUnlock()
//	vec, ok := gm.m[runID]
//
//	return vec, ok
//}

//func (gm *gaugeMapSafe) deleteGauge(runID int32) {
//	gm.mu.RLock()
//	defer gm.mu.RUnlock()
//
//	delete(gm.m, runID)
//}

// GetGaugeAnnotationOld 1 - GaugeAnnotation 6 labels: "load_testing"(title), "runID", "linc", "user", "type"
//func GetGaugeAnnotationOld(ctx context.Context, runID int32, title string) (gauge *prometheus.GaugeVec, status bool) {
//	defer func() {
//		if err := recover(); err != nil {
//			logger.Errorf(ctx, "GetGaugeAnnotation failed: '%+v'", err)
//			status = false
//		}
//	}()
//	// gaugeStorage.RLock()
//	// if g, ok := gaugeStorage.m[runID]; !ok {
//	if g, ok := gaugeStorage.GetGauge(runID); !ok {
//		logger.Infof(ctx, "Creating a new Gauge %v: %v", runID, title) // qa_alilo_load_testing_running
//
//		gauge = promauto.With(prometheus.DefaultRegisterer).NewGaugeVec(
//			prometheus.GaugeOpts{
//				Namespace: Namespace,
//				Subsystem: Subsystem,
//				Name:      RunningName,
//				Help:      RunningHelp,
//			},
//			[]string{
//				"load_testing",
//				"runID",
//				"linc",
//				"user",
//				"type",
//			})
//
//		logger.Infof(ctx, "New Gauge %v: %v registered", runID, title) // qa_alilo_load_testing_running
//
//		gaugeStorage.Append(runID, gauge)
//	} else {
//		gauge = g
//
//		//gaugeStorage.deleteGauge(runID)
//	}
//
//	logger.Infof(ctx, "Returning the gauge '%v': '%+v' ", runID, gauge)
//
//	return gauge, true
//}

var gaugeAnnotation *prometheus.GaugeVec

// GetGaugeAnnotation 1 - GaugeAnnotation 6 labels: "load_testing"(title), "runID", "linc", "user", "type"
func GetGaugeAnnotation(ctx context.Context) (*prometheus.GaugeVec, bool) {
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf(ctx, "GetGaugeAnnotation failed: '%+v'", err)
		}
	}()

	if gaugeAnnotation == nil {
		gaugeAnnotation = promauto.With(prometheus.DefaultRegisterer).NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: Namespace,
				Subsystem: Subsystem,
				Name:      RunningName,
				Help:      RunningHelp,
			},
			[]string{
				"load_testing",
				"runID",
				"linc",
				"user",
				"type",
			})
	}
	logger.Infof(ctx, "Return the gauge: '%+v' ", gaugeAnnotation)

	return gaugeAnnotation, true
}
