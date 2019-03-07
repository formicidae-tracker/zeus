package dieu

import "math"

type BoundedUnit interface {
	Value() float64
	MaxValue() float64
	MinValue() float64
}

func Clamp(u BoundedUnit) float64 {
	return math.Min(math.Max(u.Value(), u.MinValue()), u.MaxValue())
}

func IsUndefined(u BoundedUnit) bool {
	return math.IsInf(u.Value(), -1)
}

type Temperature float64

func (t Temperature) Value() float64    { return float64(t) }
func (t Temperature) MaxValue() float64 { return 40 }
func (t Temperature) MinValue() float64 { return 15 }

var UndefinedTemperature = Temperature(math.Inf(-1))

type Humidity float64

func (t Humidity) Value() float64    { return float64(t) }
func (t Humidity) MaxValue() float64 { return 85 }
func (t Humidity) MinValue() float64 { return 10 }

var UndefinedHumidity = Humidity(math.Inf(-1))

type Wind float64

func (t Wind) Value() float64    { return float64(t) }
func (t Wind) MaxValue() float64 { return 100 }
func (t Wind) MinValue() float64 { return 0 }

var UndefinedWind = Wind(math.Inf(-1))

type Light float64

func (t Light) Value() float64    { return float64(t) }
func (t Light) MaxValue() float64 { return 100 }
func (t Light) MinValue() float64 { return 0 }

var UndefinedLight = Light(math.Inf(-1))
