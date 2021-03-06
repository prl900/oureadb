package geodata

import (
	"fmt"

	"github.com/akhenakh/oureadb/loop"
	"github.com/golang/geo/s2"
	"github.com/golang/protobuf/ptypes/struct"
	spb "github.com/golang/protobuf/ptypes/struct"
	"github.com/pkg/errors"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
)

// GeoToGeoData update gd with geo data gathered from g
func GeoToGeoData(g geom.T, gd *GeoData) error {
	geo := &Geometry{}

	switch g := g.(type) {
	case *geom.Point:
		geo.Coordinates = g.Coords()
		geo.Type = Geometry_POINT
		//case *geom.MultiPolygon:
		//	geo.Type = geodata.Geometry_MULTIPOLYGON
		//	// TODO implement multipolygon
		//
		//case *geom.Polygon:
		//	geo.Type = geodata.Geometry_POLYGON
		//	// TODO implement polygon
	default:
		return errors.Errorf("unsupported geo type %T", g)
	}

	gd.Geometry = geo
	return nil
}

// GeoJSONFeatureToGeoData fill gd with the GeoJSON data f
func GeoJSONFeatureToGeoData(f *geojson.Feature, gd *GeoData) error {
	err := PropertiesToGeoData(f, gd)
	if err != nil {
		return errors.Wrap(err, "while converting feature properties to GeoData")
	}

	err = GeoToGeoData(f.Geometry, gd)
	if err != nil {
		return errors.Wrap(err, "while converting feature to GeoData")
	}

	return nil
}

// PropertiesToGeoData update gd.Properties with the properties found in f
func PropertiesToGeoData(f *geojson.Feature, gd *GeoData) error {
	for k, vi := range f.Properties {
		switch tv := vi.(type) {
		case bool:
			gd.Properties[k] = &structpb.Value{Kind: &structpb.Value_BoolValue{BoolValue: tv}}
		case int:
			gd.Properties[k] = &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: float64(tv)}}
		case string:
			gd.Properties[k] = &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: tv}}
		case float64:
			gd.Properties[k] = &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: tv}}
		case nil:
			// pass
		default:
			return fmt.Errorf("GeoJSON property %s unsupported type %T", k, tv)
		}
	}
	return nil
}

// GeoDataToFlatCellUnion generate an s2 cover for GeoData gd
func GeoDataToFlatCellUnion(gd *GeoData, coverer *s2.RegionCoverer) (s2.CellUnion, error) {
	var cu s2.CellUnion
	switch gd.Geometry.Type {
	case Geometry_POINT:
		c := s2.CellIDFromLatLng(s2.LatLngFromDegrees(gd.Geometry.Coordinates[1], gd.Geometry.Coordinates[0]))
		cu = append(cu, c.Parent(coverer.MinLevel))
	case Geometry_POLYGON:
		if len(gd.Geometry.Coordinates) < 6 {
			return nil, errors.New("invalid polygons too few coordinates")
		}
		if len(gd.Geometry.Coordinates)%2 != 0 {
			return nil, errors.New("invalid polygons odd coordinates number")
		}
		l := loop.LoopFenceFromCoordinates(gd.Geometry.Coordinates)
		if l.IsEmpty() || l.IsFull() || l.ContainsOrigin() {
			return nil, errors.New("invalid polygons")
		}

		cu = coverer.Covering(l)
	case Geometry_MULTIPOLYGON:
		for _, g := range gd.Geometry.Geometries {
			if len(g.Coordinates) < 6 {
				return nil, errors.New("invalid polygons too few coordinates")
			}
			if len(g.Coordinates)%2 != 0 {
				return nil, errors.New("invalid polygons odd coordinates number")
			}
			l := loop.LoopFenceFromCoordinates(g.Coordinates)
			if l.IsEmpty() || l.IsFull() || l.ContainsOrigin() {
				return nil, errors.New("invalid polygons")
			}

			cu = append(cu, coverer.Covering(l.RectBound())...)
		}

	default:
		return nil, errors.New("unsupported data type")
	}

	return cu, nil
}

// ToGeoJSONFeatureCollection converts a GeoData to a GeoJSON Feature Collection
func ToGeoJSONFeatureCollection(geos []*GeoData) ([]byte, error) {
	fc := geojson.FeatureCollection{}
	for _, g := range geos {
		f := &geojson.Feature{}
		switch g.Geometry.Type {
		case Geometry_POINT:
			ng := geom.NewPointFlat(geom.XY, g.Geometry.Coordinates)
			f.Geometry = ng
		case Geometry_POLYGON:
			ng := geom.NewPolygonFlat(geom.XY, g.Geometry.Coordinates, []int{len(g.Geometry.Coordinates)})
			f.Geometry = ng
		case Geometry_MULTIPOLYGON:
			mp := geom.NewMultiPolygon(geom.XY)
			for _, poly := range g.Geometry.Geometries {
				ng := geom.NewPolygonFlat(geom.XY, poly.Coordinates, []int{len(poly.Coordinates)})
				mp.Push(ng)
			}
			f.Geometry = mp
		}
		f.Properties = PropertiesToJSONMap(g.Properties)
		fc.Features = append(fc.Features, f)
	}

	return fc.MarshalJSON()
}

// PointsToGeoJSONPolyLines converts a list of GeoDatato containing points to a polylines GeoJSON
func PointsToGeoJSONPolyLines(geos []*GeoData) ([]byte, error) {
	f := geojson.Feature{}
	var flatCoords []float64

	if len(geos) == 0 {
		return f.MarshalJSON()
	}

	for _, g := range geos {
		switch g.Geometry.Type {
		case Geometry_POINT:
			flatCoords = append(flatCoords, g.Geometry.Coordinates...)
		default:
			return nil, errors.Errorf("unsupported geometry")

		}

	}
	f.Properties = PropertiesToJSONMap(geos[0].Properties)
	g := geom.NewLineStringFlat(geom.XY, flatCoords)
	f.Geometry = g

	return f.MarshalJSON()
}

// PropertiesToJSONMap converts a protobuf map to it's JSON serializable map equivalent
func PropertiesToJSONMap(src map[string]*spb.Value) map[string]interface{} {
	res := make(map[string]interface{})

	for k, v := range src {
		switch x := v.Kind.(type) {
		case *spb.Value_NumberValue:
			res[k] = x.NumberValue
		case *spb.Value_StringValue:
			res[k] = x.StringValue
		case *spb.Value_BoolValue:
			res[k] = x.BoolValue
		}
	}
	return res
}
