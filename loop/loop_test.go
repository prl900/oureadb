package loop

import (
	"testing"

	"github.com/golang/geo/s2"
	"github.com/stretchr/testify/require"
)

func TestBadLoop(t *testing.T) {
	const failingLoop = `{features:[{geometry:{type:"Polygon",coordinates:[[[2.4001533999884828,48.846457859383435],[2.4002499620546587,48.84650728362274],[2.401886152667487,48.8453952264004],[2.4018003197157896,48.84533874029373],[2.4001533999884828,48.846457859383435]]]},type:"Feature",properties:{id:"42"},id:"42"}],type:"FeatureCollection"}`

	s2pts := make([]s2.Point, 4)
	s2pts[0] = s2.PointFromLatLng(s2.LatLngFromDegrees(48.846457859383435, 2.4001533999884828))
	s2pts[1] = s2.PointFromLatLng(s2.LatLngFromDegrees(48.84650728362274, 2.4002499620546587))
	s2pts[2] = s2.PointFromLatLng(s2.LatLngFromDegrees(48.8453952264004, 2.401886152667487))
	s2pts[3] = s2.PointFromLatLng(s2.LatLngFromDegrees(48.84533874029373, 2.4018003197157896))

	if s2.RobustSign(s2pts[0], s2pts[1], s2pts[2]) == s2.Clockwise {
		t.Log("NOT CCW reversing")
		// reverse the slice
		for i := len(s2pts)/2 - 1; i >= 0; i-- {
			opp := len(s2pts) - 1 - i
			s2pts[i], s2pts[opp] = s2pts[opp], s2pts[i]
		}
	}

	for _, p := range s2pts {
		t.Log(s2.LatLngFromPoint(p).Lat.Degrees(), s2.LatLngFromPoint(p).Lng.Degrees())
	}
	l := LoopFenceFromPoints(s2pts)
	require.False(t, l.IsEmpty() || l.IsFull())
	rc := &s2.RegionCoverer{MinLevel: 18, MaxLevel: 18}
	covering := rc.Covering(l)
	t.Log(covering)
	require.Len(t, covering, 11)
}

func Test3PLoop(t *testing.T) {
	s2pts := make([]s2.Point, 3)
	s2pts[0] = s2.PointFromLatLng(s2.LatLngFromDegrees(53.51249996891279, -113.53350000023157))
	s2pts[1] = s2.PointFromLatLng(s2.LatLngFromDegrees(53.56133333693556, -113.5860000459106))
	s2pts[2] = s2.PointFromLatLng(s2.LatLngFromDegrees(53.5543333277993, -113.4728333483211))

	l := LoopFenceFromPoints(s2pts)
	t.Log(l)
	rc := &s2.RegionCoverer{MinLevel: 1, MaxLevel: 30, MaxCells: 32}
	if s2.RobustSign(s2pts[0], s2pts[1], s2pts[2]) == s2.Clockwise {
		t.Log("NOT CCW")
		s2pts[0], s2pts[1] = s2pts[1], s2pts[0]
	}
	l = LoopFenceFromPoints(s2pts)
	t.Log(l)
	covering := rc.Covering(l)
	t.Log(covering)
}

func Test4PLoop(t *testing.T) {
	//[47.61649997100236 -122.20166669972009] [47.63100000228975 -122.18449997736279] [47.64100002912518 -122.1180000087143] [47.629666669247634 -122.03883334651307]
	s2pts := make([]s2.Point, 4)
	s2pts[0] = s2.PointFromLatLng(s2.LatLngFromDegrees(47.61649997100236, -122.20166669972009))
	s2pts[1] = s2.PointFromLatLng(s2.LatLngFromDegrees(47.63100000228975, -122.18449997736279))
	s2pts[2] = s2.PointFromLatLng(s2.LatLngFromDegrees(47.64100002912518, -122.1180000087143))
	s2pts[3] = s2.PointFromLatLng(s2.LatLngFromDegrees(47.629666669247634, -122.03883334651307))
	l := LoopFenceFromPoints(s2pts)
	t.Log(l)
	rc := &s2.RegionCoverer{MinLevel: 1, MaxLevel: 30, MaxCells: 32}
	if s2.RobustSign(s2pts[0], s2pts[1], s2pts[2]) == s2.Clockwise {
		t.Log("NOT CCW")
		s2pts[0], s2pts[1] = s2pts[1], s2pts[0]
	}
	l = LoopFenceFromPoints(s2pts)
	t.Log(l)
	covering := rc.Covering(l)
	t.Log(covering)
}
