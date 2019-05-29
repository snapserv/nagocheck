package modsystem

import (
	"fmt"
	"github.com/shirou/gopsutil/host"
	"github.com/snapserv/nagocheck/nagocheck"
	"github.com/snapserv/nagopher"
	"strings"
)

type temperaturePlugin struct {
	nagocheck.Plugin
}

type temperatureResource struct {
	nagocheck.Resource

	temperatures map[string]*temperatureStats
}

type temperatureStats struct {
	value      float64
	valueRange nagopher.OptionalBounds
}

type temperatureSummarizer struct {
	nagocheck.Summarizer
}

func newTemperaturePlugin() *temperaturePlugin {
	return &temperaturePlugin{
		Plugin: nagocheck.NewPlugin("temperature",
			nagocheck.PluginDescription("Temperature Sensors"),
		),
	}
}

func (p *temperaturePlugin) DefineCheck() nagopher.Check {
	check := nagopher.NewCheck("temperature", newTemperatureSummarizer(p))
	check.AttachResources(newTemperatureResource(p))
	check.AttachContexts(
		nagopher.NewScalarContext(
			"sensor",
			nagopher.OptionalBoundsPtr(p.WarningThreshold()),
			nagopher.OptionalBoundsPtr(p.CriticalThreshold()),
		),
	)

	return check
}

func newTemperatureResource(plugin *temperaturePlugin) *temperatureResource {
	return &temperatureResource{
		Resource:     nagocheck.NewResource(plugin),
		temperatures: make(map[string]*temperatureStats),
	}
}

func (r *temperatureResource) Probe(warnings nagopher.WarningCollection) (metrics []nagopher.Metric, _ error) {
	if err := r.Collect(); err != nil {
		return metrics, err
	}

	for temperatureName, temperature := range r.temperatures {
		metrics = append(metrics,
			nagopher.MustNewNumericMetric(
				temperatureName, temperature.value, "",
				nagopher.OptionalBoundsPtr(temperature.valueRange),
				"sensor",
			),
		)
	}

	return metrics, nil
}

func (r *temperatureResource) Collect() error {
	sensorTemperatures, err := host.SensorsTemperatures()
	if err != nil {
		return err
	}

	r.temperatures = make(map[string]*temperatureStats)
	for _, sensorTemperature := range sensorTemperatures {
		keyFields := strings.Split(sensorTemperature.SensorKey, "_")
		temperatureName := strings.Join(keyFields[:len(keyFields)-1], "_")
		fieldName := keyFields[len(keyFields)-1]

		temperature, ok := r.temperatures[temperatureName]
		if !ok {
			r.temperatures[temperatureName] = &temperatureStats{}
			temperature, ok = r.temperatures[temperatureName]
			if !ok {
				return fmt.Errorf("unable to instantiate temperature: %s", temperatureName)
			}
		}

		switch fieldName {
		case "input":
			temperature.value = sensorTemperature.Temperature
		case "max":
			currentRange := temperature.valueRange.OrElse(nagopher.NewBounds())

			temperature.valueRange = nagopher.NewOptionalBounds(nagopher.NewBounds(
				nagopher.LowerBound(currentRange.Lower().OrElse(0)),
				nagopher.UpperBound(sensorTemperature.Temperature),
			))
		}
	}

	return nil
}

func newTemperatureSummarizer(plugin *temperaturePlugin) *temperatureSummarizer {
	return &temperatureSummarizer{
		Summarizer: nagocheck.NewSummarizer(plugin),
	}
}

func (s *temperatureSummarizer) Ok(check nagopher.Check) string {
	resultCollection := check.Results()
	temperatureSum := float64(0)

	for _, result := range resultCollection.Get() {
		resultMetric, err := result.Metric().Get()
		if err != nil || resultMetric == nil {
			return s.Summarizer.Ok(check)
		}

		numericMetric, ok := resultMetric.(nagopher.NumericMetric)
		if !ok {
			return s.Summarizer.Ok(check)
		}

		temperatureSum += numericMetric.Value()
	}

	averageTemperature := nagocheck.Round(temperatureSum/float64(resultCollection.Count()), 2)
	return fmt.Sprintf("average temperature is %.2f°C", averageTemperature)
}
