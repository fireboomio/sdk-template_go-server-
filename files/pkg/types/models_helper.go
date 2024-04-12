package types

import (
	"fmt"
	"github.com/spf13/cast"
	"io"
	"math"
	"strings"
)

type UploadFile struct {
	Reader io.Reader
	Name   string
}

const (
	geometryStringPrefix = "SRID=4326;POINT("
	geometryStringSuffix = ")"
)

type Geometry struct {
	Longitude float64
	Latitude  float64
}

func (g *Geometry) DistanceTo(target *Geometry) float64 {
	if g == nil || target == nil {
		return 0
	}

	p := math.Abs(g.Longitude - target.Longitude)
	q := math.Abs(g.Latitude - target.Latitude)
	return math.Hypot(p, q)
}

func (g Geometry) MarshalJSON() ([]byte, error) {
	return []byte(g.String()), nil
}

func (g *Geometry) UnmarshalJSON(dataBytes []byte) error {
	if len(dataBytes) == 0 {
		return nil
	}
	dataString := strings.TrimSuffix(string(dataBytes), geometryStringSuffix)
	dataString, ok := strings.CutPrefix(dataString, geometryStringPrefix)
	if !ok {
		return fmt.Errorf("not supported geometry data [%s]", string(dataBytes))
	}
	dataArray := strings.Split(dataString, " ")
	if dataLength := len(dataArray); dataLength != 2 {
		return fmt.Errorf("expected 2 elements but found [%d]", dataLength)
	}

	*g = Geometry{Longitude: cast.ToFloat64(dataArray[0]), Latitude: cast.ToFloat64(dataArray[1])}
	return nil
}

func (g *Geometry) String() string {
	return fmt.Sprintf("%s%f %f%s", geometryStringPrefix, g.Longitude, g.Latitude, geometryStringSuffix)
}
